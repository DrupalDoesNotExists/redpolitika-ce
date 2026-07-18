// Package usecase implements application use-cases that orchestrate the domain.
package usecase

import "fmt"

// Error is a usecase-layer error with operation context and optional wrapped cause.
type Error struct {
	Op      string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }
