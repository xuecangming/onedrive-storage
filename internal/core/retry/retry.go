package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts  int           // Maximum number of retry attempts
	InitialDelay time.Duration // Initial delay between retries
	MaxDelay     time.Duration // Maximum delay between retries
	Multiplier   float64       // Backoff multiplier
	Jitter       bool          // Whether to add jitter to delays
}

// DefaultConfig returns default retry configuration
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Operation is a function that can be retried
type Operation func() error

// OperationWithContext is a function with context that can be retried
type OperationWithContext func(ctx context.Context) error

// Do executes an operation with retry logic
func Do(op Operation, config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if this is the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		// Wait before retry
		time.Sleep(delay)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// DoWithContext executes an operation with context and retry logic
func DoWithContext(ctx context.Context, op OperationWithContext, config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if this is the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		// Wait before retry with context cancellation support
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(attempt int, config *Config) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))

	// Cap at maximum delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter if enabled
	if config.Jitter {
		// Add random jitter up to 25% of the delay
		jitter := delay * 0.25 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

// IsRetryable checks if an error should be retried
type IsRetryable func(error) bool

// DoWithRetryable executes an operation with custom retry logic
func DoWithRetryable(op Operation, config *Config, isRetryable IsRetryable) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't retry if this is the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		// Wait before retry
		time.Sleep(delay)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// DoWithContextAndRetryable executes an operation with context and custom retry logic
func DoWithContextAndRetryable(ctx context.Context, op OperationWithContext, config *Config, isRetryable IsRetryable) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't retry if this is the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(attempt, config)

		// Wait before retry with context cancellation support
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}
