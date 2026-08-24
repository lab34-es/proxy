package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/models"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminAuth_ValidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	req.Header.Set("Authorization", "Bearer my-admin-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AdminAuth("my-admin-token")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminAuth_InvalidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AdminAuth("my-admin-token")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestAdminAuth_MissingToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AdminAuth("my-admin-token")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestAdminAuth_EmptyAdminToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := AdminAuth("")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestProxyAuth_ValidKey(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ps := store.NewProviderStore(database)
	ks := store.NewAPIKeyStore(database)

	provider, err := ps.Create("test", "https://test.com", "sk")
	require.NoError(t, err)

	_, rawKey, err := ks.Create("test-key", provider.ID, 60)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := ProxyAuth(ks)(func(c echo.Context) error {
		key := GetAPIKey(c)
		assert.NotNil(t, key)
		assert.Equal(t, "test-key", key.Name)
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.NoError(t, err)
}

func TestProxyAuth_InvalidKey(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ks := store.NewAPIKeyStore(database)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer llmp-invalidkey")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := ProxyAuth(ks)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestProxyAuth_MissingKey(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ks := store.NewAPIKeyStore(database)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := ProxyAuth(ks)(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestGetAPIKey_NilContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	key := GetAPIKey(c)
	assert.Nil(t, key)
}

func TestGetAPIKey_WithKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expected := &models.APIKey{ID: "test-id", Name: "test"}
	c.Set(string(APIKeyContextKey), expected)

	key := GetAPIKey(c)
	assert.Equal(t, expected, key)
}

func TestExtractBearer(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid bearer", "Bearer my-token", "my-token"},
		{"no bearer prefix", "my-token", ""},
		{"empty", "", ""},
		{"basic auth", "Basic dXNlcjpwYXNz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.expected, extractBearer(c))
		})
	}
}
