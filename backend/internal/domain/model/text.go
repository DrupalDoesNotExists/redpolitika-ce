package model

import "strings"

// Text is a value object representing a text for analysis.
// Immutable — all methods return new instances or computed values.
type Text struct {
	content string
}

// NewText creates a new Text value object. Accepts empty text (valid state for initial session).
func NewText(content string) *Text {
	return &Text{content: content}
}

// Content returns the raw text.
func (t *Text) Content() string {
	return t.content
}

// WordCount returns the number of whitespace-delimited words.
func (t *Text) WordCount() int {
	if t.content == "" {
		return 0
	}
	return len(strings.Fields(t.content))
}

// Paragraphs splits text into paragraphs by separator.
// Empty or unknown separator defaults to "\n\n" (SPEC Q25).
// Paragraph separator is configurable per project policy.
func (t *Text) Paragraphs() []string {
	return t.ParaSplit("\n\n")
}

// ParaSplit splits text by given separator.
// Used when the paragraph separator is configured per project (Q25).
func (t *Text) ParaSplit(sep string) []string {
	if sep == "" {
		sep = "\n\n"
	}
	return strings.Split(t.content, sep)
}

// Hash returns an FNV-1a 64-bit hash of the content (A23).
func (t *Text) Hash() uint64 {
	const (
		fnvOffset64 = 14695981039346656037
		fnvPrime64  = 1099511628211
	)
	h := uint64(fnvOffset64)
	for _, c := range t.content {
		h ^= uint64(c)
		h *= fnvPrime64
	}
	return h
}
