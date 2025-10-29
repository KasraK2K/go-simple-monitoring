import { DEFAULT_THRESHOLDS, MAX_SERIES_POINTS } from '@/lib/constants';
import {
  AlertMessage,
  MetricPoint,
  NetworkDelta,
  NormalizedMetrics,
  ThresholdConfig
} from '@/lib/types';

function toNumber(value: unknown): number | null {
  const num = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(num) ? num : null;
}

function safeArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) {
    return value as T[];
  }
  if (value == null) {
    return [];
  }
  return [value as T];
}

export function normalizeMetrics(raw: any): NormalizedMetrics {
  if (!raw || typeof raw !== 'object') {
    return {};
  }

  const container = raw.body && typeof raw.body === 'object' ? raw.body : raw;

  const cpu = container.cpu ?? {};
  const ram = container.ram ?? {};
  const diskArray = container.disk_space ?? container.disk ?? [];
  const networkIO = container.network_io ?? container.network ?? {};
  const processInfo = container.process ?? {};

  const cpuUsage = container.cpu_usage
    ?? cpu.usage_percent
    ?? container.cpu_usage_percent
    ?? null;

  const memory = {
    percentage: container.memory?.percentage
      ?? ram.used_pct
      ?? container.ram_used_percent
      ?? null,
    total_bytes: ram.total_bytes ?? container.ram_total_bytes ?? null,
    used_bytes: ram.used_bytes ?? container.ram_used_bytes ?? null,
    available_bytes: ram.available_bytes ?? container.ram_available_bytes ?? null,
    buffer_bytes: ram.buffer_bytes ?? container.ram_buffer_bytes ?? null
  };

  const diskSpaces = safeArray<any>(diskArray)
    .map(disk => (disk && typeof disk === 'object' ? disk : null))
    .filter(Boolean) as any[];

  const diskTotals = diskSpaces.reduce(
    (acc, disk) => {
      const total = toNumber(disk.total_bytes) ?? 0;
      const used = toNumber(disk.used_bytes) ?? 0;
      const available = toNumber(disk.available_bytes) ?? 0;
      return {
        total: acc.total + total,
        used: acc.used + used,
        available: acc.available + available
      };
    },
    { total: 0, used: 0, available: 0 }
  );

  const disk = diskTotals.total > 0
    ? {
        percentage: diskTotals.used / diskTotals.total * 100,
        total_bytes: diskTotals.total,
        used_bytes: diskTotals.used,
        available_bytes: diskTotals.available
      }
    : {
        percentage: container.disk?.percentage
          ?? container.disk_used_percent
          ?? container.disk?.usage_percent
          ?? null,
        total_bytes: container.disk?.total_bytes ?? container.disk_total_bytes ?? null,
        used_bytes: container.disk?.used_bytes ?? container.disk_used_bytes ?? null,
        available_bytes: container.disk?.available_bytes ?? container.disk_available_bytes ?? null
      };

  const network = {
    bytes_received: networkIO.bytes_recv
      ?? networkIO.bytes_received
      ?? container.network_bytes_recv
      ?? null,
    bytes_sent: networkIO.bytes_sent
      ?? container.network_bytes_sent
      ?? null,
    packets_received: networkIO.packets_recv
      ?? networkIO.packets_received
      ?? container.network_packets_recv
      ?? null,
    packets_sent: networkIO.packets_sent
      ?? container.network_packets_sent
      ?? null,
    errors_in: networkIO.errors_in
      ?? container.network_errors_in
      ?? null,
    errors_out: networkIO.errors_out
      ?? container.network_errors_out
      ?? null,
    drops_in: networkIO.drops_in
      ?? container.network_drops_in
      ?? null,
    drops_out: networkIO.drops_out
      ?? container.network_drops_out
      ?? null
  };

  const loadAverageSource = container.load_average ?? processInfo ?? {};
  const load_average = {
    one_minute: loadAverageSource.one_minute
      ?? processInfo.load_avg_1
      ?? container.process_load_avg_1
      ?? null,
    five_minutes: loadAverageSource.five_minutes
      ?? processInfo.load_avg_5
      ?? container.process_load_avg_5
      ?? null,
    fifteen_minutes: loadAverageSource.fifteen_minutes
      ?? processInfo.load_avg_15
      ?? container.process_load_avg_15
      ?? null
  };

  const heartbeat = safeArray<any>(container.heartbeat ?? raw.heartbeat)
    .map((item: any) => ({
      name: item?.name ?? item?.service_name ?? 'Unknown service',
      url: item?.url ?? item?.endpoint,
      status: item?.status ?? 'unknown',
      last_beat: item?.last_beat ?? item?.last_check ?? item?.timestamp,
      last_duration_ms: item?.last_duration_ms ?? item?.last_duration ?? null,
      uptime_percentage: item?.uptime_percentage ?? item?.uptime ?? null,
      tags: Array.isArray(item?.tags) ? item.tags : [],
      region: item?.region,
      description: item?.description
    }));

  return {
    timestamp: raw.timestamp
      ?? container.timestamp
      ?? raw.time
      ?? container.time
      ?? null,
    cpu_usage: cpuUsage,
    memory,
    disk,
    disk_spaces: diskSpaces,
    network,
    load_average,
    heartbeat
  };
}

export function calculateNetworkDelta(previous: NormalizedMetrics | null, current: NormalizedMetrics | null): NetworkDelta | null {
  if (!previous || !current) {
    return null;
  }

  const prevRx = toNumber(previous.network?.bytes_received);
  const prevTx = toNumber(previous.network?.bytes_sent);
  const currRx = toNumber(current.network?.bytes_received);
  const currTx = toNumber(current.network?.bytes_sent);

  if (prevRx == null || prevTx == null || currRx == null || currTx == null) {
    return null;
  }

  const prevTs = previous.timestamp ? new Date(previous.timestamp).getTime() : null;
  const currTs = current.timestamp ? new Date(current.timestamp).getTime() : null;
  const durationSeconds = prevTs && currTs && currTs > prevTs ? (currTs - prevTs) / 1000 : null;

  const bytesReceived = currRx >= prevRx ? currRx - prevRx : 0;
  const bytesSent = currTx >= prevTx ? currTx - prevTx : 0;

  return {
    bytesReceived,
    bytesSent,
    durationSeconds
  };
}

export function createMetricPoint(metrics: NormalizedMetrics, delta: NetworkDelta | null): MetricPoint {
  const duration = delta?.durationSeconds && delta.durationSeconds > 0 ? delta.durationSeconds : 1;
  const networkRx = delta ? delta.bytesReceived / 1024 / 1024 / duration : null;
  const networkTx = delta ? delta.bytesSent / 1024 / 1024 / duration : null;

  return {
    timestamp: metrics.timestamp ?? new Date().toISOString(),
    cpu: metrics.cpu_usage ?? null,
    memory: metrics.memory?.percentage ?? null,
    disk: metrics.disk?.percentage ?? null,
    networkRx,
    networkTx
  };
}

export function appendPoint(series: MetricPoint[], point: MetricPoint): MetricPoint[] {
  return [...series, point].slice(-MAX_SERIES_POINTS);
}

export function evaluateThresholds(
  metrics: NormalizedMetrics | null,
  thresholds: Partial<Record<'cpu' | 'memory' | 'disk', ThresholdConfig>> = DEFAULT_THRESHOLDS
): AlertMessage[] {
  if (!metrics) {
    return [];
  }

  const checks: { key: string; label: string; value: number | null; config?: ThresholdConfig }[] = [
    { key: 'cpu', label: 'CPU', value: metrics.cpu_usage ?? null, config: thresholds.cpu ?? DEFAULT_THRESHOLDS.cpu },
    { key: 'memory', label: 'Memory', value: metrics.memory?.percentage ?? null, config: thresholds.memory ?? DEFAULT_THRESHOLDS.memory }
  ];

  if (metrics.disk_spaces && metrics.disk_spaces.length > 0) {
    metrics.disk_spaces.forEach((disk, index) => {
      checks.push({
        key: `disk_${disk.path ?? index}`,
        label: `Disk ${disk.path ?? index + 1}`,
        value: disk.used_pct ?? null,
        config: thresholds.disk ?? DEFAULT_THRESHOLDS.disk
      });
    });
  } else {
    checks.push({
      key: 'disk_overall',
      label: 'Disk',
      value: metrics.disk?.percentage ?? null,
      config: thresholds.disk ?? DEFAULT_THRESHOLDS.disk
    });
  }

  const alerts: AlertMessage[] = [];
  const now = Date.now();

  checks.forEach(check => {
    const value = check.value;
    if (value == null || check.config == null) {
      return;
    }
    const { warning, critical } = check.config;
    if (value >= critical) {
      alerts.push({
        id: `critical_${check.key}`,
        type: 'error',
        message: `${check.label} usage critical at ${value.toFixed(1)}%`,
        createdAt: now,
        persistent: true
      });
    } else if (value >= warning) {
      alerts.push({
        id: `warning_${check.key}`,
        type: 'warning',
        message: `${check.label} usage high at ${value.toFixed(1)}%`,
        createdAt: now
      });
    }
  });

  return alerts;
}
