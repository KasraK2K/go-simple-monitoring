# Go System Monitoring Service

A comprehensive Go-based monitoring service that tracks system resources, monitors remote servers, and provides heartbeat checks with a web dashboard interface.

## Features

- **System Monitoring**: CPU, memory, disk space, network I/O, disk I/O, and process statistics
- **Remote Server Monitoring**: Monitor multiple remote monitoring endpoints
- **Heartbeat Checks**: Monitor external services and APIs for availability
- **Web Dashboard**: Real-time monitoring dashboard with configurable access
- **Dual Storage**: Support for file-based logging and SQLite database storage
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

# Path Configuration
BASE_LOG_FOLDER=./logs
BASE_DATABASE_FOLDER=.

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
  "storage": "both",
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
- `BASE_DATABASE_FOLDER` - Database directory (default: .)

### Database Configuration
- `DB_MAX_CONNECTIONS` - Maximum database connections (default: 30)
- `DB_CONNECTION_TIMEOUT` - Connection timeout in seconds (default: 30)
- `DB_IDLE_TIMEOUT` - Idle connection timeout in seconds (default: 300)

### Monitoring Configuration
- `SERVER_MONITORING_TIMEOUT` - Remote server timeout (default: 15s)
- `LOG_LEVEL` - Logging level (default: INFO)

## Storage Configuration

Configure storage behavior in `configs.json`:

- `"file"` - Write logs only to log files
- `"db"` - Write logs only to SQLite database
- `"both"` - Write logs to both files and database (recommended)
- `"none"` - Disable persistence entirely

### Storage Requirements (Development)

| Interval | Daily Size | Weekly Size | Monthly Size |
| -------- | ---------- | ----------- | ------------ |
| 2s       | ~69 MB     | ~485 MB     | ~2.1 GB      |
| 5s       | ~28 MB     | ~194 MB     | ~0.83 GB     |
| 10s      | ~14 MB     | ~97 MB      | ~0.41 GB     |

**Note**: Estimates assume standard system monitoring with heartbeat checks. Actual usage varies with configuration.

## API Endpoints

| Endpoint                | Method | Description                                                        |
| ----------------------- | ------ | ------------------------------------------------------------------ |
| `/`                     | GET    | Main dashboard UI (if `HAS_DASHBOARD=true`)                       |
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