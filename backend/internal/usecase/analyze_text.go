package usecase

import (
	"context"
	"strings"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/service"
	"go.uber.org/zap"
)

// AnalyzeTextUseCase orchestrates text analysis: rules → engine → scoring → cache.
// Handles client-side rules (regex/wordlist/expr/pattern) via RuleEngine,
// server-side rules (llm/plugin/ner/pos) via optional extension point providers (A16/A27).
type AnalyzeTextUseCase struct {
	ruleRepo    ports.RuleRepository
	sessionRepo ports.SessionRepository
	cache       ports.CacheRepository
	engine      *service.RuleEngine
	calculator  *service.ScoreCalculator
	llmProvider ports.LLMProvider            // optional (A16)
	detectFunc  ports.DetectFunctionProvider // optional (A27)
	logger      *zap.Logger
}

// NewAnalyzeTextUseCase creates an AnalyzeTextUseCase.
// llmProvider and detectFunc are optional (may be nil for CE without plugins).
func NewAnalyzeTextUseCase(
	ruleRepo ports.RuleRepository,
	sessionRepo ports.SessionRepository,
	cache ports.CacheRepository,
	engine *service.RuleEngine,
	calculator *service.ScoreCalculator,
	llmProvider ports.LLMProvider,
	detectFunc ports.DetectFunctionProvider,
	logger *zap.Logger,
) *AnalyzeTextUseCase {
	return &AnalyzeTextUseCase{
		ruleRepo: ruleRepo, sessionRepo: sessionRepo, cache: cache,
		engine: engine, calculator: calculator,
		llmProvider: llmProvider, detectFunc: detectFunc,
		logger: logger,
	}
}

// AnalyzeRequest carries the input for analysis.
type AnalyzeRequest struct {
	Text      string
	SessionID model.SessionID
}

// AnalyzeResult is the output of analysis.
type AnalyzeResult struct {
	Analysis       *model.Analysis
	SessionUpdated bool
}

// Execute runs the full analysis pipeline.
func (uc *AnalyzeTextUseCase) Execute(ctx context.Context, req AnalyzeRequest) (*AnalyzeResult, error) {
	text := model.NewText(req.Text)
	textHash := text.Hash()

	// Load rules
	ruleset, err := uc.ruleRepo.LoadAll(ctx)
	if err != nil {
		return nil, &Error{Op: "AnalyzeText", Message: "load rules", Err: err}
	}

	configHash := ruleset.ConfigHash()

	// Check cache
	cached, err := uc.cache.Get(ctx, textHash, configHash.Value())
	if err == nil && cached != nil {
		return &AnalyzeResult{Analysis: cached, SessionUpdated: false}, nil
	}

	// Phase 1: client-side rules via RuleEngine (regex/wordlist/expr/pattern)
	engineResults := uc.engine.Detect(text, ruleset)

	// Phase 2: server-side rules — only rules with detectNode==nil need external providers
	var allFlags []*model.Flag

	for _, dr := range engineResults {
		allFlags = append(allFlags, dr.Flags...)
	}

	for _, rule := range ruleset.Rules() {
		if rule.DetectNode() != nil {
			continue // handled by engine
		}
		dm := rule.DetectMethod().Value()
		switch {
		case dm == "llm":
			if uc.llmProvider != nil {
				flags, err := uc.llmProvider.CheckText(ctx, req.Text, rule)
				if err != nil {
					uc.logger.Error("analyze: llm provider error", zap.String("rule", rule.ID().Value()), zap.Error(err))
					continue
				}
				allFlags = append(allFlags, flags...)
			}
		case dm == "plugin" || dm == "ner" || dm == "pos" || dm == "function" || strings.Contains(dm, "/"):
			if uc.detectFunc != nil {
				flags, err := uc.detectFunc.Detect(ctx, req.Text, rule)
				if err != nil {
					uc.logger.Error("analyze: detect provider error", zap.String("rule", rule.ID().Value()), zap.Error(err))
					continue
				}
				allFlags = append(allFlags, flags...)
			}
		}
	}

	// Build rule index for scoring
	ruleIndex := make(map[model.RuleID]*model.Rule)
	for _, r := range ruleset.Rules() {
		ruleIndex[r.ID()] = r
	}

	// Compute scores
	cleanliness, readability := uc.calculator.Compute(allFlags, ruleIndex, model.WordCountFromInt(text.WordCount()))

	// Build analysis
	analysis := model.NewAnalysis(textHash, configHash.Value(), allFlags, cleanliness, readability)

	// Cache result
	if err := uc.cache.Set(ctx, textHash, configHash.Value(), analysis); err != nil {
		_ = err // non-fatal
	}

	// Update session
	if req.SessionID.Value() != "" {
		session, err := uc.sessionRepo.FindByID(ctx, req.SessionID)
		if err != nil {
			session, err = model.NewSession(req.SessionID.String(), text, configHash.Value())
			if err != nil {
				return nil, &Error{Op: "AnalyzeText", Message: "create session", Err: err}
			}
		}
		for _, f := range allFlags {
			session.AddFlag(f)
		}
		session.SetScores(cleanliness, readability)
		if err := uc.sessionRepo.Save(ctx, session); err != nil {
			return nil, &Error{Op: "AnalyzeText", Message: "save session", Err: err}
		}
		return &AnalyzeResult{Analysis: analysis, SessionUpdated: true}, nil
	}

	return &AnalyzeResult{Analysis: analysis, SessionUpdated: false}, nil
}
