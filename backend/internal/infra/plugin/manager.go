// Package plugin implements HashiCorp go-plugin lifecycle management (A15/A27).
package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/configutil"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/fix"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/infra/rules"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/identity"
)

var handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "REDPOLITIKA_PLUGIN",
	MagicCookieValue: "ce_v1",
}

// Manager manages plugin lifecycle.
// No archetypes (A15) — plugins are opaque binaries speaking gRPC.
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
// After handshake, calls GetCapabilities to discover extension points (A15/A27).
// Per-plugin CLI flags passed via PLUGIN_<NAME>_FLAGS env vars (e.g. PLUGIN_PAGES_FLAGS).
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

		// Load per-plugin flags from PLUGIN_<NAME>_FLAGS env var.
		// e.g. redpolitika-pages → PLUGIN_PAGES_FLAGS
		pluginSuffix := strings.TrimPrefix(name, "redpolitika-")
		envKey := "PLUGIN_" + strings.ToUpper(strings.ReplaceAll(pluginSuffix, "-", "_")) + "_FLAGS"
		var pluginArgs []string
		if flags := os.Getenv(envKey); flags != "" {
			pluginArgs = strings.Fields(flags)
			m.logger.Info("plugin flags", zap.String("name", name), zap.String("env", envKey), zap.Strings("args", pluginArgs))
		}

		p := &Plugin{}
		client := goplugin.NewClient(&goplugin.ClientConfig{
			HandshakeConfig:  handshake,
			Plugins:          goplugin.PluginSet{name: p},
			Cmd:              exec.CommandContext(context.Background(), bin, pluginArgs...),
			AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
			Logger:           NewZapHCLogAdapter(m.logger.Named("plugin." + name)),
			StartTimeout:     30 * time.Second,
		})

		raw, err := client.Client()
		if err != nil {
			client.Kill()
			m.logger.Error("plugin connect failed, skipping", zap.String("name", name), zap.Error(err))
			continue
		}
		plug, err := raw.Dispense(name)
		if err != nil {
			client.Kill()
			m.logger.Error("plugin dispense failed, skipping", zap.String("name", name), zap.Error(err))
			continue
		}
		conn, ok := plug.(*grpc.ClientConn)
		if !ok {
			client.Kill()
			m.logger.Error("plugin unexpected type, skipping", zap.String("name", name), zap.String("type", fmt.Sprintf("%T", plug)))
			continue
		}

		// Call GetCapabilities to discover extension points (A15/A27)
		info := &PluginInfo{Name: name, Conn: conn}
		capCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		idClient := identity.NewPluginIdentityClient(conn)
		resp, err := idClient.GetCapabilities(capCtx, &identity.GetCapabilitiesRequest{})
		cancel()
		if err != nil {
			m.logger.Warn("plugin GetCapabilities failed, registering without capabilities",
				zap.String("name", name), zap.Error(err))
		} else {
			info.Capabilities = resp.Capabilities
			info.Methods = resp.Methods
			m.logger.Info("plugin capabilities",
				zap.String("name", name),
				zap.Strings("capabilities", info.Capabilities),
				zap.Strings("methods", info.Methods))

			// Register each scoped method as an ExternalNode in the detect registry
			// so the parser can build them and the adapter can extract ConfigJSON.
			for _, method := range info.Methods {
				methodName := method // capture
				pluginName := name
				detect.Register(methodName, func(args map[string]interface{}, children []detect.Node) (detect.Node, error) {
					return &detect.ExternalNode{
						PluginName: pluginName,
						ConfigJSON: configutil.MarshalConfig(args),
					}, nil
				})
				fix.Register(methodName, func(args map[string]interface{}, children []fix.Node) (fix.Node, error) {
					return &fix.ExternalFixNode{
						PluginName: pluginName,
						MethodName: methodName,
						ConfigJSON: configutil.MarshalConfig(args),
					}, nil
				})
				rules.RegisterKnownMethod(methodName)
				m.logger.Debug("registered plugin detect method",
					zap.String("plugin", name), zap.String("method", methodName))
			}
		}

		m.mu.Lock()
		m.clients[name] = client
		m.mu.Unlock()

		reg.RegisterPlugin(info)
		registered = append(registered, name)
		m.logger.Info("plugin started", zap.String("name", name))
	}

	return registered, nil
}

// Status returns names of currently running plugins.
func (m *Manager) Status() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
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
