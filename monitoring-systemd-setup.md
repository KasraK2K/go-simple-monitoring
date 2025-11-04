# Run `monitoring` as a systemd service

This guide sets up your Go-built binary **`monitoring`** to run as a systemd service on Linux. It will:

- Write monitoring logs to `/var/log/monitoring` (configurable in `configs.json`)
- Run as a single service instance with automatic logging
- Auto-restart on crash
- Monitor system resources and configured heartbeat targets

> Assumptions
>
> - You already have a compiled binary named `monitoring` (no extension).
> - You have root (or sudo) access.
> - Your `configs.json` file is configured properly.
> - When rebuilding, run `templ generate ./web/views` before `go build` so the dashboard templ files are compiled into the binary.

---

## 1) Create a dedicated user and directories

```bash
# Create service user (no login)
sudo useradd --system --no-create-home --shell /usr/sbin/nologin monitoring || true

# App and logs directories
sudo mkdir -p /opt/monitoring
sudo mkdir -p /var/log/monitoring

# Place your compiled binary and config files here (adjust source path)
sudo cp /path/to/monitoring /opt/monitoring/monitoring
sudo cp /path/to/configs.json /opt/monitoring/configs.json

# Permissions
sudo chown -R monitoring:monitoring /opt/monitoring
sudo chown -R monitoring:monitoring /var/log/monitoring
sudo chmod 0755 /opt/monitoring/monitoring
sudo chmod 0644 /opt/monitoring/configs.json

# Note: The SQLite database file (monitoring.db) will be automatically created 
# in /opt/monitoring/ when the service starts for the first time
```

Create your `configs.json` configuration file:

```bash
sudo tee /opt/monitoring/configs.json >/dev/null <<'EOF'
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

# Path Configuration (use environment-based paths)
BASE_LOG_FOLDER=/var/log/monitoring
BASE_DATABASE_FOLDER=/opt/monitoring

# Database Configuration
DB_MAX_CONNECTIONS=30
DB_CONNECTION_TIMEOUT=30
DB_IDLE_TIMEOUT=300

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

## 2) systemd service unit

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
ReadWritePaths=/var/log/monitoring /opt/monitoring

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

## 3) Enable and start the service

```bash
# Enable service to start on boot
sudo systemctl enable monitoring.service

# Start the service
sudo systemctl start monitoring.service

# Check service status
sudo systemctl status monitoring.service
```

---

## 4) Manage the service

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

## 5) API mode (optional)

If you want to run the monitoring service in API mode (HTTP server), modify the `ExecStart` in the service unit:

```ini
# For API mode on port 8080
ExecStart=/opt/monitoring/monitoring --mode api --port 8080

# Or use environment variables
Environment=MODE=api
Environment=PORT=8080
ExecStart=/opt/monitoring/monitoring
```

Then add network access to the hardening section:

```ini
# Add if running API mode
PrivateNetwork=false
```

---

## 6) Log rotation

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

## 7) Verify monitoring output

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

## 8) Security notes

- `ProtectSystem=full` makes root FS read-only for the service; only paths in `ReadWritePaths` are writable.
- Service runs as dedicated `monitoring` user with minimal privileges.
- Environment file permissions are restricted (0640) to prevent unauthorized access to secrets.
- If running API mode, consider adding firewall rules to restrict access.

---

## 9) Storage management

Based on your `refresh_time` setting, monitor disk usage:

| Interval | Daily Size | Weekly Size | Monthly Size |
| -------- | ---------- | ----------- | ------------ |
| 2s       | ~59 MB     | ~413 MB     | ~1.8 GB      |
| 5s       | ~23.6 MB   | ~165 MB     | ~708 MB      |
| 10s      | ~11.8 MB   | ~83 MB      | ~354 MB      |

Use the built-in log cleanup or logrotate to manage storage:

```bash
# Check current disk usage
du -sh /var/log/monitoring/

# Manual cleanup (keeps last 7 days)
find /var/log/monitoring/ -name "*.log" -mtime +7 -delete
```

---

## 10) Updating the binary

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

## 11) Troubleshooting

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

## 12) Files to upload and deploy

For deployment, you need to upload these files to your server:

### Required files:
1. **Compiled binary**: `monitoring` (built with `go build`)
2. **Configuration file**: `configs.json` 
3. **Environment file**: `.env` (create based on `.env.example`)

### File locations on server:
- `/opt/monitoring/monitoring` - The compiled Go binary
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
```

### Database file:
- The SQLite database file `monitoring.db` will be automatically created in `/opt/monitoring/` when the service starts
- No manual upload required for the database

---

That's it! You now have a systemd-managed monitoring service that automatically logs system metrics and server heartbeats to daily log files.
