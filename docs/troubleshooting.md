# Troubleshooting Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [📊 Dashboard Guide](dashboard-guide.md)

Common issues, solutions, and debugging techniques for the monitoring service.

## 🔍 Quick Diagnostics

### Health Check Commands
```bash
# Check service status
curl http://localhost:3500/health

# Check database connectivity  
curl http://localhost:3500/health/db

# View service logs
sudo journalctl -u monitoring.service -f

# Check database connection manually
psql -U monitoring -d monitoring -h localhost
```

## 🚫 Common Issues

### 1. Service Won't Start

#### **Symptoms**
- Service fails to start
- "connection refused" errors
- Port already in use

#### **Diagnosis**
```bash
# Check if service is running
sudo systemctl status monitoring.service

# Check port availability
sudo netstat -tlnp | grep 3500

# Check configuration
sudo -u monitoring /opt/monitoring/bin/monitoring-server --config-check
```

#### **Solutions**

**Port Conflict**
```bash
# Find process using port
sudo lsof -i :3500

# Kill conflicting process
sudo kill -9 PID

# Or change port in .env
PORT=3501
```

**Permission Issues**
```bash
# Fix file permissions
sudo chown -R monitoring:monitoring /opt/monitoring
sudo chmod +x /opt/monitoring/bin/monitoring-server
sudo chmod 600 /opt/monitoring/config/.env
```

**Configuration Errors**
```bash
# Validate .env file
cat /opt/monitoring/config/.env | grep -E '(POSTGRES|PORT|AES|JWT)'

# Check for missing values
if [ -z "$POSTGRES_PASSWORD" ]; then echo "Missing POSTGRES_PASSWORD"; fi
```

### 2. Database Connection Issues

#### **Symptoms**
- "failed to connect to database" 
- PostgreSQL connection timeout
- TimescaleDB extension errors

#### **Diagnosis**
```bash
# Test PostgreSQL connection
sudo -u monitoring psql -U monitoring -d monitoring -h localhost -c '\dt'

# Check PostgreSQL service
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-*.log

# Verify TimescaleDB extension
sudo -u postgres psql -d monitoring -c "SELECT * FROM pg_extension WHERE extname = 'timescaledb';"
```

#### **Solutions**

**PostgreSQL Not Running**
```bash
# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Check if listening on correct port
sudo ss -tlnp | grep 5432
```

**Authentication Failed**
```bash
# Reset password
sudo -u postgres psql
ALTER USER monitoring WITH PASSWORD 'new_password';
\q

# Update .env file
POSTGRES_PASSWORD=new_password
```

**Connection Refused**
```bash
# Edit postgresql.conf
sudo nano /etc/postgresql/14/main/postgresql.conf
# Uncomment and set: listen_addresses = 'localhost'

# Edit pg_hba.conf  
sudo nano /etc/postgresql/14/main/pg_hba.conf
# Add: local   monitoring   monitoring   md5

# Restart PostgreSQL
sudo systemctl restart postgresql
```

**TimescaleDB Issues**
```bash
# Install TimescaleDB if missing
sudo apt install timescaledb-2-postgresql-14

# Add to postgresql.conf
echo "shared_preload_libraries = 'timescaledb'" | sudo tee -a /etc/postgresql/14/main/postgresql.conf

# Restart and create extension
sudo systemctl restart postgresql
sudo -u postgres psql -d monitoring -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"
```

### 3. Dashboard Access Issues

#### **Symptoms**
- Dashboard not loading
- 502 Bad Gateway (Nginx)
- Authentication failures

#### **Diagnosis**
```bash
# Check if dashboard is enabled
grep HAS_DASHBOARD /opt/monitoring/config/.env

# Test direct access
curl http://localhost:3500

# Check Nginx status
sudo systemctl status nginx

# Check Nginx logs
sudo tail -f /var/log/nginx/error.log
```

#### **Solutions**

**Dashboard Disabled**
```bash
# Enable dashboard in .env
HAS_DASHBOARD=true

# Restart service
sudo systemctl restart monitoring.service
```

**Nginx Configuration Issues**
```bash
# Test Nginx config
sudo nginx -t

# Check proxy settings
sudo nano /etc/nginx/sites-available/monitoring

# Ensure proxy_pass points to correct port
proxy_pass http://127.0.0.1:3500;

# Reload Nginx
sudo systemctl reload nginx
```

**SSL Certificate Problems**
```bash
# Check certificate status
sudo certbot certificates

# Renew certificate
sudo certbot renew

# Test SSL configuration
openssl s_client -connect monitoring.your-domain.com:443
```

### 4. Performance Issues

#### **Symptoms**
- Slow dashboard loading
- High memory usage
- Database query timeouts

#### **Diagnosis**
```bash
# Check system resources
top -p $(pgrep monitoring-server)

# Monitor database performance
sudo -u postgres psql -d monitoring -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE query LIKE '%monitoring%' 
ORDER BY mean_exec_time DESC LIMIT 10;
"

# Check table sizes
sudo -u postgres psql -d monitoring -c "
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables WHERE schemaname = 'public';
"
```

#### **Solutions**

**High Memory Usage**
```bash
# Reduce downsampling points
MONITORING_DOWNSAMPLE_MAX_POINTS=100

# Enable log rotation
echo '*/5 * * * * find /opt/monitoring/logs -name "*.log" -mtime +7 -delete' | sudo crontab -

# Optimize PostgreSQL
sudo nano /etc/postgresql/14/main/postgresql.conf
# Set: shared_buffers = 256MB
# Set: effective_cache_size = 1GB
```

**Slow Queries**
```bash
# Check if tables are hypertables
sudo -u postgres psql -d monitoring -c "
SELECT hypertable_name, num_chunks 
FROM timescaledb_information.hypertables;
"

# Create missing indexes
sudo -u postgres psql -d monitoring -c "
CREATE INDEX IF NOT EXISTS idx_monitoring_timestamp 
ON monitoring_table (timestamp DESC);
"

# Vacuum analyze
sudo -u postgres psql -d monitoring -c "VACUUM ANALYZE;"
```

**Database Disk Space**
```bash
# Clean old data
sudo -u postgres psql -d monitoring -c "
DELETE FROM monitoring_table 
WHERE timestamp < NOW() - INTERVAL '30 days';
"

# Set up automatic cleanup
SELECT add_retention_policy('monitoring_table', INTERVAL '30 days');
```

## 🔧 Advanced Debugging

### 1. Enable Debug Logging

```bash
# Update .env
LOG_LEVEL=DEBUG

# Restart service
sudo systemctl restart monitoring.service

# View debug logs
sudo journalctl -u monitoring.service -f | grep DEBUG
```

### 2. Database Debugging

```sql
-- Check connection status
SELECT state, count(*) 
FROM pg_stat_activity 
WHERE datname = 'monitoring' 
GROUP BY state;

-- Monitor query performance
SELECT query, calls, total_time, mean_time 
FROM pg_stat_statements 
ORDER BY mean_time DESC LIMIT 10;

-- Check locks
SELECT * FROM pg_locks 
WHERE NOT GRANTED;

-- Check hypertable status
SELECT hypertable_name, num_chunks, compression_enabled
FROM timescaledb_information.hypertables;
```

### 3. Network Debugging

```bash
# Check listening ports
sudo ss -tlnp | grep monitoring

# Test connectivity
telnet localhost 3500

# Check firewall
sudo ufw status
sudo iptables -L

# Monitor network traffic
sudo tcpdump -i lo port 3500
```

### 4. Application Profiling

```bash
# Memory profiling
go tool pprof http://localhost:3500/debug/pprof/heap

# CPU profiling  
go tool pprof http://localhost:3500/debug/pprof/profile

# Goroutine analysis
go tool pprof http://localhost:3500/debug/pprof/goroutine
```

## 🛠️ Recovery Procedures

### 1. Database Recovery

```bash
# Backup current data
sudo -u postgres pg_dump monitoring > /tmp/monitoring_backup.sql

# Reset database
sudo -u postgres dropdb monitoring
sudo -u postgres createdb monitoring
sudo -u postgres psql -d monitoring -c "CREATE EXTENSION timescaledb;"

# Restore data (if needed)
sudo -u postgres psql monitoring < /tmp/monitoring_backup.sql
```

### 2. Service Recovery

```bash
# Reset service state
sudo systemctl stop monitoring.service
sudo systemctl reset-failed monitoring.service
sudo systemctl start monitoring.service

# Clear cache files
sudo rm -rf /opt/monitoring/tmp/*
sudo rm -rf /opt/monitoring/logs/*.lock
```

### 3. Configuration Reset

```bash
# Backup current config
sudo cp /opt/monitoring/config/.env /opt/monitoring/config/.env.backup

# Reset to defaults
sudo cp .env.example /opt/monitoring/config/.env
sudo nano /opt/monitoring/config/.env  # Update with your values

# Restart service
sudo systemctl restart monitoring.service
```

## 📊 Monitoring the Monitor

### 1. External Health Checks

```bash
# Create external monitor script
cat > /usr/local/bin/monitor-health.sh << 'EOF'
#!/bin/bash
URL="http://localhost:3500/health"
if ! curl -f -s "$URL" > /dev/null; then
    echo "$(date): Monitoring service unhealthy" | logger -t monitor-health
    systemctl restart monitoring.service
fi
EOF

chmod +x /usr/local/bin/monitor-health.sh

# Add to cron
echo "*/5 * * * * /usr/local/bin/monitor-health.sh" | sudo crontab -
```

### 2. Resource Monitoring

```bash
# Monitor service resources
watch 'systemctl status monitoring.service; echo ""; ps aux | grep monitoring-server | grep -v grep'

# Database monitoring
watch 'sudo -u postgres psql -d monitoring -c "SELECT count(*) FROM monitoring_table;"'
```

## 🔗 Getting Help

### 1. Information to Collect

Before reporting issues, collect:

```bash
# System information
uname -a
cat /etc/os-release

# Service status
sudo systemctl status monitoring.service

# Recent logs
sudo journalctl -u monitoring.service --since="1 hour ago"

# Configuration (remove sensitive data)
cat /opt/monitoring/config/.env | sed 's/PASSWORD=.*/PASSWORD=***REDACTED***/'

# Database status
sudo -u postgres psql -d monitoring -c "\dt"
```

### 2. Support Channels

- **🐛 GitHub Issues**: [Create Issue](https://github.com/your-repo/issues/new)
- **📖 Documentation**: [Main README](../README.md)
- **🚀 Deployment**: [Production Guide](production-deployment.md)
- **🗄️ Database**: [PostgreSQL Setup](postgresql-setup.md)

### 3. Emergency Contacts

For production issues:
1. Check this troubleshooting guide first
2. Review service logs: `sudo journalctl -u monitoring.service -f`
3. Test basic connectivity: `curl http://localhost:3500/health`
4. Report with collected information above

---
**🔍 Most issues are resolved by checking logs, verifying configuration, and ensuring services are running. When in doubt, restart services and check permissions.**