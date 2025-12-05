package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/goccy/go-yaml"
)

// Config holds all application configuration
type Config struct {
	UniFi     UniFiConfig     `json:"unifi" yaml:"unifi" envPrefix:"UNIFI_" title:"UniFi Controller" description:"UniFi controller connection settings"`
	Storage   StorageConfig   `json:"storage" yaml:"storage" envPrefix:"STORAGE_" title:"Storage Backend" description:"Backup storage backend configuration"`
	Logging   LoggingConfig   `json:"logging" yaml:"logging" envPrefix:"LOG_" title:"Logging" description:"Application logging configuration"`
	Retention RetentionConfig `json:"retention" yaml:"retention" envPrefix:"RETENTION_" title:"Retention Policy" description:"Backup retention settings"`
}

// UniFiConfig holds UniFi controller configuration
type UniFiConfig struct {
	URL                string `json:"url" yaml:"url" env:"URL" title:"Controller URL" description:"URL of your UniFi controller" example:"https://unifi.example.com" format:"uri"`
	Username           string `json:"username" yaml:"username" env:"USER" title:"Username" description:"UniFi controller username (must be Administrator, not just Site Administrator)" example:"admin"`
	Password           string `json:"password" yaml:"password" env:"PASS" title:"Password" description:"UniFi controller password" writeOnly:"true"`
	Site               string `json:"site" yaml:"site" env:"SITE" title:"Site Name" description:"UniFi site name" default:"default" example:"default"`
	IncludeDays        int    `json:"includeDays" yaml:"includeDays" env:"INCLUDE_DAYS" title:"Include Days" description:"Number of days of history to include in backup (0 for current state only)" default:"0" minimum:"0" example:"0"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify" yaml:"insecure_skip_verify" env:"INSECURE" title:"Insecure Skip Verify" description:"Skip TLS certificate verification (useful for self-signed certificates)" default:"false"`
	Timeout            string `json:"timeout" yaml:"timeout" env:"TIMEOUT" title:"Timeout" description:"HTTP timeout for backup operations" default:"10m" example:"10m" pattern:"^[0-9]+(ns|us|ms|s|m|h)$"`
	MaxRetries         int    `json:"max_retries" yaml:"max_retries" env:"MAX_RETRIES" title:"Max Retries" description:"Maximum number of retry attempts for failed operations" default:"3" minimum:"0" example:"3"`
}

// StorageConfig holds storage backend configuration
type StorageConfig struct {
	URL string `json:"url" yaml:"url" env:"URL" title:"Storage URL" description:"Storage backend URL" example:"file://./backups" format:"uri"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level" yaml:"level" env:"LEVEL" title:"Log Level" description:"Log level" enum:"debug,info,warn,error" default:"info" example:"info"`
	Format string `json:"format" yaml:"format" env:"FORMAT" title:"Log Format" description:"Log output format" enum:"pretty,text,json" default:"pretty" example:"pretty"`
}

// RetentionConfig holds backup retention configuration
type RetentionConfig struct {
	KeepLast int `json:"keepLast" yaml:"keepLast" env:"KEEP_LAST" title:"Keep Last" description:"Number of backups to keep (0 for unlimited)" default:"7" minimum:"0" example:"7"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		UniFi: UniFiConfig{
			URL:                "https://unifi.my-site.com",
			Username:           "admin",
			Password:           "changeme",
			Site:               "default",
			IncludeDays:        0,
			InsecureSkipVerify: false,
			Timeout:            "10m",
			MaxRetries:         3,
		},
		Storage: StorageConfig{
			URL: "file://./backups",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "pretty",
		},
		Retention: RetentionConfig{
			KeepLast: 7,
		},
	}
}

// ParseSlogLevel converts a string log level to slog.Level
func ParseSlogLevel(v string) (slog.Level, error) {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", v)
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []string

	// UniFi validation
	if c.UniFi.URL == "" {
		errs = append(errs, "unifi.url is required")
	}
	if c.UniFi.Username == "" {
		errs = append(errs, "unifi.username is required")
	}
	if c.UniFi.Password == "" {
		errs = append(errs, "unifi.password is required")
	}
	if c.UniFi.Site == "" {
		errs = append(errs, "unifi.site is required")
	}
	if c.UniFi.Timeout == "" {
		errs = append(errs, "unifi.timeout is required")
	} else {
		if _, err := time.ParseDuration(c.UniFi.Timeout); err != nil {
			errs = append(errs, fmt.Sprintf("unifi.timeout is invalid: %v (examples: 10m, 1h, 30s)", err))
		}
	}
	if c.UniFi.MaxRetries < 0 {
		errs = append(errs, "unifi.max_retries must be non-negative")
	}

	// Storage validation
	if c.Storage.URL == "" {
		errs = append(errs, "storage.url is required")
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "warning": true, "error": true}
	if !validLevels[strings.ToLower(c.Logging.Level)] {
		errs = append(errs, "logging.level must be one of: debug, info, warn, error")
	}

	validFormats := map[string]bool{"text": true, "json": true, "pretty": true}
	if !validFormats[strings.ToLower(c.Logging.Format)] {
		errs = append(errs, "logging.format must be one of: pretty, text, json")
	}

	// Retention validation
	if c.Retention.KeepLast < 0 {
		errs = append(errs, "retention.keepLast must be non-negative (0 for unlimited)")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// LoadConfig loads configuration from multiple sources in the following order:
// 1. Default values (from struct tags)
// 2. Config file (YAML or JSON)
// 3. Environment variables (highest priority)
func LoadConfig(configPath string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Load from file if provided
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	} else {
		// Try to auto-detect config file
		for _, name := range []string{"config.yaml", "config.yml", "config.json"} {
			if _, err := os.Stat(name); err == nil {
				if err := loadFromFile(cfg, name); err != nil {
					return nil, fmt.Errorf("failed to load config file %s: %w", name, err)
				}
				break
			}
		}
	}

	// Parse environment variables (overrides file config and defaults)
	opts := env.Options{
		// Don't use RequiredIfNoDef to allow file/default values
		UseFieldNameByDefault: false,
	}

	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, cfg)
	case ".json":
		return json.Unmarshal(data, cfg)
	default:
		return fmt.Errorf("unsupported config file format: %s (supported: .yaml, .yml, .json)", ext)
	}
}
