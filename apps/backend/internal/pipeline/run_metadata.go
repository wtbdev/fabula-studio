package pipeline

import "context"

// RunMetadata carries stable correlation identifiers for a pipeline run.
type RunMetadata struct {
	ProjectID string
	JobID     string
	RunID     string
	TraceID   string
}

type runMetadataContextKey struct{}

// WithRunMetadata attaches correlation metadata to ctx for Pipeline.Convert.
func WithRunMetadata(ctx context.Context, meta RunMetadata) context.Context {
	return context.WithValue(ctx, runMetadataContextKey{}, meta)
}

// RunMetadataFromContext returns pipeline run metadata attached to ctx.
func RunMetadataFromContext(ctx context.Context) (RunMetadata, bool) {
	meta, ok := ctx.Value(runMetadataContextKey{}).(RunMetadata)
	return meta, ok
}
