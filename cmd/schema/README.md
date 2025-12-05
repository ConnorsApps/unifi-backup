# UniFi Backup Configuration Schema Generator

This tool generates a JSON Schema for the UniFi backup configuration file format. The generated schema provides IDE autocomplete and inline documentation when editing configuration files.

## Usage

### Build the tool

```bash
go build -o unifi-backup-schema ./cmd/schema
```

### Generate schema to stdout

```bash
./unifi-backup-schema
```

### Generate schema to a file

```bash
./unifi-backup-schema -output config.schema.json
```

### Options

- `-output`: Output file path (default: stdout)
- `-pretty`: Pretty-print the JSON output (default: true)

## Using the Schema in VS Code

For YAML files, install the YAML extension and add to your VS Code settings:

```json
{
  "yaml.schemas": {
    "./config.schema.json": ["config.yaml", "config.yml"]
  }
}
```

This enables autocomplete and inline documentation while editing your configuration.

## Regenerating the Schema

If you modify the configuration structure in `pkg/config/config.go`, regenerate the schema:

```bash
go run ./cmd/schema -output config.schema.json
```

