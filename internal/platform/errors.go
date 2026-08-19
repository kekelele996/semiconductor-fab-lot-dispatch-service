package platform

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource version conflict")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrCapacity          = errors.New("capacity exhausted")
	ErrInvariant         = errors.New("fab invariant violated")
)

type OperationError struct {
	Operation, Resource, ID string
	Err                     error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s %s %s: %v", e.Operation, e.Resource, e.ID, e.Err)
}
func (e *OperationError) Unwrap() error { return e.Err }
func Wrap(op, resource, id string, err error) error {
	if err == nil {
		return nil
	}
	return &OperationError{op, resource, id, err}
}
