// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds everything the server needs to start.
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from the environment. It returns an error if a
// required variable is missing so main can fail fast with a clear message.
func Load() (Config, error) {
	cfg := Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

// getenv returns the value of key, or fallback when it is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
