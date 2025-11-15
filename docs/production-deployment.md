# Production Deployment Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [🌐 Nginx Setup](nginx-setup.md) | [🛠️ Systemd Setup](monitoring-systemd-setup.md)

Complete guide for deploying Go System Monitoring Service to production environments.

## 🚀 Quick Deployment Checklist

- [ ] **Server Setup**: Linux server with Go 1.21+
- [ ] **Database**: PostgreSQL/TimescaleDB configured
- [ ] **Application**: Binary built and configured
- [ ] **Reverse Proxy**: Nginx with SSL
- [ ] **Process Manager**: Systemd service
- [ ] **Security**: Firewall and authentication
- [ ] **Monitoring**: Health checks and logs

## 🛠️ Server Prerequisites

### System Requirements

| **Component** | **Minimum** | **Recommended** | **Notes** |
|---------------|-------------|-----------------|-----------|
| **CPU** | 1 core | 2+ cores | More cores for multiple monitored servers |
| **RAM** | 512MB | 2GB+ | Depends on data retention period |
| **Storage** | 5GB | 20GB+ | For logs and database storage |
| **OS** | Ubuntu 20.04+ | Ubuntu 22.04 LTS | Or CentOS 8+, RHEL 8+ |

### Required Software

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y \
    curl \
    wget \
    unzip \
    nginx \
    postgresql \
    postgresql-contrib \
    certbot \
    python3-certbot-nginx

# CentOS/RHEL/Rocky Linux
sudo dnf install -y \
    curl \
    wget \
    unzip \
    nginx \
    postgresql \
    postgresql-server \
    postgresql-contrib \
    certbot \
    python3-certbot-nginx
```

## 📦 Application Deployment

### 1. Create Deployment User

```bash
# Create dedicated user for security
sudo useradd -r -s /bin/false -d /opt/monitoring monitoring

# Create application directory
sudo mkdir -p /opt/monitoring/{bin,config,logs,data}
sudo chown -R monitoring:monitoring /opt/monitoring
```

### 2. Build and Deploy Application

```bash
# On your development machine
git clone https://github.com/your-repo/api.go-monitoring.git
cd api.go-monitoring

# Build for Linux (if building on different OS)
GOOS=linux GOARCH=amd64 go build -o monitoring-server cmd/main.go

# Or build locally on server
go build -o monitoring-server cmd/main.go

# Copy to server
scp monitoring-server user@your-server:/tmp/
ssh user@your-server

# On server: move to production location
sudo mv /tmp/monitoring-server /opt/monitoring/bin/
sudo chown monitoring:monitoring /opt/monitoring/bin/monitoring-server
sudo chmod +x /opt/monitoring/bin/monitoring-server
```

### 3. Configuration Setup

```bash
# Create configuration files
sudo cp configs_example.json /opt/monitoring/config/configs.json
sudo cp .env.example /opt/monitoring/config/.env

# Edit configuration
sudo nano /opt/monitoring/config/.env
```

### Production `.env` Configuration

```bash
# Server Configuration
PORT=3500
ENVIRONMENT=production

# Security (Generate strong secrets!)
AES_SECRET=your-32-character-aes-secret-here
JWT_SECRET=your-32-character-jwt-secret-here

# Database Configuration
POSTGRES_USER=monitoring
POSTGRES_PASSWORD=your_secure_db_password
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=monitoring

# Logging and Storage
BASE_LOG_FOLDER=/opt/monitoring/logs
LOG_LEVEL=INFO
MONITORING_DOWNSAMPLE_MAX_POINTS=200

# Security
CHECK_TOKEN=true
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20

# Dashboard (disable if using separate frontend)
HAS_DASHBOARD=true
DASHBOARD_DEFAULT_RANGE=24h

# CORS (adjust for your domain)
CORS_ALLOWED_ORIGINS=https://your-domain.com,https://www.your-domain.com
```

### Production `configs.json`

```json
{
  "refresh_time": "30s",
  "storage": ["postgresql"],
  "persist_server_logs": true,
  "logrotate": {
    "enabled": true,
    "max_age_days": 30
  },
  "heartbeat": [
    {
      "name": "Main Website",
      "url": "https://your-website.com",
      "timeout": 5
    },
    {
      "name": "API Health",
      "url": "https://api.your-website.com/health",
      "timeout": 3
    }
  ],
  "servers": [
    {
      "name": "Production Server 1",
      "address": "https://server1.your-domain.com:3500",
      "table_name": "server1_monitoring"
    },
    {
      "name": "Database Server",
      "address": "https://db.your-domain.com:3500",
      "table_name": "database_monitoring"
    }
  ]
}
```

## 🔐 Security Setup

### 1. Generate Strong Secrets

```bash
# Generate AES secret (32 characters)
openssl rand -base64 24

# Generate JWT secret (32 characters)
openssl rand -base64 24

# Update .env file with generated secrets
sudo nano /opt/monitoring/config/.env
```

### 2. Set Proper Permissions

```bash
# Secure configuration files
sudo chmod 600 /opt/monitoring/config/.env
sudo chmod 640 /opt/monitoring/config/configs.json
sudo chown monitoring:monitoring /opt/monitoring/config/*

# Create log directories
sudo mkdir -p /opt/monitoring/logs
sudo chown monitoring:monitoring /opt/monitoring/logs
sudo chmod 750 /opt/monitoring/logs
```

### 3. Firewall Configuration

```bash
# Ubuntu (UFW)
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw deny 3500/tcp  # Block direct access to app port
sudo ufw enable

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-service=ssh
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

## 🌐 Nginx Reverse Proxy Setup

### 1. Create Nginx Configuration

```bash
sudo nano /etc/nginx/sites-available/monitoring
```

```nginx
server {
    listen 80;
    server_name monitoring.your-domain.com;
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name monitoring.your-domain.com;

    # SSL Configuration (will be configured by Certbot)
    ssl_certificate /etc/letsencrypt/live/monitoring.your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitoring.your-domain.com/privkey.pem;
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;

    # Proxy settings
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Main application
    location / {
        proxy_pass http://127.0.0.1:3500;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # WebSocket support (if needed)
    location /ws {
        proxy_pass http://127.0.0.1:3500;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # Health check endpoint
    location /health {
        proxy_pass http://127.0.0.1:3500/health;
        access_log off;
    }
}
```

### 2. Enable Site and Get SSL Certificate

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/monitoring /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo certbot --nginx -d monitoring.your-domain.com

# Auto-renewal setup
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

## 🛠️ Systemd Service Setup

### 1. Create Service File

```bash
sudo nano /etc/systemd/system/monitoring.service
```

```ini
[Unit]
Description=Go System Monitoring Service
Documentation=https://github.com/your-repo/api.go-monitoring
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=monitoring
Group=monitoring
ExecStart=/opt/monitoring/bin/monitoring-server
WorkingDirectory=/opt/monitoring/config
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Environment
Environment=GIN_MODE=release
EnvironmentFile=/opt/monitoring/config/.env

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/monitoring/logs /tmp
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=monitoring

[Install]
WantedBy=multi-user.target
```

### 2. Enable and Start Service

```bash
# Reload systemd and enable service
sudo systemctl daemon-reload
sudo systemctl enable monitoring.service

# Start service
sudo systemctl start monitoring.service

# Check status
sudo systemctl status monitoring.service

# View logs
sudo journalctl -u monitoring.service -f
```

## 📊 Health Checks and Monitoring

### 1. Service Health Check Script

```bash
sudo nano /opt/monitoring/bin/health-check.sh
```

```bash
#!/bin/bash
# Health check script for monitoring service

SERVICE_URL="http://localhost:3500/health"
SERVICE_NAME="monitoring"

# Check if service is responding
if curl -f -s "$SERVICE_URL" > /dev/null; then
    echo "✅ $SERVICE_NAME is healthy"
    exit 0
else
    echo "❌ $SERVICE_NAME is unhealthy"
    
    # Restart service
    sudo systemctl restart monitoring.service
    sleep 5
    
    # Check again
    if curl -f -s "$SERVICE_URL" > /dev/null; then
        echo "✅ $SERVICE_NAME restarted successfully"
        exit 0
    else
        echo "❌ $SERVICE_NAME restart failed"
        exit 1
    fi
fi
```

```bash
sudo chmod +x /opt/monitoring/bin/health-check.sh

# Add to crontab for automated health checks
sudo crontab -e
# Add: */5 * * * * /opt/monitoring/bin/health-check.sh >> /opt/monitoring/logs/health-check.log 2>&1
```

### 2. Log Rotation

```bash
sudo nano /etc/logrotate.d/monitoring
```

```
/opt/monitoring/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 monitoring monitoring
    postrotate
        systemctl reload monitoring.service
    endscript
}
```

## 🔍 Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check service status and logs
sudo systemctl status monitoring.service
sudo journalctl -u monitoring.service -n 50

# Check configuration
sudo -u monitoring /opt/monitoring/bin/monitoring-server --help

# Test configuration manually
cd /opt/monitoring/config && sudo -u monitoring /opt/monitoring/bin/monitoring-server
```

#### Database Connection Issues
```bash
# Test PostgreSQL connection
sudo -u monitoring psql -U monitoring -d monitoring -h localhost

# Check PostgreSQL service
sudo systemctl status postgresql
sudo journalctl -u postgresql -n 20
```

#### Nginx/SSL Issues
```bash
# Test Nginx configuration
sudo nginx -t

# Check SSL certificate
sudo certbot certificates

# Renew SSL certificate
sudo certbot renew --dry-run
```

### Performance Monitoring

```bash
# Monitor application resources
sudo systemctl status monitoring.service
sudo ps aux | grep monitoring-server

# Monitor database performance
sudo -u postgres psql -c "
SELECT 
    datname,
    numbackends,
    xact_commit,
    xact_rollback,
    blks_read,
    blks_hit
FROM pg_stat_database 
WHERE datname = 'monitoring';
"

# Monitor disk space
df -h /opt/monitoring
du -sh /opt/monitoring/*
```

## 🔄 Updates and Maintenance

### 1. Application Updates

```bash
# Stop service
sudo systemctl stop monitoring.service

# Backup current version
sudo cp /opt/monitoring/bin/monitoring-server /opt/monitoring/bin/monitoring-server.backup

# Deploy new version
sudo cp /tmp/monitoring-server-new /opt/monitoring/bin/monitoring-server
sudo chown monitoring:monitoring /opt/monitoring/bin/monitoring-server
sudo chmod +x /opt/monitoring/bin/monitoring-server

# Start service
sudo systemctl start monitoring.service
sudo systemctl status monitoring.service
```

### 2. Database Maintenance

```bash
# Vacuum and analyze database
sudo -u postgres psql -d monitoring -c "VACUUM ANALYZE;"

# Check database size
sudo -u postgres psql -d monitoring -c "
SELECT 
    pg_size_pretty(pg_database_size('monitoring')) as database_size;
"
```

## 🔗 Next Steps

- **✅ Production Deployment Complete!**
- **🌐 [Nginx Setup Guide](nginx-setup.md)** - Advanced proxy configuration
- **🛠️ [Systemd Setup](monitoring-systemd-setup.md)** - Service management
- **🔧 [CLI Usage](cli-usage.md)** - Command line tools
- **🔍 [Troubleshooting](troubleshooting.md)** - Problem solving

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/your-repo/issues)
- **Security**: [Security Policy](../SECURITY.md)
- **Documentation**: [Main README](../README.md)

---
**🚀 Congratulations! Your monitoring service is now running in production with enterprise-grade security and performance.**