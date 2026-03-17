package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Marlliton/slogpretty"
)

func (cfg *Config) SetupLogger() {
	// Setup structured logging based on config
	logLevel, err := ParseSlogLevel(cfg.Logging.Level)
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
