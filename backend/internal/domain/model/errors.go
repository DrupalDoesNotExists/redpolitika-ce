// Package model defines domain entities, value objects, and microtypes.
package model

import "fmt"

// DomainError is a domain-layer error with operation context.
type DomainError struct {
	Op      string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}
