// Package pipeline orchestrates the full novel-to-screenplay conversion.
package pipeline

// Config holds tunable parameters for the conversion pipeline.
type Config struct {
	MaxChunkSize       int     `json:"max_chunk_size"`       // max characters per story node (default 3000)
	MaxRecursionDepth  int     `json:"max_recursion_depth"`  // max tree depth (default 5)
	MaxRetries         int     `json:"max_retries"`          // retry count for agent calls (default 3)
	MaxConcurrency     int     `json:"max_concurrency"`      // goroutine pool size for scene writing (default 5)
	ContextWindowRatio float64 `json:"context_window_ratio"` // context package max fraction of model window (default 0.5)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxChunkSize:       3000,
		MaxRecursionDepth:  5,
		MaxRetries:         3,
		MaxConcurrency:     5,
		ContextWindowRatio: 0.5,
	}
}
