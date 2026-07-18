// Package service provides pure domain services orchestrating entities.
package service

import (
	"sort"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// FixApplier applies flag suggestions to text.
// Pure domain service — transforms Text by applying accepted Flag replacements
// and adjusting remaining flags' spans.
type FixApplier struct{}

// NewFixApplier creates a FixApplier.
func NewFixApplier() *FixApplier {
	return &FixApplier{}
}

// ApplyFixResult contains the result of applying a single fix.
type ApplyFixResult struct {
	Text           *model.Text
	RemainingFlags []*model.Flag
}

// Apply applies a flag's suggestion to the text.
// Returns modified text with the fix applied, and remaining flags with adjusted spans.
// Flags whose spans overlap the replaced area are excluded (conflict).
// Flags in paragraphs before the applied flag are untouched.
// Flags in the same paragraph after the replaced span are shifted by diff.
func (a *FixApplier) Apply(text *model.Text, flag *model.Flag, allFlags []*model.Flag) ApplyFixResult {
	paras := text.Paragraphs()
	paraIdx := flag.ParagraphIndex().Value()

	if paraIdx < 0 || paraIdx >= len(paras) {
		return ApplyFixResult{Text: text, RemainingFlags: allFlags}
	}

	para := paras[paraIdx]
	sp := flag.Span()
	start, end := sp.Start(), sp.End()

	if start < 0 || end > len(para) || start >= end {
		return ApplyFixResult{Text: text, RemainingFlags: allFlags}
	}

	// Apply replacement
	suggestion := flag.Suggestion().Value()
	newPara := para[:start] + suggestion + para[end:]
	delta := len(suggestion) - (end - start)

	// Adjust remaining flags
	var remaining []*model.Flag
	for _, f := range allFlags {
		if f == flag || f.State() == model.FlagStateRejected {
			continue
		}

		fParaIdx := f.ParagraphIndex().Value()
		fSp := f.Span()

		if fParaIdx == paraIdx {
			if fSp.Start() >= end {
				// After replaced area — shift by delta
				f.AdjustSpan(delta, paraIdx)
				remaining = append(remaining, f)
			} else if fSp.End() <= start {
				// Before replaced area — unchanged
				remaining = append(remaining, f)
			} else {
				// Overlaps replaced area — conflict, skip
				continue
			}
		} else {
			// Different paragraph — unchanged
			remaining = append(remaining, f)
		}
	}

	// Rebuild text
	newParas := make([]string, len(paras))
	copy(newParas, paras)
	newParas[paraIdx] = newPara

	fullText := buildText(newParas)

	return ApplyFixResult{
		Text:           model.NewText(fullText),
		RemainingFlags: remaining,
	}
}

// ApplyAll applies multiple flags, sorted by position (paragraph then span).
// First-in-position flags are applied first; conflicts skip later ones.
func (a *FixApplier) ApplyAll(text *model.Text, flags []*model.Flag) ApplyFixResult {
	if len(flags) == 0 {
		return ApplyFixResult{Text: text, RemainingFlags: nil}
	}

	// Stable sort by paragraph then span start
	sorted := make([]*model.Flag, len(flags))
	copy(sorted, flags)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi, pj := sorted[i].ParagraphIndex().Value(), sorted[j].ParagraphIndex().Value()
		if pi != pj {
			return pi < pj
		}
		return sorted[i].Span().Start() < sorted[j].Span().Start()
	})

	current := text
	var pending []*model.Flag

	for _, f := range sorted {
		if f.State() == model.FlagStateRejected {
			continue
		}
		result := a.Apply(current, f, pending)
		current = result.Text
		pending = result.RemainingFlags
	}

	return ApplyFixResult{Text: current, RemainingFlags: pending}
}

func buildText(paras []string) string {
	if len(paras) == 0 {
		return ""
	}
	var b []byte
	for i, p := range paras {
		if i > 0 {
			b = append(b, '\n', '\n')
		}
		b = append(b, p...)
	}
	return string(b)
}
