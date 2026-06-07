// Package config provides application configuration loaded from environment.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configurable parameters for the backend service.
type Config struct {
	Addr                   string
	ModelName              string
	APIKey                 string
	BaseURL                string
	LogLevel               string
	OTLPEndpoint           string
	DatabaseURL            string
	JWTSecret              string
	JWTTTL                 time.Duration
	PipelineMaxConcurrency int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		ttl = 24 * time.Hour
	}
	maxConcurrency := getEnvInt("PIPELINE_MAX_CONCURRENCY", 3000)
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}

	return Config{
		Addr:                   getEnv("LISTEN_ADDR", ":8080"),
		ModelName:              getEnv("LLM_MODEL", "deepseek-v4-flash"),
		APIKey:                 os.Getenv("OPENAI_API_KEY"),
		BaseURL:                getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		OTLPEndpoint:           getEnv("OTLP_ENDPOINT", "localhost:4317"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://fabula:fabula@localhost:5432/fabula?sslmode=disable"),
		JWTSecret:              getEnv("JWT_SECRET", "fabula-local-development-secret"),
		JWTTTL:                 ttl,
		PipelineMaxConcurrency: maxConcurrency,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}
