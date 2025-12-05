package storage

import (
	"bytes"
	"io"
	"testing"
)

func TestProgressReader(t *testing.T) {
	// Create a test reader with 100 bytes
	data := bytes.Repeat([]byte("a"), 100)
	reader := bytes.NewReader(data)

	// Create progress reader with 1MB interval (won't trigger during this small read)
	pr := newProgressReader(reader, int64(len(data)), 1)

	// Read all data
	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(result) != len(data) {
		t.Errorf("Expected %d bytes, got %d", len(data), len(result))
	}

	if pr.read != int64(len(data)) {
		t.Errorf("Expected read counter to be %d, got %d", len(data), pr.read)
	}
}

func TestProgressReaderWithUnknownSize(t *testing.T) {
	data := bytes.Repeat([]byte("b"), 50)
	reader := bytes.NewReader(data)

	// Create progress reader with unknown size (0)
	pr := newProgressReader(reader, 0, 1)

	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(result) != len(data) {
		t.Errorf("Expected %d bytes, got %d", len(data), len(result))
	}
}
