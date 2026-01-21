package errors

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts (including initial)
	InitialWait time.Duration // Initial wait before first retry
	MaxWait     time.Duration // Maximum wait between retries
	Multiplier  float64       // Backoff multiplier (e.g., 2.0 for exponential)
}

// DefaultRetryConfig returns sensible defaults for retry behavior
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
	}
}

// Retry executes fn with exponential backoff retry logic
// It returns the result of fn or the last error encountered
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Execute function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is permanent (no retry)
		var permErr *PermanentError
		if errors.As(err, &permErr) {
			return err
		}

		// Don't retry on last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Check context before waiting
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Wait before retry
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-timer.C:
		}

		// Calculate next wait time (exponential backoff)
		wait = time.Duration(float64(wait) * cfg.Multiplier)
		if wait > cfg.MaxWait {
			wait = cfg.MaxWait
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// RetryWithResult executes fn with exponential backoff retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Execute function
		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		// Check if error is permanent (no retry)
		var permErr *PermanentError
		if errors.As(err, &permErr) {
			return result, err
		}

		// Don't retry on last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Check context before waiting
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Wait before retry
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-timer.C:
		}

		// Calculate next wait time (exponential backoff)
		wait = time.Duration(float64(wait) * cfg.Multiplier)
		if wait > cfg.MaxWait {
			wait = cfg.MaxWait
		}
	}

	return result, fmt.Errorf("retry failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
