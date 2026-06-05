// Package observability provides OpenTelemetry tracing and structured logging.
package observability

import (
	"context"
	"log"

	atrace "trpc.group/trpc-go/trpc-agent-go/telemetry/trace"
)

// Telemetry wraps the framework's trace cleanup function.
type Telemetry struct {
	cleanup func() error
}

// Config for telemetry initialization.
type Config struct {
	ServiceName  string
	OTLPEndpoint string // e.g., "localhost:4317"
}

// New initializes OpenTelemetry via the framework's atrace.Start().
// It sets up the global tracer provider with OTLP gRPC exporter.
func New(ctx context.Context, cfg Config) (*Telemetry, error) {
	opts := []atrace.Option{
		atrace.WithServiceName(cfg.ServiceName),
	}
	if cfg.OTLPEndpoint != "" {
		opts = append(opts, atrace.WithEndpoint(cfg.OTLPEndpoint))
	}

	clean, err := atrace.Start(ctx, opts...)
	if err != nil {
		return nil, err
	}

	log.Printf("[Telemetry] Initialized: service=%s endpoint=%s", cfg.ServiceName, cfg.OTLPEndpoint)
	return &Telemetry{cleanup: clean}, nil
}

// Shutdown flushes and stops the telemetry providers.
func (t *Telemetry) Shutdown() {
	if t.cleanup != nil {
		t.cleanup()
	}
}
