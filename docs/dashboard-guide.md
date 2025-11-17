# Dashboard User Guide

**📚 Navigation:** [🏠 Main README](../README.md) | [🚀 Production Deployment](production-deployment.md) | [🔧 CLI Usage](cli-usage.md) | [🌐 Nginx Setup](nginx-setup.md)

Complete guide to using the web dashboard for monitoring your systems and servers.

## 🎯 Dashboard Overview

The monitoring dashboard provides real-time and historical views of:
- **System Metrics**: CPU, memory, disk, network usage
- **Server Monitoring**: Multiple remote servers
- **Heartbeat Checks**: External service availability
- **Time-Series Charts**: Historical performance data

## 📊 Main Interface

### 1. Navigation Bar
- **Server Selector**: Switch between local and remote servers
- **Time Range Picker**: Select data timeframes (1h, 6h, 24h, 7d, 30d)
- **Refresh Controls**: Manual refresh and auto-refresh settings

### 2. Metrics Overview Cards
```
┌─────────────┬─────────────┬─────────────┬─────────────┐
│  CPU Usage  │ Memory Used │  Disk Used  │ Network I/O │
│    45.2%    │    68.1%    │    82.3%    │  ↑15 ↓8 MB │
└─────────────┴─────────────┴─────────────┴─────────────┘
```

### 3. Interactive Charts
- **System Performance**: CPU and memory over time
- **Network Activity**: Upload/download trends
- **Usage Distribution**: Resource allocation pie charts

## 🔧 Configuration Options

### 1. Time Range Selection
- **1h**: Real-time monitoring (5-second intervals)
- **6h**: Recent trends (30-second intervals)  
- **24h**: Daily patterns (2-minute intervals)
- **7d**: Weekly trends (30-minute intervals)
- **30d**: Monthly overview (2-hour intervals)

### 2. Data Downsampling
- Automatically adjusts detail level based on time range
- **TimescaleDB**: Optimal time-bucket aggregation
- **PostgreSQL**: Count-based sampling
- **Configurable**: `MONITORING_DOWNSAMPLE_MAX_POINTS=200`

### 3. Historical Query Storage
The monitoring service supports independent database selection for historical data queries:

- **Current Data Queries**: Real-time monitoring uses default storage preference
- **Historical Data Queries**: Time range selections use dedicated storage backend
- **Environment Variable**: `HISTORICAL_QUERY_STORAGE=postgresql|sqlite`
- **Performance Optimization**: Use PostgreSQL for large historical datasets, SQLite for smaller data
- **Automatic Fallback**: System falls back gracefully when preferred storage unavailable

### 4. Auto-Refresh Settings
```javascript
// Available refresh intervals
const refreshIntervals = [
  { label: '5 seconds', value: 5000 },
  { label: '30 seconds', value: 30000 },
  { label: '1 minute', value: 60000 },
  { label: '5 minutes', value: 300000 }
];
```

## 🖥️ Multi-Server Monitoring

### 1. Server Configuration
```json
{
  "servers": [
    {
      "name": "Production Web",
      "address": "https://web.example.com:3500",
      "table_name": "web_server_monitoring"
    },
    {
      "name": "Database Server", 
      "address": "https://db.example.com:3500",
      "table_name": "database_monitoring"
    }
  ]
}
```

### 2. Server Status Indicators
- 🟢 **Online**: Server responding normally
- 🟡 **Warning**: High resource usage
- 🔴 **Offline**: Server not responding
- ⚪ **Unknown**: No recent data

### 3. Server Switching
1. Click server name in navigation
2. Dashboard automatically loads server-specific data
3. Historical data preserved per server
4. Independent time range selection

## 💓 Heartbeat Monitoring

### 1. Service Status Overview
```
┌─────────────────────────────────────────────┐
│ External Services Health                    │
├─────────────────┬──────────┬────────────────┤
│ Google          │ ✅ 100ms │ Last: 2 min ago │
│ GitHub API      │ ✅ 150ms │ Last: 2 min ago │
│ Local API       │ ❌ Error │ Last: 5 min ago │
└─────────────────┴──────────┴────────────────┘
```

### 2. Response Time Charts
- Real-time latency monitoring
- Historical response time trends
- Availability percentage calculation
- Downtime detection and alerts

### 3. Configuration
```json
{
  "heartbeat": [
    {
      "name": "Main Website",
      "url": "https://example.com",
      "timeout": 5
    },
    {
      "name": "API Server",
      "url": "https://api.example.com", 
      "timeout": 3
    }
  ]
}
```

## 📈 Charts and Visualizations

### 1. System Performance Chart
- **Dual-axis**: CPU percentage and memory percentage
- **Time-based**: X-axis shows time progression
- **Interactive**: Hover for exact values
- **Responsive**: Adapts to screen size

### 2. Network Activity Chart
- **Upload/Download**: Separate lines for TX/RX
- **Units**: Automatically scales (B/s, KB/s, MB/s)
- **Real-time**: Updates with live data
- **Historical**: Shows trends over selected timeframe

### 3. Usage Donut Charts
- **Resource Distribution**: CPU, Memory, Disk usage
- **Color-coded**: Green (low), yellow (medium), red (high)
- **Percentage Display**: Current utilization levels
- **Responsive**: Adapts to container size

## 🎨 Theme and Customization

### 1. Dark/Light Mode
- **Auto-detection**: Respects system preference
- **Manual Toggle**: Switch themes manually
- **Persistent**: Remembers user preference
- **Responsive**: All charts adapt to theme

### 2. Color Schemes
```css
/* Light theme */
--bg-color: #ffffff;
--text-color: #1a1a1a;
--chart-bg: #f8f9fa;

/* Dark theme */
--bg-color: #1a1a1a;
--text-color: #ffffff; 
--chart-bg: #2d2d2d;
```

### 3. Layout Options
- **Responsive Grid**: Adapts to screen width
- **Mobile-friendly**: Touch-optimized controls
- **Fullscreen Charts**: Expandable visualizations
- **Compact Mode**: Dense information display

## 🔍 Data Analysis Features

### 1. Time Range Analysis
- **Zoom Controls**: Focus on specific time periods
- **Custom Ranges**: Select exact start/end times
- **Quick Presets**: 1h, 6h, 24h, 7d, 30d buttons
- **URL Sharing**: Shareable links with time selections

### 2. Performance Insights
- **Peak Detection**: Identifies usage spikes
- **Trend Analysis**: Shows performance patterns
- **Baseline Comparison**: Compare against averages
- **Anomaly Highlighting**: Unusual patterns

### 3. Export Options
```javascript
// Chart data export
const exportData = {
  timeRange: '24h',
  metrics: ['cpu', 'memory', 'network'],
  format: 'json', // or 'csv'
  server: 'production-web'
};
```

## 🚨 Alerts and Notifications

### 1. Visual Indicators
- **Color-coded Metrics**: Red for high usage
- **Blinking Alerts**: Attention-grabbing indicators
- **Status Badges**: Clear state representation
- **Toast Notifications**: Non-intrusive alerts

### 2. Threshold Configuration
```javascript
const thresholds = {
  cpu: { warning: 70, critical: 90 },
  memory: { warning: 80, critical: 95 },
  disk: { warning: 85, critical: 95 },
  response_time: { warning: 1000, critical: 5000 }
};
```

### 3. Alert Types
- **Performance**: High resource usage
- **Connectivity**: Server unreachable
- **Heartbeat**: Service down
- **Data**: Missing or stale data

## 📱 Mobile Experience

### 1. Responsive Design
- **Touch-friendly**: Large tap targets
- **Swipe Gestures**: Navigate between charts
- **Zoom Support**: Pinch to zoom charts
- **Orientation**: Works in portrait/landscape

### 2. Mobile-Optimized Features
- **Simplified Navigation**: Collapsible menus
- **Essential Metrics**: Priority information first
- **Fast Loading**: Optimized data requests
- **Offline Indicators**: Connection status

## ⚙️ Advanced Configuration

### 1. URL Parameters
```
https://monitoring.example.com/?
  server=production&
  range=24h&
  refresh=30s&
  theme=dark
```

### 2. Local Storage Settings
```javascript
const userPreferences = {
  defaultTimeRange: '6h',
  autoRefresh: true,
  refreshInterval: 30000,
  theme: 'auto',
  compactMode: false
};
```

### 3. Dashboard Customization
```json
{
  "dashboard": {
    "defaultRange": "6h",
    "chartsPerRow": 2,
    "showLegends": true,
    "animateTransitions": true,
    "enableExport": true
  }
}
```

## 🔗 Integration Options

### 1. API Access
```bash
# Get current metrics
curl https://monitoring.example.com/api/v1/monitoring

# Get historical data
curl "https://monitoring.example.com/api/v1/monitoring" \
  -d '{"from":"2023-01-01T00:00:00Z","to":"2023-01-02T00:00:00Z"}'
```

### 2. Webhook Support
```javascript
// Real-time updates via WebSocket
const ws = new WebSocket('wss://monitoring.example.com/ws');
ws.onmessage = (event) => {
  const metrics = JSON.parse(event.data);
  updateDashboard(metrics);
};
```

### 3. External Monitoring
- **Grafana Integration**: Custom datasource
- **Prometheus Export**: Metrics endpoint
- **InfluxDB**: Time-series data export
- **Custom Scripts**: API automation

## 🔗 Related Documentation

- [🚀 Production Deployment](production-deployment.md) - Server setup
- [🗄️ PostgreSQL Setup](postgresql-setup.md) - Database configuration  
- [🔧 CLI Usage](cli-usage.md) - Command line tools
- [🏠 Main README](../README.md) - Getting started

---
**🎯 The dashboard is designed for both real-time monitoring and historical analysis. Use time ranges and server switching to get the insights you need.**