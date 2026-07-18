// Package plugin implements HashiCorp go-plugin lifecycle management (A15/A27).
package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "REDPOLITIKA_PLUGIN",
	MagicCookieValue: "ce_v1",
}

// Manager manages plugin lifecycle.
// No archetypes (A15) — plugins are opaque binaries speaking gRPC.
// The caller decides which service to call on the connection.
type Manager struct {
	logger  *zap.Logger
	clients map[string]*goplugin.Client
	mu      sync.Mutex
}

// NewManager creates a plugin manager.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger:  logger,
		clients: make(map[string]*goplugin.Client),
	}
}

// ScanDir discovers and starts all plugin binaries in dir.
// Any binary matching "redpolitika-*" is launched as a go-plugin process.
// Its *grpc.ClientConn is registered in Registry under the binary name.
func (m *Manager) ScanDir(ctx context.Context, reg *Registry, dir string) ([]string, error) {
	binaries, err := goplugin.Discover("redpolitika-*", dir)
	if err != nil {
		return nil, fmt.Errorf("plugin discover: %w", err)
	}

	var registered []string
	for _, bin := range binaries {
		name := filepath.Base(bin)
		if _, exists := m.clients[name]; exists {
			continue
		}

		p := &Plugin{}
		client := goplugin.NewClient(&goplugin.ClientConfig{
			HandshakeConfig:  handshake,
			Plugins:          goplugin.PluginSet{name: p},
			Cmd:              exec.CommandContext(ctx, bin),
			AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
			Logger: hclog.New(&hclog.LoggerOptions{
				Name:  "plugin." + name,
				Level: hclog.Info,
			}),
			StartTimeout: 30 * time.Second,
		})

		raw, err := client.Client()
		if err != nil {
			client.Kill()
			return nil, fmt.Errorf("plugin connect %s: %w", name, err)
		}
		plug, err := raw.Dispense(name)
		if err != nil {
			client.Kill()
			return nil, fmt.Errorf("plugin dispense %s: %w", name, err)
		}
		conn, ok := plug.(*grpc.ClientConn)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("plugin %s: unexpected type %T", name, plug)
		}

		m.mu.Lock()
		m.clients[name] = client
		m.mu.Unlock()

		reg.Register(name, conn)
		registered = append(registered, name)
		m.logger.Info("plugin started", zap.String("name", name))
	}

	return registered, nil
}

// StopAll kills all plugin processes.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.clients {
		client.Kill()
		delete(m.clients, name)
		m.logger.Info("plugin stopped", zap.String("name", name))
	}
}
