# Go System Monitoring Service

A comprehensive Go-based monitoring service that tracks system resources, monitors remote servers, and provides heartbeat checks with a web dashboard interface.

## 📚 Documentation Navigation

| **Quick Start** | **Deployment** | **Configuration** | **Advanced** |
|------------------|----------------|-------------------|--------------|
| [Local Development](#local-development) | [🚀 Production Deployment](docs/production-deployment.md) | [🔧 CLI Usage](docs/cli-usage.md) | [🐳 Docker Setup](docs/docker-deployment.md) |
| [Configuration](#configuration) | [🌐 Nginx Setup](docs/nginx-setup.md) | [🗄️ PostgreSQL/TimescaleDB](docs/postgresql-setup.md) | [🛠️ Systemd Setup](docs/monitoring-systemd-setup.md) |
| [API Reference](#api-endpoints) | [☁️ Cloud Deployment](docs/cloud-deployment.md) | [📊 Dashboard Guide](docs/dashboard-guide.md) | [🔍 Troubleshooting](docs/troubleshooting.md) |

> **🔗 All documentation is cross-linked for easy navigation. Missing a document? [Create an issue](https://github.com/your-repo/issues/new)!**

## Features

- **System Monitoring**: CPU, memory, disk space, network I/O, disk I/O, and process statistics
- **Remote Server Monitoring**: Monitor multiple remote monitoring endpoints
- **Heartbeat Checks**: Monitor external services and APIs for availability
- **Web Dashboard**: Real-time monitoring dashboard with configurable access
- **Pluggable Storage**: File, SQLite, and PostgreSQL (TimescaleDB) backends
- **Rate Limiting**: Built-in request rate limiting with configurable thresholds
- **CORS Support**: Configurable cross-origin resource sharing
- **Log Rotation**: Automatic cleanup of old logs and database records
- **Security**: JWT-based authentication with AES encryption

## Prerequisites

- Go 1.21 or newer (module target: Go 1.24.6)
- [Templ CLI](https://templ.guide) for generating dashboard components:

  ```bash
  go install github.com/a-h/templ/cmd/templ@latest
  ```

  Ensure the `templ` binary is on your `PATH` so generated Go files stay in sync with `.templ` sources.

## Project Structure

```
├── cmd/                    # Main application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/       # HTTP request handlers
│   │   ├── logics/         # Business logic
│   │   └── models/         # Data models
│   ├── config/             # Configuration management
│   └── utils/              # Utility functions
├── web/
│   └── views/              # Templ template files
├── configs.json            # Monitoring configuration
├── .env.example            # Environment variables template
└── DEPLOYMENT.md           # Production deployment guide
```

## Local Development

### 1. Environment Setup

Copy the environment template and configure your settings:

```bash
cp .env.example .env
```

**IMPORTANT**: Set required security variables in your `.env` file:

```bash
# Required Secrets (generate secure random strings)
AES_SECRET=your-32-character-aes-secret-key
JWT_SECRET=your-16-character-jwt-secret

# Server Configuration
PORT=3500

# Dashboard
HAS_DASHBOARD=true
DASHBOARD_DEFAULT_RANGE=24h # 1h, 6h, 24h, 7d, 30d. Leave empty for live data only.

# Path Configuration
BASE_LOG_FOLDER=./logs
SQLITE_DNS=./monitoring.db

# CORS Configuration (for development)
CORS_ALLOWED_ORIGINS=http://localhost:3500,http://127.0.0.1:3500

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
```

### 2. Configuration File

Create or modify `configs.json` for monitoring settings:

```json
{
  "path": "./logs",
  "refresh_time": "5s",
  "storage": ["file", "sqlite", "postgresql"],
  "persist_server_logs": true,
  "logrotate": {
    "enabled": true,
    "max_age_days": 30
  },
  "heartbeat": [
    {
      "name": "Google",
      "url": "https://www.google.com",
      "timeout": 5
    },
    {
      "name": "GitHub API",
      "url": "https://api.github.com",
      "timeout": 5
    }
  ],
  "servers": [
    {
      "name": "Production API",
      "address": "https://api.example.com/monitoring",
      "table_name": "production_monitoring"
    }
  ]
}
```

### 3. Generate Templates and Run

```bash
# Generate templ bindings (required before first run and after template changes)
templ generate ./web/views

# Start the development server
go run ./cmd
```

Or use the development watcher:

```bash
./scripts/dev.sh
```

The dashboard will be available at http://localhost:3500.

### 4. Custom Configuration Path

To use a different configuration file:

```bash
MONITOR_CONFIG_PATH=/path/to/your/config.json go run ./cmd
```

## Environment Configuration

The application uses centralized environment configuration. All available variables:

### Required Variables

- `AES_SECRET` - AES encryption key (minimum 32 characters)
- `JWT_SECRET` - JWT signing key (minimum 16 characters)

### Server Configuration

- `PORT` - Server port (default: 3500)
- `GO_ENV` - Environment mode (development/production)

### Security & Access

- `CORS_ALLOWED_ORIGINS` - Comma-separated allowed origins
- `CHECK_TOKEN` - Enable token validation (default: false)
- `HAS_DASHBOARD` - Enable/disable dashboard access (default: true)

### Rate Limiting

- `RATE_LIMIT_ENABLED` - Enable rate limiting (default: true)
- `RATE_LIMIT_RPS` - Requests per second (default: 10)
- `RATE_LIMIT_BURST` - Burst capacity (default: 20)

### Storage Paths

- `BASE_LOG_FOLDER` - Log files directory (default: ./logs)

### Database Configuration

- `DB_MAX_CONNECTIONS` - Maximum database connections (default: 30)
- `DB_CONNECTION_TIMEOUT` - Connection timeout in seconds (default: 30)
- `DB_IDLE_TIMEOUT` - Idle connection timeout in seconds (default: 300)
- `SQLITE_DNS` - SQLite database file path/DSN (default: `./monitoring.db`)
- PostgreSQL (used when `storage` contains `"postgresql"`):
  - `POSTGRES_USER` (default: `monitoring`)
  - `POSTGRES_PASSWORD` (default: `monitoring`)
  - `POSTGRES_HOST` (default: `localhost`)
  - `POSTGRES_PORT` (default: `5432`)
  - `POSTGRES_DB` (default: `monitoring`)
  - The app automatically builds a DSN from these variables.

### Monitoring Configuration

- `SERVER_MONITORING_TIMEOUT` - Remote server timeout (default: 15s)
- `LOG_LEVEL` - Logging level (default: INFO)

### Downsampling (historical queries)

- `MONITORING_DOWNSAMPLE_MAX_POINTS` - Target number of points for historical queries (default: 150).
  - If `0`: downsampling disabled (all rows returned).
  - If `> 0`: smart downsampling applies when data exceeds this threshold.
  - Uses TimescaleDB `time_bucket()` when available, falls back to `ntile()` for standard PostgreSQL.
  - Recommended values: 100-500 for optimal performance/detail balance.

## Storage Configuration

Configure storage behavior in `configs.json` using an array:

- `"file"` - Write logs to log files
- `"sqlite"` - Write logs to SQLite database
- `"postgresql"` - Write logs to PostgreSQL (TimescaleDB recommended)

Notes:

- Set multiple backends at once, e.g. `["file", "sqlite"]`.
- An empty array disables persistence entirely.
- Breaking change: previous `"db"` and `"both"` values are removed. Use the array form instead, e.g. `["sqlite"]`.

Example:

```json
{
  "path": "./logs",
  "refresh_time": "5s",
  "storage": ["file", "sqlite"],
  "persist_server_logs": true
}
```

### PostgreSQL + TimescaleDB (recommended)

PostgreSQL persistence works out of the box. TimescaleDB is recommended for optimal time-series performance:

- **Without TimescaleDB**: Uses standard PostgreSQL with `ntile()` downsampling
- **With TimescaleDB**: Automatically uses `time_bucket()` for efficient time-based downsampling
- **Auto-detection**: System automatically detects TimescaleDB extension and chooses optimal strategy
- **Performance**: TimescaleDB provides significantly better performance for large time-series datasets

#### Quick Setup with Docker

```bash
# Start TimescaleDB with one command
./scripts/postgres-timescale-up.sh

# Stop when done
./scripts/postgres-timescale-up.sh --stop
```

#### Configuration

Set `"storage": ["postgresql"]` in `configs.json` and configure `.env`:

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=monitoring
POSTGRES_USER=monitoring
POSTGRES_PASSWORD=your_secure_password
```

**📖 For detailed setup instructions, see [PostgreSQL Setup Guide](docs/postgresql-setup.md)**

#### Advanced Setup

For production deployment, manual installation, or cloud setup, see the comprehensive guides:

- 🗄️ **[PostgreSQL Setup Guide](docs/postgresql-setup.md)** - Complete setup instructions
- 🚀 **[Production Deployment](docs/production-deployment.md)** - Server deployment
- ☁️ **[Cloud Deployment](docs/cloud-deployment.md)** - AWS, GCP, Azure options

> **💡 The application automatically creates and optimizes database tables, including TimescaleDB hypertables when available. No manual SQL setup required!**

### What Gets Persisted (Compact Format)

To reduce storage, only UI‑used fields are persisted for local monitoring snapshots and remote server payloads:

- `time` (added automatically)
- `cpu_usage_percent`
- `ram_used_percent`
- `disk_spaces` array: `path`, `device`, `filesystem`, `total_bytes`, `used_bytes`, `available_bytes`, `used_pct`
- `network_bytes_sent`, `network_bytes_recv`
- `process_load_avg_1`, `process_load_avg_5`, `process_load_avg_15`
- `heartbeat` (name, url, status, response_ms, response_time, last_checked)
- `server_metrics` (unchanged, already compact)

Example single entry (pretty‑printed):

```json
{
  "time": "2024-11-14T12:34:56Z",
  "cpu_usage_percent": 12.3,
  "ram_used_percent": 45.6,
  "disk_spaces": [
    {
      "path": "/",
      "device": "/dev/disk1s1",
      "filesystem": "apfs",
      "total_bytes": 500000000000,
      "used_bytes": 300000000000,
      "available_bytes": 200000000000,
      "used_pct": 60.0
    }
  ],
  "network_bytes_sent": 123456789,
  "network_bytes_recv": 987654321,
  "process_load_avg_1": 0.42,
  "process_load_avg_5": 0.37,
  "process_load_avg_15": 0.31,
  "heartbeat": [
    {
      "name": "Google",
      "url": "https://www.google.com",
      "status": "up",
      "response_ms": 120,
      "response_time": "120ms",
      "last_checked": "2024-11-14T12:34:56Z"
    }
  ],
  "server_metrics": []
}
```

### Storage Requirements (Compact, Typical)

Estimates below are per storage backend. If you select multiple backends in `storage`, the storage footprint is roughly additive. Values assume 1–2 disks, ≤5 heartbeats, ≤5 remote servers.

File (JSON logs)

| Interval | Daily Size | Weekly Size | Monthly Size |
| -------- | ---------- | ----------- | ------------ |
| 2s       | ~35–45 MB  | ~245–315 MB | ~1.0–1.4 GB  |
| 5s       | ~14–18 MB  | ~98–126 MB  | ~0.4–0.6 GB  |
| 10s      | ~7–9 MB    | ~49–63 MB   | ~0.2–0.3 GB  |

SQLite (JSON per row + indexes)

| Interval | Daily Size | Weekly Size | Monthly Size |
| -------- | ---------- | ----------- | ------------ |
| 2s       | ~40–55 MB  | ~280–385 MB | ~1.2–1.7 GB  |
| 5s       | ~16–22 MB  | ~112–154 MB | ~0.5–0.75 GB |
| 10s      | ~8–11 MB   | ~56–77 MB   | ~0.25–0.35 GB|

PostgreSQL (raw hypertable, no compression)

| Interval | Daily Size | Weekly Size | Monthly Size |
| -------- | ---------- | ----------- | ------------ |
| 2s       | ~40–60 MB  | ~280–420 MB | ~1.2–1.8 GB  |
| 5s       | ~16–24 MB  | ~112–168 MB | ~0.5–0.8 GB  |
| 10s      | ~8–12 MB   | ~56–84 MB   | ~0.25–0.4 GB |

PostgreSQL with TimescaleDB downsampling (example policy)

- Keep raw data 7 days, then maintain 1h aggregates indefinitely.
- Raw segment ~ the “PostgreSQL (raw)” weekly line (e.g., 280–420 MB at 2s).
- 1h continuous aggregates typically compress by 80–95% vs raw; monthly total is often ≤ 0.3–0.6 GB depending on retention and policies.

Notes:

- These are indicative ranges; real usage varies with retention, number of disks, and remote servers.
- For PostgreSQL, enabling compression and adjusting retention can materially reduce footprint.

## API Endpoints

| Endpoint                | Method | Description                                                        |
| ----------------------- | ------ | ------------------------------------------------------------------ |
| `/`                     | GET    | Main dashboard UI (if `HAS_DASHBOARD=true`)                        |
| `/api/v1/server-config` | GET    | Server configuration including refresh interval and server list    |
| `/api/v1/tables`        | GET    | Available database table names and count                           |
| `/monitoring`           | POST   | System monitoring data with optional filtering and table selection |

## API Testing

### Basic Monitoring Data

```bash
# Get current system metrics
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Date Range Filtering

```bash
# Get metrics for date range
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2024-01-01",
    "to": "2024-01-31"
  }'
```

### Table-Specific Queries

```bash
# List available tables
curl -X GET http://localhost:3500/api/v1/tables

# Query specific server table
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "production_monitoring",
    "from": "2024-01-01"
  }'
```

### With Authentication (Production)

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{}'
```

## Build for Production

### CLI Monitor Tool

Build the monitoring CLI tool for different platforms using the build script:

```bash
# Build for specific platforms
./scripts/build-monitor.sh windows      # Builds monitor-windows.exe
./scripts/build-monitor.sh linux       # Builds monitor-linux
./scripts/build-monitor.sh mac         # Builds monitor-mac

# Build for multiple platforms at once
./scripts/build-monitor.sh windows linux mac
```

The CLI monitor tool allows you to monitor remote monitoring services from the command line:

```bash
# Local monitoring (current system)
./bin/monitor-linux

# Remote monitoring
./bin/monitor-linux -url http://localhost:3500/monitoring

# With authentication
./bin/monitor-linux -url http://localhost:3500/monitoring -token YOUR_TOKEN

# Additional options
./bin/monitor-linux -compact           # Compact display mode
./bin/monitor-linux -details          # Show detailed information
./bin/monitor-linux -refresh 1s       # Custom refresh rate
```

### Main Service

Always generate templates before building the main service:

```bash
# Generate templates
templ generate ./web/views

# Build for Linux
GOOS=linux GOARCH=amd64 \
CGO_ENABLED=0 \
go build \
  -ldflags="-s -w" \
  -trimpath \
  -o monitoring \
  ./cmd
```

## Server Configuration

The application can monitor remote servers that expose a `/monitoring` endpoint. Configure in `configs.json`:

```json
{
  "servers": [
    {
      "name": "Production API",
      "address": "https://api.example.com",
      "table_name": "production_monitoring"
    },
    {
      "name": "Staging Environment",
      "address": "https://staging.example.com",
      "table_name": "staging_monitoring"
    }
  ]
}
```

**Notes:**

- Each server must expose a `/monitoring` endpoint returning system metrics
- `table_name` creates separate storage for each server's data
- Unavailable servers won't break overall functionality
- Server monitoring includes automatic error handling and timeouts

## Development Commands

```bash
# Generate templates (run after .templ file changes)
templ generate ./web/views

# Run development server
go run ./cmd

# Run with custom config
MONITOR_CONFIG_PATH=custom-config.json go run ./cmd

# Build for current platform
go build -o monitoring ./cmd

# Run tests
go test ./...

# Check for issues
go vet ./...
```

## Deployment

For production deployment instructions, see [DEPLOYMENT.md](./DEPLOYMENT.md).

## Security Considerations

- Always set strong, unique values for `AES_SECRET` and `JWT_SECRET`
- Configure `CORS_ALLOWED_ORIGINS` appropriately for your environment
- Use `HAS_DASHBOARD=false` to disable dashboard in API-only deployments
- Enable `CHECK_TOKEN=true` for production token validation
- Monitor rate limiting settings based on your traffic patterns

## Contributing

1. Ensure all templates are generated: `templ generate ./web/views`
2. Run tests: `go test ./...`
3. Check code quality: `go vet ./...`
4. Follow existing code patterns and centralized configuration approach
