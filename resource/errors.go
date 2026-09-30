package resource

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAmbiguous     = errors.New("resource name is ambiguous")
	ErrUnsupported   = errors.New("operation is unsupported")
	ErrInvalidOption = errors.New("invalid option")
	ErrFailedState   = errors.New("resource entered a failed state")
)

// NotFoundError preserves a Gophercloud error when one is available.
type NotFoundError struct {
	Resource  string
	Reference string
	Cause     error
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q: %v", e.Resource, e.Reference, ErrNotFound)
}
func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }
func (e *NotFoundError) Unwrap() error        { return e.Cause }

// AmbiguousError reports the matching IDs instead of selecting arbitrarily.
type AmbiguousError struct {
	Resource string
	Name     string
	IDs      []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%s %q: %v (%v)", e.Resource, e.Name, ErrAmbiguous, e.IDs)
}
func (e *AmbiguousError) Unwrap() error { return ErrAmbiguous }

// FailedStateError is returned immediately when a waiter observes failure.
type FailedStateError struct {
	Resource string
	ID       string
	Status   string
}

func (e *FailedStateError) Error() string {
	return fmt.Sprintf("%s %q: %v (%s)", e.Resource, e.ID, ErrFailedState, e.Status)
}
func (e *FailedStateError) Unwrap() error { return ErrFailedState }

// OperationError adds context while preserving errors.Is/errors.As behavior.
type OperationError struct {
	Operation string
	Resource  string
	Cause     error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Operation, e.Resource, e.Cause)
}
func (e *OperationError) Unwrap() error { return e.Cause }

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidOption, fmt.Sprintf(format, args...))
}
