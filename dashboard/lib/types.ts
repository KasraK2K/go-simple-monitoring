export type HeartbeatStatus = 'healthy' | 'degraded' | 'offline' | 'unknown';

export interface HeartbeatEntry {
  name: string;
  url?: string;
  status: HeartbeatStatus;
  last_beat?: string;
  last_duration_ms?: number;
  uptime_percentage?: number;
  tags?: string[];
  region?: string;
  description?: string;
}

export interface NetworkStats {
  bytes_received?: number;
  bytes_sent?: number;
  packets_received?: number;
  packets_sent?: number;
  errors_in?: number;
  errors_out?: number;
  drops_in?: number;
  drops_out?: number;
}

export interface DiskSpace {
  path?: string;
  filesystem?: string;
  total_bytes?: number;
  used_bytes?: number;
  available_bytes?: number;
  used_pct?: number;
}

export interface MemoryStats {
  percentage?: number;
  total_bytes?: number;
  used_bytes?: number;
  available_bytes?: number;
  buffer_bytes?: number;
}

export interface LoadAverage {
  one_minute?: number;
  five_minutes?: number;
  fifteen_minutes?: number;
}

export interface NormalizedMetrics {
  timestamp?: string;
  cpu_usage?: number;
  memory?: MemoryStats;
  disk?: {
    percentage?: number;
    total_bytes?: number;
    used_bytes?: number;
    available_bytes?: number;
  };
  disk_spaces?: DiskSpace[];
  network?: NetworkStats;
  load_average?: LoadAverage;
  heartbeat?: HeartbeatEntry[];
}

export interface MetricPoint {
  timestamp: string;
  cpu: number | null;
  memory: number | null;
  disk: number | null;
  networkRx?: number | null;
  networkTx?: number | null;
}

export interface NetworkDelta {
  bytesReceived: number;
  bytesSent: number;
  durationSeconds?: number | null;
}

export interface AlertMessage {
  id: string;
  type: 'info' | 'warning' | 'error';
  message: string;
  createdAt: number;
  persistent?: boolean;
}

export interface MonitorServerOption {
  name: string;
  address: string;
}

export interface ServerConfig {
  refresh_interval_seconds?: number;
  servers?: MonitorServerOption[];
  thresholds?: {
    cpu?: ThresholdConfig;
    memory?: ThresholdConfig;
    disk?: ThresholdConfig;
  };
}

export interface ThresholdConfig {
  warning: number;
  critical: number;
}

export interface DateRangeFilter {
  from?: string | null;
  to?: string | null;
}
