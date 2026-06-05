// Package observability provides OpenTelemetry tracing and structured logging.
package observability

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry holds the telemetry providers and utilities.
type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	Tracer         trace.Tracer
}

// Config for telemetry initialization.
type Config struct {
	ServiceName    string
	OTLPEndpoint   string // e.g., "localhost:4317"
	Environment    string
}

// New initializes OpenTelemetry with OTLP gRPC exporter.
func New(ctx context.Context, cfg Config) (*Telemetry, error) {
	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = "localhost:4317"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Create OTLP trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // For local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service info
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global provider
	otel.SetTracerProvider(tp)

	return &Telemetry{
		TracerProvider: tp,
		Tracer:         tp.Tracer(cfg.ServiceName),
	}, nil
}

// Shutdown flushes and stops the telemetry providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.TracerProvider != nil {
		return t.TracerProvider.Shutdown(ctx)
	}
	return nil
}

// Logger provides structured logging with context.
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a structured logger.
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "[Fabula] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Info logs an info message with optional key-value pairs.
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("%s %v", msg, formatKeysAndValues(keysAndValues...))
}

// Error logs an error message with optional key-value pairs.
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Printf("ERROR: %s %v", msg, formatKeysAndValues(keysAndValues...))
}

// Debug logs a debug message with optional key-value pairs.
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if os.Getenv("LOG_LEVEL") == "debug" {
		l.logger.Printf("DEBUG: %s %v", msg, formatKeysAndValues(keysAndValues...))
	}
}

// WithContext extracts trace context for logging.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return &Logger{
			logger: log.New(os.Stdout, fmt.Sprintf("[Fabula][%s] ", span.SpanContext().TraceID().String()[:8]), log.LstdFlags|log.Lmsgprefix),
		}
	}
	return l
}

func formatKeysAndValues(keysAndValues ...interface{}) string {
	if len(keysAndValues) == 0 {
		return ""
	}
	result := ""
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%v=%v", keysAndValues[i], keysAndValues[i+1])
	}
	return result
}
