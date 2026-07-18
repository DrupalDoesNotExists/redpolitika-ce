package model

import (
	"fmt"
	"strings"
)

// Domain value objects — struct with private field prevents external construction.
// All constructors use From* naming and validate or wrap raw values.

type (
	RuleID         struct{ value string }
	SessionID      struct{ value string }
	FlagID         struct{ value uint64 }
	ConfigHash     struct{ value uint64 }
	WordCount      struct{ value int }
	ParagraphIndex struct{ value int }
	Occurrence     struct{ value int }
	MatchText      struct{ value string }
	Suggestion     struct{ value string }
	Severity       struct{ value int }
	Category       struct{ value string }
	DetectMethod   struct{ value string }
)

// --- Constructors ---

func RuleIDFromString(v string) (RuleID, error) {
	if v == "" {
		return RuleID{}, &DomainError{Op: "RuleIDFromString", Message: "rule ID must not be empty"}
	}
	return RuleID{value: v}, nil
}

func SessionIDFromString(v string) (SessionID, error) {
	if v == "" {
		return SessionID{}, &DomainError{Op: "SessionIDFromString", Message: "session ID must not be empty"}
	}
	return SessionID{value: v}, nil
}

func FlagIDFromUint64(v uint64) FlagID { return FlagID{value: v} }

func ConfigHashFromUint64(v uint64) ConfigHash { return ConfigHash{value: v} }

func WordCountFromInt(v int) WordCount { return WordCount{value: v} }

func ParagraphIndexFromInt(v int) ParagraphIndex { return ParagraphIndex{value: v} }

func OccurrenceFromInt(v int) Occurrence { return Occurrence{value: v} }

func MatchTextFromString(v string) MatchText { return MatchText{value: v} }

func SuggestionFromString(v string) Suggestion { return Suggestion{value: v} }

func SeverityFromInt(v int) (Severity, error) {
	if v < 1 || v > 10 {
		return Severity{}, &DomainError{Op: "SeverityFromInt", Message: "severity must be between 1 and 10"}
	}
	return Severity{value: v}, nil
}

func CategoryFromString(v string) (Category, error) {
	if v != "cleanliness" && v != "readability" {
		return Category{}, &DomainError{Op: "CategoryFromString", Message: "invalid category: " + v}
	}
	return Category{value: v}, nil
}

func DetectMethodFromString(v string) (DetectMethod, error) {
	switch v {
	case "regex", "wordlist", "expr", "llm", "plugin", "pattern", "function", "ner", "pos":
		return DetectMethod{value: v}, nil
	default:
		// Accept scoped names (e.g., "plugin/method", "custom/ner")
		if strings.Contains(v, "/") {
			return DetectMethod{value: v}, nil
		}
		return DetectMethod{}, &DomainError{Op: "DetectMethodFromString", Message: "invalid detect method: " + v}
	}
}

// --- Value getters ---

func (id RuleID) Value() string      { return id.value }
func (id SessionID) Value() string   { return id.value }
func (id FlagID) Value() uint64      { return id.value }
func (c ConfigHash) Value() uint64   { return c.value }
func (w WordCount) Value() int       { return w.value }
func (p ParagraphIndex) Value() int  { return p.value }
func (o Occurrence) Value() int      { return o.value }
func (m MatchText) Value() string    { return m.value }
func (s Suggestion) Value() string   { return s.value }
func (s Severity) Value() int        { return s.value }
func (c Category) Value() string     { return c.value }
func (d DetectMethod) Value() string { return d.value }

// --- Helpers ---

func (w WordCount) IsZero() bool { return w.value <= 0 }

func (id RuleID) String() string    { return id.value }
func (id SessionID) String() string { return id.value }
func (id FlagID) String() string    { return fmt.Sprintf("%016x", id.value) }

// Predefined values for enum-like types.
var (
	DetectMethodRegex    = DetectMethod{value: "regex"}
	DetectMethodWordlist = DetectMethod{value: "wordlist"}
	DetectMethodExpr     = DetectMethod{value: "expr"}
	DetectMethodLLM      = DetectMethod{value: "llm"}
	DetectMethodPlugin   = DetectMethod{value: "plugin"}
	DetectMethodPattern  = DetectMethod{value: "pattern"}
	DetectMethodFunction = DetectMethod{value: "function"}
	DetectMethodNER      = DetectMethod{value: "ner"}
	DetectMethodPOS      = DetectMethod{value: "pos"}
	CategoryCleanliness  = Category{value: "cleanliness"}
	CategoryReadability  = Category{value: "readability"}
)
