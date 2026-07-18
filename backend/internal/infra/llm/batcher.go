// Package llm provides the LLM rule batcher per A16/A16.1.
// Core only orchestrates; the LLM provider is a plugin via extension points.
package llm

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// Batcher collects LLM-type rules and dispatches them to the LLM provider.
// Per A16/A16.1: core only orchestrates, provider is a plugin.
type Batcher struct {
	logger *zap.Logger
}

// NewBatcher creates an LLM rule batcher.
func NewBatcher(logger *zap.Logger) *Batcher {
	return &Batcher{logger: logger}
}

// BatchRequest groups an LLM rule with the text to analyze.
type BatchRequest struct {
	Rule *model.Rule
	Text string
}

// BatchResult holds the flags produced by an LLM rule (or error).
type BatchResult struct {
	RuleID string
	Flags  []*model.Flag
	Error  error
}

// LLMRules filters rules with detect method "llm" from a RuleSet.
func LLMRules(rs *model.RuleSet) []*model.Rule {
	var out []*model.Rule
	for _, r := range rs.Rules() {
		if r.DetectMethod() == model.DetectMethodLLM {
			out = append(out, r)
		}
	}
	return out
}

// NewBatchRequests creates a BatchRequest per LLM rule for the given text.
func NewBatchRequests(text string, rules []*model.Rule) []BatchRequest {
	reqs := make([]BatchRequest, 0, len(rules))
	for _, r := range rules {
		reqs = append(reqs, BatchRequest{Rule: r, Text: text})
	}
	return reqs
}

// Execute sends each LLM request to the provider and collects results.
// Each LLM rule is analyzed individually (future: group by provider/model).
// Returns one BatchResult per request, preserving order.
func (b *Batcher) Execute(
	ctx context.Context,
	provider ports.LLMProvider,
	requests []BatchRequest,
) []BatchResult {
	if provider == nil {
		b.logger.Warn("llm batcher: no provider configured, skipping LLM rules")
		return nil
	}
	if len(requests) == 0 {
		return nil
	}

	results := make([]BatchResult, 0, len(requests))
	for _, req := range requests {
		select {
		case <-ctx.Done():
			results = append(results, BatchResult{
				RuleID: req.Rule.ID().Value(),
				Error:  fmt.Errorf("llm batcher: context cancelled: %w", ctx.Err()),
			})
			continue
		default:
		}

		flags, err := provider.CheckText(ctx, req.Text, req.Rule)
		if err != nil {
			b.logger.Error("llm batcher: provider error",
				zap.String("rule", req.Rule.ID().Value()),
				zap.Error(err),
			)
			results = append(results, BatchResult{
				RuleID: req.Rule.ID().Value(),
				Error:  fmt.Errorf("llm batcher: provider CheckText: %w", err),
			})
			continue
		}

		results = append(results, BatchResult{
			RuleID: req.Rule.ID().Value(),
			Flags:  flags,
		})
	}

	return results
}
