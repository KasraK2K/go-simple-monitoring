# Cloud Deployment Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🗄️ PostgreSQL Setup](postgresql-setup.md) | [🐳 Docker Setup](docker-deployment.md)

Deploy the monitoring service to major cloud providers with automated scaling and managed databases.

## ☁️ Deployment Options

| **Provider** | **Compute** | **Database** | **Complexity** | **Cost** |
|--------------|-------------|--------------|----------------|----------|
| **AWS** | EC2, ECS, Lambda | RDS PostgreSQL | ⭐⭐⭐ | $$$ |
| **Google Cloud** | Compute Engine, Cloud Run | Cloud SQL | ⭐⭐⭐ | $$$ |
| **Azure** | Virtual Machines, Container Instances | PostgreSQL | ⭐⭐⭐ | $$$ |
| **DigitalOcean** | Droplets, App Platform | Managed PostgreSQL | ⭐⭐ | $$ |
| **Railway** | Container Hosting | Built-in PostgreSQL | ⭐ | $ |

## 🚀 AWS Deployment

### 1. EC2 + RDS Setup

```bash
# Launch EC2 instance
aws ec2 run-instances \
  --image-id ami-0c02fb55956c7d316 \
  --instance-type t3.micro \
  --key-name your-key-pair \
  --security-group-ids sg-xxxxxxxx

# Create RDS PostgreSQL instance
aws rds create-db-instance \
  --db-instance-identifier monitoring-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 14.9 \
  --master-username monitoring \
  --master-user-password YourSecurePassword \
  --allocated-storage 20
```

### 2. ECS Fargate Deployment

```yaml
# docker-compose.aws.yml
version: '3.8'
services:
  monitoring:
    image: your-account.dkr.ecr.region.amazonaws.com/monitoring:latest
    ports:
      - "3500:3500"
    environment:
      POSTGRES_HOST: monitoring-db.xxxxx.region.rds.amazonaws.com
      POSTGRES_USER: monitoring
      POSTGRES_PASSWORD: !Ref DatabasePassword
    logging:
      driver: awslogs
      options:
        awslogs-group: /ecs/monitoring
        awslogs-region: us-east-1
```

## 🟦 Google Cloud Deployment

### 1. Cloud Run + Cloud SQL

```bash
# Create Cloud SQL instance
gcloud sql instances create monitoring-postgres \
  --database-version=POSTGRES_14 \
  --tier=db-f1-micro \
  --region=us-central1

# Deploy to Cloud Run
gcloud run deploy monitoring \
  --image gcr.io/your-project/monitoring:latest \
  --platform managed \
  --region us-central1 \
  --set-env-vars POSTGRES_HOST=10.x.x.x
```

### 2. Kubernetes Engine

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: monitoring-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: monitoring
  template:
    metadata:
      labels:
        app: monitoring
    spec:
      containers:
      - name: monitoring
        image: gcr.io/your-project/monitoring:latest
        ports:
        - containerPort: 3500
        env:
        - name: POSTGRES_HOST
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: host
---
apiVersion: v1
kind: Service
metadata:
  name: monitoring-service
spec:
  selector:
    app: monitoring
  ports:
  - port: 80
    targetPort: 3500
  type: LoadBalancer
```

## 🔷 Azure Deployment

### 1. Container Instances + PostgreSQL

```bash
# Create resource group
az group create --name monitoring-rg --location eastus

# Create PostgreSQL server
az postgres server create \
  --resource-group monitoring-rg \
  --name monitoring-postgres \
  --location eastus \
  --admin-user monitoring \
  --admin-password YourSecurePassword \
  --sku-name B_Gen5_1

# Deploy container
az container create \
  --resource-group monitoring-rg \
  --name monitoring-app \
  --image your-registry.azurecr.io/monitoring:latest \
  --dns-name-label monitoring-app \
  --ports 3500
```

## 🌊 DigitalOcean Deployment

### 1. App Platform (Easiest)

```yaml
# .do/app.yaml
name: monitoring-service
services:
- name: monitoring
  source_dir: /
  github:
    repo: your-username/api.go-monitoring
    branch: main
  run_command: ./monitoring-server
  environment_slug: go
  instance_count: 1
  instance_size_slug: basic-xxs
  envs:
  - key: POSTGRES_HOST
    value: ${monitoring-db.HOSTNAME}
  - key: POSTGRES_USER
    value: ${monitoring-db.USERNAME}
  - key: POSTGRES_PASSWORD
    value: ${monitoring-db.PASSWORD}
  routes:
  - path: /
databases:
- name: monitoring-db
  engine: PG
  size: db-s-1vcpu-1gb
```

### 2. Droplet + Managed Database

```bash
# Create droplet
doctl compute droplet create monitoring-server \
  --size s-1vcpu-1gb \
  --image ubuntu-20-04-x64 \
  --region nyc3 \
  --ssh-keys your-ssh-key-id

# Create managed database
doctl databases create monitoring-postgres \
  --engine postgres \
  --size db-s-1vcpu-1gb \
  --region nyc3
```

## 🚂 Railway Deployment (Fastest)

### 1. One-Click Deploy

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login and deploy
railway login
railway init
railway add --database postgresql
railway up
```

### 2. railway.json Configuration

```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "NIXPACKS"
  },
  "deploy": {
    "numReplicas": 1,
    "sleepApplication": false,
    "restartPolicyType": "ON_FAILURE"
  }
}
```

## 🔧 Environment Configuration for Cloud

### Production .env Template

```bash
# Cloud-specific settings
ENVIRONMENT=production
PORT=3500

# Database (use cloud database connection string)
POSTGRES_HOST=${{ secrets.DB_HOST }}
POSTGRES_USER=${{ secrets.DB_USER }}
POSTGRES_PASSWORD=${{ secrets.DB_PASSWORD }}
POSTGRES_DB=monitoring
POSTGRES_PORT=5432

# Security
AES_SECRET=${{ secrets.AES_SECRET }}
JWT_SECRET=${{ secrets.JWT_SECRET }}
CHECK_TOKEN=true

# Performance
MONITORING_DOWNSAMPLE_MAX_POINTS=200
RATE_LIMIT_ENABLED=true

# Logging
LOG_LEVEL=INFO
```

### Cloud-Specific Environment Variables

#### AWS
```bash
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=${{ secrets.AWS_ACCESS_KEY }}
AWS_SECRET_ACCESS_KEY=${{ secrets.AWS_SECRET_KEY }}
```

#### Google Cloud
```bash
GOOGLE_APPLICATION_CREDENTIALS=/app/service-account.json
GOOGLE_CLOUD_PROJECT=your-project-id
```

#### Azure
```bash
AZURE_CLIENT_ID=${{ secrets.AZURE_CLIENT_ID }}
AZURE_CLIENT_SECRET=${{ secrets.AZURE_CLIENT_SECRET }}
AZURE_TENANT_ID=${{ secrets.AZURE_TENANT_ID }}
```

## 📊 Monitoring and Observability

### Health Check Endpoints

```bash
# Application health
curl https://your-app.com/health

# Database connectivity
curl https://your-app.com/health/db

# Metrics endpoint
curl https://your-app.com/metrics
```

### Cloud Monitoring Integration

#### AWS CloudWatch
```go
import "github.com/aws/aws-sdk-go/service/cloudwatch"

// Custom metrics
cloudwatch.PutMetricData(&cloudwatch.PutMetricDataInput{
    Namespace: aws.String("Monitoring/Application"),
    MetricData: []*cloudwatch.MetricDatum{
        {
            MetricName: aws.String("ActiveConnections"),
            Value:      aws.Float64(float64(activeConns)),
        },
    },
})
```

#### Google Cloud Monitoring
```yaml
# monitoring.yaml
resources:
- name: monitoring-alert
  type: monitoring.v1.alertPolicy
  properties:
    displayName: "High CPU Usage"
    conditions:
    - displayName: "CPU > 80%"
      conditionThreshold:
        filter: 'resource.type="gce_instance"'
        comparison: COMPARISON_GREATER
        thresholdValue: 0.8
```

## 🔗 Related Documentation

- [🚀 Production Deployment](production-deployment.md) - Traditional server deployment
- [🐳 Docker Setup](docker-deployment.md) - Container deployment
- [🗄️ PostgreSQL Setup](postgresql-setup.md) - Database configuration
- [🏠 Main README](../README.md) - Getting started

---
**☁️ Choose the cloud provider that best fits your needs: Railway for simplicity, DigitalOcean for balance, or AWS/GCP/Azure for enterprise features.**