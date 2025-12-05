package storage

import (
	"testing"
)

func TestParseSMBURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *smbConfig
		wantErr bool
	}{
		{
			name: "basic URL with host and share",
			url:  "smb://server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with custom port",
			url:  "smb://server:139/share",
			want: &smbConfig{
				Host:     "server",
				Port:     139,
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with username and password",
			url:  "smb://user:pass@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "pass",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with domain (DOMAIN;username format)",
			url:  "smb://DOMAIN;user:pass@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "pass",
				Domain:   "DOMAIN",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with domain (DOMAIN\\username format - URL encoded as %5C)",
			url:  "smb://DOMAIN%5Cuser:pass@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "pass",
				Domain:   "DOMAIN",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with email-style username (@ not treated as domain separator)",
			url:  "smb://user@company.com:pass@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user@company.com",
				Password: "pass",
				Domain:   "",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with base path",
			url:  "smb://server/share/path/to/backups",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Share:    "share",
				BasePath: "path/to/backups",
			},
			wantErr: false,
		},
		{
			name: "URL with username and base path",
			url:  "smb://user:pass@server/share/backups",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "pass",
				Share:    "share",
				BasePath: "backups",
			},
			wantErr: false,
		},
		{
			name: "complex URL with all components (semicolon domain separator)",
			url:  "smb://DOMAIN;admin:secret@192.168.1.100:139/backup/unifi/daily",
			want: &smbConfig{
				Host:     "192.168.1.100",
				Port:     139,
				Username: "admin",
				Password: "secret",
				Domain:   "DOMAIN",
				Share:    "backup",
				BasePath: "unifi/daily",
			},
			wantErr: false,
		},
		{
			name: "URL with IP address",
			url:  "smb://10.0.0.50/share",
			want: &smbConfig{
				Host:     "10.0.0.50",
				Port:     445,
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with username only (no password)",
			url:  "smb://user@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name:    "invalid scheme (http)",
			url:     "http://server/share",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid scheme (ftp)",
			url:     "ftp://server/share",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing share name",
			url:     "smb://server/",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing share name (no trailing slash)",
			url:     "smb://server",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "malformed URL",
			url:     "not a url at all",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			want:    nil,
			wantErr: true,
		},
		{
			name: "URL with special characters in password",
			url:  "smb://user:p@ss%21word@server/share",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Username: "user",
				Password: "p@ss!word",
				Share:    "share",
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name: "URL with space in base path (URL encoded)",
			url:  "smb://server/share/my%20backups",
			want: &smbConfig{
				Host:     "server",
				Port:     445,
				Share:    "share",
				BasePath: "my backups",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSMBURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSMBURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got == nil {
				t.Errorf("parseSMBURL() returned nil, want %v", tt.want)
				return
			}
			if got.Host != tt.want.Host {
				t.Errorf("parseSMBURL() Host = %v, want %v", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("parseSMBURL() Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.Username != tt.want.Username {
				t.Errorf("parseSMBURL() Username = %v, want %v", got.Username, tt.want.Username)
			}
			if got.Password != tt.want.Password {
				t.Errorf("parseSMBURL() Password = %v, want %v", got.Password, tt.want.Password)
			}
			if got.Domain != tt.want.Domain {
				t.Errorf("parseSMBURL() Domain = %v, want %v", got.Domain, tt.want.Domain)
			}
			if got.Share != tt.want.Share {
				t.Errorf("parseSMBURL() Share = %v, want %v", got.Share, tt.want.Share)
			}
			if got.BasePath != tt.want.BasePath {
				t.Errorf("parseSMBURL() BasePath = %v, want %v", got.BasePath, tt.want.BasePath)
			}
		})
	}
}
