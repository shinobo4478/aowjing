// Package config loads runtime configuration from environment variables.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadDotEnv reads simple KEY=VALUE lines from path into the process
// environment. A missing file is not an error. Existing variables are not
// overridden, so real environment config still wins in deployed setups. Call
// it before Load.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

// Config holds everything the server needs to start.
type Config struct {
	Port        string
	DatabaseURL string

	// RedisAddr is the asynq broker, shared by the API (enqueue) and the
	// worker (process).
	RedisAddr string

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
		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
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
