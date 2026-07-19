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
	registry *Registry
}

// NewFixAdapter creates a FixAdapter.
// Plugin lookup deferred to call time — plugins load during OnStart, after construction.
func NewFixAdapter(registry *Registry) ports.FixFunctionProvider {
	return &FixAdapter{registry: registry}
}

func (a *FixAdapter) Fix(ctx context.Context, text string, flag *model.Flag) (string, error) {
	plugins := a.registry.FindByCapability(CapFixProvider)
	if len(plugins) == 0 {
		return "", fmt.Errorf("fix: no plugin with %q capability", CapFixProvider)
	}
	client := fix.NewFixServiceClient(plugins[0].Conn)
	resp, err := client.Fix(ctx, &fix.FixRequest{
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
