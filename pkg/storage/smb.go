package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"github.com/jfjallid/go-smb/smb"
	"github.com/jfjallid/go-smb/spnego"
)

type smbStore struct {
	session  *smb.Connection
	share    string
	basePath string
}

func (s *smbStore) Put(ctx context.Context, key string, r io.Reader) (written int64, err error) {
	fullPath := path.Join(s.basePath, key)

	// Ensure the parent directory exists
	dir := path.Dir(fullPath)
	if dir != "." && dir != "/" {
		// Create directory structure if needed
		// MkdirAll is idempotent - it succeeds if the directory already exists
		// We only log non-critical errors since the subsequent PutFile will fail
		// if there's a real permission issue
		if err := s.session.MkdirAll(s.share, dir); err != nil {
			// Check if the error is because the directory already exists
			// The go-smb library returns STATUS_OBJECT_NAME_COLLISION for existing dirs
			if !strings.Contains(err.Error(), "COLLISION") && !strings.Contains(err.Error(), "exists") {
				// Log unexpected errors but continue - PutFile will fail if there's a real issue
				slog.Debug("mkdir warning (may be ignorable)", "dir", dir, "error", err)
			}
		}
	}

	// Use PutFile with a callback that reads from the reader
	const offset = uint64(0)
	bytesWritten := int64(0)
	err = s.session.PutFile(s.share, fullPath, offset, func(buffer []byte) (int, error) {
		byteCount, readErr := r.Read(buffer)
		bytesWritten += int64(byteCount)
		return byteCount, readErr
	})
	if err != nil {
		return bytesWritten, fmt.Errorf("write SMB file %q: %w", fullPath, err)
	}

	return bytesWritten, nil
}

func (s *smbStore) List(ctx context.Context) ([]string, error) {
	var backups []string

	// List files in the base path (third argument is search pattern)
	entries, err := s.session.ListDirectory(s.share, s.basePath, "*")
	if err != nil {
		return nil, fmt.Errorf("list SMB directory %q: %w", s.basePath, err)
	}

	// Filter for .unf files only
	for _, entry := range entries {
		if !entry.IsDir && strings.HasSuffix(entry.Name, ".unf") {
			// Return filename without path prefix for consistency
			backups = append(backups, entry.Name)
		}
	}

	return backups, nil
}

func (s *smbStore) Delete(ctx context.Context, key string) error {
	fullPath := path.Join(s.basePath, key)
	err := s.session.DeleteFile(s.share, fullPath)
	if err != nil {
		return fmt.Errorf("delete SMB file %q: %w", fullPath, err)
	}
	return nil
}

func (s *smbStore) Close() error {
	if s.session != nil {
		if err := s.session.TreeDisconnect(s.share); err != nil {
			slog.Warn("SMB tree disconnect failed", "error", err)
		}
		s.session.Close()
	}
	return nil
}

// smbConfig holds the parsed SMB connection configuration (unexported, internal use only)
type smbConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Domain   string
	Share    string
	BasePath string
}

// parseSMBURL parses URLs in the format:
// smb://[username[:password]@]host[:port]/share[/path]
// Domain can be specified as: DOMAIN;username, DOMAIN\username (URL-encoded as %5C), or username@domain
func parseSMBURL(smbURL string) (*smbConfig, error) {
	u, err := url.Parse(smbURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	if u.Scheme != "smb" {
		return nil, fmt.Errorf("expected smb:// scheme, got %s://", u.Scheme)
	}

	cfg := &smbConfig{
		Host: u.Hostname(),
		Port: 445, // default SMB port
	}

	if p := u.Port(); p != "" {
		fmt.Sscanf(p, "%d", &cfg.Port)
	}

	if u.User != nil {
		cfg.Username = u.User.Username()
		cfg.Password, _ = u.User.Password()

		// Check if username contains domain (DOMAIN;username or DOMAIN\username format)
		// We support two explicit domain formats:
		//   - DOMAIN;username (URL-safe format)
		//   - DOMAIN\username (URL-encoded as %5C)
		// Note: We do NOT parse user@domain format as it conflicts with email-style
		// usernames (e.g., john@company.com). If you need UPN-style authentication,
		// pass the full UPN as the username without domain extraction.
		if strings.Contains(cfg.Username, ";") {
			parts := strings.SplitN(cfg.Username, ";", 2)
			cfg.Domain = parts[0]
			cfg.Username = parts[1]
		} else if strings.Contains(cfg.Username, "\\") {
			parts := strings.SplitN(cfg.Username, "\\", 2)
			cfg.Domain = parts[0]
			cfg.Username = parts[1]
		}
	}

	// Path format: /share/path/to/file
	pathParts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	if len(pathParts) == 0 || pathParts[0] == "" {
		return nil, fmt.Errorf("share name required in path")
	}

	cfg.Share = pathParts[0]
	if len(pathParts) > 1 {
		cfg.BasePath = pathParts[1]
	}

	return cfg, nil
}

// OpenSMBStore opens an SMB/CIFS storage backend from the given URL
func OpenSMBStore(smbURL string) (ObjectStore, error) {
	cfg, err := parseSMBURL(smbURL)
	if err != nil {
		return nil, fmt.Errorf("parse SMB URL: %w", err)
	}

	options := smb.Options{
		Host: cfg.Host,
		Port: cfg.Port,
		Initiator: &spnego.NTLMInitiator{
			User:     cfg.Username,
			Password: cfg.Password,
			Domain:   cfg.Domain,
		},
	}

	session, err := smb.NewConnection(options)
	if err != nil {
		return nil, fmt.Errorf("connect to SMB server %s:%d: %w", cfg.Host, cfg.Port, err)
	}

	if !session.IsAuthenticated() {
		session.Close()
		return nil, fmt.Errorf("SMB authentication failed for user %s", cfg.Username)
	}

	if err := session.TreeConnect(cfg.Share); err != nil {
		session.Close()
		return nil, fmt.Errorf("connect to share %q: %w", cfg.Share, err)
	}

	slog.Debug("SMB connection established",
		"host", cfg.Host,
		"port", cfg.Port,
		"share", cfg.Share,
		"base_path", cfg.BasePath,
	)

	return &smbStore{
		session:  session,
		share:    cfg.Share,
		basePath: cfg.BasePath,
	}, nil
}
