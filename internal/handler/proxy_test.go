package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/middleware"
	"github.com/lab34-es/proxy/internal/models"
	"github.com/lab34-es/proxy/internal/proxy"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatCompletion_NoAPIKey(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ps := store.NewProviderStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	f := proxy.NewForwarder(us, gs, ges)
	h := NewProxyHandler(f, ps)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.ChatCompletion(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestChatCompletion_ProviderNotFound(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ps := store.NewProviderStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	f := proxy.NewForwarder(us, gs, ges)
	h := NewProxyHandler(f, ps)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set a key with a non-existent provider.
	c.Set(string(middleware.APIKeyContextKey), &models.APIKey{
		ID:         "key1",
		ProviderID: "nonexistent-provider",
	})

	err = h.ChatCompletion(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, he.Code)
}

func TestListModels_NoAPIKey(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ps := store.NewProviderStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	f := proxy.NewForwarder(us, gs, ges)
	h := NewProxyHandler(f, ps)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.ListModels(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestListModels_ProviderNotFound(t *testing.T) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	defer database.Close()

	ps := store.NewProviderStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	f := proxy.NewForwarder(us, gs, ges)
	h := NewProxyHandler(f, ps)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set(string(middleware.APIKeyContextKey), &models.APIKey{
		ID:         "key1",
		ProviderID: "nonexistent-provider",
	})

	err = h.ListModels(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, he.Code)
}
