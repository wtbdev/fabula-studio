// Package config provides application configuration loaded from environment.
package config

import "os"

// Config holds all configurable parameters for the backend service.
type Config struct {
	Addr       string
	ModelName  string
	APIKey     string
	BaseURL    string
	LogLevel   string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		Addr:      getEnv("LISTEN_ADDR", ":8080"),
		ModelName: getEnv("LLM_MODEL", "deepseek-v4-flash"),
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		BaseURL:   getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
