# Deployment Guide

This guide shows how to deploy the Go monitoring service to production using systemd on Linux.

## Overview

Deploy your compiled `monitoring` binary as a systemd service that will:

- Run as a dedicated system user with minimal privileges
- Write monitoring logs and database to configurable locations
- Auto-restart on crash with proper resource limits
- Monitor system resources and configured heartbeat targets
- Serve the dashboard web interface (if enabled)

## Prerequisites

- Compiled binary named `monitoring` (no extension)
- Root or sudo access on the target server
- Properly configured `configs.json` file
- Environment variables configured in `.env` file

> **Build Requirements**: Run `templ generate ./web/views` before `go build` so the dashboard templ files are compiled into the binary.

---

## 1) Create system user and directories

```bash
# Create service user (no login)
sudo useradd --system --no-create-home --shell /usr/sbin/nologin monitoring || true

# App, logs, and database directories
sudo mkdir -p /opt/monitoring
sudo mkdir -p /var/syslogs/log
sudo mkdir -p /var/syslogs/database

# Place your compiled binary and config files here (adjust source paths)
sudo cp /path/to/monitoring /opt/monitoring/monitoring
sudo cp /path/to/configs.json /opt/monitoring/configs.json

# CRITICAL: Set proper ownership for all directories
sudo chown -R monitoring:monitoring /opt/monitoring
sudo chown -R monitoring:monitoring /var/syslogs

# File permissions
sudo chmod 0755 /opt/monitoring/monitoring
sudo chmod 0644 /opt/monitoring/configs.json

# Note: SQLite database and log files will be automatically created
# when the service starts for the first time
```

---

## 2) Configuration files

### configs.json

Create your `configs.json` configuration file:

```bash
sudo tee /opt/monitoring/configs.json >/dev/null <<'EOF'
{
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
      "name": "Production",
      "address": "https://monitoring.example.com",
      "table_name": "production_monitoring"
    },
    {
      "name": "Staging",
      "address": "https://staging-monitor.example.com",
      "table_name": "staging_monitoring"
    }
  ]
}
EOF
sudo chown monitoring:monitoring /opt/monitoring/configs.json
```

> **Note**: The `path` field in `configs.json` will be automatically overridden by the `BASE_LOG_FOLDER` environment variable, so the actual log path will be `/var/log/monitoring` as configured in the `.env` file.

### Environment Configuration

Create the environment configuration file `/opt/monitoring/.env`:

```bash
sudo tee /opt/monitoring/.env >/dev/null <<'EOF'
# Required Secrets
AES_SECRET=your-aes-secret-here
JWT_SECRET=your-jwt-secret-here

# Server Configuration
PORT=3500

# Environment
GO_ENV=production

# CORS Configuration
CORS_ALLOWED_ORIGINS=*

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20

# Logging
LOG_LEVEL=INFO

# Token Validation
CHECK_TOKEN=false

# Dashboard
HAS_DASHBOARD=true

# Path Configuration
BASE_LOG_FOLDER=/var/syslogs/log
SQLITE_DNS=/var/syslogs/database/monitoring.db

# Database Configuration
DB_MAX_CONNECTIONS=30
DB_CONNECTION_TIMEOUT=30
DB_IDLE_TIMEOUT=300

# PostgreSQL (optional when using storage ["postgresql"]) 
# POSTGRES_USER=monitoring
# POSTGRES_PASSWORD=monitoring
# POSTGRES_HOST=localhost
# POSTGRES_PORT=5432
# POSTGRES_DB=monitoring

# Server Monitoring
SERVER_MONITORING_TIMEOUT=15s

# HTTP Client Configuration (optional, uses defaults if not set)
# HTTP_MAX_CONNS_PER_HOST=10
# HTTP_MAX_IDLE_CONNS=100
# HTTP_MAX_IDLE_CONNS_PER_HOST=5
# HTTP_IDLE_CONN_TIMEOUT=90s
# HTTP_CONNECT_TIMEOUT=10s
# HTTP_REQUEST_TIMEOUT=30s
# HTTP_RESPONSE_HEADER_TIMEOUT=10s
# HTTP_MAX_RESPONSE_SIZE=10485760
# HTTP_TLS_HANDSHAKE_TIMEOUT=10s

# Time Configuration (optional)
# DISABLE_UTC_ENFORCEMENT=false
# DEFAULT_TIMEZONE=UTC
EOF
sudo chown monitoring:monitoring /opt/monitoring/.env
sudo chmod 0640 /opt/monitoring/.env
```

---

## 3) systemd service unit

Create the service unit at `/etc/systemd/system/monitoring.service`:

```ini
# /etc/systemd/system/monitoring.service
[Unit]
Description=System Monitoring Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=monitoring
Group=monitoring
WorkingDirectory=/opt/monitoring

# Load optional env file if present
EnvironmentFile=-/opt/monitoring/.env

# Start the monitoring service
ExecStart=/opt/monitoring/monitoring

# Hardening (tweak if your app needs more access)
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true
# Allow writes to logs and database paths
ReadWritePaths=/opt/monitoring /var/syslogs

# Restart policy
Restart=always
RestartSec=5s

# Resource limits (adjust as needed)
LimitNOFILE=1048576

# Logging
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Reload systemd units:

```bash
sudo systemctl daemon-reload
```

---

## 4) Enable and start the service

```bash
# Enable service to start on boot
sudo systemctl enable monitoring.service

# Start the service
sudo systemctl start monitoring.service

# Restart the service
sudo systemctl restart monitoring.service

# Check service status
sudo systemctl status monitoring.service
```

---

## 5) Service management

```bash
# View service status
sudo systemctl status monitoring.service

# View live logs
sudo journalctl -u monitoring.service -f

# View recent logs
sudo journalctl -u monitoring.service --since "1 hour ago"

# Stop the service
sudo systemctl stop monitoring.service

# Restart the service
sudo systemctl restart monitoring.service

# Disable auto-start on boot
sudo systemctl disable monitoring.service
```

---

## 6) API mode configuration

The monitoring service runs both API and dashboard on the same port (3500 by default). The current configuration already includes:

```ini
# Current service configuration (API + Dashboard on port 3500)
ExecStart=/opt/monitoring/monitoring

# Port is configured via .env file:
# PORT=3500
```

To change the port, update your `.env` file:

```bash
# Edit the port in .env
sudo nano /opt/monitoring/.env

# Change PORT=3500 to your desired port
PORT=3500
```

Then restart the service:

```bash
sudo systemctl restart monitoring
```

**Note**: The service already has network access enabled - no additional hardening changes needed.

---

## 7) Log rotation

Set up log rotation to manage disk space for monitoring logs:

```bash
sudo tee /etc/logrotate.d/monitoring >/dev/null <<'EOF'
/var/log/monitoring/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    su monitoring monitoring
}
EOF
```

---

## 8) Verify deployment

```bash
# Check if monitoring logs are being created
ls -lah /var/log/monitoring/

# View recent monitoring data
tail -f /var/log/monitoring/$(date +%Y-%m-%d).log

# Check service resource usage
systemctl show monitoring.service --property=MainPID
ps -p $(systemctl show monitoring.service --property=MainPID --value) -o pid,ppid,%cpu,%mem,cmd
```

---

## 9) Storage management

Storage usage depends on your configuration settings and the amount of data collected.

### 9.1) File Storage (Log Files)

**Base system monitoring data** (CPU, memory, disk, network, processes):

- Each entry: ~1.2 KB (JSON format)
- Storage by refresh interval:

| Refresh Interval | Entries/Day | Daily Size | Weekly Size | Monthly Size |
| ---------------- | ----------- | ---------- | ----------- | ------------ |
| 2s               | 43,200      | ~52 MB     | ~364 MB     | ~1.6 GB      |
| 5s               | 17,280      | ~21 MB     | ~147 MB     | ~630 MB      |
| 10s              | 8,640       | ~10 MB     | ~73 MB      | ~315 MB      |
| 30s              | 2,880       | ~3.5 MB    | ~25 MB      | ~105 MB      |
| 60s              | 1,440       | ~1.7 MB    | ~12 MB      | ~52 MB       |

### 9.2) Database Storage (SQLite)

**Database size estimates** (includes indexes and overhead):

- Each system monitoring record: ~2 KB (normalized tables)
- Database growth by refresh interval:

| Refresh Interval | Records/Day | Daily Growth | Weekly Growth | Monthly Growth |
| ---------------- | ----------- | ------------ | ------------- | -------------- |
| 2s               | 43,200      | ~86 MB       | ~602 MB       | ~2.6 GB        |
| 5s               | 17,280      | ~35 MB       | ~245 MB       | ~1.0 GB        |
| 10s              | 8,640       | ~17 MB       | ~120 MB       | ~515 MB        |
| 30s              | 2,880       | ~6 MB        | ~42 MB        | ~180 MB        |
| 60s              | 1,440       | ~3 MB        | ~21 MB        | ~90 MB         |

### 9.3) Heartbeat Services Impact

**Each heartbeat service adds** (per check):

- File storage: ~200 bytes per heartbeat result
- Database storage: ~350 bytes per heartbeat result

**Additional storage per heartbeat service**:

| Check Interval | Checks/Day | Daily Size (File) | Daily Size (DB) | Monthly Size (File) | Monthly Size (DB) |
| -------------- | ---------- | ----------------- | --------------- | ------------------- | ----------------- |
| 30s            | 2,880      | ~580 KB           | ~1 MB           | ~18 MB              | ~31 MB            |
| 60s            | 1,440      | ~290 KB           | ~500 KB         | ~9 MB               | ~15 MB            |
| 300s (5min)    | 288        | ~58 KB            | ~100 KB         | ~1.8 MB             | ~3 MB             |

**Example**: With 5 heartbeat services checking every 60s:

- Additional file storage: ~1.5 MB/day, ~45 MB/month
- Additional database storage: ~2.5 MB/day, ~75 MB/month

### 9.4) Remote Server Monitoring Impact

**Each remote server adds** (per monitoring call):

- File storage: ~1.5 KB per server response
- Database storage: ~2.5 KB per server response

**Additional storage per remote server**:

| Monitoring Interval | Calls/Day | Daily Size (File) | Daily Size (DB) | Monthly Size (File) | Monthly Size (DB) |
| ------------------- | --------- | ----------------- | --------------- | ------------------- | ----------------- |
| 30s                 | 2,880     | ~4.3 MB           | ~7.2 MB         | ~130 MB             | ~216 MB           |
| 60s                 | 1,440     | ~2.2 MB           | ~3.6 MB         | ~66 MB              | ~108 MB           |
| 300s (5min)         | 288       | ~430 KB           | ~720 KB         | ~13 MB              | ~22 MB            |

**Example**: With 3 remote servers monitored every 60s:

- Additional file storage: ~6.6 MB/day, ~198 MB/month
- Additional database storage: ~10.8 MB/day, ~324 MB/month

### 9.5) Storage Management Commands

```bash
# Check current disk usage
du -sh /var/log/monitoring/
du -sh /opt/monitoring/monitoring.db

# Check database size and record counts
sqlite3 /opt/monitoring/monitoring.db "
SELECT
    name as table_name,
    (SELECT COUNT(*) FROM \`default\`) as system_records,
    (SELECT COUNT(*) FROM heartbeat_results) as heartbeat_records
FROM sqlite_master WHERE type='table';
"

# File cleanup (keeps last 7 days)
find /var/log/monitoring/ -name "*.log" -mtime +7 -delete

# Database cleanup (keeps last 30 days of data)
sqlite3 /opt/monitoring/monitoring.db "
DELETE FROM \`default\` WHERE datetime(timestamp) < datetime('now', '-30 days');
DELETE FROM heartbeat_results WHERE datetime(timestamp) < datetime('now', '-30 days');
VACUUM;
"
```

### 9.6) Recommended Storage Planning

**For production environments**:

- Monitor storage usage weekly
- Set up automated cleanup (via cron or systemd timers)
- Consider disk space requirements based on your specific configuration
- Plan for 2-3x the calculated storage as safety margin

**Example total storage estimate** (5s refresh, 5 heartbeats/60s, 3 servers/60s):

- Files: ~21 MB + 1.5 MB + 6.6 MB = ~29 MB/day (~870 MB/month)
- Database: ~35 MB + 2.5 MB + 10.8 MB = ~48 MB/day (~1.4 GB/month)
- **Total: ~77 MB/day (~2.3 GB/month)**

---

## 10) Security considerations

- `ProtectSystem=full` makes root FS read-only for the service; only paths in `ReadWritePaths` are writable.
- Service runs as dedicated `monitoring` user with minimal privileges.
- Environment file permissions are restricted (0640) to prevent unauthorized access to secrets.
- If running API mode, consider adding firewall rules to restrict access.

---

## 11) Updating the binary

```bash
# Regenerate templ output (if any UI changes were made)
templ generate ./web/views

# Copy new binary
sudo cp /path/to/new/monitoring /opt/monitoring/monitoring
sudo chown monitoring:monitoring /opt/monitoring/monitoring
sudo chmod 0755 /opt/monitoring/monitoring

# Restart service to use new binary
sudo systemctl restart monitoring.service

# Verify new version is running
sudo systemctl status monitoring.service
```

---

## 12) Files to upload and deploy

For deployment, you need to upload these files to your server:

### Required files:

1. **Compiled binary**: `monitoring` (built with `go build` - includes embedded web assets)
2. **Configuration file**: `configs.json`
3. **Environment file**: `.env` (create based on `.env.example`)

### File locations on server:

- `/opt/monitoring/monitoring` - The compiled Go binary (with embedded web assets)
- `/opt/monitoring/configs.json` - JSON configuration file
- `/opt/monitoring/.env` - Environment variables file
- `/etc/systemd/system/monitoring.service` - Systemd service unit

### Build and upload steps:

```bash
# 1. On your development machine, build the binary
templ generate ./web/views  # Generate templ files
go build -o monitoring ./cmd

# 2. Upload files to server (adjust paths as needed)
scp monitoring user@server:/tmp/
scp configs.json user@server:/tmp/
scp .env user@server:/tmp/

# 3. On the server, move files to correct locations (as shown in step 1)
sudo cp /tmp/monitoring /opt/monitoring/monitoring
sudo cp /tmp/configs.json /opt/monitoring/configs.json
sudo cp /tmp/.env /opt/monitoring/.env
sudo chown -R monitoring:monitoring /opt/monitoring
```

### Database file:

- The SQLite database file will be automatically created in `/var/syslogs/database/` when the service starts
- No manual upload required for the database

### Important Permission Note:

**CRITICAL**: After deployment, if you see "database is closed" errors in logs, ensure proper ownership:

```bash
# Fix database directory permissions
sudo chown -R monitoring:monitoring /var/syslogs
sudo systemctl restart monitoring
```

---

## 13) Troubleshooting

### Common Issues

#### NAMESPACE Error (status=226)

If you get `status=226/NAMESPACE` error when starting the service, it's usually caused by systemd security restrictions. For servers that only need network access (no local database/logs), use this simplified service configuration:

```ini
# Simplified /etc/systemd/system/monitoring.service for API-only servers
[Unit]
Description=System Monitoring Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=monitoring
Group=monitoring
WorkingDirectory=/opt/monitoring

# Load optional env file if present
EnvironmentFile=-/opt/monitoring/.env

# Start the monitoring service
ExecStart=/opt/monitoring/monitoring

# Minimal hardening - remove problematic restrictions
NoNewPrivileges=true
# Remove: ProtectSystem, ProtectHome, PrivateTmp, ReadWritePaths

# Restart policy
Restart=always
RestartSec=5s

# Resource limits
LimitNOFILE=1048576

# Logging
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

After updating the service file:

```bash
sudo systemctl daemon-reload
sudo systemctl restart monitoring.service
```

#### Other Troubleshooting Commands

```bash
# Check service logs for errors
sudo journalctl -u monitoring.service --since "10 minutes ago"

# Verify config file is valid JSON
sudo -u monitoring cat /opt/monitoring/configs.json | jq .

# Test binary manually
sudo -u monitoring /opt/monitoring/monitoring --help

# Check file permissions
ls -la /opt/monitoring/
ls -la /var/log/monitoring/
```

---

That's it! You now have a systemd-managed monitoring service that automatically logs system metrics and server heartbeats to daily log files and database.
