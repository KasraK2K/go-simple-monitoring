# Docker Deployment Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [🌐 Nginx Setup](nginx-setup.md)

Deploy the monitoring service using Docker containers for easy scaling and management.

## 🐳 Quick Docker Setup

### 1. Using Docker Compose (Recommended)

```bash
# Clone repository
git clone https://github.com/your-repo/api.go-monitoring.git
cd api.go-monitoring

# Start with PostgreSQL + TimescaleDB
docker-compose -f scripts/docker-compose.timescale.yml up -d

# Build and run monitoring service
docker-compose up -d
```

### 2. Manual Docker Commands

```bash
# Build application image
docker build -t monitoring-service .

# Run with external PostgreSQL
docker run -d \
  --name monitoring \
  -p 3500:3500 \
  -e POSTGRES_HOST=your-db-host \
  -e POSTGRES_USER=monitoring \
  -e POSTGRES_PASSWORD=your-password \
  -v /opt/monitoring/logs:/app/logs \
  monitoring-service
```

## 📋 Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o monitoring-server cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/monitoring-server .
COPY --from=builder /app/configs_example.json ./configs.json

EXPOSE 3500
CMD ["./monitoring-server"]
```

## 🌐 Production Docker Compose

```yaml
version: '3.8'

services:
  monitoring:
    build: .
    ports:
      - "3500:3500"
    environment:
      - POSTGRES_HOST=timescaledb
      - POSTGRES_USER=monitoring
      - POSTGRES_PASSWORD=secure_password
      - ENVIRONMENT=production
      - HISTORICAL_QUERY_STORAGE=postgresql
    volumes:
      - ./configs.json:/app/configs.json
      - monitoring_logs:/app/logs
    depends_on:
      - timescaledb
    restart: unless-stopped

  timescaledb:
    image: timescale/timescaledb:latest-pg16
    environment:
      - POSTGRES_USER=monitoring
      - POSTGRES_PASSWORD=secure_password
      - POSTGRES_DB=monitoring
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./scripts/timescale-init:/docker-entrypoint-initdb.d
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl/certs
    depends_on:
      - monitoring
    restart: unless-stopped

volumes:
  postgres_data:
  monitoring_logs:
```

## 🔗 Related Documentation

- [🗄️ PostgreSQL Setup](postgresql-setup.md) - Database configuration
- [🚀 Production Deployment](production-deployment.md) - Traditional deployment
- [🏠 Main README](../README.md) - Getting started