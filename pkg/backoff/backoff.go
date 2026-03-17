package backoff

import (
	"context"
	"log/slog"
	"math"
	"time"
)

const defaultRetryInitialDelay = 1 * time.Second

// retryWithBackoff attempts an operation with exponential backoff
func Retry(ctx context.Context, maxRetries int, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff: 1s, 2s, 4s, 8s...
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * defaultRetryInitialDelay
			if delay > 30*time.Second {
				delay = 30 * time.Second // Cap at 30 seconds
			}

			slog.Info("Retrying operation",
				"attempt", attempt+1,
				"max_attempts", maxRetries+1,
				"delay", delay,
			)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 {
				slog.Info("Operation succeeded after retry", "attempts", attempt+1)
			}
			return nil
		}

		slog.Warn("Operation failed",
			"attempt", attempt+1,
			"error", lastErr,
		)
	}

	return lastErr
}
