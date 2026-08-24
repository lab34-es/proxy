package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere.
	os.Unsetenv("ADDR")
	os.Unsetenv("DSN")
	os.Unsetenv("ADMIN_TOKEN")
	os.Unsetenv("SESSION_SECRET")

	cfg := Load()

	assert.Equal(t, ":8080", cfg.Addr)
	assert.Equal(t, "proxy.db", cfg.DSN)
	assert.Equal(t, "", cfg.AdminToken)
	assert.Equal(t, "", cfg.SessionSecret)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("DSN", "test.db")
	t.Setenv("ADMIN_TOKEN", "my-token")
	t.Setenv("SESSION_SECRET", "my-secret")

	cfg := Load()

	assert.Equal(t, ":9090", cfg.Addr)
	assert.Equal(t, "test.db", cfg.DSN)
	assert.Equal(t, "my-token", cfg.AdminToken)
	assert.Equal(t, "my-secret", cfg.SessionSecret)
}

func TestLoad_SessionSecretDefaultsToAdminToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "admin-tok")
	os.Unsetenv("SESSION_SECRET")

	cfg := Load()

	assert.Equal(t, "admin-tok", cfg.AdminToken)
	assert.Equal(t, "admin-tok", cfg.SessionSecret)
}

func TestEnvOr(t *testing.T) {
	os.Unsetenv("TEST_ENV_OR_KEY")
	assert.Equal(t, "fallback", envOr("TEST_ENV_OR_KEY", "fallback"))

	t.Setenv("TEST_ENV_OR_KEY", "value")
	assert.Equal(t, "value", envOr("TEST_ENV_OR_KEY", "fallback"))
}
