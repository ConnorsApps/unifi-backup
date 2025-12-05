package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/ConnorsApps/unifi-backup/pkg/config"
	"github.com/Marlliton/slogpretty"
)

const (
	defaultRetryInitialDelay = 1 * time.Second
)

// retryWithBackoff attempts an operation with exponential backoff
func retryWithBackoff(ctx context.Context, maxRetries int, operation func() error) error {
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

// formatBytes converts bytes to human-readable format (B, KB, MB, GB, TB)
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// formatSpeed converts bytes per second to human-readable format
func formatSpeed(bytesPerSecond float64) string {
	const unit = 1024
	if bytesPerSecond < unit {
		return fmt.Sprintf("%.0f B/s", bytesPerSecond)
	}

	div, exp := float64(unit), 0
	for n := bytesPerSecond / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB/s", "MB/s", "GB/s", "TB/s"}
	return fmt.Sprintf("%.2f %s", bytesPerSecond/div, units[exp])
}

func setupLogger(cfg *config.Config) {
	// Setup structured logging based on config
	logLevel, err := config.ParseSlogLevel(cfg.Logging.Level)
	if err != nil {
		slog.Error("Invalid log level, using INFO", "error", err)
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Logging.Format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else if strings.EqualFold(cfg.Logging.Format, "text") {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		handler = slogpretty.New(os.Stdout, &slogpretty.Options{
			Level:      logLevel,
			TimeFormat: time.Kitchen,
			Colorful:   true,
			Multiline:  true,
		})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
