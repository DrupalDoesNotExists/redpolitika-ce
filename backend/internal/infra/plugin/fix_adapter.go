package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/fix"
)

// FixAdapter adapts plugin FixService into domain FixFunctionProvider.
type FixAdapter struct {
	client fix.FixServiceClient
}

// NewFixAdapter creates a FixAdapter from the first plugin with fix.provider capability.
func NewFixAdapter(registry *Registry) ports.FixFunctionProvider {
	plugins := registry.FindByCapability(CapFixProvider)
	if len(plugins) == 0 {
		return nil
	}
	return &FixAdapter{client: fix.NewFixServiceClient(plugins[0].Conn)}
}

func (a *FixAdapter) Fix(ctx context.Context, text string, flag *model.Flag) (string, error) {
	resp, err := a.client.Fix(ctx, &fix.FixRequest{
		Text:      text,
		RuleId:    flag.RuleID().Value(),
		MatchText: flag.MatchText().Value(),
		Start:     uint32(flag.Span().Start()),
		End:       uint32(flag.Span().End()),
	})
	if err != nil {
		return "", fmt.Errorf("fix plugin: %w", err)
	}
	return resp.FixedText, nil
}
