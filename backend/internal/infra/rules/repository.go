package rules

import (
	"context"
	"sync"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// Repository implements ports.RuleRepository.
// Loads once and caches in memory.
// Validation happens in model.NewRule constructor — no separate validator (H1).
type Repository struct {
	loader *Loader
	mu     sync.Mutex
	cached *model.RuleSet
}

// NewRepository creates a RuleRepository backed by the YAML loader.
func NewRepository(loader *Loader) *Repository {
	return &Repository{loader: loader}
}

// LoadAll loads (or reloads) rules from disk. Cached after first load.
func (r *Repository) LoadAll(ctx context.Context) (*model.RuleSet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != nil {
		return r.cached, nil
	}

	rs, err := r.loader.LoadAll(ctx)
	if err != nil {
		return nil, err
	}

	r.cached = rs
	return rs, nil
}

// Watch returns a reload notification channel.
func (r *Repository) Watch(ctx context.Context) <-chan struct{} {
	return r.loader.Watch(ctx)
}
