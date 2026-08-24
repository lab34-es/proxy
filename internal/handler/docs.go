package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// DocsHandler serves the OpenAPI spec and a Swagger UI page.
type DocsHandler struct {
	spec []byte
}

func NewDocsHandler(spec []byte) *DocsHandler {
	return &DocsHandler{spec: spec}
}

// Spec serves the raw OpenAPI YAML file.
func (h *DocsHandler) Spec(c echo.Context) error {
	return c.Blob(http.StatusOK, "application/x-yaml", h.spec)
}

// SwaggerUI serves an HTML page that loads Swagger UI from a CDN and points it
// at /openapi.yaml. No local assets required.
func (h *DocsHandler) SwaggerUI(c echo.Context) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>lab34/proxy — API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: '#swagger-ui',
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset,
      ],
      layout: "BaseLayout",
      deepLinking: true,
    });
  </script>
</body>
</html>`)
	return c.HTML(http.StatusOK, html)
}
