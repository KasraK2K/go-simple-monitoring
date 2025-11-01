import { state } from './state.js';
import { formatLastChecked, sanitizeBaseUrl } from './utils.js';

function normalizeAddress(address) {
  return sanitizeBaseUrl(address || '');
}

function formatPercent(value) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? `${numeric.toFixed(1)}%` : '—';
}

function formatLoadAverage(value) {
  if (!value && value !== 0) return '—';
  if (Array.isArray(value)) {
    const first = value.find((item) => Number.isFinite(Number(item)) || typeof item === 'string');
    return first ? String(first) : '—';
  }
  if (typeof value === 'string') {
    const parts = value.split(/[,\s]+/).filter(Boolean);
    return parts.length > 0 ? parts[0] : value || '—';
  }
  const numeric = Number(value);
  if (Number.isFinite(numeric)) {
    return numeric.toFixed(2);
  }
  return String(value);
}

function statusLabel(status) {
  switch ((status || '').toLowerCase()) {
    case 'ok':
    case 'online':
    case 'healthy':
      return 'Online';
    case 'error':
    case 'offline':
      return 'Error';
    case 'stale':
    case 'degraded':
      return 'Stale';
    default:
      return 'Unknown';
  }
}

function statusClass(status) {
  switch ((status || '').toLowerCase()) {
    case 'ok':
    case 'online':
    case 'healthy':
      return 'server-status--ok';
    case 'error':
    case 'offline':
      return 'server-status--error';
    case 'stale':
    case 'degraded':
      return 'server-status--stale';
    default:
      return 'server-status--unknown';
  }
}

function createMetric(label, value) {
  const wrapper = document.createElement('div');
  wrapper.className = 'server-card__metric';

  const labelEl = document.createElement('span');
  labelEl.className = 'server-card__metric-label';
  labelEl.textContent = label;

  const valueEl = document.createElement('span');
  valueEl.className = 'server-card__metric-value';
  valueEl.textContent = value;

  wrapper.append(labelEl, valueEl);
  return wrapper;
}

function createServerCard(metric) {
  const card = document.createElement('article');
  card.className = 'server-card';

  const header = document.createElement('header');
  header.className = 'server-card__header';

  const titleWrap = document.createElement('div');

  const title = document.createElement('h3');
  title.className = 'server-card__title';
  title.textContent = metric.name || 'Unnamed Server';

  const address = document.createElement('p');
  address.className = 'server-card__address';
  address.textContent = metric.address || '—';

  titleWrap.append(title, address);

  const status = document.createElement('span');
  status.className = `server-status ${statusClass(metric.status)}`;

  const statusDot = document.createElement('span');
  statusDot.className = 'status-dot';

  const statusText = document.createElement('span');
  statusText.className = 'status-text';
  statusText.textContent = statusLabel(metric.status);

  status.append(statusDot, statusText);

  const actions = document.createElement('div');
  actions.className = 'server-card__actions';
  actions.append(status);

  const normalizedAddress = normalizeAddress(metric.address || metric.raw?.address || '');
  const activeAddress = normalizeAddress(state.selectedBaseUrl);
  if (normalizedAddress) {
    const switchBtn = document.createElement('button');
    switchBtn.type = 'button';
    switchBtn.className = 'server-card__switch';

    const isActive = normalizedAddress === activeAddress && activeAddress !== '';
    if (isActive) {
      switchBtn.textContent = 'Viewing live';
      switchBtn.classList.add('server-card__switch--active');
      switchBtn.disabled = true;
    } else {
      switchBtn.textContent = 'View remotely';
      switchBtn.addEventListener('click', () => {
        if (typeof state.handleServerSelection === 'function') {
          state.handleServerSelection({ name: metric.name, address: normalizedAddress });
        }
      });
    }

    actions.appendChild(switchBtn);
  }

  header.append(titleWrap, actions);

  const metricsWrap = document.createElement('div');
  metricsWrap.className = 'server-card__metrics';
  metricsWrap.append(
    createMetric('CPU', formatPercent(metric.cpu_usage)),
    createMetric('Memory', formatPercent(metric.memory_used_percent)),
    createMetric('Disk', formatPercent(metric.disk_used_percent)),
    createMetric('Load', formatLoadAverage(metric.load_average))
  );

  const footer = document.createElement('footer');
  footer.className = 'server-card__footer';

  const updated = document.createElement('span');
  updated.className = 'server-card__timestamp';
  if (metric.timestamp) {
    updated.textContent = `Updated ${formatLastChecked(metric.timestamp)}`;
    const date = new Date(metric.timestamp);
    if (!Number.isNaN(date.getTime())) {
      updated.title = date.toLocaleString();
    }
  } else {
    updated.textContent = 'Awaiting first sample';
  }

  footer.append(updated);

  if (metric.message && metric.message.trim()) {
    const message = document.createElement('span');
    message.className = 'server-card__message';
    message.textContent = metric.message;
    footer.append(message);
  }

  card.append(header, metricsWrap, footer);
  return card;
}

function ensureDisplayMetric(rawMetric, fallbackName = '', fallbackAddress = '') {
  const metric = { ...rawMetric };
  metric.name = metric.name || fallbackName;
  metric.address = metric.address || fallbackAddress;
  metric.status = metric.status || 'unknown';
  if (metric.timestamp === undefined) {
    metric.timestamp = '';
  }
  return metric;
}

function mergeConfiguredWithMetrics(metrics, configured) {
  const normalizedMap = new Map();
  metrics.forEach((metric) => {
    if (!metric) return;
    const key = normalizeAddress(metric.address || metric.name || '');
    if (!normalizedMap.has(key)) {
      normalizedMap.set(key, metric);
    }
  });

  const output = [];
  const hasConfigured = Array.isArray(configured) && configured.length > 0;

  if (hasConfigured) {
    configured.forEach((server) => {
      if (!server) return;
      const key = normalizeAddress(server.address || server.name || '');
      const metric = normalizedMap.get(key);
      if (metric) {
        output.push(ensureDisplayMetric(metric, server.name, key));
        normalizedMap.delete(key);
      } else {
        output.push({
          name: server.name || 'Unnamed Server',
          address: key,
          status: 'error',
          message: 'No data yet',
          cpu_usage: NaN,
          memory_used_percent: NaN,
          disk_used_percent: NaN,
          load_average: '',
          timestamp: ''
        });
      }
    });
  }

  normalizedMap.forEach((metric) => {
    output.push(ensureDisplayMetric(metric));
  });

  return output;
}

export function updateServerMetricsSection(metrics = [], configuredServers = []) {
  const container = document.getElementById('serverMetricsList');
  const summary = document.getElementById('serverMetricsSummary');
  if (!container || !summary) {
    return;
  }

  const merged = mergeConfiguredWithMetrics(metrics || [], configuredServers || []);
  state.serverMetrics = merged;

  if (merged.length === 0) {
    const message = Array.isArray(configuredServers) && configuredServers.length > 0
      ? 'Waiting for remote metrics…'
      : 'No servers configured yet';
    const empty = document.createElement('div');
    empty.id = 'serverMetricsEmpty';
    empty.className = 'servers-empty';
    empty.textContent = message;
    container.replaceChildren(empty);
    summary.textContent = message;
    return;
  }

  const fragment = document.createDocumentFragment();
  merged.forEach((metric) => {
    fragment.appendChild(createServerCard(metric));
  });
  container.replaceChildren(fragment);

  const healthy = merged.filter((metric) => (metric.status || '').toLowerCase() === 'ok').length;
  summary.textContent = `${healthy} healthy / ${merged.length} total`;
}
