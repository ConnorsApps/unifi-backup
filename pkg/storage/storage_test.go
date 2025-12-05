package storage

import (
	"testing"
	"time"
)

func TestParseBackupFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		expectedAge time.Duration // approximate age from now
	}{
		{
			name:     "valid filename with format",
			filename: "unifi-backup-2024-01-15T10-30-00Z.unf",
			wantErr:  false,
		},
		{
			name:     "valid recent filename",
			filename: "unifi-backup-2025-12-05T00-00-00Z.unf",
			wantErr:  false,
		},
		{
			name:     "invalid prefix",
			filename: "backup-2024-01-15T10-30-00Z.unf",
			wantErr:  true,
		},
		{
			name:     "invalid suffix",
			filename: "unifi-backup-2024-01-15T10-30-00Z.bak",
			wantErr:  true,
		},
		{
			name:     "invalid timestamp",
			filename: "unifi-backup-not-a-timestamp.unf",
			wantErr:  true,
		},
		{
			name:     "missing prefix",
			filename: "2024-01-15T10-30-00Z.unf",
			wantErr:  true,
		},
		{
			name:     "empty string",
			filename: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, err := ParseBackupFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBackupFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if timestamp.IsZero() {
					t.Error("ParseBackupFilename() returned zero timestamp for valid filename")
				}
			}
		})
	}
}

func TestGenerateBackupFilename(t *testing.T) {
	filename := GenerateBackupFilename()

	// Check prefix and suffix
	if len(filename) == 0 {
		t.Fatal("GenerateBackupFilename() returned empty string")
	}

	// Should be parseable
	timestamp, err := ParseBackupFilename(filename)
	if err != nil {
		t.Errorf("Generated filename %q is not parseable: %v", filename, err)
	}

	// Timestamp should be recent (within last minute)
	if time.Since(timestamp) > time.Minute {
		t.Errorf("Generated timestamp is too old: %v", timestamp)
	}
}
