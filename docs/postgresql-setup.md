# PostgreSQL & TimescaleDB Setup Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🌐 Nginx Setup](nginx-setup.md) | [🔧 CLI Usage](cli-usage.md)

This guide covers setting up PostgreSQL with optional TimescaleDB extension for optimal time-series performance.

## 🎯 Quick Setup Options

| **Method** | **Best For** | **Setup Time** | **Performance** |
|------------|--------------|----------------|-----------------|
| [Docker Compose](#docker-compose-setup) | Development, Testing | 5 minutes | ⭐⭐⭐⭐ |
| [Native Installation](#native-installation) | Production | 15 minutes | ⭐⭐⭐⭐⭐ |
| [Cloud Services](#cloud-deployment) | Scalable Production | 10 minutes | ⭐⭐⭐⭐⭐ |

## 🐳 Docker Compose Setup (Recommended for Development)

### 1. Quick Start with Docker

```bash
# Clone and navigate to project
cd api.go-monitoring

# Start TimescaleDB with Docker Compose
./scripts/postgres-timescale-up.sh

# Or manually:
docker-compose -f scripts/docker-compose.timescale.yml up -d
```

### 2. Configuration

Create your `.env` file:
```bash
cp .env.example .env
```

Update PostgreSQL settings in `.env`:
```bash
# PostgreSQL Configuration
POSTGRES_USER=monitoring
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=monitoring

# Historical Query Storage Selection
# Which database to use for historical queries with date ranges
# Valid values: "sqlite", "postgresql" (default: postgresql)
HISTORICAL_QUERY_STORAGE=postgresql

# Enable PostgreSQL storage
# Edit configs.json: "storage": ["postgresql"]
```

### 3. Verify Setup

```bash
# Check if TimescaleDB is running
docker ps | grep timescaledb

# Connect to database
docker exec -it monitoring-timescaledb psql -U monitoring -d monitoring

# Verify TimescaleDB extension
monitoring=# \dx
```

## 🖥️ Native Installation

### Ubuntu/Debian

```bash
# Add PostgreSQL repository
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" | sudo tee /etc/apt/sources.list.d/pgdg.list

# Add TimescaleDB repository
echo "deb https://packagecloud.io/timescale/timescaledb/ubuntu/ $(lsb_release -c -s) main" | sudo tee /etc/apt/sources.list.d/timescaledb.list
wget --quiet -O - https://packagecloud.io/timescale/timescaledb/gpgkey | sudo apt-key add -

# Install PostgreSQL and TimescaleDB
sudo apt update
sudo apt install postgresql-14 timescaledb-2-postgresql-14

# Configure TimescaleDB
sudo timescaledb-tune

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### CentOS/RHEL/Rocky Linux

```bash
# Install PostgreSQL
sudo dnf install postgresql-server postgresql-contrib

# Add TimescaleDB repository
echo '[timescaledb]
name=TimescaleDB
baseurl=https://packagecloud.io/timescale/timescaledb/el/8/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packagecloud.io/timescale/timescaledb/gpgkey
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300' | sudo tee /etc/yum.repos.d/timescaledb.repo

# Install TimescaleDB
sudo dnf install timescaledb-2-postgresql-14

# Initialize and start PostgreSQL
sudo postgresql-setup --initdb
sudo systemctl enable postgresql --now

# Configure TimescaleDB
sudo timescaledb-tune
```

### macOS (Homebrew)

```bash
# Install PostgreSQL
brew install postgresql

# Install TimescaleDB
brew tap timescale/tap
brew install timescaledb

# Start PostgreSQL service
brew services start postgresql

# Configure TimescaleDB
timescaledb-tune
```

## 🔧 Database Configuration

### 1. Create Database and User

```sql
-- Connect as postgres user
sudo -u postgres psql

-- Create database and user
CREATE DATABASE monitoring;
CREATE USER monitoring WITH ENCRYPTED PASSWORD 'your_secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE monitoring TO monitoring;
\q
```

### 2. Enable TimescaleDB Extension

```sql
-- Connect to your database
psql -U monitoring -d monitoring -h localhost

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Verify installation
SELECT * FROM timescaledb_information.license;
\q
```

### 3. Initialize Tables (Automatic)

The application automatically creates and configures tables when it starts:

```bash
# Your application will create:
# - Regular PostgreSQL tables with proper indexes
# - TimescaleDB hypertables when extension is available
# - Optimal configurations for time-series data

# Start your monitoring service
./monitoring-server
```

## 🔄 Historical Query Storage Configuration

### Advanced Storage Selection

The monitoring service supports independent database selection for historical queries (queries with date ranges). This allows you to optimize performance by using different databases for current data versus historical analysis.

#### Environment Variable Configuration

```bash
# Historical Query Storage Selection
# Valid values: "sqlite", "postgresql" (default: postgresql)
HISTORICAL_QUERY_STORAGE=postgresql
```

#### Use Cases

**Use PostgreSQL for Historical Queries** (Recommended)
```bash
HISTORICAL_QUERY_STORAGE=postgresql
```
- ✅ **Best for**: Large datasets, complex queries, production environments
- ✅ **Performance**: Optimized for time-series data with TimescaleDB
- ✅ **Scalability**: Handles millions of records efficiently
- ✅ **Advanced Features**: Compression, retention policies, continuous aggregates

**Use SQLite for Historical Queries**
```bash
HISTORICAL_QUERY_STORAGE=sqlite
```
- ✅ **Best for**: Small to medium datasets, development environments
- ✅ **Simplicity**: Single file database, easy backup and migration
- ✅ **Resource Usage**: Lower memory footprint for smaller datasets
- ⚠️ **Limitation**: Less efficient for very large historical datasets

#### Important Notes

1. **Storage Configuration Required**: The selected storage type must be included in your `configs.json` storage array:
   ```json
   {
     "storage": ["postgresql", "sqlite"]
   }
   ```

2. **Automatic Fallback**: If the preferred storage isn't available, the system automatically falls back to the original storage selection logic.

3. **Current vs Historical**: Non-historical queries (current data) continue using the original preference logic (SQLite preferred).

## ⚡ Performance Optimization

### 1. PostgreSQL Configuration

Edit `postgresql.conf`:
```ini
# Memory settings
shared_buffers = 256MB                    # 25% of RAM for dedicated server
effective_cache_size = 1GB                # 75% of available RAM
work_mem = 4MB                            # Per operation memory

# TimescaleDB settings
max_connections = 100
shared_preload_libraries = 'timescaledb'

# WAL settings for better write performance
wal_buffers = 16MB
checkpoint_timeout = 10min
```

### 2. TimescaleDB Policies (Optional)

```sql
-- Connect to your database
psql -U monitoring -d monitoring

-- Set up compression for older data (saves 90% storage)
SELECT add_compression_policy('your_table_name', INTERVAL '7 days');

-- Set up data retention (automatically delete old data)
SELECT add_retention_policy('your_table_name', INTERVAL '1 year');

-- Create continuous aggregates for faster queries
CREATE MATERIALIZED VIEW hourly_stats
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 hour', timestamp) AS bucket,
  AVG((data->>'cpu_usage_percent')::numeric) as avg_cpu,
  MAX((data->>'ram_used_percent')::numeric) as max_ram
FROM your_table_name
GROUP BY bucket;
```

## 🌐 Cloud Deployment Options

### AWS RDS with TimescaleDB

```bash
# AWS CLI setup
aws rds create-db-instance \
    --db-instance-identifier monitoring-timescaledb \
    --db-instance-class db.t3.micro \
    --engine postgres \
    --master-username monitoring \
    --master-user-password YourSecurePassword \
    --allocated-storage 20

# Update your .env with RDS endpoint
POSTGRES_HOST=your-rds-endpoint.amazonaws.com
POSTGRES_PORT=5432
```

### Google Cloud SQL

```bash
# Create instance
gcloud sql instances create monitoring-postgres \
    --database-version=POSTGRES_14 \
    --tier=db-f1-micro \
    --region=us-central1

# Create database and user
gcloud sql databases create monitoring --instance=monitoring-postgres
gcloud sql users create monitoring --instance=monitoring-postgres --password=YourSecurePassword
```

### DigitalOcean Managed Databases

```bash
# Use DigitalOcean control panel or API
# Select PostgreSQL 14+ with TimescaleDB
# Update connection details in .env
```

## 🔍 Troubleshooting

### Common Issues

#### "TimescaleDB extension not found"
```bash
# Verify TimescaleDB installation
psql -U postgres -c "SELECT * FROM pg_available_extensions WHERE name = 'timescaledb';"

# If not found, reinstall TimescaleDB
```

#### "Connection refused"
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check if PostgreSQL is listening
sudo netstat -tlnp | grep 5432

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-*.log
```

#### "Permission denied"
```bash
# Ensure user has proper permissions
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE monitoring TO monitoring;"
sudo -u postgres psql -c "GRANT ALL ON SCHEMA public TO monitoring;"
```

#### "Table not empty" errors
```bash
# These are expected for existing tables with data
# The application handles this automatically
# See scripts/migrate-to-hypertables.sql for manual migration
```

### Performance Issues

```sql
-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public';

-- Check if tables are hypertables
SELECT 
    hypertable_name,
    num_chunks,
    compression_enabled
FROM timescaledb_information.hypertables;

-- Monitor query performance
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE query LIKE '%monitoring%' 
ORDER BY mean_exec_time DESC;
```

## 🔗 Next Steps

- **✅ PostgreSQL Setup Complete!**
- **🚀 [Production Deployment Guide](production-deployment.md)** - Deploy to your server
- **🌐 [Nginx Setup](nginx-setup.md)** - Configure reverse proxy
- **🛠️ [Systemd Setup](monitoring-systemd-setup.md)** - Run as a service
- **🔧 [CLI Usage](cli-usage.md)** - Command line tools

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/your-repo/issues)
- **Performance**: Check [troubleshooting guide](troubleshooting.md)
- **Documentation**: [Main README](../README.md)

---
**⚡ Pro Tip**: Use TimescaleDB for production deployments - it provides 10-100x better performance for time-series data!