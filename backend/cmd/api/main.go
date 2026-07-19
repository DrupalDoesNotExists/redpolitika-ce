package main

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/service"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/cache"
	dbinfra "github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/db"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/plugin"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/rules"
	sessionstore "github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/session"
	httptransport "github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/http"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/ws"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
	"github.com/drupaldoesnotexists/redpolitika/ce/pkg/config"
)

func main() {
	fx.New(
		// Bootstrap: config + logger
		fx.Provide(
			config.ProvideConfig,
			config.ProvideLogger,
		),

		// Domain services
		fx.Provide(
			service.NewRuleEngine,
			service.NewScoreCalculator,
			service.NewFixApplier,
		),

		// Infra — rules
		fx.Provide(
			func(cfg *config.Config) *rules.Loader {
				return rules.NewLoader(cfg.RulesDir, cfg.RulesProjectDir, cfg.RulesOverrideDir)
			},
			fx.Annotate(rules.NewRepository, fx.As(new(ports.RuleRepository))),
		),

		// Infra — session
		fx.Provide(
			fx.Annotate(sessionstore.NewMemoryStore, fx.As(new(ports.SessionRepository))),
		),

		// Infra — cache
		fx.Provide(
			fx.Annotate(
				func() *cache.LRUCache { return cache.NewLRUCache(1000, 5*time.Minute) },
				fx.As(new(ports.CacheRepository)),
			),
		),

		// Infra — database (framework for plugins - A6)
		fx.Provide(
			func(cfg *config.Config) dbinfra.Config {
				return dbinfra.Config{Driver: cfg.DBDriver, DSN: cfg.DBDSN}
			},
			dbinfra.NewConnector,
			dbinfra.NewMigrator,
		),
		fx.Invoke(func(lc fx.Lifecycle, m *dbinfra.Migrator, log *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					log.Info("running database migrations")
					return m.Migrate()
				},
			})
		}),

		// Infra — plugin system
		fx.Provide(
			func(cfg *config.Config) string { return cfg.PluginsDir },
			plugin.NewManager,
			plugin.NewRegistry,
			plugin.NewPluginRegistrar,
			fx.Annotate(plugin.NewPagesAdapter, fx.As(new(ports.StaticPagesProvider))),
		),
		fx.Invoke(func(lc fx.Lifecycle, m *plugin.Manager, r *plugin.Registry, log *zap.Logger, cfg *config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if cfg.PluginsDir == "" {
						log.Info("no plugins dir configured, skipping plugin scan")
						return nil
					}
					started, err := m.ScanDir(ctx, r, cfg.PluginsDir)
					if err != nil {
						return err
					}
					log.Info("plugin scan complete", zap.Strings("started", started))
					return nil
				},
				OnStop: func(context.Context) error {
					m.StopAll()
					return nil
				},
			})
		}),

		// Usecases
		fx.Provide(
			func(
				ruleRepo ports.RuleRepository,
				sessionRepo ports.SessionRepository,
				cache ports.CacheRepository,
				engine *service.RuleEngine,
				calculator *service.ScoreCalculator,
				logger *zap.Logger,
				reg *plugin.Registry,
			) *usecase.AnalyzeTextUseCase {
				// Build plugin adapters from registry (A27 extension points)
				// Returns nil if no plugin provides the capability — fallback to core-only.
				llmProvider := plugin.NewLLMAdapter(reg)
				detectFunc := plugin.NewDetectAdapter(reg)
				return usecase.NewAnalyzeTextUseCase(ruleRepo, sessionRepo, cache, engine, calculator, llmProvider, detectFunc, logger)
			},
			usecase.NewAcceptRejectFlagUseCase,
			usecase.NewApplyFixUseCase,
			usecase.NewRegisterPluginUseCase,
		),

		// Transport layers
		ws.Module,
		httptransport.Module,
	).Run()
}
