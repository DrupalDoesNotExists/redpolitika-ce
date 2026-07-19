package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/proto/detect"
)

// DetectAdapter adapts plugin DetectService into domain DetectFunctionProvider.
type DetectAdapter struct {
	registry *Registry
}

// NewDetectAdapter creates a DetectAdapter if any plugin provides detect capability.
func NewDetectAdapter(registry *Registry) ports.DetectFunctionProvider {
	if len(registry.FindByCapability(CapDetectProvider)) == 0 {
		return nil
	}
	return &DetectAdapter{registry: registry}
}

func (a *DetectAdapter) Detect(ctx context.Context, text string, rule *model.Rule) ([]*model.Flag, error) {
	dm := rule.DetectMethod().Value()

	var info *PluginInfo

	if info = a.registry.LookupMethod(dm); info != nil {
		return a.callPlugin(ctx, info, text, rule)
	}

	if capID, ok := BuiltinMethods[dm]; ok && capID == CapDetectProvider {
		plugins := a.registry.FindByCapability(CapDetectProvider)
		if len(plugins) == 0 {
			return nil, fmt.Errorf("detect: no plugin with %q for method %q", CapDetectProvider, dm)
		}
		return a.callPlugin(ctx, plugins[0], text, rule)
	}

	return nil, fmt.Errorf("detect: no plugin registered method %q", dm)
}

func (a *DetectAdapter) callPlugin(ctx context.Context, info *PluginInfo, text string, rule *model.Rule) ([]*model.Flag, error) {
	client := detect.NewDetectServiceClient(info.Conn)
	resp, err := client.Detect(ctx, &detect.DetectRequest{
		Text:   text,
		RuleId: rule.ID().Value(),
	})
	if err != nil {
		return nil, fmt.Errorf("plugin %s detect: %w", info.Name, err)
	}

	paras := model.NewText(text).Paragraphs()

	var flags []*model.Flag
	for _, m := range resp.Matches {
		start := int(m.Start)
		end := int(m.End)
		paraIdx, startInPara := findParagraph(paras, start)
		matchStr := m.MatchText
		if matchStr == "" && startInPara >= 0 {
			matchStr = extractMatch(paras, paraIdx, startInPara, end-start)
		}
		occ := occurrenceInParagraph(paras, paraIdx, matchStr, startInPara)

		fid := model.NewFlagID(rule.ID().Value(), matchStr, paraIdx, occ)

		msg := rule.Suggestion().Value()
		if msg == "" {
			msg = "Match found: '" + matchStr + "'"
		}

		sp, spErr := model.NewSpan(start, end)
		if spErr != nil {
			continue
		}
		flag, err := model.NewFlag(
			fid.Value(), rule.ID().Value(), matchStr,
			rule.Suggestion().Value(), nil,
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

// findParagraph returns paragraph index and offset within paragraph for a byte offset.
func findParagraph(paras []string, offset int) (int, int) {
	at := 0
	for i, p := range paras {
		if offset >= at && offset < at+len(p) {
			return i, offset - at
		}
		at += len(p) + 2
	}
	if len(paras) > 0 && offset >= 0 {
		p := paras[len(paras)-1]
		return len(paras) - 1, min(offset, len(p))
	}
	return 0, 0
}

// extractMatch extracts matchStr from paragraph by offset and length.
func extractMatch(paras []string, paraIdx, start, length int) string {
	if paraIdx < 0 || paraIdx >= len(paras) {
		return ""
	}
	para := paras[paraIdx]
	if start+length > len(para) {
		length = len(para) - start
	}
	if start < 0 || start >= len(para) {
		return ""
	}
	return para[start : start+length]
}

// occurrenceInParagraph counts non-overlapping occurrences of matchStr in paragraph before start offset.
func occurrenceInParagraph(paras []string, paraIdx int, matchStr string, start int) int {
	if paraIdx < 0 || paraIdx >= len(paras) || matchStr == "" || start <= 0 {
		return 0
	}
	para := paras[paraIdx]
	occ := 0
	searchFrom := 0
	for searchFrom < start && searchFrom < len(para) {
		idx := indexOf(para, matchStr, searchFrom)
		if idx < 0 || idx >= start {
			break
		}
		occ++
		searchFrom = idx + len(matchStr)
	}
	return occ
}

func indexOf(s, sub string, from int) int {
	if from > len(s) {
		return -1
	}
	idx := indexAt(s, sub, from)
	return idx
}

func indexAt(s, sub string, start int) int {
	if start > len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
