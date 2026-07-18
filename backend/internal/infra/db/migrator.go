package db

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator applies database migrations on startup.
// Core provides DB infra for plugins (A6). CE domain doesn't use DB directly.
// Plugins receive Migrator access via gRPC (A13) for their own schema migrations.
type Migrator struct {
	conn   *Connector
	logger *zap.Logger
}

// NewMigrator creates a Migrator.
func NewMigrator(conn *Connector, logger *zap.Logger) *Migrator {
	return &Migrator{conn: conn, logger: logger}
}

// Migrate applies all pending CE framework migrations.
func (m *Migrator) Migrate() error {
	return m.migrateFS()
}

// migrateFS applies embedded .sql migrations in order.
func (m *Migrator) migrateFS() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrator: read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("migrator: read %s: %w", name, err)
		}

		if _, err := m.conn.DB().Exec(string(data)); err != nil {
			return fmt.Errorf("migrator: execute %s: %w", name, err)
		}

		m.logger.Info("migration applied", zap.String("file", name))
	}

	return nil
}
