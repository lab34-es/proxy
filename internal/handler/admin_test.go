package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAdminHandler(t *testing.T) (*AdminHandler, *store.ProviderStore, *store.APIKeyStore, *store.UsageStore) {
	t.Helper()
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	ps := store.NewProviderStore(database)
	ks := store.NewAPIKeyStore(database)
	us := store.NewUsageStore(database)
	gs := store.NewGuardrailStore(database)
	ges := store.NewGuardrailEventStore(database)
	h := NewAdminHandler(ps, ks, us, gs, ges)
	return h, ps, ks, us
}

// ── Providers ──

func TestCreateProvider_Success(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	body := `{"name":"openai","base_url":"https://api.openai.com","api_key":"sk-test"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/providers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateProvider(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "openai", resp["name"])
}

func TestCreateProvider_MissingFields(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	body := `{"name":"openai"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/providers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateProvider(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestListProviders_Empty(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListProviders(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "[]\n", rec.Body.String())
}

func TestListProviders_WithData(t *testing.T) {
	h, ps, _, _ := setupAdminHandler(t)
	e := echo.New()

	_, err := ps.Create("p1", "https://p1.com", "k1")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.ListProviders(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var list []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &list)
	assert.Len(t, list, 1)
}

func TestGetProvider_Found(t *testing.T) {
	h, ps, _, _ := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/providers/"+p.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(p.ID)

	err = h.GetProvider(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetProvider_NotFound(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/providers/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.GetProvider(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestUpdateProvider_Success(t *testing.T) {
	h, ps, _, _ := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("original", "https://orig.com", "key")
	require.NoError(t, err)

	body := `{"name":"updated"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/providers/"+p.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(p.ID)

	err = h.UpdateProvider(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "updated", resp["name"])
}

func TestUpdateProvider_NotFound(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	body := `{"name":"x"}`
	req := httptest.NewRequest(http.MethodPut, "/admin/providers/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.UpdateProvider(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestDeleteProvider_Success(t *testing.T) {
	h, ps, _, _ := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("todel", "https://del.com", "key")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/admin/providers/"+p.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(p.ID)

	err = h.DeleteProvider(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteProvider_NotFound(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodDelete, "/admin/providers/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.DeleteProvider(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

// ── API Keys ──

func TestCreateAPIKey_Success(t *testing.T) {
	h, ps, _, _ := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)

	body := `{"name":"my-key","provider_id":"` + p.ID + `","rate_limit_rpm":60}`
	req := httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.CreateAPIKey(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "my-key", resp["name"])
	assert.NotEmpty(t, resp["key"])
}

func TestCreateAPIKey_MissingFields(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	body := `{"name":"my-key"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateAPIKey(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateAPIKey_ProviderNotFound(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	body := `{"name":"my-key","provider_id":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateAPIKey(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestListAPIKeys_Empty(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListAPIKeys(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRevokeAPIKey_Success(t *testing.T) {
	h, ps, ks, _ := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)
	k, _, err := ks.Create("torevoke", p.ID, 0)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/admin/keys/"+k.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(k.ID)

	err = h.RevokeAPIKey(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRevokeAPIKey_NotFound(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodDelete, "/admin/keys/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.RevokeAPIKey(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

// ── Usage ──

func TestQueryUsage_Empty(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/usage", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.QueryUsage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQueryUsage_WithFilters(t *testing.T) {
	h, ps, ks, us := setupAdminHandler(t)
	e := echo.New()

	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)
	k, _, err := ks.Create("k", p.ID, 0)
	require.NoError(t, err)
	err = us.Record(k.ID, p.ID, "gpt-4", 100, 50, 150)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?api_key_id="+k.ID+"&provider_id="+p.ID+"&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.QueryUsage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestQueryUsage_InvalidStartTime(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?start=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.QueryUsage(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestQueryUsage_InvalidEndTime(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?end=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.QueryUsage(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestQueryUsage_ValidTimes(t *testing.T) {
	h, _, _, _ := setupAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?start=2024-01-01T00:00:00Z&end=2025-01-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.QueryUsage(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ── intParam ──

func TestIntParam(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name     string
		query    string
		param    string
		def      int
		expected int
	}{
		{"empty uses default", "", "limit", 100, 100},
		{"valid number", "limit=50", "limit", 100, 50},
		{"non-numeric uses default", "limit=abc", "limit", 100, 100},
		{"zero", "limit=0", "limit", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/admin/usage"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.expected, intParam(c, tt.param, tt.def))
		})
	}
}
