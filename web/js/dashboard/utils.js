import { state } from './state.js';

export function sanitizeBaseUrl(url) {
  if (!url) return '';
  return String(url).replace(/\/+$/, '');
}

export function parseTimestamp(value) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function escapeHtml(value) {
  if (value === undefined || value === null) return '';
  const content = String(value);
  return content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function bytesToMb(bytes) {
  if (bytes === undefined || bytes === null) return undefined;
  return bytes / 1024 / 1024;
}

export function bytesToMbPerSecond(bytes, seconds = null) {
  if (bytes === undefined || bytes === null) return undefined;
  const intervalSeconds = Number.isFinite(seconds) && seconds > 0
    ? seconds
    : state.refreshInterval / 1000;
  if (!Number.isFinite(intervalSeconds) || intervalSeconds <= 0) {
    return bytesToMb(bytes);
  }
  return (bytes / 1024 / 1024) / intervalSeconds;
}

export function toFiniteNumber(value) {
  if (value === undefined || value === null) return NaN;
  const numeric = Number(value);
  return Number.isFinite(numeric) ? numeric : NaN;
}

export function formatResponseTime(ms) {
  const value = Number(ms);
  if (!Number.isFinite(value) || value < 0) return '—';
  if (value < 1000) return `${Math.round(value)} ms`;
  return `${(value / 1000).toFixed(2)} s`;
}

export function formatLastChecked(timestamp) {
  if (!timestamp) return '—';
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return '—';

  const diffSeconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (diffSeconds < 5) return 'moments ago';
  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

export function statusClass(status) {
  if (status === 'up') return 'status-up';
  if (status === 'down') return 'status-down';
  return 'status-warning';
}

export function statusLabel(status) {
  if (status === 'up') return 'Online';
  if (status === 'down') return 'Offline';
  return status ? status.charAt(0).toUpperCase() + status.slice(1) : 'Unknown';
}

export function getHeartbeatKey(server) {
  if (!server) return '';
  return server.url || server.name || '';
}
