package plugin

import (
	"context"
	"fmt"
	"net/rpc"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// Plugin is a universal go-plugin GRPCPlugin returning *grpc.ClientConn.
// No archetypes (A15) — one type fits all plugin binaries.
type Plugin struct{}

func (p *Plugin) Server(*goplugin.MuxBroker) (any, error)                  { return nil, nil }
func (p *Plugin) Client(_ *goplugin.MuxBroker, _ *rpc.Client) (any, error) { return nil, nil }

func (p *Plugin) GRPCServer(*goplugin.GRPCBroker, *grpc.Server) error { return nil }

func (p *Plugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return conn, nil
}

// PluginRegistrar implements usecase.PluginRegistrar.
// Adapts *grpc.ClientConn storage behind the Registrar abstraction.
type PluginRegistrar struct {
	registry *Registry
}

// NewPluginRegistrar creates a PluginRegistrar.
func NewPluginRegistrar(registry *Registry) *PluginRegistrar {
	return &PluginRegistrar{registry: registry}
}

// RegisterPlugin stores a plugin connection by name.
func (r *PluginRegistrar) RegisterPlugin(ctx context.Context, name string, conn any) error {
	grpcConn, ok := conn.(*grpc.ClientConn)
	if !ok {
		return fmt.Errorf("plugin connection type %T not supported", conn)
	}
	r.registry.Register(name, grpcConn)
	return nil
}

// ListPlugins returns all registered plugin names.
func (r *PluginRegistrar) ListPlugins(ctx context.Context) ([]string, error) {
	return r.registry.List(), nil
}
