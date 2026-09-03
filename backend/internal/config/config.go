// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds everything the server needs to start.
type Config struct {
	Port        string
	DatabaseURL string

	// Single-admin credentials. No user table yet (Phase 1 scope).
	AdminUsername string
	AdminPassword string

	// CookieSecure marks the session cookie Secure + SameSite=None, required
	// when the frontend and API are served from different sites over HTTPS.
	// Leave false for local http.
	CookieSecure bool

	// CORSOrigin is the single browser origin allowed to call the API with
	// credentials (the frontend dev server by default).
	CORSOrigin string
}

// Load reads configuration from the environment. It returns an error if a
// required variable is missing so main can fail fast with a clear message.
func Load() (Config, error) {
	cfg := Config{
		Port:          getenv("PORT", "8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		AdminUsername: getenv("ADMIN_USERNAME", "admin"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		CookieSecure:  getenvBool("COOKIE_SECURE", false),
		CORSOrigin:    getenv("CORS_ORIGIN", "http://localhost:3000"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD is required")
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

// getenvBool parses key as a bool ("1", "t", "true", ... per strconv), falling
// back on unset or unparseable values.
func getenvBool(key string, fallback bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
