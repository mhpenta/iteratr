package errors

import (
	"errors"
	"testing"
)

func TestValidationError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		err := NewValidationError("session_name", "bad.name", "dots not allowed")
		expected := `validation failed for session_name: dots not allowed (value: "bad.name")`
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Is ErrInvalidInput", func(t *testing.T) {
		err := NewValidationError("field", "value", "message")
		if !errors.Is(err, ErrInvalidInput) {
			t.Error("ValidationError should match ErrInvalidInput")
		}
	})
}

func TestTransientError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		inner := errors.New("connection refused")
		err := NewTransientError("connect", inner)
		expected := "transient error in connect: connection refused"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		inner := errors.New("connection refused")
		err := NewTransientError("connect", inner)
		if errors.Unwrap(err) != inner {
			t.Error("TransientError should unwrap to inner error")
		}
	})

	t.Run("IsTransient", func(t *testing.T) {
		err := NewTransientError("op", errors.New("temp"))
		if !IsTransient(err) {
			t.Error("IsTransient should return true for TransientError")
		}

		regularErr := errors.New("regular")
		if IsTransient(regularErr) {
			t.Error("IsTransient should return false for regular error")
		}
	})
}

func TestPermanentError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		inner := errors.New("disk full")
		err := NewPermanentError("write", inner)
		expected := "permanent error in write: disk full"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		inner := errors.New("disk full")
		err := NewPermanentError("write", inner)
		if errors.Unwrap(err) != inner {
			t.Error("PermanentError should unwrap to inner error")
		}
	})
}

func TestMultiError(t *testing.T) {
	t.Run("Empty MultiError", func(t *testing.T) {
		me := &MultiError{}
		if me.ErrorOrNil() != nil {
			t.Error("Empty MultiError should return nil")
		}
	})

	t.Run("Single error", func(t *testing.T) {
		me := &MultiError{}
		me.Append(errors.New("error 1"))
		expected := "error 1"
		if me.Error() != expected {
			t.Errorf("expected %q, got %q", expected, me.Error())
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		me := &MultiError{}
		me.Append(errors.New("error 1"))
		me.Append(errors.New("error 2"))
		if len(me.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(me.Errors))
		}
	})

	t.Run("Append nil does nothing", func(t *testing.T) {
		me := &MultiError{}
		me.Append(nil)
		if len(me.Errors) != 0 {
			t.Error("Appending nil should not add to errors")
		}
	})

	t.Run("NewMultiError filters nil", func(t *testing.T) {
		errs := []error{
			errors.New("error 1"),
			nil,
			errors.New("error 2"),
			nil,
		}
		me := NewMultiError(errs)
		if len(me.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(me.Errors))
		}
	})
}
