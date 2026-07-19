package http

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/pkg/config"
)

// NewEcho creates and configures an Echo instance.
func NewEcho(cfg *config.Config, logger *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.HTTPErrorHandler = HTTPErrorHandler

	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request",
				zap.String("uri", v.URI),
				zap.String("method", v.Method),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.Error(v.Error),
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	return e
}

// StartEcho starts the Echo HTTP server.
// Used as an fx.Invoke lifecycle hook.
func StartEcho(lc fx.Lifecycle, e *echo.Echo, cfg *config.Config, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%d", cfg.Port)
			logger.Info("starting HTTP server", zap.String("addr", addr))
			go func() {
				if err := e.Start(addr); err != nil {
					logger.Fatal("server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down HTTP server")
			return e.Shutdown(ctx)
		},
	})
}

// Module is the FX module for the transport layer.
var Module = fx.Module("transport",
	fx.Provide(
		NewEcho,
		NewHealthHandler,
		NewClientRulesHandler,
		NewAnalyzeHandler,
		NewRulesHandler,
		NewPagesHandler,
	),
	fx.Invoke(
		RegisterRoutes,
		StartEcho,
	),
)
