import { state } from './state.js';
import { escapeHtml, formatLastChecked, formatResponseTime } from './utils.js';

function mapServerStatus(status) {
  const s = String(status || '').toLowerCase();
  if (s === 'ok' || s === 'online' || s === 'healthy') {
    return { label: 'Online', className: 'status-up' };
  }
  if (s === 'error' || s === 'offline') {
    return { label: 'Offline', className: 'status-down' };
  }
  if (s === 'stale' || s === 'degraded') {
    return { label: 'Stale', className: 'status-warning' };
  }
  return { label: 'Unknown', className: 'status-warning' };
}

function formatPercent(value) {
  const n = Number(value);
  return Number.isFinite(n) ? `${n.toFixed(1)}%` : '—';
}

function formatLoad(value) {
  if (value === undefined || value === null) return '—';
  if (typeof value === 'string') {
    const parts = value.split(/[\s,]+/).filter(Boolean);
    return parts.length ? parts[0] : value;
  }
  if (Array.isArray(value) && value.length) {
    const first = Number(value[0]);
    return Number.isFinite(first) ? first.toFixed(2) : String(value[0]);
  }
  const n = Number(value);
  return Number.isFinite(n) ? n.toFixed(2) : '—';
}

function toNumber(value) {
  const n = Number(value);
  return Number.isFinite(n) ? n : NaN;
}

function firstLoadNumber(value) {
  if (value == null) return NaN;
  if (typeof value === 'string') {
    const parts = value.split(/[\s,]+/).filter(Boolean);
    return toNumber(parts[0]);
  }
  if (Array.isArray(value) && value.length) {
    return toNumber(value[0]);
  }
  if (typeof value === 'object' && value.one_minute !== undefined) {
    return toNumber(value.one_minute);
  }
  return toNumber(value);
}

function getSeverityFor(type, raw) {
  const NEAR_DANGER_FRACTION = 0.35;
  let value = toNumber(raw);
  if (type === 'load') {
    value = firstLoadNumber(raw);
  }
  if (!Number.isFinite(value)) return '';

  if (type === 'cpu' || type === 'memory' || type === 'disk') {
    const thresholds = state.thresholds?.[type];
    if (!thresholds) return '';
    const { warning, critical } = thresholds;
    if (!Number.isFinite(warning) || !Number.isFinite(critical)) return '';
    if (value >= critical) return 'danger';
    const buffer = Math.max(2, (critical - warning) * NEAR_DANGER_FRACTION);
    if (value >= critical - buffer) return 'caution';
    if (value >= warning) return 'warning';
    return 'safe';
  }

  if (type === 'latency') {
    // ms thresholds
    const warning = 200; // 200ms
    const critical = 1000; // 1s
    if (value >= critical) return 'danger';
    const buffer = Math.max(50, (critical - warning) * NEAR_DANGER_FRACTION);
    if (value >= critical - buffer) return 'caution';
    if (value >= warning) return 'warning';
    return 'safe';
  }

  if (type === 'load') {
    // Simple load thresholds (1, 2, 4)
    const warning = 1;
    const critical = 4;
    if (value >= critical) return 'danger';
    const buffer = Math.max(0.25, (critical - warning) * NEAR_DANGER_FRACTION);
    if (value >= critical - buffer) return 'caution';
    if (value >= warning) return 'warning';
    return 'safe';
  }

  return '';
}

function renderServersTable() {
  const tbody = document.getElementById('compactServersBody');
  if (!tbody) return;

  const serverRows = Array.isArray(state.serverMetrics) ? state.serverMetrics : [];

  // Build a top row from local real-time metrics if available
  const local = state.previousMetrics || null;
  const localRow = local
    ? {
        name: 'Local System',
        address: 'local',
        status: state.connectionState === 'offline' ? 'error' : 'ok',
        cpu_usage: local.cpu_usage,
        memory_used_percent: local.memory?.percentage,
        disk_used_percent: local.disk?.percentage,
        load_average: local.load_average,
        timestamp: local.timestamp || ''
      }
    : null;

  const rows = localRow ? [localRow, ...serverRows] : serverRows;
  if (rows.length === 0) {
    tbody.innerHTML = '<tr><td colspan="8">No servers configured</td></tr>';
    return;
  }

  const fragment = document.createDocumentFragment();
  // helpers: disk breakdown (all disks)
  function calcDiskPct(disk = {}) {
    const v = Number(disk.used_pct);
    if (Number.isFinite(v)) return v;
    const used = Number(disk.used_bytes);
    const total = Number(disk.total_bytes);
    if (Number.isFinite(used) && Number.isFinite(total) && total > 0) {
      return (used / total) * 100;
    }
    return NaN;
  }

  function resolveAllDisks(metric) {
    const disks = metric?.disk_space ?? metric?.diskSpace ?? [];
    if (!Array.isArray(disks) || disks.length === 0) return [];
    const normalized = disks
      .map((d) => ({ ...d, used_pct: calcDiskPct(d) }))
      .filter((d) => Number.isFinite(d.used_pct));
    if (normalized.length === 0) return [];
    // Group per device root where possible to avoid per-partition spam
    const groupKey = (d) => {
      const dev = String(d.device || '').trim();
      if (!dev) return d.path || '';
      const mSd = dev.match(/^\/dev\/sd([a-z])[0-9]+$/);
      if (mSd) return `/dev/sd${mSd[1]}`;
      const mNv = dev.match(/^(\/dev\/nvme\d+n\d+)p\d+$/);
      if (mNv) return mNv[1];
      const mOsx = dev.match(/^(\/dev\/disk\d+)s\d+$/);
      if (mOsx) return mOsx[1];
      return dev;
    };
    const grouped = new Map();
    normalized.forEach((d) => {
      const key = groupKey(d);
      const g = grouped.get(key) || { total_bytes: 0, used_bytes: 0, paths: [], device: key || d.device };
      g.total_bytes += Number(d.total_bytes) || 0;
      g.used_bytes += Number(d.used_bytes) || 0;
      if (d.path) g.paths.push(d.path);
      grouped.set(key, g);
    });
    const ROOTS = ['/', '\\', '/System/Volumes/Data'];
    const list = Array.from(grouped.values()).map((g) => {
      const value = g.total_bytes > 0 ? (g.used_bytes / g.total_bytes) * 100 : NaN;
      const path = g.paths.find((p) => ROOTS.includes(p)) || g.paths[0] || '';
      const device = g.device || '';
      return { path, device, value };
    }).filter((x) => Number.isFinite(x.value));
    const primary = list.find((d) => ROOTS.includes(d.path));
    const others = list.filter((d) => d !== primary);
    const out = [];
    if (primary) out.push({ type: 'primary', path: primary.path, device: primary.device, value: primary.value });
    others.forEach((d) => out.push({ type: 'mounted', path: d.path, device: d.device, value: d.value }));
    if (!primary && out.length > 0) out[0].type = 'primary';
    return out;
  }

  rows.forEach((metric) => {
    const tr = document.createElement('tr');
    const status = mapServerStatus(metric.status);

    const cpuSeverity = getSeverityFor('cpu', metric.cpu_usage);
    const memSeverity = getSeverityFor('memory', metric.memory_used_percent);
    const loadSeverity = getSeverityFor('load', metric.load_average);

    const disks = resolveAllDisks(metric);
    const diskBadgesHtml = disks.length
      ? `<div class="disk-badges">${disks
          .map((d) => {
            const sev = getSeverityFor('disk', d.value);
            const icon = d.type === 'primary' ? '★' : '⦿';
            const tooltip = (d.device && String(d.device).trim()) || (d.path || 'Mounted');
            return `<span class="disk-badge"><span class="disk-badge__icon" aria-hidden="true" data-tooltip="${escapeHtml(tooltip)}">${icon}</span><span class="${sev ? `server-card__metric-value--${sev}` : ''}">${formatPercent(d.value)}</span></span>`;
          })
          .join('')}</div>`
      : `<span class="${getSeverityFor('disk', metric.disk_used_percent) ? `server-card__metric-value--${getSeverityFor('disk', metric.disk_used_percent)}` : ''}">${formatPercent(metric.disk_used_percent)}</span>`;

    tr.innerHTML = `
      <td>${escapeHtml(metric.name || '')}</td>
      <td class="mono">${escapeHtml(metric.address || '')}</td>
      <td>
        <span class="server-status ${status.className}"><span class="status-dot"></span><span class="status-text">${escapeHtml(status.label)}</span></span>
      </td>
      <td class="num"><span class="${cpuSeverity ? `server-card__metric-value--${cpuSeverity}` : ''}">${formatPercent(metric.cpu_usage)}</span></td>
      <td class="num"><span class="${memSeverity ? `server-card__metric-value--${memSeverity}` : ''}">${formatPercent(metric.memory_used_percent)}</span></td>
      <td class="num disk-cell">${diskBadgesHtml}</td>
      <td class="num"><span class="${loadSeverity ? `server-card__metric-value--${loadSeverity}` : ''}">${formatLoad(metric.load_average)}</span></td>
      <td>${escapeHtml(formatLastChecked(metric.timestamp))}</td>
    `;
    fragment.appendChild(tr);
  });

  tbody.replaceChildren(fragment);
}

function renderHeartbeatsTable() {
  const tbody = document.getElementById('compactHeartbeatsBody');
  if (!tbody) return;

  const hb = (state.previousMetrics && Array.isArray(state.previousMetrics.heartbeat))
    ? state.previousMetrics.heartbeat
    : [];

  if (hb.length === 0) {
    tbody.innerHTML = '<tr><td colspan="5">No heartbeat data</td></tr>';
    return;
  }
  const fragment = document.createDocumentFragment();

  hb.forEach((server) => {
    const status = String(server.status || '').toLowerCase();

    const tr = document.createElement('tr');
    const statusClass = status === 'up' ? 'status-up' : status === 'down' ? 'status-down' : 'status-warning';

    const latencyMs = toNumber(server.response_ms);
    const latencySeverity = getSeverityFor('latency', latencyMs);

    tr.innerHTML = `
      <td>${escapeHtml(server.name || '')}</td>
      <td class="mono">${escapeHtml(server.url || '')}</td>
      <td><span class="server-status ${statusClass}"><span class="status-dot"></span><span class="status-text">${escapeHtml(status === 'up' ? 'Online' : status === 'down' ? 'Offline' : (server.status || 'Unknown'))}</span></span></td>
      <td class="num"><span class="${latencySeverity ? `server-card__metric-value--${latencySeverity}` : ''}">${escapeHtml(formatResponseTime(server.response_ms))}</span></td>
      <td>${escapeHtml(formatLastChecked(server.last_checked))}</td>
    `;
    fragment.appendChild(tr);
  });

  tbody.replaceChildren(fragment);
}

export function updateCompactView() {
  const container = document.getElementById('compactView');
  if (!container) return;
  const isCompact = document.body.getAttribute('data-theme') === 'compact';
  container.style.display = isCompact ? '' : 'none';
  if (!isCompact) return;
  renderServersTable();
  renderHeartbeatsTable();
}
