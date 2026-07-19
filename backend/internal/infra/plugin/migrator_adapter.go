package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/migrator"
)

// MigratorAdapter adapts plugin MigratorService into domain Migrator port (A13).
type MigratorAdapter struct {
	client migrator.MigratorServiceClient
}

// NewMigratorAdapter creates a MigratorAdapter from the first plugin with migrator capability.
func NewMigratorAdapter(registry *Registry) ports.Migrator {
	plugins := registry.FindByCapability(CapMigrator)
	if len(plugins) == 0 {
		return nil
	}
	return &MigratorAdapter{client: migrator.NewMigratorServiceClient(plugins[0].Conn)}
}

func (a *MigratorAdapter) Migrate(ctx context.Context, req ports.MigrateRequest) (ports.MigrateResult, error) {
	resp, err := a.client.Migrate(ctx, &migrator.MigrateRequest{
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
