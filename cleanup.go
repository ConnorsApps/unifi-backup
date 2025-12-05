package main

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/ConnorsApps/unifi-backup/pkg/storage"
)

// backupInfo holds information about a backup file
type backupInfo struct {
	filename  string
	timestamp time.Time
}

// cleanupOldBackups removes old backups keeping only the last n backups
func cleanupOldBackups(ctx context.Context, store storage.ObjectStore, keepLast int) error {
	slog.Info("Checking for old backups to cleanup", "keep_last", keepLast)

	// List all backup files
	files, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	if len(files) <= keepLast {
		slog.Info("No cleanup needed", "backup_count", len(files), "keep_last", keepLast)
		return nil
	}

	// Parse timestamps from filenames
	var backups []backupInfo
	for _, filename := range files {
		timestamp, err := storage.ParseBackupFilename(filename)
		if err != nil {
			slog.Debug("Skipping file with unparseable format", "filename", filename, "error", err)
			continue
		}
		backups = append(backups, backupInfo{
			filename:  filename,
			timestamp: timestamp,
		})
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].timestamp.After(backups[j].timestamp)
	})

	// Delete backups beyond the keepLast count
	deletedCount := 0
	failedCount := 0
	for i := keepLast; i < len(backups); i++ {
		backup := backups[i]
		slog.Info("Deleting old backup", "filename", backup.filename, "timestamp", backup.timestamp)
		if err := store.Delete(ctx, backup.filename); err != nil {
			slog.Warn("failed to delete backup", "filename", backup.filename, "error", err)
			failedCount++
			// Continue trying to delete other files even if one fails
		} else {
			deletedCount++
		}
	}

	slog.Info("Cleanup completed",
		"deleted_count", deletedCount,
		"failed_count", failedCount,
		"remaining_count", keepLast+failedCount,
	)
	return nil
}
