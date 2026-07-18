package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/plugin"
)

// HealthHandler handles health, version, and metrics.
type HealthHandler struct {
	logger  *zap.Logger
	version string
	plugins *plugin.Manager

	analyzeLatency prometheus.Histogram
	flagsTotal     prometheus.Counter
	analyzeTotal   prometheus.Counter
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *zap.Logger, plugins *plugin.Manager) *HealthHandler {
	h := &HealthHandler{
		logger:  logger,
		version: "0.1.0-dev",
		plugins: plugins,
	}
	h.analyzeLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "redpolitika_analyze_latency_seconds",
		Help:    "Latency of text analysis",
		Buckets: prometheus.DefBuckets,
	})
	h.flagsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "redpolitika_flags_total",
		Help: "Total flags raised by analyze",
	})
	h.analyzeTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "redpolitika_analyze_total",
		Help: "Total analyze requests",
	})
	return h
}

// ObserveAnalyze records analyze metrics (call from analyze handler).
func (h *HealthHandler) ObserveAnalyze(d time.Duration, flagCount int) {
	h.analyzeTotal.Inc()
	h.analyzeLatency.Observe(d.Seconds())
	h.flagsTotal.Add(float64(flagCount))
}

// Health responds with service health status including plugins.
func (h *HealthHandler) Health(c echo.Context) error {
	plugins := []string{}
	if h.plugins != nil {
		plugins = h.plugins.Status()
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"plugins": plugins,
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

// Metrics exposes Prometheus metrics.
func (h *HealthHandler) Metrics() echo.HandlerFunc {
	handler := promhttp.Handler()
	return func(c echo.Context) error {
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
