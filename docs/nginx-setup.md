# Nginx Authentication Setup for API Monitoring Dashboard

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [🔧 CLI Usage](cli-usage.md)

This guide explains how to configure nginx with basic authentication for the dashboard (`/` path) while keeping other API endpoints open.

## Overview

The nginx configuration protects only the dashboard (root path `/`) with basic authentication, while all other paths remain publicly accessible for API endpoints.

## Setup Steps

### 1. Install htpasswd utility and create htpasswd file

First, install the `htpasswd` utility:

```bash
# On Ubuntu/Debian:
sudo apt-get install apache2-utils

# On CentOS/RHEL/Fedora:
sudo yum install httpd-tools
# or on newer versions:
sudo dnf install httpd-tools

# On macOS:
brew install httpd
```

Then create the htpasswd file with your credentials:

```bash
sudo htpasswd -c /etc/nginx/htpasswd-api-monitoring <username>
```

Or create it manually with this sample content:

```
# Sample htpasswd file for dashboard basic auth.

# Replace the hash with one generated via `htpasswd` or `openssl passwd -apr1`.

dashboard:$apr1$s7.F32Y2$zMSpxtQrqXVyL4x//OtOl/
```

The sample credentials are:

- **Username**: `dashboard`
- **Password**: `SamplePass123`

To generate your own password hash:

```bash
openssl passwd -apr1 '<your-password>'
```

### 2. Create nginx configuration

Create the nginx configuration file at `/etc/nginx/conf.d/<YOUR_DOMAIN>-api-monitoring.conf`:

```nginx
# Nginx config for <YOUR_DOMAIN>
# Place this file in /etc/nginx/conf.d/ (or sites-enabled) and reload nginx.
# Create the htpasswd file referenced below with `htpasswd -c /etc/nginx/htpasswd-api-monitoring <user>`.

upstream api_monitoring_app {
    server 127.0.0.1:3500;
    keepalive 16;
}

server {
    listen 80;
    listen [::]:80;
    server_name api-monitoring.techsuite.ai;

    # --- Gzip compression ---
    # It's best to place these in the http {} block globally,
    # but including them here in server {} also works.
    gzip on;
    gzip_comp_level 5;              # balance CPU and size
    gzip_min_length 1024;           # only compress responses >= 1KB
    gzip_vary on;                   # add Vary: Accept-Encoding
    gzip_proxied any;               # allow compression for proxied responses
    gzip_types
        text/plain
        text/css
        text/xml
        application/json
        application/javascript
        application/x-javascript
        application/xml
        application/xml+rss
        image/svg+xml;

    # Increase default buffer sizes for larger responses
    client_max_body_size 20m;

    # Handle large responses (log exports with date ranges)
    proxy_buffering on;
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;
    proxy_max_temp_file_size 1024m;

    # Exact match for dashboard entry point; everything else stays open.
    location = / {
        auth_basic "TechSuite Monitoring";
        auth_basic_user_file /etc/nginx/htpasswd-api-monitoring;

        proxy_pass http://api_monitoring_app;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # Let nginx handle compression to clients (avoid upstream compression)
        proxy_set_header Accept-Encoding "";
        proxy_set_header Connection "";
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }

    # Unauthenticated access for APIs, static assets, etc.
    location / {
        auth_basic off;

        proxy_pass http://api_monitoring_app;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # Let nginx handle compression to clients (avoid upstream compression)
        proxy_set_header Accept-Encoding "";
        proxy_set_header Connection "";
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

**Important**: Replace `<YOUR_DOMAIN>` with your actual domain name.

### 3. Configuration Details

The nginx config includes:

- **Protected path**: `location = /` (exact match for dashboard)

  - Requires basic authentication
  - Uses htpasswd file at `/etc/nginx/htpasswd-api-monitoring`

- **Public paths**: `location /` (all other paths)

  - No authentication required
  - API endpoints remain accessible

- **Upstream configuration**:

  - Proxies to local application on `127.0.0.1:3500`
  - Includes proper headers and timeouts

- **Gzip compression**:
  - Enabled at the `server {}` level via `gzip on;`
  - `gzip_types` includes common text and JSON/JS content
  - `gzip_vary on;` adds `Vary: Accept-Encoding` for proper caching
  - `gzip_proxied any;` compresses proxied responses
  - `proxy_set_header Accept-Encoding "";` ensures the app returns uncompressed responses so nginx can compress at the edge

### 4. Apply configuration

Test and reload nginx:

```bash
# Test configuration syntax
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
# OR
sudo nginx -s reload
```

### 5. Verify setup

1. Visit `https://<YOUR_DOMAIN>/` - should prompt for username/password
2. Test API endpoints like `https://<YOUR_DOMAIN>/api/...` - should work without authentication
3. Use the credentials from your htpasswd file to access the dashboard

### Verify Gzip

Use curl to confirm gzip is active:

```bash
curl -I -H 'Accept-Encoding: gzip' https://<YOUR_DOMAIN>/api/v1/server-config
# Look for:  Content-Encoding: gzip
# And:       Vary: Accept-Encoding
```

## Security Notes

- Change the default password before production use
- Consider using SSL/TLS certificates for HTTPS
- The htpasswd file should have restricted permissions (readable only by nginx user)

```bash
sudo chown root:nginx /etc/nginx/htpasswd-api-monitoring
sudo chmod 640 /etc/nginx/htpasswd-api-monitoring
```

## SSL/HTTPS Configuration (Recommended)

For production use, add SSL configuration to your nginx config:

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name <YOUR_DOMAIN>;

    ssl_certificate /path/to/your/certificate.crt;
    ssl_certificate_key /path/to/your/private.key;

    # ... rest of configuration same as above
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name <YOUR_DOMAIN>;
    return 301 https://$server_name$request_uri;
}
```
