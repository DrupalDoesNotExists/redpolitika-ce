package http

import (
	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/ws"
	"github.com/drupaldoesnotexists/redpolitika/ce/pkg/config"
)

// RegisterRoutes registers all HTTP and WS routes on the Echo instance.
func RegisterRoutes(
	e *echo.Echo,
	cfg *config.Config,
	health *HealthHandler,
	clientRules *ClientRulesHandler,
	analyze *AnalyzeHandler,
	rules *RulesHandler,
	live *ws.LiveHandler,
) {
	// Health
	e.GET("/health", health.Health)
	e.GET("/healthz", health.Health)
	e.GET("/version", health.Version)
	e.GET("/metrics", health.Metrics())

	// API
	e.GET("/api/client-rules", clientRules.Handle)
	e.POST("/api/analyze", analyze.Handle)
	e.GET("/api/rules", rules.Handle)

	// WebSocket
	e.GET("/ws/live", live.Handle)

	// Static file serving — frontend static export
	e.Static("/", cfg.StaticDir)
}
