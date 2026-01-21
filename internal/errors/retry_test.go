package errors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Run("Success on first attempt", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		attempts := 0
		err := Retry(ctx, cfg, func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("Success after retries", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		attempts := 0
		err := Retry(ctx, cfg, func() error {
			attempts++
			if attempts < 3 {
				return NewTransientError("op", errors.New("temp"))
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("Permanent error stops retry", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		attempts := 0
		err := Retry(ctx, cfg, func() error {
			attempts++
			return NewPermanentError("op", errors.New("permanent"))
		})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("Max attempts exhausted", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		attempts := 0
		err := Retry(ctx, cfg, func() error {
			attempts++
			return NewTransientError("op", errors.New("always fails"))
		})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cfg := RetryConfig{
			MaxAttempts: 5,
			InitialWait: 50 * time.Millisecond,
			MaxWait:     500 * time.Millisecond,
			Multiplier:  2.0,
		}

		attempts := 0
		errChan := make(chan error, 1)

		go func() {
			errChan <- Retry(ctx, cfg, func() error {
				attempts++
				return NewTransientError("op", errors.New("temp"))
			})
		}()

		// Cancel after a short delay
		time.Sleep(30 * time.Millisecond)
		cancel()

		err := <-errChan
		if err == nil {
			t.Error("expected error due to cancellation")
		}
		if attempts > 2 {
			t.Errorf("expected at most 2 attempts before cancellation, got %d", attempts)
		}
	})
}

func TestRetryWithResult(t *testing.T) {
	t.Run("Success with result", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 3,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		result, err := RetryWithResult(ctx, cfg, func() (string, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != "success" {
			t.Errorf("expected 'success', got %q", result)
		}
	})

	t.Run("Failure returns zero value", func(t *testing.T) {
		ctx := context.Background()
		cfg := RetryConfig{
			MaxAttempts: 2,
			InitialWait: 10 * time.Millisecond,
			MaxWait:     100 * time.Millisecond,
			Multiplier:  2.0,
		}

		result, err := RetryWithResult(ctx, cfg, func() (int, error) {
			return 0, errors.New("failed")
		})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialWait != 100*time.Millisecond {
		t.Errorf("expected InitialWait=100ms, got %v", cfg.InitialWait)
	}
	if cfg.MaxWait != 2*time.Second {
		t.Errorf("expected MaxWait=2s, got %v", cfg.MaxWait)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("expected Multiplier=2.0, got %f", cfg.Multiplier)
	}
}
