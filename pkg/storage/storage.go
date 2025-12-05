package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"gocloud.dev/blob"
)

const (
	// BackupPrefix is the prefix used for backup filenames
	BackupPrefix = "unifi-backup-"
	// BackupSuffix is the file extension for backup files
	BackupSuffix = ".unf"
	// ProgressLogIntervalMB is the interval in MB for logging download progress
	ProgressLogIntervalMB = 10
	// TimeFormat is the timestamp format used in backup filenames
	// Uses hyphens instead of colons for SMB/Windows filesystem compatibility
	TimeFormat = "2006-01-02T15-04-05Z"
)

// ObjectStore provides an abstraction for storing and retrieving backup files
// across different storage backends (local filesystem, SMB, S3, GCS, etc.)
type ObjectStore interface {
	// Put writes data from the reader to the storage backend with the given key
	Put(ctx context.Context, key string, r io.Reader) (written int64, err error)
	// List returns all backup file names from the storage backend
	List(ctx context.Context) ([]string, error)
	// Delete removes a backup file from the storage backend
	Delete(ctx context.Context, key string) error
	// Close releases any resources held by the storage backend
	Close() error
}

// Open opens an object store from a URL.
//
// Supported URL schemes:
//   - file://    - Local filesystem (via gocloud.dev/blob/fileblob)
//   - gs://      - Google Cloud Storage (via gocloud.dev/blob/gcsblob)
//   - s3://      - Amazon S3 (via gocloud.dev/blob/s3blob)
//   - smb://     - SMB/CIFS network shares (via github.com/jfjallid/go-smb)
//
// SMB URL format:
//
//	smb://[DOMAIN\]username[:password]@host[:port]/share[/path]
//	or
//	smb://username[:password]@host[:port]/share[/path]
//
// Examples:
//
//	smb://admin:password@192.168.1.10/backups
//	smb://admin:password@192.168.1.10/backups/unifi
//	smb://DOMAIN\user:password@nas.local:445/share/path
//
// For other schemes, see: https://gocloud.dev/concepts/urls/
func Open(ctx context.Context, storageURL string) (ObjectStore, error) {
	// Check if it's an SMB URL
	if strings.HasPrefix(storageURL, "smb://") {
		return OpenSMBStore(storageURL)
	}

	// Fall back to gocloud blob store
	b, err := blob.OpenBucket(ctx, storageURL)
	if err != nil {
		return nil, fmt.Errorf("open bucket %q: %w", storageURL, err)
	}
	return &blobStore{b: b}, nil
}

// GenerateBackupFilename generates a backup filename with the current UTC timestamp.
//
// Format: unifi-backup-YYYY-MM-DDTHH-MM-SSZ.unf
//
// Example: unifi-backup-2025-12-05T00-57-39Z.unf
//
// The timestamp uses hyphens instead of colons for compatibility with
// SMB/CIFS and Windows filesystems.
func GenerateBackupFilename() string {
	return BackupPrefix + time.Now().UTC().Format(TimeFormat) + BackupSuffix
}

// ParseBackupFilename extracts the timestamp from a backup filename.
//
// Expected format: unifi-backup-YYYY-MM-DDTHH-MM-SSZ.unf
//
// Example: unifi-backup-2025-12-05T00-57-39Z.unf returns 2025-12-05 00:57:39 UTC
//
// Returns an error if the filename doesn't match the expected format or
// contains an invalid timestamp.
func ParseBackupFilename(filename string) (time.Time, error) {
	// Strip the prefix and suffix
	if !strings.HasPrefix(filename, BackupPrefix) || !strings.HasSuffix(filename, BackupSuffix) {
		return time.Time{}, fmt.Errorf("filename %q does not match expected format %s*%s", filename, BackupPrefix, BackupSuffix)
	}

	timestampStr := strings.TrimPrefix(filename, BackupPrefix)
	timestampStr = strings.TrimSuffix(timestampStr, BackupSuffix)

	timestamp, err := time.Parse(TimeFormat, timestampStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp from filename %q: %w", filename, err)
	}

	return timestamp, nil
}
