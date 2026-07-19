package main

import (
	"context"

	identitypb "github.com/drupaldoesnotexists/redpolitika/ce/plugins/pages/proto/identity"
)

// identityService implements PluginIdentityServer for static.pages capability.
type identityService struct {
	identitypb.UnimplementedPluginIdentityServer
}

// GetCapabilities returns the static.pages extension point capability.
func (s *identityService) GetCapabilities(_ context.Context, _ *identitypb.GetCapabilitiesRequest) (*identitypb.GetCapabilitiesResponse, error) {
	return &identitypb.GetCapabilitiesResponse{
		Capabilities: []string{"static.pages"},
		Methods:      []string{},
	}, nil
}
