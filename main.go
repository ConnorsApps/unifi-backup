package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ConnorsApps/unifi-backup/pkg/config"
	"github.com/ConnorsApps/unifi-backup/pkg/storage"
	"github.com/ConnorsApps/unifi-backup/pkg/unifi"

	_ "github.com/joho/godotenv/autoload"
)

// Version information set via ldflags at build time
var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to configuration file (YAML or JSON)")
	showVersion := flag.Bool("version", false, "Show version information and exit")
	flag.Parse()

	// Show version and exit if requested
	if *showVersion || len(os.Args) > 1 && strings.EqualFold(os.Args[1], "version") {
		slog.Info("UniFi Backup Tool", "version", Version, "commit", Commit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	setupLogger(cfg)

	// Extract config values
	storageURL := cfg.Storage.URL

	slog.Info("Starting UniFi backup",
		"version", Version,
		"baseURL", cfg.UniFi.URL,
		"site", cfg.UniFi.Site,
		"includeDays", cfg.UniFi.IncludeDays,
	)

	// Parse timeout duration
	timeout, err := time.ParseDuration(cfg.UniFi.Timeout)
	if err != nil {
		slog.Error("Invalid timeout duration", "error", err)
		os.Exit(1)
	}

	// Create UniFi client
	client, err := unifi.NewClient(cfg.UniFi.URL, unifi.ClientOptions{
		Site:               cfg.UniFi.Site,
		InsecureSkipVerify: cfg.UniFi.InsecureSkipVerify,
		Timeout:            timeout,
	})
	if err != nil {
		slog.Error("Failed to create UniFi client", "error", err)
		os.Exit(1)
	}

	// 1. Login with timeout
	loginCtx, loginCancel := context.WithTimeout(ctx, 30*time.Second)
	defer loginCancel()

	if err := client.Login(loginCtx, cfg.UniFi.Username, cfg.UniFi.Password); err != nil {
		slog.Error("Login failed", "error", err)
		os.Exit(1)
	}

	// 2. Trigger backup with timeout
	backupCtx, backupCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer backupCancel()

	backupURL, err := client.CreateBackup(backupCtx, cfg.UniFi.Username, cfg.UniFi.IncludeDays)
	if err != nil {
		slog.Error("Backup creation failed", "error", err)
		os.Exit(1)
	}

	// 3. Download backup with retry logic
	var dlResp *unifi.DownloadResponse
	downloadCtx, downloadCancel := context.WithTimeout(ctx, timeout)
	defer downloadCancel()

	err = retryWithBackoff(downloadCtx, cfg.UniFi.MaxRetries, func() error {
		var err error
		dlResp, err = client.DownloadBackup(downloadCtx, backupURL)
		return err
	})
	if err != nil {
		slog.Error("Failed to download backup after retries", "error", err)
		os.Exit(1)
	}
	defer dlResp.Body.Close()

	outName := storage.GenerateBackupFilename()

	store, err := storage.Open(ctx, storageURL)
	if err != nil {
		slog.Error("Error opening storage", "error", err)
		os.Exit(1)
	}

	defer store.Close()

	// Wrap the response body with a progress reader for logging
	progressReader := storage.NewProgressReader(dlResp.Body, dlResp.ContentLength)

	written, err := store.Put(ctx, outName, progressReader)
	if err != nil {
		slog.Error("Failed to save backup", "error", err)
		os.Exit(1)
	}

	// Verify backup size matches expected
	if dlResp.ContentLength > 0 && written != dlResp.ContentLength {
		slog.Warn("Backup size mismatch",
			"expected_bytes", dlResp.ContentLength,
			"written_bytes", written,
		)
	}

	slog.Info(
		"Backup saved successfully",
		"filename", outName,
		"size_bytes", written,
		"expected_bytes", dlResp.ContentLength,
	)

	// 4. Perform backup cleanup if enabled
	if cfg.Retention.KeepLast > 0 {
		if err := cleanupOldBackups(ctx, store, cfg.Retention.KeepLast); err != nil {
			slog.Warn("Failed to cleanup old backups", "error", err)
			// Don't fail the entire backup process on cleanup error
		}
	}
}
