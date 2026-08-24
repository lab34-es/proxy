package middleware

import (
	"net/http"
	"strings"

	"github.com/lab34-es/proxy/internal/models"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/labstack/echo/v4"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

// AdminAuth returns middleware that validates the Authorization bearer token
// against the configured admin token.
func AdminAuth(adminToken string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if adminToken == "" {
				// No admin token configured — reject all admin requests.
				return echo.NewHTTPError(http.StatusUnauthorized, "admin token not configured")
			}
			token := extractBearer(c)
			if token != adminToken {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid admin token")
			}
			return next(c)
		}
	}
}

// ProxyAuth returns middleware that validates the Authorization bearer token
// as a proxy API key by looking it up (hashed) in the database.
func ProxyAuth(keys *store.APIKeyStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := extractBearer(c)
			if raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing api key")
			}
			k, err := keys.Lookup(raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "key lookup failed")
			}
			if k == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
			}
			c.Set(string(APIKeyContextKey), k)
			return next(c)
		}
	}
}

// GetAPIKey retrieves the authenticated API key from the echo context.
func GetAPIKey(c echo.Context) *models.APIKey {
	v := c.Get(string(APIKeyContextKey))
	if v == nil {
		return nil
	}
	k, _ := v.(*models.APIKey)
	return k
}

func extractBearer(c echo.Context) string {
	h := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
