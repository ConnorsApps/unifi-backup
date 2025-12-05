# Configuration

Configuration can be provided via:
1. **Environment variables** (highest priority)
2. **Config file** (YAML or JSON)
3. **Default values**

## Config File

Pass a config file with `-config path/to/config.yaml`. Supports `.yaml`, `.yml`, or `.json`.

If no `-config` flag is provided, the app auto-detects `config.yaml`, `config.yml`, or `config.json` in the current directory.

A `.env` file in the current directory is also loaded automatically.

See `config.example.yaml` for a full example.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `UNIFI_URL` | UniFi controller URL | |
| `UNIFI_USER` | Username (must be Administrator role) | |
| `UNIFI_PASS` | Password | |
| `UNIFI_SITE` | Site name | `default` |
| `UNIFI_INCLUDE_DAYS` | Days of history to include (0 = current state only) | `0` |
| `UNIFI_INSECURE` | Skip TLS verification for self-signed certs | `false` |
| `UNIFI_TIMEOUT` | HTTP timeout for backup operations (e.g., 10m, 1h, 30s) | `10m` |
| `UNIFI_MAX_RETRIES` | Maximum number of retry attempts | `3` |
| `STORAGE_URL` | Storage backend URL (see below) | `file://./backups` |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `LOG_FORMAT` | Log format: `text`, `json` | `text` |
| `RETENTION_KEEP_LAST` | Number of backups to keep (0 = unlimited) | `7` |

## Storage Backends

| Scheme | Description | Example |
|--------|-------------|---------|
| `file://` | Local filesystem | `file://./backups` |
| `smb://` | SMB/CIFS network share | `smb://user:pass@server/share/path` |
| `s3://` | Amazon S3 | `s3://bucket-name` |
| `gs://` | Google Cloud Storage | `gs://bucket-name` |

### SMB URL Format

```
smb://[DOMAIN\]username[:password]@host[:port]/share[/path]
```

Examples:
- `smb://admin:password@192.168.1.10/backups`
- `smb://admin:password@nas.local:445/backups/unifi`
- `smb://DOMAIN\user:password@nas.local/share`
