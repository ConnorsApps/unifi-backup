package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob" // file://
	_ "gocloud.dev/blob/gcsblob"  // gs://
	_ "gocloud.dev/blob/s3blob"   // s3://
)

type blobStore struct {
	b *blob.Bucket
}

func (s *blobStore) Put(ctx context.Context, key string, r io.Reader) (written int64, err error) {
	writer, err := s.b.NewWriter(ctx, key, nil)
	if err != nil {
		return 0, fmt.Errorf("create writer: %w", err)
	}

	bytesWritten, err := io.Copy(writer, r)
	if err != nil {
		_ = writer.Close()
		return bytesWritten, fmt.Errorf("write data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return bytesWritten, fmt.Errorf("close writer: %w", err)
	}

	return bytesWritten, nil
}

func (s *blobStore) List(ctx context.Context) ([]string, error) {
	var backups []string
	iter := s.b.List(&blob.ListOptions{})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate objects: %w", err)
		}
		// Filter for .unf files only
		if strings.HasSuffix(obj.Key, ".unf") {
			backups = append(backups, obj.Key)
		}
	}
	return backups, nil
}

func (s *blobStore) Delete(ctx context.Context, key string) error {
	if err := s.b.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *blobStore) Close() error {
	return s.b.Close()
}
