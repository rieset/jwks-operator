package utils

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
	OnRetry     func(attempt int, err error) // Optional callback for retry attempts
}

// RetryWithDelay retries a function with fixed delay between attempts
func RetryWithDelay(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for i := 0; i < config.MaxAttempts; i++ {
		if i > 0 {
			// Check context before sleeping
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(config.Delay):
			}

			// Call retry callback if provided
			if config.OnRetry != nil && lastErr != nil {
				config.OnRetry(i, lastErr)
			}
		}

		// Check context before attempting
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", config.MaxAttempts, lastErr)
}
