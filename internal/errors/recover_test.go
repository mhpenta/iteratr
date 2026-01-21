package errors

import (
	"errors"
	"testing"
)

func TestRecover(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		err := Recover(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Regular error", func(t *testing.T) {
		expectedErr := errors.New("regular error")
		err := Recover(func() error {
			return expectedErr
		})
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Panic recovery", func(t *testing.T) {
		err := Recover(func() error {
			panic("something went wrong")
		})
		if err == nil {
			t.Error("expected error from panic, got nil")
		}
		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Error("expected PanicError type")
		}
		if panicErr.Value != "something went wrong" {
			t.Errorf("expected panic value 'something went wrong', got %v", panicErr.Value)
		}
		if panicErr.StackTrace == "" {
			t.Error("expected non-empty stack trace")
		}
	})
}

func TestRecoverWithResult(t *testing.T) {
	t.Run("Success with result", func(t *testing.T) {
		result, err := RecoverWithResult(func() (string, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != "success" {
			t.Errorf("expected 'success', got %q", result)
		}
	})

	t.Run("Panic returns zero value", func(t *testing.T) {
		result, err := RecoverWithResult(func() (int, error) {
			panic("crash")
		})
		if err == nil {
			t.Error("expected error from panic, got nil")
		}
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Error("expected PanicError type")
		}
	})
}

func TestSafeGo(t *testing.T) {
	t.Run("Normal execution", func(t *testing.T) {
		errChan := make(chan error, 1)
		SafeGo(func() error {
			return nil
		}, errChan)
		// No error should be sent for successful execution
		select {
		case err := <-errChan:
			t.Errorf("expected no error, got %v", err)
		default:
			// Expected: no error
		}
	})

	t.Run("Error propagation", func(t *testing.T) {
		errChan := make(chan error, 1)
		expectedErr := errors.New("test error")
		SafeGo(func() error {
			return expectedErr
		}, errChan)
		// Wait for goroutine to finish
		err := <-errChan
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("Panic recovery", func(t *testing.T) {
		errChan := make(chan error, 1)
		SafeGo(func() error {
			panic("goroutine panic")
		}, errChan)
		// Wait for goroutine to finish
		err := <-errChan
		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Error("expected PanicError type")
		}
		if panicErr.Value != "goroutine panic" {
			t.Errorf("expected panic value 'goroutine panic', got %v", panicErr.Value)
		}
	})
}

func TestPanicError(t *testing.T) {
	t.Run("Error message includes value and stack", func(t *testing.T) {
		pe := &PanicError{
			Value:      "test panic",
			StackTrace: "stack trace here",
		}
		errMsg := pe.Error()
		if errMsg == "" {
			t.Error("expected non-empty error message")
		}
		// Should include both value and stack
		if errMsg != "panic: test panic\nstack trace here" {
			t.Errorf("unexpected error message format: %s", errMsg)
		}
	})
}
