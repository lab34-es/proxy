package main

import (
	_ "embed"
	"log"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/lab34-es/proxy/internal/config"
	"github.com/lab34-es/proxy/internal/db"
	"github.com/lab34-es/proxy/internal/handler"
	"github.com/lab34-es/proxy/internal/middleware"
	"github.com/lab34-es/proxy/internal/proxy"
	"github.com/lab34-es/proxy/internal/store"
	"github.com/lab34-es/proxy/internal/web"
)

//go:embed openapi.yaml
var specBytes []byte

func main() {
	cfg := config.Load()

	if cfg.AdminToken == "" {
		log.Fatal("ADMIN_TOKEN environment variable is required")
	}

	database, err := db.Open(cfg.DSN)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// Stores.
	providerStore := store.NewProviderStore(database)
	keyStore := store.NewAPIKeyStore(database)
	usageStore := store.NewUsageStore(database)
	guardrailStore := store.NewGuardrailStore(database)
	guardrailEventStore := store.NewGuardrailEventStore(database)

	// Handlers.
	adminH := handler.NewAdminHandler(providerStore, keyStore, usageStore, guardrailStore, guardrailEventStore)
	forwarder := proxy.NewForwarder(usageStore, guardrailStore, guardrailEventStore)
	proxyH := handler.NewProxyHandler(forwarder, providerStore)
	docsH := handler.NewDocsHandler(specBytes)

	// Middleware.
	rateLimiter := middleware.NewRateLimiter()

	// Echo setup.
	e := echo.New()
	e.HideBanner = true
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())

	// ── Documentation (public) ─────────────────────────────────────────
	e.GET("/openapi.yaml", docsH.Spec)
	e.GET("/docs", docsH.SwaggerUI)

	// ── Dashboard (React SPA) ──────────────────────────────────────────
	spaHandler := web.SPAHandler()
	e.GET("/dashboard", spaHandler)
	e.GET("/dashboard/*", spaHandler)

	// ── Admin routes (admin token auth) ────────────────────────────────
	admin := e.Group("/admin", middleware.AdminAuth(cfg.AdminToken))

	admin.POST("/providers", adminH.CreateProvider)
	admin.GET("/providers", adminH.ListProviders)
	admin.GET("/providers/:id", adminH.GetProvider)
	admin.PUT("/providers/:id", adminH.UpdateProvider)
	admin.DELETE("/providers/:id", adminH.DeleteProvider)

	admin.POST("/keys", adminH.CreateAPIKey)
	admin.GET("/keys", adminH.ListAPIKeys)
	admin.DELETE("/keys/:id", adminH.RevokeAPIKey)

	admin.GET("/usage", adminH.QueryUsage)

	admin.POST("/guardrails", adminH.CreateGuardrail)
	admin.GET("/guardrails", adminH.ListGuardrails)
	admin.GET("/guardrails/:id", adminH.GetGuardrail)
	admin.DELETE("/guardrails/:id", adminH.DeleteGuardrail)

	admin.GET("/guardrail-events", adminH.ListGuardrailEvents)
	admin.GET("/guardrail-events/:id", adminH.GetGuardrailEvent)
	admin.DELETE("/guardrail-events/:id", adminH.DeleteGuardrailEvent)

	// ── Proxy routes (proxy API key auth + rate limit) ─────────────────
	v1 := e.Group("/v1",
		middleware.ProxyAuth(keyStore),
		rateLimiter.Middleware(),
	)

	v1.POST("/chat/completions", proxyH.ChatCompletion)
	v1.GET("/models", proxyH.ListModels)

	// Start.
	log.Printf("proxy listening on %s", cfg.Addr)
	log.Printf("Swagger UI: http://localhost%s/docs", cfg.Addr)
	log.Printf("Dashboard:  http://localhost%s/dashboard", cfg.Addr)
	if err := e.Start(cfg.Addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
