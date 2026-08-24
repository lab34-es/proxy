package config

import (
	"os"
)

// Config holds all application configuration sourced from environment variables.
type Config struct {
	// Addr is the address the server listens on.
	Addr string
	// DSN is the SQLite data source name (file path).
	DSN string
	// AdminToken is the bearer token required for /admin/* endpoints.
	AdminToken string
	// SessionSecret is used to sign dashboard session cookies.
	// Defaults to AdminToken if not set.
	SessionSecret string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	adminToken := envOr("ADMIN_TOKEN", "")
	return Config{
		Addr:          envOr("ADDR", ":8080"),
		DSN:           envOr("DSN", "proxy.db"),
		AdminToken:    adminToken,
		SessionSecret: envOr("SESSION_SECRET", adminToken),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
