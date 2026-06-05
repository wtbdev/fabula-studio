// Package server implements the HTTP API for novel-to-screenplay conversion.
package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/fabula-studio/backend/internal/config"
	"github.com/fabula-studio/backend/internal/observability"
	"github.com/fabula-studio/backend/internal/pipeline"
	"github.com/fabula-studio/backend/internal/schema"
)

// Server holds dependencies and registered handlers.
type Server struct {
	config    config.Config
	pipeline  *pipeline.Pipeline
	eventBus  *observability.EventBus
	telemetry *observability.Telemetry
	agui      *AGUIServer
	http      *http.Server
}

// New creates and returns a configured Server.
func New(cfg config.Config) *Server {
	// Initialize event bus
	eventBus := observability.NewEventBus()

	// Initialize telemetry
	ctx := context.Background()
	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:  "fabula-studio",
		OTLPEndpoint: cfg.OTLPEndpoint,
	})
	if err != nil {
		log.Printf("Warning: Failed to initialize telemetry: %v", err)
	}
	p := pipeline.New(pipeline.DefaultConfig(), cfg.ModelName, cfg.APIKey, cfg.BaseURL, eventBus)

	// Initialize AG-UI server
	agui := NewAGUIServer(cfg.ModelName, cfg.APIKey, cfg.BaseURL, eventBus)

	srv := &Server{
		config:    cfg,
		pipeline:  p,
		eventBus:  eventBus,
		telemetry: telemetry,
		agui:      agui,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/convert", srv.handleConvert)
	mux.HandleFunc("/api/health", srv.handleHealth)
	mux.HandleFunc("/api/events", eventBus.EventHandler())
	mux.HandleFunc("/api/events/stream", eventBus.SSEHandler())
	mux.HandleFunc("/api/trace", srv.handleTraceInfo)

	// Register AG-UI routes
	agui.Routes(mux)

	srv.http = &http.Server{
		Addr:         cfg.Addr,
		Handler:      withCORS(withLogging(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 600 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv
}

// Start begins listening. Blocks until the server stops.
func (s *Server) Start() error {
	log.Printf("fabula-studio backend starting on %s (model: %s)", s.config.Addr, s.config.ModelName)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/convert         - Convert novel chapters to screenplay")
	log.Printf("  GET  /api/health          - Health check")
	log.Printf("  GET  /api/events          - Get recent events")
	log.Printf("  GET  /api/events/stream   - SSE stream of real-time events")
	log.Printf("  GET  /api/trace           - Trace information")
	log.Printf("")
	log.Printf("AG-UI Protocol:")
	log.Printf("  GET  /api/sessions        - List sessions")
	log.Printf("  POST /api/sessions        - Create session")
	log.Printf("  POST /api/chat            - Non-streaming chat")
	log.Printf("  POST /api/chat/stream     - SSE streaming chat")
	log.Printf("")
	log.Printf("Observability:")
	log.Printf("  Jaeger UI: http://localhost:16686")
	log.Printf("  Grafana:   http://localhost:3000")
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.telemetry != nil {
		s.telemetry.Shutdown()
	}
	return s.http.Shutdown(ctx)
}

// ---- handlers ----

// handleConvert accepts a JSON body with novel chapters and returns a screenplay.
func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持 POST 方法"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "读取请求体失败: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req schema.ConversionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON 解析失败: " + err.Error()})
		return
	}

	if len(req.Chapters) < 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "至少需要3个章节，当前仅提供 " + itoa(len(req.Chapters)) + " 个",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 600*time.Second)
	defer cancel()

	scr, err := s.pipeline.Convert(ctx, req.Title, req.Author, req.Chapters)
	if err != nil {
		log.Printf("conversion failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, schema.ConversionResponse{
			Error: err.Error(),
		})
		return
	}

	yamlBytes, _ := yaml.Marshal(scr)
	resp := schema.ConversionResponse{
		Screenplay: scr,
		YAML:       string(yamlBytes),
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleHealth returns a simple health check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleTraceInfo returns trace information.
func (s *Server) handleTraceInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jaeger_ui": "http://localhost:16686",
		"grafana":   "http://localhost:3000",
		"otlp_endpoint": "localhost:4317",
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ---- middleware ----

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
