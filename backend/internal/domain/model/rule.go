package model

import (
	"strings"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/fix"
)

// Examples contains before/after text samples for a rule.
type Examples struct {
	Bad  []string `json:"bad,omitempty"`
	Good []string `json:"good,omitempty"`
}

// Related links to related rules or documentation.
type Related struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Rule — aggregate root, immutable after loading.
// Has a composable Detect/Fix method tree per SPEC §8 (A26/Q33).
type Rule struct {
	id            RuleID
	priority      int          // higher = runs first (0 default)
	severity      Severity
	category      Category
	enabled       bool
	detectMethod  DetectMethod // "regex", "wordlist", "llm", "plugin", etc. — "" for tree-based
	detectNode    detect.Node  // nil for server-side rules (llm/plugin/ner/pos)
	fixNode       fix.Node     // nil when no fix method
	suggestion    Suggestion
	name          string
	url           string
	examples      Examples
	related       []Related
	clientSide    bool
}

// NewRule creates a Rule from raw values.
// detectMethod: flat method name ("regex", "wordlist", "llm", "plugin", "ner", "pos", "function", "expr").
// detectNode: compiled detection tree (nil for server-side methods like llm/plugin).
// fixNode: compiled fix tree (nil if no autofix).
func NewRule(
	id string,
	severity int,
	category string,
	detectMethod string,
	detectNode detect.Node,
	fixNode fix.Node,
	priority int,
	suggestion string,
	name string,
	url string,
	examples Examples,
	related []Related,
) (*Rule, error) {
	rid, err := RuleIDFromString(id)
	if err != nil {
		return nil, err
	}
	sev, err := SeverityFromInt(severity)
	if err != nil {
		return nil, err
	}
	cat, err := CategoryFromString(category)
	if err != nil {
		return nil, err
	}
	var dm DetectMethod
	if detectMethod != "" {
		dm, err = DetectMethodFromString(detectMethod)
		if err != nil {
			return nil, err
		}
	} else {
		dm = DetectMethod{value: ""}
	}

	return &Rule{
		id: rid, priority: priority, severity: sev, category: cat, enabled: true,
		detectMethod: dm, detectNode: detectNode,
		fixNode: fixNode,
		suggestion: SuggestionFromString(suggestion),
		name: name, url: url,
		examples: examples, related: related,
		clientSide: detectNode != nil && detect.IsNodeSync(detectNode),
	}, nil
}

// --- Getters ---

func (r *Rule) ID() RuleID                 { return r.id }
func (r *Rule) Priority() int              { return r.priority }
func (r *Rule) Severity() Severity         { return r.severity }
func (r *Rule) Category() Category         { return r.category }
func (r *Rule) IsEnabled() bool            { return r.enabled }
func (r *Rule) DetectMethod() DetectMethod { return r.detectMethod }
func (r *Rule) DetectNode() detect.Node    { return r.detectNode }
func (r *Rule) FixNode() fix.Node          { return r.fixNode }
func (r *Rule) Suggestion() Suggestion     { return r.suggestion }
func (r *Rule) Name() string               { return r.name }
func (r *Rule) URL() string                { return r.url }
func (r *Rule) Examples() Examples         { return r.examples }
func (r *Rule) Related() []Related         { return r.related }

// IsClientSide checks if rule can run client-side (A29).
func (r *Rule) IsClientSide() bool {
	return r.enabled && r.detectNode != nil && r.clientSide
}

// HasMethodTree returns true if this rule uses a composable method tree.
// False for server-side-only rules (llm, plugin, ner, pos, function).
func (r *Rule) HasMethodTree() bool {
	return r.detectNode != nil
}

// Disable creates a disabled copy for override merging.
func (r *Rule) Disable() *Rule {
	return &Rule{
		id: r.id, priority: r.priority, severity: r.severity, category: r.category, enabled: false,
		detectMethod: r.detectMethod, detectNode: r.detectNode,
		fixNode: r.fixNode,
		suggestion: r.suggestion,
		name: r.name, url: r.url,
		examples: r.examples, related: r.related,
		clientSide: r.clientSide,
	}
}

// Detect runs this rule's detection against text.
// Uses the method tree for detection and fix computation.
// Returns matching Flags with per-paragraph occurrence tracking.
// Inline suppressions (<!-- rp:disable -->) are honoured (E3).
func (r *Rule) Detect(text *Text) []*Flag {
	if !r.enabled || r.detectNode == nil {
		return nil
	}

	paras := text.Paragraphs()
	var flags []*Flag

	paraOffset := 0
	for paraIdx, para := range paras {
		if para == "" {
			// empty paragraph still advances offset past the separator (except last)
			if paraIdx < len(paras)-1 {
				paraOffset += 2 // \n\n
			}
			continue
		}

		matches := r.detectNode.Detect(para)
		if len(matches) == 0 {
			paraOffset += len(para)
			if paraIdx < len(paras)-1 {
				paraOffset += 2
			}
			continue
		}

		for _, m := range matches {
			start, end := m.Start, m.End
			if start < 0 || end > len(para) || start >= end {
				continue
			}

			matchStr := para[start:end]
			// A23: occurrence = N-th copy of match_text in the paragraph
			// (not among detect hits). Frontend resolves span via indexOf.
			occ := occurrenceInParagraph(para, matchStr, start)

			span, err := NewSpan(start, end)
			if err != nil {
				continue
			}

			flagID := NewFlagID(r.id.Value(), matchStr, paraIdx, occ)

			var autoFix *string
			if r.fixNode != nil {
				ctx := fix.Context{
					Text:   para,
					Start:  start,
					End:    end,
					Groups: m.Groups,
				}
				fixed := r.fixNode.Fix(matchStr, ctx)
				autoFix = &fixed
			}

			msg := r.suggestion.Value()
			if msg == "" {
				msg = "Match found: '" + matchStr + "'"
			}

			flag, err := NewFlag(
				flagID.Value(), r.id.Value(), matchStr,
				r.suggestion.Value(), autoFix,
				r.severity.Value(), r.category.Value(),
				occ, msg, span, paraIdx,
				r.name, r.url, r.examples, r.related,
			)
			if err != nil {
				continue
			}
			flags = append(flags, flag)
		}

		paraOffset += len(para)
		if paraIdx < len(paras)-1 {
			paraOffset += 2
		}
	}
	return flags
}

// occurrenceInParagraph returns how many times matchStr appears in para
// before byte offset start (non-overlapping, same as frontend indexOf loop).
func occurrenceInParagraph(para, matchStr string, start int) int {
	if matchStr == "" || start <= 0 {
		return 0
	}
	occ := 0
	searchFrom := 0
	for searchFrom < start {
		rel := strings.Index(para[searchFrom:], matchStr)
		if rel < 0 {
			break
		}
		abs := searchFrom + rel
		if abs >= start {
			break
		}
		occ++
		searchFrom = abs + len(matchStr)
	}
	return occ
}
