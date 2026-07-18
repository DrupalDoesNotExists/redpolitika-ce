package plugin

import (
	"sync"

	"google.golang.org/grpc"
)

// Registry holds *grpc.ClientConn for all running plugins.
// Flat map, no archetypes (A15) — any plugin can provide any service.
// Callers create typed gRPC clients from the connection as needed.
type Registry struct {
	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		conns: make(map[string]*grpc.ClientConn),
	}
}

// Register stores a plugin connection under its binary name.
func (r *Registry) Register(name string, conn *grpc.ClientConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[name] = conn
}

// Conn returns a plugin's gRPC connection by binary name.
func (r *Registry) Conn(name string) *grpc.ClientConn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conns[name]
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, name)
}

// List returns all registered plugin binary names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.conns))
	for n := range r.conns {
		names = append(names, n)
	}
	return names
}
