// Package session implements in-memory SessionRepository per A33.
package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// MemoryStore implements ports.SessionRepository using an in-memory sync.Map.
// Sessions are keyed by SessionID. TTL/eviction is not yet implemented.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[model.SessionID]*model.Session
}

// NewMemoryStore creates an in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[model.SessionID]*model.Session),
	}
}

// Save stores a session. Overwrites existing session with same ID.
func (s *MemoryStore) Save(ctx context.Context, session *model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[session.ID()] = session
	return nil
}

// FindByID retrieves a session. Returns error if not found.
func (s *MemoryStore) FindByID(ctx context.Context, id model.SessionID) (*model.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.data[id]
	if !ok {
		return nil, &Error{Op: "FindByID", Message: fmt.Sprintf("session %s not found", id)}
	}
	return session, nil
}

// Delete removes a session.
func (s *MemoryStore) Delete(ctx context.Context, id model.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

// ListActive returns all session IDs.
func (s *MemoryStore) ListActive(ctx context.Context) ([]model.SessionID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]model.SessionID, 0, len(s.data))
	for id := range s.data {
		ids = append(ids, id)
	}
	return ids, nil
}
