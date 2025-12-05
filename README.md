# UniFi Backup

A Golang application to automatically backup your UniFi controller configuration. Supports local filesystem, SMB/CIFS shares, S3, and Google Cloud Storage.

Use SMB to back up directly to a [UniFi NAS](https://ui.com/integrations/network-storage) or any network share.

See [CONFIGURATION.md](CONFIGURATION.md) for setup details.

## Quick Start

```bash
# Set credentials
export UNIFI_URL=https://your-unifi-controller
export UNIFI_USER=admin
export UNIFI_PASS=your-password

# Run (backups save to ./backups by default)
go run github.com/ConnorsApps/unifi-backup

# Or use a config file
go run github.com/ConnorsApps/unifi-backup -config config.yaml
```

## Requirements

- UniFi controller (Network Application)
- User account with **Administrator** role (not just Site Administrator)
