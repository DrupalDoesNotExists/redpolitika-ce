package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HealthHandler handles health and version requests.
type HealthHandler struct {
	logger  *zap.Logger
	version string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger:  logger,
		version: "0.1.0-dev",
	}
}

// Health responds with service health status.
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Version responds with version information.
func (h *HealthHandler) Version(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"version":   h.version,
		"module":    "ce",
		"component": "redpolitika",
	})
}
