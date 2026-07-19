package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/migrator"
)

// MigratorAdapter adapts plugin MigratorService into domain Migrator port (A13).
type MigratorAdapter struct {
	registry *Registry
}

// NewMigratorAdapter creates a MigratorAdapter.
// Plugin lookup deferred to call time — plugins load during OnStart, after construction.
func NewMigratorAdapter(registry *Registry) ports.Migrator {
	return &MigratorAdapter{registry: registry}
}

func (a *MigratorAdapter) Migrate(ctx context.Context, req ports.MigrateRequest) (ports.MigrateResult, error) {
	plugins := a.registry.FindByCapability(CapMigrator)
	if len(plugins) == 0 {
		return ports.MigrateResult{}, fmt.Errorf("migrator: no plugin with %q capability", CapMigrator)
	}
	client := migrator.NewMigratorServiceClient(plugins[0].Conn)
	resp, err := client.Migrate(ctx, &migrator.MigrateRequest{
		Dialect:       req.Dialect,
		Dsn:           req.DSN,
		TargetVersion: req.TargetVersion,
		Direction:     req.Direction,
	})
	if err != nil {
		return ports.MigrateResult{}, fmt.Errorf("migrator plugin: %w", err)
	}
	return ports.MigrateResult{
		CurrentVersion: resp.CurrentVersion,
		ErrMsg:         resp.Error,
	}, nil
}
