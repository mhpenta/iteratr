package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure conditions
var (
	// ErrInvalidInput indicates invalid user input or configuration
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("not found")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrShutdown indicates the system is shutting down
	ErrShutdown = errors.New("system shutting down")

	// ErrAgentFailed indicates the AI agent subprocess failed
	ErrAgentFailed = errors.New("agent failed")

	// ErrStateCorruption indicates session state is corrupted
	ErrStateCorruption = errors.New("state corruption detected")
)

// ValidationError represents an input validation failure
type ValidationError struct {
	Field   string // Field that failed validation
	Value   string // Invalid value
	Message string // Human-readable message
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s (value: %q)", e.Field, e.Message, e.Value)
}

// Is implements error comparison for errors.Is
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// TransientError represents a temporary failure that can be retried
type TransientError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("transient error in %s: %v", e.Op, e.Err)
}

func (e *TransientError) Unwrap() error {
	return e.Err
}

// NewTransientError creates a new transient error
func NewTransientError(op string, err error) *TransientError {
	return &TransientError{Op: op, Err: err}
}

// IsTransient checks if an error is transient and can be retried
func IsTransient(err error) bool {
	var te *TransientError
	return errors.As(err, &te)
}

// PermanentError represents a non-recoverable failure
type PermanentError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *PermanentError) Error() string {
	return fmt.Sprintf("permanent error in %s: %v", e.Op, e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// NewPermanentError creates a new permanent error
func NewPermanentError(op string, err error) *PermanentError {
	return &PermanentError{Op: op, Err: err}
}

// MultiError aggregates multiple errors
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors: %v", len(e.Errors), e.Errors)
}

// Append adds an error to the multi-error if it's non-nil
func (e *MultiError) Append(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// ErrorOrNil returns the MultiError if it has errors, otherwise nil
func (e *MultiError) ErrorOrNil() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return e
}

// NewMultiError creates a new multi-error from a slice of errors
func NewMultiError(errs []error) *MultiError {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	return &MultiError{Errors: filtered}
}
