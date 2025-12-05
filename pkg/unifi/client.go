package unifi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Minute
)

type backupResp struct {
	Meta struct {
		Rc  string `json:"rc"`
		Msg string `json:"msg,omitempty"`
	} `json:"meta"`
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

// Client represents a UniFi controller API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	site       string
}

// ClientOptions configures the UniFi API client
type ClientOptions struct {
	Site               string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// NewClient creates a new UniFi API client
func NewClient(baseURL string, opts ClientOptions) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	timeout := defaultHTTPTimeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	httpClient := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify},
		},
		Timeout: timeout,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		site:       opts.Site,
	}, nil
}

// Login authenticates with the UniFi controller
func (c *Client) Login(ctx context.Context, username, password string) error {
	slog.Info("Logging in to UniFi controller", "username", username)

	loginPayload := map[string]string{"username": username, "password": password}
	loginBody, err := json.Marshal(loginPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/login",
		strings.NewReader(string(loginBody)),
	)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	loginResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(loginResp.Body)
		return fmt.Errorf("login failed with status %s: %s", loginResp.Status, string(body))
	}

	var loginResult backupResp
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResult.Meta.Rc != "ok" {
		return fmt.Errorf("login failed: %s", loginResult.Meta.Msg)
	}

	slog.Info("Successfully logged in")
	return nil
}

// CreateBackup triggers a backup on the UniFi controller and returns the backup URL
func (c *Client) CreateBackup(ctx context.Context, username string, includeDays int) (string, error) {
	slog.Info("Triggering backup", "includeDays", includeDays)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/s/%s/cmd/backup", c.baseURL, c.site),
		strings.NewReader(fmt.Sprintf(`{"cmd":"backup","days":%d}`, includeDays)),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create backup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("backup request failed: %w", err)
	}
	defer resp.Body.Close()

	var backupResult backupResp
	if err := json.NewDecoder(resp.Body).Decode(&backupResult); err != nil {
		return "", fmt.Errorf("failed to decode backup response: %w", err)
	}

	if backupResult.Meta.Rc != "ok" || len(backupResult.Data) == 0 || backupResult.Data[0].URL == "" {
		if backupResult.Meta.Msg == "api.err.NoPermission" {
			slog.Info(fmt.Sprintf("Make sure the user '%s' is an Administrator rather than just a Site Administrator", username))
		}
		return "", fmt.Errorf("backup failed: response_code=%s, message=%s, data_length=%d",
			backupResult.Meta.Rc, backupResult.Meta.Msg, len(backupResult.Data))
	}

	backupURL := c.baseURL + backupResult.Data[0].URL
	slog.Info("Backup created successfully", "url", backupURL)

	return backupURL, nil
}

// DownloadResponse contains the backup file and metadata
type DownloadResponse struct {
	Body          io.ReadCloser
	ContentLength int64
}

// DownloadBackup downloads the backup file from the given URL
func (c *Client) DownloadBackup(ctx context.Context, backupURL string) (*DownloadResponse, error) {
	slog.Info("Downloading backup file")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, backupURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	downloadResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download backup: %w", err)
	}

	if downloadResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(downloadResp.Body)
		downloadResp.Body.Close()
		return nil, fmt.Errorf("download failed with status %s: %s", downloadResp.Status, string(body))
	}

	contentLength := downloadResp.ContentLength
	slog.Info("Backup download started", "size_bytes", contentLength)

	return &DownloadResponse{
		Body:          downloadResp.Body,
		ContentLength: contentLength,
	}, nil
}
