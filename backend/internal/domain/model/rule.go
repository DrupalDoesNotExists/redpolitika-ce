package model

import (
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
		id: rid, severity: sev, category: cat, enabled: true,
		detectMethod: dm, detectNode: detectNode,
		fixNode: fixNode,
		suggestion: SuggestionFromString(suggestion),
		name: name, url: url,
		examples: examples, related: related,
	}, nil
}

// --- Getters ---

func (r *Rule) ID() RuleID                 { return r.id }
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

// IsClientSide — regex/wordlist leaf rules can run on client (A29).
// Composite tree rules return false (server-side only for now).
func (r *Rule) IsClientSide() bool {
	if !r.enabled || r.detectNode == nil {
		return false
	}
	switch r.detectNode.(type) {
	case *detect.RegexNode, *detect.WordlistNode:
		return true
	default:
		return false
	}
}

// HasMethodTree returns true if this rule uses a composable method tree.
// False for server-side-only rules (llm, plugin, ner, pos, function).
func (r *Rule) HasMethodTree() bool {
	return r.detectNode != nil
}

// Disable creates a disabled copy for override merging.
func (r *Rule) Disable() *Rule {
	return &Rule{
		id: r.id, severity: r.severity, category: r.category, enabled: false,
		detectMethod: r.detectMethod, detectNode: r.detectNode,
		fixNode: r.fixNode,
		suggestion: r.suggestion,
		name: r.name, url: r.url,
		examples: r.examples, related: r.related,
	}
}

// Detect runs this rule's detection against text.
// Uses the method tree for detection and fix computation.
// Returns matching Flags with per-paragraph occurrence tracking.
func (r *Rule) Detect(text *Text) []*Flag {
	if !r.enabled || r.detectNode == nil {
		return nil
	}

	paras := text.Paragraphs()
	var flags []*Flag

	for paraIdx, para := range paras {
		if para == "" {
			continue
		}

		matches := r.detectNode.Detect(para)
		if len(matches) == 0 {
			continue
		}

		occurrences := make(map[string]int)
		for _, m := range matches {
			start, end := m.Start, m.End
			if start < 0 || end > len(para) || start >= end {
				continue
			}
			matchStr := para[start:end]

			key := matchStr
			occ := occurrences[key]
			occurrences[key]++

			span, err := NewSpan(start, end)
			if err != nil {
				continue
			}

			flagID := NewFlagID(r.id.Value(), matchStr, paraIdx, occ)

			var autoFix *string
			if r.fixNode != nil {
				fixed := r.fixNode.Fix(matchStr)
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
	}
	return flags
}
