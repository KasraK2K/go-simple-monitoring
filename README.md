# Log

## Prerequisites

- Go 1.21 or newer (module target: Go 1.24.6).
- [Templ CLI](https://templ.guide) for generating the dashboard components:

  ```bash
  go install github.com/a-h/templ/cmd/templ@latest
  ```

  Ensure the `templ` binary is on your `PATH` so generated Go files stay in sync with `.templ` sources.

## Local Development

1. Generate templ bindings (rerun whenever `web/views/*.templ` changes):

   ```bash
   templ generate ./web/views
   ```

2. Start the API/UI server:

   ```bash
   go run ./cmd
   ```

   To point at a custom configuration file, set `MONITOR_CONFIG_PATH` (defaults to `configs.json` in the project root):

   ```bash
   MONITOR_CONFIG_PATH=/path/to/your/config.json go run ./cmd
   ```

   or use the watcher:

   ```bash
   ./scripts/dev.sh
   ```

   The dashboard is available at http://localhost:3500.

## Build for Linux

Always regenerate templ output before creating release binaries:

```bash
templ generate ./web/views

GOOS=linux GOARCH=amd64 \
CGO_ENABLED=0 \
go build \
  -ldflags="-s -w" \
  -trimpath \
  -o monitoring \
  ./cmd
```

---

## Log File Storage

The system automatically logs monitoring data to daily files in `YYYY-MM-DD.log` format based on the `configs.json` configuration.

### Storage Configuration

In `configs.json`, you can control where logs are stored using the `storage` field:

- `"file"` - Write logs only to log files (recommended for production)
- `"db"` - Write logs only to database
- `"both"` - Write logs to both files and database
- `"none"` - Disable persistence entirely (no files or database writes and log rotation is skipped)

### Server Log Persistence

- `"persist_server_logs"` - When `true`, fetch `/monitoring` from each configured server and persist the response.
- Each server entry accepts `"table_name"`; when populated, a subdirectory is created under `path/servers/<table_name>` for file storage and the same identifier is used for the SQLite table. If `table_name` is empty the server is skipped.

### Automatic Log Rotation

Configure the `logrotate` block to prune old log files and database rows automatically:

```json
"logrotate": {
  "enabled": true,
  "max_age_days": 30
}
```

- `enabled`: set to `false` to disable builtin cleanup.
- `max_age_days`: number of days to retain logs (default 30). The scheduler runs once per day and cleans up immediately on startup (files and SQLite rows).

### Storage Requirements

| Interval | Daily Size | Weekly Size | Monthly Size | Yearly Size |
| -------- | ---------- | ----------- | ------------ | ----------- |
| 2s       | ~69 MB     | ~485 MB     | ~2.1 GB      | ~24.7 GB    |
| 5s       | ~28 MB     | ~194 MB     | ~0.83 GB     | ~9.9 GB     |
| 10s      | ~14 MB     | ~97 MB      | ~0.41 GB     | ~4.9 GB     |

**Notes**:

- Estimates assume each log entry is ~1.65 KB (two disks + three heartbeat targets). Actual usage will vary with disk count, heartbeat list, and payload size.
- Use the `CleanOldLogs()` function to automatically remove logs older than specified days to manage disk space.

---

## API Information

| Endpoint                | Method | Description                                                        |
| ----------------------- | ------ | ------------------------------------------------------------------ |
| `/`                     | GET    | Main dashboard UI with monitoring interface                        |
| `/api/v1/server-config` | GET    | Server configuration including refresh interval and server list    |
| `/api/v1/tables`        | GET    | Available database table names and count                           |
| `/monitoring`           | POST   | System monitoring data with optional filtering and table selection |

---

## API Testing

### Test Monitoring Endpoint

**Test without filters (current metrics):**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Test with date range filter:**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2024-01-01",
    "to": "2024-01-31"
  }'
```

**Test with from date only:**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2024-01-01"
  }'
```

**Test with to date only:**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "to": "2024-01-31"
  }'
```

**Test with Authorization (Production Mode):**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{}'
```

### Table-Specific Queries

The monitoring endpoint supports querying specific tables using the `table_name` parameter. This allows you to get data from specific server tables or the default monitoring table.

**List available tables:**

```bash
curl -X GET http://localhost:3500/api/v1/tables
```

Response:

```json
{
  "tables": ["default", "server_api_prod", "server_web_staging"],
  "count": 3
}
```

**Query specific table (all data):**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "server_api_prod"
  }'
```

**Query specific table with date range:**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "server_api_prod",
    "from": "2024-01-01",
    "to": "2024-01-31"
  }'
```

**Query default table explicitly:**

```bash
curl -X POST http://localhost:3500/monitoring \
  -H "Content-Type: application/json" \
  -d '{
    "table_name": "default",
    "from": "2024-01-01"
  }'
```

**Notes:**

- If `table_name` is omitted or empty, the default monitoring table is queried
- Use `"table_name": "default"` to explicitly query the main monitoring table
- Server tables are created automatically when servers are configured with `table_name` in `configs.json`
- Use `/api/v1/tables` endpoint to discover available table names
- Table-specific queries support all the same date filtering options as default queries
