package service

import (
	"math"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// ScoreCalculator computes cleanliness and readability scores.
// Formula: 10.0 - (total_severity_penalty / normalised_words) * 10.0
// Normalisation: per 100 words.
type ScoreCalculator struct{}

// NewScoreCalculator creates a ScoreCalculator.
func NewScoreCalculator() *ScoreCalculator { return &ScoreCalculator{} }

// ScoredFlag pairs a flag with its rule's severity for scoring.
type ScoredFlag struct {
	Severity model.Severity
	Flag     *model.Flag
}

// ComputeFromMap computes scores from category-grouped scored flags.
func (c *ScoreCalculator) ComputeFromMap(
	grouped map[model.Category][]ScoredFlag,
	wordCount model.WordCount,
) (cleanliness, readability model.Score) {
	if wordCount.IsZero() {
		return model.NewScoreUnsafe(model.MaxScore), model.NewScoreUnsafe(model.MaxScore)
	}

	// Floor at 100 words so short texts don't get disproportionately penalised
	effectiveWords := float64(wordCount.Value())
	if effectiveWords < 100 {
		effectiveWords = 100
	}
	norm := effectiveWords / 100.0

	for category, flags := range grouped {
		var penalty float64
		for _, sf := range flags {
			if !sf.Flag.IsPending() {
				continue
			}
			penalty += float64(sf.Severity.Value())
		}

		score := model.MaxScore - penalty/norm
		if score < 0 {
			score = 0
		}

		switch {
		case category == model.CategoryCleanliness:
			cleanliness = model.NewScoreUnsafe(math.Round(score*10) / 10)
		case category == model.CategoryReadability:
			readability = model.NewScoreUnsafe(math.Round(score*10) / 10)
		}
	}

	// Default to max when category had no flags
	if cleanliness.Value() == 0 && grouped[model.CategoryCleanliness] == nil {
		cleanliness = model.NewScoreUnsafe(model.MaxScore)
	}
	if readability.Value() == 0 && grouped[model.CategoryReadability] == nil {
		readability = model.NewScoreUnsafe(model.MaxScore)
	}

	return
}

// Compute computes scores from a flat flag slice, looking up rule details from a rule index.
// Convenience wrapper for use-cases.
func (c *ScoreCalculator) Compute(flags []*model.Flag, ruleIndex map[model.RuleID]*model.Rule, wordCount model.WordCount) (cleanliness, readability model.Score) {
	grouped := make(map[model.Category][]ScoredFlag)

	for _, f := range flags {
		rule, ok := ruleIndex[f.RuleID()]
		if !ok {
			continue
		}
		if !f.IsPending() {
			continue
		}
		cat := rule.Category()
		grouped[cat] = append(grouped[cat], ScoredFlag{Severity: rule.Severity(), Flag: f})
	}

	return c.ComputeFromMap(grouped, wordCount)
}
