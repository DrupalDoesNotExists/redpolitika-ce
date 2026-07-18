package model

import "fmt"

// Score is a value object for a normalised score 0.0–10.0.
// Two scores exist per analysis: cleanliness and readability.
type Score struct {
	value float64
}

// MaxScore is the maximum possible score.
const MaxScore = 10.0

// NewScore creates a Score with validation.
func NewScore(value float64) (Score, error) {
	if value < 0 || value > MaxScore {
		return Score{}, &DomainError{
			Op:      "NewScore",
			Message: fmt.Sprintf("score must be between 0 and %v, got %v", MaxScore, value),
		}
	}
	return Score{value: value}, nil
}

// NewScoreUnsafe creates a Score without validation, clamping to [0, MaxScore].
// Convenience for internal use where bounds are known.
func NewScoreUnsafe(value float64) Score {
	if value < 0 {
		value = 0
	}
	if value > MaxScore {
		value = MaxScore
	}
	return Score{value: value}
}

// Value returns the raw score value.
func (s Score) Value() float64 { return s.value }

// Sum adds two scores, clamping to [0, MaxScore].
func (s Score) Sum(other Score) Score {
	return NewScoreUnsafe(s.value + other.value)
}

// String implements fmt.Stringer.
func (s Score) String() string { return fmt.Sprintf("%.1f", s.value) }
