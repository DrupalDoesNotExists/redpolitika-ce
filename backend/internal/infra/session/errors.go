package session

import "fmt"

// Error is an infra-layer error for the session package.
type Error struct {
	Op      string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("infra/session: %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("infra/session: %s: %s", e.Op, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }
