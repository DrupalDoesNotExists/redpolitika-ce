// Package usecase implements application use-cases that orchestrate the domain.
package usecase

import (
	"context"
	"fmt"
)

// PluginRegistrar defines the port for registering external plugins.
// Implementation lives in infra/plugin; this abstraction keeps the use case
// decoupled from gRPC types. Per A15: no archetypes, flat registration.
type PluginRegistrar interface {
	RegisterPlugin(ctx context.Context, name string, conn any) error
	ListPlugins(ctx context.Context) ([]string, error)
}

// RegisterPluginUseCase handles the registration of external plugins.
// In CE, plugins are launched by Manager.ScanDir and registered automatically.
// This use case provides the API for dynamic registration (used by EE).
type RegisterPluginUseCase struct {
	registrar PluginRegistrar
}

// NewRegisterPluginUseCase creates a RegisterPluginUseCase.
func NewRegisterPluginUseCase(registrar PluginRegistrar) *RegisterPluginUseCase {
	return &RegisterPluginUseCase{registrar: registrar}
}

// RegisterRequest contains the plugin registration data.
type RegisterRequest struct {
	Name string
	Conn any // *grpc.ClientConn or adapter
}

// Execute registers a plugin in the registry.
func (uc *RegisterPluginUseCase) Execute(ctx context.Context, req RegisterRequest) error {
	if req.Name == "" {
		return &Error{Op: "RegisterPlugin", Message: "plugin name must not be empty"}
	}
	if req.Conn == nil {
		return &Error{Op: "RegisterPlugin", Message: "plugin connection must not be nil"}
	}
	if err := uc.registrar.RegisterPlugin(ctx, req.Name, req.Conn); err != nil {
		return &Error{Op: "RegisterPlugin", Message: fmt.Sprintf("register %s", req.Name), Err: err}
	}
	return nil
}

// List returns all registered plugin names.
func (uc *RegisterPluginUseCase) List(ctx context.Context) ([]string, error) {
	return uc.registrar.ListPlugins(ctx)
}
