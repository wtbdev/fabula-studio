// Package server implements the AG-UI protocol for the fabula-studio backend.
// AG-UI (Agent-UI) is a protocol for streaming agent interactions to web clients.
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"

	"github.com/fabula-studio/backend/internal/observability"
)

// AGUIEventType represents the type of AG-UI event.
type AGUIEventType string

const (
	AGUIEventTypeRunStart         AGUIEventType = "RUN_START"
	AGUIEventTypeRunFinished      AGUIEventType = "RUN_FINISHED"
	AGUIEventTypeRunError         AGUIEventType = "RUN_ERROR"
	AGUIEventTypeTextMessageStart AGUIEventType = "TEXT_MESSAGE_START"
	AGUIEventTypeTextMessageDelta AGUIEventType = "TEXT_MESSAGE_CONTENT"
	AGUIEventTypeTextMessageEnd   AGUIEventType = "TEXT_MESSAGE_END"
	AGUIEventTypeToolCallStart    AGUIEventType = "TOOL_CALL_START"
	AGUIEventTypeToolCallArgs     AGUIEventType = "TOOL_CALL_ARGS"
	AGUIEventTypeToolCallEnd      AGUIEventType = "TOOL_CALL_END"
	AGUIEventTypeCustom           AGUIEventType = "CUSTOM"
)

// AGUIEvent is the event envelope for the AG-UI protocol.
type AGUIEvent struct {
	Type      AGUIEventType   `json:"type"`
	Timestamp int64           `json:"timestamp"`
	RunID     string          `json:"run_id"`
	MessageID string          `json:"message_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolID    string          `json:"tool_id,omitempty"`
	Args      json.RawMessage `json:"args,omitempty"`
	Error     string          `json:"error,omitempty"`
	Meta      interface{}     `json:"meta,omitempty"`
}

// AGUISession represents an active AG-UI session.
type AGUISession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// AGUIServer implements the AG-UI protocol endpoints.
type AGUIServer struct {
	modelName string
	apiKey    string
	baseURL   string
	eventBus  *observability.EventBus
	sessions  map[string]*AGUISession
	mu        sync.RWMutex
}

// NewAGUIServer creates a new AG-UI protocol server.
func NewAGUIServer(modelName, apiKey, baseURL string, eventBus *observability.EventBus) *AGUIServer {
	return &AGUIServer{
		modelName: modelName,
		apiKey:    apiKey,
		baseURL:   baseURL,
		eventBus:  eventBus,
		sessions:  make(map[string]*AGUISession),
	}
}

// Routes registers the AG-UI protocol routes on the provided mux.
func (s *AGUIServer) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/chat/stream", s.handleChatStream)
}

// handleSessions handles GET (list) and POST (create) for sessions.
func (s *AGUIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSessions(w, r)
	case http.MethodPost:
		s.handleCreateSession(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleSessionByID handles GET for a specific session.
func (s *AGUIServer) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// Extract session ID from path: /api/sessions/{id}
	id := r.URL.Path[len("/api/sessions/"):]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session ID required"})
		return
	}

	s.mu.RLock()
	session, exists := s.sessions[id]
	s.mu.RUnlock()

	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// handleListSessions returns all active sessions.
func (s *AGUIServer) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*AGUISession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	writeJSON(w, http.StatusOK, sessions)
}

// handleCreateSession creates a new AG-UI session.
func (s *AGUIServer) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.UserID == "" {
		req.UserID = "default-user"
	}

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	session := &AGUISession{
		ID:        sessionID,
		UserID:    req.UserID,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, session)
}

// ChatRequest is the request body for the chat endpoint.
type ChatRequest struct {
	SessionID string        `json:"session_id"`
	Message   string        `json:"message"`
	AgentName string        `json:"agent_name,omitempty"`
	History   []ChatMessage `json:"history,omitempty"`
}

// ChatMessage represents a message in the conversation history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is the response body for the chat endpoint.
type ChatResponse struct {
	RunID   string `json:"run_id"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// handleChat handles non-streaming chat requests.
func (s *AGUIServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())

	// Create agent
	agt := s.createAgent(req.AgentName)
	if agt == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent name"})
		return
	}

	// Run the agent
	r2 := runner.NewRunner("fabula-studio", agt)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	msg := model.NewUserMessage(req.Message)
	eventChan, err := r2.Run(r.Context(), "default-user", sessionID, msg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ChatResponse{
			RunID: runID,
			Error: err.Error(),
		})
		return
	}

	// Collect response
	var content string
	for evt := range eventChan {
		if evt.Error != nil {
			writeJSON(w, http.StatusInternalServerError, ChatResponse{
				RunID: runID,
				Error: evt.Error.Message,
			})
			return
		}
		if len(evt.Response.Choices) > 0 {
			content += evt.Response.Choices[0].Message.Content
		}
	}

	writeJSON(w, http.StatusOK, ChatResponse{
		RunID:   runID,
		Content: content,
	})
}

// handleChatStream handles SSE streaming chat requests.
func (s *AGUIServer) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	// Send RUN_START
	s.sendSSE(flusher, w, AGUIEvent{
		Type:      AGUIEventTypeRunStart,
		Timestamp: time.Now().UnixMilli(),
		RunID:     runID,
	})

	// Create agent
	agt := s.createAgent(req.AgentName)
	if agt == nil {
		s.sendSSE(flusher, w, AGUIEvent{
			Type:      AGUIEventTypeRunError,
			Timestamp: time.Now().UnixMilli(),
			RunID:     runID,
			Error:     "invalid agent name",
		})
		return
	}

	// Run the agent
	r2 := runner.NewRunner("fabula-studio", agt)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	msg := model.NewUserMessage(req.Message)
	eventChan, err := r2.Run(r.Context(), "default-user", sessionID, msg)
	if err != nil {
		s.sendSSE(flusher, w, AGUIEvent{
			Type:      AGUIEventTypeRunError,
			Timestamp: time.Now().UnixMilli(),
			RunID:     runID,
			Error:     err.Error(),
		})
		return
	}

	// Track message state
	messageStarted := false

	// Process events
	for evt := range eventChan {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		if evt.Error != nil {
			s.sendSSE(flusher, w, AGUIEvent{
				Type:      AGUIEventTypeRunError,
				Timestamp: time.Now().UnixMilli(),
				RunID:     runID,
				Error:     evt.Error.Message,
			})
			return
		}

		// Handle text content
		if len(evt.Response.Choices) > 0 {
			choice := evt.Response.Choices[0]

			// Check for tool calls
			if len(choice.Message.ToolCalls) > 0 {
				for _, tc := range choice.Message.ToolCalls {
					s.sendSSE(flusher, w, AGUIEvent{
						Type:      AGUIEventTypeToolCallStart,
						Timestamp: time.Now().UnixMilli(),
						RunID:     runID,
						ToolName:  tc.Function.Name,
						ToolID:    tc.ID,
					})
					s.sendSSE(flusher, w, AGUIEvent{
						Type:      AGUIEventTypeToolCallArgs,
						Timestamp: time.Now().UnixMilli(),
						RunID:     runID,
						ToolID:    tc.ID,
						Args:      json.RawMessage(tc.Function.Arguments),
					})
					s.sendSSE(flusher, w, AGUIEvent{
						Type:      AGUIEventTypeToolCallEnd,
						Timestamp: time.Now().UnixMilli(),
						RunID:     runID,
						ToolID:    tc.ID,
					})
				}
			}

			// Handle text content
			if choice.Message.Content != "" {
				if !messageStarted {
					s.sendSSE(flusher, w, AGUIEvent{
						Type:      AGUIEventTypeTextMessageStart,
						Timestamp: time.Now().UnixMilli(),
						RunID:     runID,
						MessageID: messageID,
					})
					messageStarted = true
				}

				s.sendSSE(flusher, w, AGUIEvent{
					Type:      AGUIEventTypeTextMessageDelta,
					Timestamp: time.Now().UnixMilli(),
					RunID:     runID,
					MessageID: messageID,
					Content:   choice.Message.Content,
				})
			}
		}

		// Check for runner completion
		if evt.IsRunnerCompletion() {
			break
		}
	}

	// Close message if started
	if messageStarted {
		s.sendSSE(flusher, w, AGUIEvent{
			Type:      AGUIEventTypeTextMessageEnd,
			Timestamp: time.Now().UnixMilli(),
			RunID:     runID,
			MessageID: messageID,
		})
	}

	// Send pipeline progress events if available
	if s.eventBus != nil {
		for _, pe := range s.eventBus.GetEventLog() {
			metaJSON, _ := json.Marshal(pe)
			s.sendSSE(flusher, w, AGUIEvent{
				Type:      AGUIEventTypeCustom,
				Timestamp: time.Now().UnixMilli(),
				RunID:     runID,
				Meta:      json.RawMessage(metaJSON),
			})
		}
	}

	// Send RUN_FINISHED
	s.sendSSE(flusher, w, AGUIEvent{
		Type:      AGUIEventTypeRunFinished,
		Timestamp: time.Now().UnixMilli(),
		RunID:     runID,
	})
}

// sendSSE writes an AG-UI event as an SSE message.
func (s *AGUIServer) sendSSE(flusher http.Flusher, w http.ResponseWriter, evt AGUIEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("[AG-UI] Error marshaling event: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// createAgent creates an LLMAgent for the given agent name.
func (s *AGUIServer) createAgent(name string) *llmagent.LLMAgent {
	if name == "" {
		name = "default"
	}

	opts := []openai.Option{}
	if s.apiKey != "" {
		opts = append(opts, openai.WithAPIKey(s.apiKey))
	}
	if s.baseURL != "" {
		opts = append(opts, openai.WithBaseURL(s.baseURL))
	}
	m := openai.New(s.modelName, opts...)

	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(4096),
		Temperature: floatPtr(0.7),
	}

	return llmagent.New(name,
		llmagent.WithModel(m),
		llmagent.WithDescription("Fabula Studio conversational agent"),
		llmagent.WithInstruction("You are a helpful assistant for novel-to-screenplay adaptation. Answer questions about the conversion process, characters, scenes, and story structure."),
		llmagent.WithGenerationConfig(genConfig),
	)
}

// AGUIEventHandler returns an SSE handler that streams AG-UI events from the event bus.
func (s *AGUIServer) AGUIEventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		client := s.eventBus.Subscribe()
		defer s.eventBus.Unsubscribe(client)

		// Send initial connection event
		s.sendSSE(flusher, w, AGUIEvent{
			Type:      AGUIEventTypeCustom,
			Timestamp: time.Now().UnixMilli(),
			RunID:     "system",
			Meta:      map[string]string{"status": "connected"},
		})

		for {
			select {
			case <-r.Context().Done():
				return
			case evt, ok := <-client:
				if !ok {
					return
				}
				metaJSON, _ := json.Marshal(evt)
				s.sendSSE(flusher, w, AGUIEvent{
					Type:      AGUIEventTypeCustom,
					Timestamp: time.Now().UnixMilli(),
					RunID:     "pipeline",
					Meta:      json.RawMessage(metaJSON),
				})
			}
		}
	}
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int { return &i }

// floatPtr returns a pointer to a float64.
func floatPtr(f float64) *float64 { return &f }
