package model

// Span represents a text span [Start, End) within a paragraph.
// Zero-based, end-exclusive. Immutable value object.
type Span struct {
	start int
	end   int
}

// NewSpan creates a Span with validation.
func NewSpan(start, end int) (Span, error) {
	if start < 0 || end < 0 {
		return Span{}, &DomainError{Op: "NewSpan", Message: "span positions must be non-negative"}
	}
	if start >= end {
		return Span{}, &DomainError{Op: "NewSpan", Message: "start must be less than end"}
	}
	return Span{start: start, end: end}, nil
}

// Start returns the start offset.
func (s Span) Start() int { return s.start }

// End returns the end offset.
func (s Span) End() int { return s.end }

// Length returns the span length.
func (s Span) Length() int { return s.end - s.start }

// Overlaps returns true if spans share any position.
func (s Span) Overlaps(other Span) bool {
	return s.start < other.end && other.start < s.end
}

// Contains returns true if offset is within the span.
func (s Span) Contains(offset int) bool {
	return s.start <= offset && offset < s.end
}

// IsZero returns true for an invalid/default span.
func (s Span) IsZero() bool { return s.start == 0 && s.end == 0 }
