package storage

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

// progressReader wraps an io.Reader and logs progress periodically
type progressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	lastLogged  int64
	logInterval int64
	startTime   time.Time
	lastLogTime time.Time
}

// NewProgressReader creates a new progress reader that logs download progress
func NewProgressReader(r io.Reader, totalSize int64) *progressReader {
	return newProgressReader(r, totalSize, ProgressLogIntervalMB)
}

// newProgressReader creates a new progress reader that logs every logIntervalMB megabytes
func newProgressReader(r io.Reader, totalSize int64, logIntervalMB int64) *progressReader {
	now := time.Now()
	return &progressReader{
		reader:      r,
		total:       totalSize,
		logInterval: logIntervalMB * 1024 * 1024,
		startTime:   now,
		lastLogTime: now,
	}
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	// Log progress at intervals
	if pr.read-pr.lastLogged >= pr.logInterval || err == io.EOF {
		pr.logProgress()
		pr.lastLogged = pr.read
	}

	return n, err
}

// FormatBytes converts bytes to human-readable format (B, KB, MB, GB, TB)
func FormatBytes(bytes int64) string {
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

func (pr *progressReader) logProgress() {
	elapsed := time.Since(pr.startTime)
	elapsedSeconds := elapsed.Seconds()

	// Calculate speed in bytes per second
	speedBytesPerSec := float64(0)
	if elapsedSeconds > 0 {
		speedBytesPerSec = float64(pr.read) / elapsedSeconds
	}

	attrs := []any{
		"downloaded", FormatBytes(pr.read),
		"elapsed", elapsed.Round(time.Second),
		"speed", formatSpeed(speedBytesPerSec),
	}

	if pr.total > 0 && speedBytesPerSec > 0 {
		percentage := float64(pr.read) / float64(pr.total) * 100
		remaining := time.Duration(float64(pr.total-pr.read)/speedBytesPerSec) * time.Second
		attrs = append(attrs,
			"total", FormatBytes(pr.total),
			"percentage", fmt.Sprintf("%.1f%%", percentage),
			"estimated_remaining", remaining.Round(time.Second),
		)
	} else if pr.total > 0 {
		percentage := float64(pr.read) / float64(pr.total) * 100
		attrs = append(attrs,
			"total", FormatBytes(pr.total),
			"percentage", fmt.Sprintf("%.1f%%", percentage),
		)
	}

	slog.Info("Download progress", attrs...)
}
