// Package ports defines repository and extension point interfaces (A27).
// These are the "ports" in ports-and-adapters. Domain depends on these interfaces;
// infra provides implementations. Domain does not import infra.
package ports

import (
	"context"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// --- Repositories ---

// RuleRepository loads rules from disk storage.
// Implementation: infra/rules/ on disk.
type RuleRepository interface {
	// LoadAll reads all rule YAML files, deep-merges base→project→override layers (A24).
	LoadAll(ctx context.Context) (*model.RuleSet, error)

	// Watch returns a channel of reload notifications (optional — A25).
	Watch(ctx context.Context) <-chan struct{}
}

// SessionRepository manages sessions in memory.
// Implementation: infra/session/.
type SessionRepository interface {
	// Save persists a session.
	Save(ctx context.Context, session *model.Session) error

	// FindByID retrieves a session by ID.
	FindByID(ctx context.Context, id model.SessionID) (*model.Session, error)

	// Delete removes a session.
	Delete(ctx context.Context, id model.SessionID) error

	// ListActive returns all active session IDs.
	ListActive(ctx context.Context) ([]model.SessionID, error)
}

// CacheRepository caches analysis results for unchanged text+config (A33).
// Implementation: infra/cache/.
type CacheRepository interface {
	// Get retrieves cached analysis for a given text+config hash pair.
	Get(ctx context.Context, textHash, configHash uint64) (*model.Analysis, error)

	// Set stores an analysis result.
	Set(ctx context.Context, textHash, configHash uint64, analysis *model.Analysis) error

	// Invalidate clears cache entries for a config hash (when rules reload).
	Invalidate(ctx context.Context, configHash uint64) error
}
