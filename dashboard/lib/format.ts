import { format } from 'date-fns';

export function formatBytes(value?: number | null, digits = 1): string {
  if (!value || !Number.isFinite(value)) {
    return '--';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let size = value;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  return `${size.toFixed(digits)} ${units[unitIndex]}`;
}

export function formatPercent(value?: number | null, digits = 1): string {
  if (value == null || !Number.isFinite(value)) {
    return '--';
  }
  return `${value.toFixed(digits)}%`;
}

export function formatNumber(value?: number | null, digits = 0): string {
  if (value == null || !Number.isFinite(value)) {
    return '--';
  }
  return value.toFixed(digits);
}

export function formatDateTime(value?: string | number | Date | null): string {
  if (!value) {
    return '--';
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '--';
  }
  return format(date, 'MMM d, yyyy HH:mm:ss');
}

export function formatRelativeDate(value?: string | number | Date | null): string {
  if (!value) {
    return '--';
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '--';
  }
  return format(date, 'HH:mm:ss');
}

export function formatDuration(ms?: number | null): string {
  if (ms == null || !Number.isFinite(ms)) {
    return '--';
  }
  if (ms < 1000) {
    return `${ms.toFixed(0)} ms`;
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(1)} s`;
  }
  const minutes = ms / 60_000;
  return `${minutes.toFixed(1)} min`;
}
