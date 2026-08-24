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

func setupGuardrailAdminHandler(t *testing.T) (*AdminHandler, *store.GuardrailStore, *store.GuardrailEventStore, *store.ProviderStore, *store.APIKeyStore) {
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
	return h, gs, ges, ps, ks
}

// ── Guardrails ──

func TestCreateGuardrail_Success(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	body := `{"pattern":"\\bpassword\\b","mode":"reject"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/guardrails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateGuardrail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, `\bpassword\b`, resp["pattern"])
	assert.Equal(t, "reject", resp["mode"])
}

func TestCreateGuardrail_Replace(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	body := `{"pattern":"secret","mode":"replace","replace_by":"[REDACTED]"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/guardrails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateGuardrail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "replace", resp["mode"])
	assert.Equal(t, "[REDACTED]", resp["replace_by"])
}

func TestCreateGuardrail_MissingFields(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	body := `{"pattern":""}`
	req := httptest.NewRequest(http.MethodPost, "/admin/guardrails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateGuardrail(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateGuardrail_InvalidMode(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	body := `{"pattern":"test","mode":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/guardrails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateGuardrail(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateGuardrail_InvalidRegex(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	body := `{"pattern":"[invalid","mode":"reject"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/guardrails", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateGuardrail(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestListGuardrails_Empty(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrails", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListGuardrails(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "[]\n", rec.Body.String())
}

func TestListGuardrails_WithData(t *testing.T) {
	h, gs, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	_, err := gs.Create("pattern1", "reject", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrails", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.ListGuardrails(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var list []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &list)
	assert.Len(t, list, 1)
}

func TestGetGuardrail_Found(t *testing.T) {
	h, gs, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	g, err := gs.Create("test", "reject", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrails/"+g.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(g.ID)

	err = h.GetGuardrail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetGuardrail_NotFound(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrails/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.GetGuardrail(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestDeleteGuardrail_Success(t *testing.T) {
	h, gs, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	g, err := gs.Create("todel", "reject", "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/admin/guardrails/"+g.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(g.ID)

	err = h.DeleteGuardrail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteGuardrail_NotFound(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodDelete, "/admin/guardrails/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.DeleteGuardrail(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

// ── Guardrail Events ──

func TestListGuardrailEvents_Empty(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrail-events", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListGuardrailEvents(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListGuardrailEvents_WithFilters(t *testing.T) {
	h, gs, ges, ps, ks := setupGuardrailAdminHandler(t)
	e := echo.New()

	g, err := gs.Create("test", "reject", "")
	require.NoError(t, err)
	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)
	k, _, err := ks.Create("k", p.ID, 0)
	require.NoError(t, err)
	_, err = ges.Record(g.ID, k.ID, g.Pattern, g.Mode, "input")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrail-events?guardrail_id="+g.ID+"&api_key_id="+k.ID+"&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.ListGuardrailEvents(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetGuardrailEvent_Found(t *testing.T) {
	h, gs, ges, ps, ks := setupGuardrailAdminHandler(t)
	e := echo.New()

	g, err := gs.Create("test", "reject", "")
	require.NoError(t, err)
	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)
	k, _, err := ks.Create("k", p.ID, 0)
	require.NoError(t, err)
	ev, err := ges.Record(g.ID, k.ID, g.Pattern, g.Mode, "input")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrail-events/"+ev.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ev.ID)

	err = h.GetGuardrailEvent(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetGuardrailEvent_NotFound(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/admin/guardrail-events/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.GetGuardrailEvent(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestDeleteGuardrailEvent_Success(t *testing.T) {
	h, gs, ges, ps, ks := setupGuardrailAdminHandler(t)
	e := echo.New()

	g, err := gs.Create("test", "reject", "")
	require.NoError(t, err)
	p, err := ps.Create("test", "https://test.com", "key")
	require.NoError(t, err)
	k, _, err := ks.Create("k", p.ID, 0)
	require.NoError(t, err)
	ev, err := ges.Record(g.ID, k.ID, g.Pattern, g.Mode, "input")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/admin/guardrail-events/"+ev.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(ev.ID)

	err = h.DeleteGuardrailEvent(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteGuardrailEvent_NotFound(t *testing.T) {
	h, _, _, _, _ := setupGuardrailAdminHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodDelete, "/admin/guardrail-events/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := h.DeleteGuardrailEvent(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}
