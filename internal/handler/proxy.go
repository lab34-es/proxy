package handler

import (
	"errors"
	"net/http"

	"github.com/lab34-es/proxy/internal/middleware"
	"github.com/lab34-es/proxy/internal/proxy"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
)

type ProxyHandler struct {
	forwarder *proxy.Forwarder
	providers *store.ProviderStore
}

func NewProxyHandler(f *proxy.Forwarder, p *store.ProviderStore) *ProxyHandler {
	return &ProxyHandler{forwarder: f, providers: p}
}

func (h *ProxyHandler) ChatCompletion(c echo.Context) error {
	apiKey := middleware.GetAPIKey(c)
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing api key")
	}

	provider, err := h.providers.GetByID(apiKey.ProviderID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to look up provider")
	}
	if provider == nil {
		return echo.NewHTTPError(http.StatusBadGateway, "provider not found for this key")
	}

	if err := h.forwarder.ForwardChatCompletion(c.Response().Writer, c.Request(), provider, apiKey); err != nil {
		var grErr *proxy.GuardrailRejectedError
		if errors.As(err, &grErr) {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": map[string]interface{}{
					"message": grErr.Error(),
					"type":    "guardrail_rejected",
					"code":    "guardrail_rejected",
				},
			})
		}
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	return nil
}

func (h *ProxyHandler) ListModels(c echo.Context) error {
	apiKey := middleware.GetAPIKey(c)
	if apiKey == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing api key")
	}

	provider, err := h.providers.GetByID(apiKey.ProviderID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to look up provider")
	}
	if provider == nil {
		return echo.NewHTTPError(http.StatusBadGateway, "provider not found for this key")
	}

	if err := h.forwarder.ForwardListModels(c.Response().Writer, c.Request(), provider); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	return nil
}
