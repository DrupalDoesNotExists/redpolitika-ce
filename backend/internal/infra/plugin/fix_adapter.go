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

func (a *FixAdapter) Fix(ctx context.Context, text string, flag *model.Flag, config string, methodName string) (string, error) {
	info := a.registry.LookupMethod(methodName)
	if info == nil {
		return "", fmt.Errorf("fix: no plugin registered method %q", methodName)
	}
	client := fix.NewFixServiceClient(info.Conn)
	resp, err := client.Fix(ctx, &fix.FixRequest{
		Text:       text,
		RuleId:     flag.RuleID().Value(),
		MatchText:  flag.MatchText().Value(),
		Start:      uint32(flag.Span().Start()),
		End:        uint32(flag.Span().End()),
		Config:     config,
		MethodName: methodName,
	})
	if err != nil {
		return "", fmt.Errorf("fix plugin %s: %w", info.Name, err)
	}
	return resp.FixedText, nil
}
