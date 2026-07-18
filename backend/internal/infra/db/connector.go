package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"go.uber.org/zap"
)

// Config holds database configuration.
type Config struct {
	// Driver is "sqlite" or "postgres". Default: "sqlite" (A2).
	Driver string
	// DSN is the connection string.
	DSN string
	// MaxOpenConns is max open connections. Default: 25.
	MaxOpenConns int
	// MaxIdleConns is max idle connections. Default: 5.
	MaxIdleConns int
	// ConnMaxLifetime is max connection lifetime. Default: 5min.
	ConnMaxLifetime time.Duration
}

// Connector opens and manages the *sql.DB connection.
type Connector struct {
	cfg    Config
	db     *sql.DB
	logger *zap.Logger
}

// NewConnector opens a database connection based on Config.
func NewConnector(cfg Config, logger *zap.Logger) (*Connector, error) {
	// Default to sqlite
	if cfg.Driver == "" {
		cfg.Driver = "sqlite"
	}
	if cfg.DSN == "" {
		cfg.DSN = "file:redpolitika.db?cache=shared&_journal_mode=WAL"
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}

	// Map driver names
	driverName := cfg.Driver
	switch cfg.Driver {
	case "sqlite":
		driverName = "sqlite" // modernc.org/sqlite registers as "sqlite"
	case "postgres":
		driverName = "pgx" // pgx stdlib registers as "pgx"
	}

	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db connector: open %s: %w", cfg.Driver, err)
	}

	// Configure pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connectivity
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db connector: ping %s: %w", cfg.Driver, err)
	}

	logger.Info("database connected", zap.String("driver", cfg.Driver))

	return &Connector{cfg: cfg, db: db, logger: logger}, nil
}

// DB returns the underlying *sql.DB.
func (c *Connector) DB() *sql.DB { return c.db }

// Close closes the database connection.
func (c *Connector) Close() error {
	return c.db.Close()
}
