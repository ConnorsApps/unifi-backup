package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.UniFi.URL != "https://unifi.my-site.com" {
		t.Errorf("Expected default URL, got %s", cfg.UniFi.URL)
	}
	if cfg.UniFi.Site != "default" {
		t.Errorf("Expected default site, got %s", cfg.UniFi.Site)
	}
	if cfg.UniFi.Timeout != "10m" {
		t.Errorf("Expected default timeout '10m', got %s", cfg.UniFi.Timeout)
	}
	if cfg.UniFi.MaxRetries != 3 {
		t.Errorf("Expected default max retries 3, got %d", cfg.UniFi.MaxRetries)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got %v", cfg.Logging.Level)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "default config should be valid",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing URL",
			cfg: &Config{
				UniFi: UniFiConfig{
					Username: "admin",
					Password: "pass",
					Site:     "default",
				},
				Storage: StorageConfig{URL: "file://./backups"},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
		{
			name: "missing username",
			cfg: &Config{
				UniFi: UniFiConfig{
					URL:      "https://example.com",
					Password: "pass",
					Site:     "default",
				},
				Storage: StorageConfig{URL: "file://./backups"},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromYAML(t *testing.T) {
	yamlContent := `
unifi:
  url: https://test.example.com
  username: testuser
  password: testpass
  site: testsite
  includeDays: 7
  insecure_skip_verify: true
  timeout: 5m
  max_retries: 2
storage:
  url: file://./test-backups
logging:
  level: debug
  format: json
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UniFi.URL != "https://test.example.com" {
		t.Errorf("Expected URL 'https://test.example.com', got %s", cfg.UniFi.URL)
	}
	if cfg.UniFi.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", cfg.UniFi.Username)
	}
	if cfg.UniFi.IncludeDays != 7 {
		t.Errorf("Expected IncludeDays 7, got %d", cfg.UniFi.IncludeDays)
	}
	if !cfg.UniFi.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
	if cfg.UniFi.Timeout != "5m" {
		t.Errorf("Expected timeout '5m', got %s", cfg.UniFi.Timeout)
	}
	if cfg.UniFi.MaxRetries != 2 {
		t.Errorf("Expected max retries 2, got %d", cfg.UniFi.MaxRetries)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got %v", cfg.Logging.Level)
	}
}

func TestLoadFromJSON(t *testing.T) {
	jsonContent := `{
  "unifi": {
    "url": "https://json.example.com",
    "username": "jsonuser",
    "password": "jsonpass",
    "site": "jsonsite",
    "includeDays": 14,
    "insecure_skip_verify": false,
    "timeout": "20m",
    "max_retries": 4
  },
  "storage": {
    "url": "smb://server/share"
  },
  "logging": {
    "level": "warn",
    "format": "text"
  }
}`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UniFi.URL != "https://json.example.com" {
		t.Errorf("Expected URL 'https://json.example.com', got %s", cfg.UniFi.URL)
	}
	if cfg.UniFi.Username != "jsonuser" {
		t.Errorf("Expected username 'jsonuser', got %s", cfg.UniFi.Username)
	}
	if cfg.UniFi.IncludeDays != 14 {
		t.Errorf("Expected IncludeDays 14, got %d", cfg.UniFi.IncludeDays)
	}
	if cfg.Storage.URL != "smb://server/share" {
		t.Errorf("Expected storage URL 'smb://server/share', got %s", cfg.Storage.URL)
	}
	if cfg.UniFi.Timeout != "20m" {
		t.Errorf("Expected timeout '20m', got %s", cfg.UniFi.Timeout)
	}
	if cfg.UniFi.MaxRetries != 4 {
		t.Errorf("Expected max retries 4, got %d", cfg.UniFi.MaxRetries)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	t.Setenv("UNIFI_URL", "https://env.example.com")
	t.Setenv("UNIFI_USER", "envuser")
	t.Setenv("UNIFI_PASS", "envpass")
	t.Setenv("UNIFI_SITE", "envsite")
	t.Setenv("UNIFI_INCLUDE_DAYS", "30")
	t.Setenv("UNIFI_TIMEOUT", "30m")
	t.Setenv("UNIFI_MAX_RETRIES", "5")
	t.Setenv("STORAGE_URL", "file://./env-backups")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("LOG_FORMAT", "json")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UniFi.URL != "https://env.example.com" {
		t.Errorf("Expected URL from env, got %s", cfg.UniFi.URL)
	}
	if cfg.UniFi.Username != "envuser" {
		t.Errorf("Expected username from env, got %s", cfg.UniFi.Username)
	}
	if cfg.UniFi.IncludeDays != 30 {
		t.Errorf("Expected IncludeDays 30 from env, got %d", cfg.UniFi.IncludeDays)
	}
	if cfg.UniFi.Timeout != "30m" {
		t.Errorf("Expected timeout '30m' from env, got %s", cfg.UniFi.Timeout)
	}
	if cfg.UniFi.MaxRetries != 5 {
		t.Errorf("Expected max retries 5 from env, got %d", cfg.UniFi.MaxRetries)
	}
	if cfg.Logging.Level != "error" {
		t.Errorf("Expected log level 'error' from env, got %v", cfg.Logging.Level)
	}
}

func TestParseSlogLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{"debug", "debug", slog.LevelDebug, false},
		{"info", "info", slog.LevelInfo, false},
		{"warn", "warn", slog.LevelWarn, false},
		{"warning", "warning", slog.LevelWarn, false},
		{"error", "error", slog.LevelError, false},
		{"DEBUG uppercase", "DEBUG", slog.LevelDebug, false},
		{"INFO uppercase", "INFO", slog.LevelInfo, false},
		{"invalid", "invalid", slog.LevelInfo, true},
		{"empty", "", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSlogLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSlogLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSlogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
