package unifi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginUsesUniFiOSAuthEndpointAndStoresCSRF(t *testing.T) {
	t.Parallel()

	var sawAuthLogin bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed decoding payload: %v", err)
		}
		if payload["username"] != "backup" || payload["password"] != "secret" {
			t.Fatalf("unexpected payload: %#v", payload)
		}

		sawAuthLogin = true
		w.Header().Set("x-updated-csrf-token", "csrf-token-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, ClientOptions{Site: "default"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.Login(context.Background(), "backup", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if !sawAuthLogin {
		t.Fatal("expected /api/auth/login to be called")
	}
	if client.csrfToken != "csrf-token-123" {
		t.Fatalf("csrf token not stored, got %q", client.csrfToken)
	}
}

func TestCreateBackupUsesProxyEndpointAndCSRF(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/login":
			w.Header().Set("x-updated-csrf-token", "csrf-token-abc")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		case "/proxy/network/api/s/default/cmd/backup":
			if got := r.Header.Get("X-Csrf-Token"); got != "csrf-token-abc" {
				t.Fatalf("unexpected X-Csrf-Token header: %q", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading request body: %v", err)
			}
			if !strings.Contains(string(body), `"cmd":"backup"`) {
				t.Fatalf("expected backup command in body, got: %s", string(body))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[{"url":"/dl/backup/test.unf"}]}`))
			return
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, ClientOptions{Site: "default"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.Login(context.Background(), "backup", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	backupURL, err := client.CreateBackup(context.Background(), "backup", 0)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	wantURL := server.URL + "/proxy/network/dl/backup/test.unf"
	if backupURL != wantURL {
		t.Fatalf("CreateBackup() URL = %q, want %q", backupURL, wantURL)
	}
}

func TestDownloadBackupNormalizesRelativeURLToProxyPath(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxy/network/dl/backup/test.unf" {
			t.Fatalf("unexpected download path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backup-bytes"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, ClientOptions{Site: "default"})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.DownloadBackup(context.Background(), "/dl/backup/test.unf")
	if err != nil {
		t.Fatalf("DownloadBackup() error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(body) != "backup-bytes" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
