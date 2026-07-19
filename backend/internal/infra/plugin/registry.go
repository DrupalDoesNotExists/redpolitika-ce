package plugin

import (
	"sync"

	"google.golang.org/grpc"
)

// Registry holds PluginInfo for all running plugins.
// Flat map, no archetypes (A15) — any plugin is defined by its capabilities.
type Registry struct {
	mu      sync.RWMutex
	infoMap map[string]*PluginInfo
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		infoMap: make(map[string]*PluginInfo),
	}
}

// Register stores a plugin connection under its binary name (legacy, no capabilities).
func (r *Registry) Register(name string, conn *grpc.ClientConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.infoMap[name]; ok {
		existing.Conn = conn
		return
	}
	r.infoMap[name] = &PluginInfo{Name: name, Conn: conn}
}

// RegisterPlugin stores full plugin metadata.
func (r *Registry) RegisterPlugin(info *PluginInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.infoMap[info.Name] = info
}

// Get returns PluginInfo by binary name, or nil.
func (r *Registry) Get(name string) *PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.infoMap[name]
}

// Conn returns a plugin's gRPC connection by binary name.
func (r *Registry) Conn(name string) *grpc.ClientConn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info := r.infoMap[name]
	if info == nil {
		return nil
	}
	return info.Conn
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.infoMap, name)
}

// List returns all registered plugin binary names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.infoMap))
	for n := range r.infoMap {
		names = append(names, n)
	}
	return names
}

// LookupMethod finds which plugin registered a detect/fix method name (A37).
// Returns empty string if not found.
func (r *Registry) LookupMethod(name string) *PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, info := range r.infoMap {
		for _, m := range info.Methods {
			if m == name {
				return info
			}
		}
	}
	return nil
}

// FindByCapability returns all plugins that have a given capability.
func (r *Registry) FindByCapability(cap string) []*PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*PluginInfo
	for _, info := range r.infoMap {
		for _, c := range info.Capabilities {
			if c == cap {
				out = append(out, info)
			}
		}
	}
	return out
}
