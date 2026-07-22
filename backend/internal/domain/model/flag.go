package model

import (
	"hash/fnv"
	"strconv"
)

// NewFlagID computes a FlagID per A23: FNV-1a 64 of rule_id ‖ match_text ‖ paragraph_index ‖ occurrence.
func NewFlagID(ruleID string, matchText string, paraIdx int, occ int) FlagID {
	h := fnv.New64a()
	_, _ = h.Write([]byte(ruleID))
	_, _ = h.Write([]byte("‖"))
	_, _ = h.Write([]byte(matchText))
	_, _ = h.Write([]byte("‖"))
	_, _ = h.Write([]byte(strconv.Itoa(paraIdx)))
	_, _ = h.Write([]byte("‖"))
	_, _ = h.Write([]byte(strconv.Itoa(occ)))
	return FlagIDFromUint64(h.Sum64())
}

// FlagState — lifecycle state machine.
type FlagState int

const (
	FlagStateRaised FlagState = iota
	FlagStateAccepted
	FlagStateRejected
	FlagStateApplied
)

// String returns the JSON-safe string representation of FlagState.
func (s FlagState) String() string {
	switch s {
	case FlagStateRaised:
		return "raised"
	case FlagStateAccepted:
		return "accepted"
	case FlagStateRejected:
		return "rejected"
	case FlagStateApplied:
		return "applied"
	default:
		return "unknown"
	}
}

// Flag — entity, rule match on text fragment.
type Flag struct {
	id             FlagID
	ruleID         RuleID
	matchText      MatchText
	suggestion     Suggestion
	autofix        *string
	severity       Severity
	category       Category
	occurrence     Occurrence
	message        string
	span           Span
	paragraphIndex ParagraphIndex
	state          FlagState
	ruleName       string
	ruleUrl        string
	examples       Examples
	related        []Related
}

// NewFlag from raw values.
func NewFlag(
	id uint64, ruleID string, matchText string,
	suggestion string, autofix *string, severity int, category string,
	occurrence int, message string,
	span Span, paraIdx int,
	ruleName string,
	ruleUrl string,
	examples Examples,
	related []Related,
) (*Flag, error) {
	if span.IsZero() {
		return nil, &DomainError{Op: "NewFlag", Message: "span must not be zero"}
	}
	rid, err := RuleIDFromString(ruleID)
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
	if message == "" {
		message = suggestion
	}
	return &Flag{
		id: FlagIDFromUint64(id), ruleID: rid,
		matchText:  MatchTextFromString(matchText),
		suggestion: SuggestionFromString(suggestion),
		autofix:    autofix,
		severity:   sev, category: cat,
		occurrence: OccurrenceFromInt(occurrence), message: message,
		span: span, paragraphIndex: ParagraphIndexFromInt(paraIdx),
		state:    FlagStateRaised,
		ruleName: ruleName,
		ruleUrl:  ruleUrl,
		examples: examples,
		related:  related,
	}, nil
}

func (f *Flag) ID() FlagID                     { return f.id }
func (f *Flag) RuleID() RuleID                 { return f.ruleID }
func (f *Flag) MatchText() MatchText           { return f.matchText }
func (f *Flag) Suggestion() Suggestion         { return f.suggestion }
func (f *Flag) Autofix() *string               { return f.autofix }
func (f *Flag) Severity() Severity             { return f.severity }
func (f *Flag) Category() Category             { return f.category }
func (f *Flag) Occurrence() Occurrence         { return f.occurrence }
func (f *Flag) Message() string                { return f.message }
func (f *Flag) Span() Span                     { return f.span }
func (f *Flag) ParagraphIndex() ParagraphIndex { return f.paragraphIndex }
func (f *Flag) State() FlagState               { return f.state }
func (f *Flag) IsPending() bool                { return f.state == FlagStateRaised }
func (f *Flag) RuleName() string               { return f.ruleName }
func (f *Flag) RuleURL() string                { return f.ruleUrl }
func (f *Flag) Examples() Examples             { return f.examples }
func (f *Flag) Related() []Related             { return f.related }

// SetSuggestion updates the flag's suggestion (replacement text).
func (f *Flag) SetSuggestion(s Suggestion) { f.suggestion = s }

// SetAutofix updates the flag's autofix.
func (f *Flag) SetAutofix(s *string) { f.autofix = s }

func (f *Flag) Accept() error {
	if f.state != FlagStateRaised {
		return &DomainError{Op: "Flag.Accept", Message: "can only accept a raised flag"}
	}
	f.state = FlagStateAccepted
	return nil
}

func (f *Flag) Reject() error {
	if f.state != FlagStateRaised {
		return &DomainError{Op: "Flag.Reject", Message: "can only reject a raised flag"}
	}
	f.state = FlagStateRejected
	return nil
}

func (f *Flag) Apply() error {
	if f.state != FlagStateRaised && f.state != FlagStateAccepted {
		return &DomainError{Op: "Flag.Apply", Message: "can only apply a raised or accepted flag"}
	}
	f.state = FlagStateApplied
	return nil
}

// AdjustSpan shifts the flag's span by delta and updates paragraph index.
// Used by FixApplier when preceding text is modified.
func (f *Flag) AdjustSpan(delta int, paraIdx int) {
	newStart := f.span.start + delta
	newEnd := f.span.end + delta
	if newStart < 0 || newStart >= newEnd {
		return
	}
	f.span = Span{start: newStart, end: newEnd}
	f.paragraphIndex = ParagraphIndexFromInt(paraIdx)
}
