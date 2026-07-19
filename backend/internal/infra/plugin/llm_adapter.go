package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/llm"
)

// LLMAdapter adapts plugin LLMService into domain LLMProvider.
type LLMAdapter struct {
	registry *Registry
}

// NewLLMAdapter creates an LLMAdapter.
// Plugin lookup deferred to call time — plugins load during OnStart, after construction.
func NewLLMAdapter(registry *Registry) ports.LLMProvider {
	return &LLMAdapter{registry: registry}
}

func (a *LLMAdapter) CheckText(ctx context.Context, text string, rule *model.Rule) ([]*model.Flag, error) {
	plugins := a.registry.FindByCapability(CapLLMProvider)
	if len(plugins) == 0 {
		return nil, fmt.Errorf("llm: no plugin with %q capability", CapLLMProvider)
	}
	client := llm.NewLLMServiceClient(plugins[0].Conn)
	resp, err := client.Analyze(ctx, &llm.AnalyzeRequest{
		Text:   text,
		RuleId: rule.ID().Value(),
	})
	if err != nil {
		return nil, fmt.Errorf("llm plugin: %w", err)
	}

	paras := model.NewText(text).Paragraphs()
	var flags []*model.Flag

	for _, m := range resp.Matches {
		start := int(m.Start)
		end := int(m.End)
		matchStr := m.MatchText
		sugg := m.Suggestion
		if sugg == "" {
			sugg = rule.Suggestion().Value()
		}

		paraIdx, startInPara := findParagraph(paras, start)
		if matchStr == "" && startInPara >= 0 {
			length := end - start
			if length <= 0 {
				length = len(matchStr)
			}
			matchStr = extractMatch(paras, paraIdx, startInPara, length)
		}
		occ := occurrenceInParagraph(paras, paraIdx, matchStr, startInPara)

		sp, spErr := model.NewSpan(start, end)
		if spErr != nil {
			continue
		}

		msg := sugg
		if msg == "" {
			msg = "Match found: '" + matchStr + "'"
		}

		flag, err := model.NewFlag(
			model.NewFlagID(rule.ID().Value(), matchStr, paraIdx, occ).Value(),
			rule.ID().Value(), matchStr,
			sugg, nil,
			rule.Severity().Value(), rule.Category().Value(),
			occ, msg,
			sp, paraIdx,
			rule.Name(), rule.URL(),
			rule.Examples(), rule.Related(),
		)
		if err != nil {
			continue
		}
		flags = append(flags, flag)
	}
	return flags, nil
}
