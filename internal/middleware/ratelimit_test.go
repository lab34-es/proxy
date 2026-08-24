package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lab34-es/proxy/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_NoKey(t *testing.T) {
	rl := NewRateLimiter()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_UnlimitedKey(t *testing.T) {
	rl := NewRateLimiter()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set(string(APIKeyContextKey), &models.APIKey{ID: "key1", RateLimitRPM: 0})

	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter()
	e := echo.New()

	key := &models.APIKey{ID: "key1", RateLimitRPM: 60}

	// First request should always be allowed (burst).
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(APIKeyContextKey), key)

	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter()
	e := echo.New()

	key := &models.APIKey{ID: "rate-test", RateLimitRPM: 1}

	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request — uses the burst allowance.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(APIKeyContextKey), key)
	err := handler(c)
	assert.NoError(t, err)

	// Second request — should be rate limited.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(string(APIKeyContextKey), key)
	err = handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusTooManyRequests, he.Code)
}

func TestRateLimiter_GetLimiter_Reuse(t *testing.T) {
	rl := NewRateLimiter()

	l1 := rl.getLimiter("key1", 60)
	l2 := rl.getLimiter("key1", 60)
	l3 := rl.getLimiter("key2", 120)

	assert.Same(t, l1, l2)    // same key -> same limiter
	assert.NotSame(t, l1, l3) // different key -> different limiter
}
