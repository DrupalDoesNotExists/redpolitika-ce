package service

import (
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// RuleEngine detects rule violations in text.
// Delegates to Rule.Detect() for internal tree-based rules.
// Rules with external detect nodes (plugin-delegated) are handled by the use case.
type RuleEngine struct{}

// NewRuleEngine creates a RuleEngine.
func NewRuleEngine() *RuleEngine { return &RuleEngine{} }

// DetectResult groups flags produced by one rule.
type DetectResult struct {
	RuleID model.RuleID
	Flags  []*model.Flag
}

// Detect runs all client-side rules against text. Returns per-rule results.
// Server-side rules (llm, plugin, ner, pos) return nil and are handled externally.
func (e *RuleEngine) Detect(text *model.Text, rules *model.RuleSet) []DetectResult {
	var results []DetectResult

	for _, rule := range rules.Rules() {
		if !rule.IsEnabled() {
			continue
		}
		if detect.IsExternal(rule.DetectNode()) {
			continue // plugin-delegated — handled by usecase
		}
		if rule.Severity().Value() == 0 {
			continue // helper rules for ref only — no flags produced
		}

		flags := rule.Detect(text)
		if len(flags) > 0 {
			results = append(results, DetectResult{RuleID: rule.ID(), Flags: flags})
		}
	}
	return results
}
