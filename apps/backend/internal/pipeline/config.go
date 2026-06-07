// Package pipeline orchestrates the full novel-to-screenplay conversion.
package pipeline

// Config holds tunable parameters for the conversion pipeline.
type Config struct {
	MaxConcurrency int `json:"max_concurrency"` // max concurrent story-beat extraction windows (default 2000)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{MaxConcurrency: 2000}
}
