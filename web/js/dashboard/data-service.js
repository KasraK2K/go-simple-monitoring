import { checkThresholds, showAlert } from './alerts.js';
import { LOCAL_SERVER_OPTION } from './constants.js';
import { renderHistoricalCharts, resetCharts, updateCharts } from './charts.js';
import { calculateHeartbeatUptime, updateHeartbeat } from './heartbeat.js';
import { calculateNetworkDelta } from './network.js';
import { state } from './state.js';
import { setLoadingState, showErrorState, updateConnectionStatus, updateRefreshDisplay, updateStatus } from './ui.js';
import { sanitizeBaseUrl } from './utils.js';
import { updateMetrics, updateTrends } from './metrics.js';

export async function fetchMetrics() {
  try {
    if (state.isInitialLoad) {
      setLoadingState('initial', true);
    }

    const monitoringUrl = buildEndpoint('/monitoring');
    const filterPayload = state.pendingFilter;
    const requestBody = filterPayload
      ? JSON.stringify({
        from: filterPayload.from || undefined,
        to: filterPayload.to || undefined
      })
      : null;

    const response = await fetch(monitoringUrl, {
      method: 'POST',
      cache: 'no-store',
      headers: requestBody ? { 'Content-Type': 'application/json' } : undefined,
      body: requestBody
    });

    if (!response.ok) {
      throw new Error(`Request failed with status ${response.status}`);
    }

    const payload = await response.json();
    let entries = payload;

    if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
      entries = payload.data ?? [];
    }

    if (!Array.isArray(entries)) {
      entries = entries ? [entries] : [];
    }

    const data = entries[0] || null;

    if (!data) {
      throw new Error('No monitoring data available');
    }

    const normalizedList = entries
      .map((entry) => normalizeMetrics(entry))
      .filter((item) => item && typeof item === 'object');

    if (normalizedList.length === 0) {
      throw new Error('No monitoring data available');
    }

    if (filterPayload) {
      state.historicalSeries = normalizedList;
    } else if (state.historicalMode) {
      const latest = normalizedList[0];
      const latestTimestamp = latest.timestamp;
      state.historicalSeries = [
        latest,
        ...state.historicalSeries.filter((item) => item.timestamp !== latestTimestamp)
      ];
    }

    const activeSeries = state.historicalMode && state.historicalSeries.length > 0
      ? state.historicalSeries
      : normalizedList;

    const latestNormalized = activeSeries[0];
    const comparisonBaseline = activeSeries[1] || state.previousMetrics;
    const networkDelta = calculateNetworkDelta(latestNormalized, comparisonBaseline);

    latestNormalized.network_delta = networkDelta;

    updateMetrics(latestNormalized, networkDelta);

    if (state.historicalMode) {
      renderHistoricalCharts(activeSeries);
    } else {
      updateCharts(latestNormalized, networkDelta);
    }

    updateTrends(latestNormalized, networkDelta, comparisonBaseline);
    updateStatus(true);
    updateConnectionStatus('online');

    checkThresholds(latestNormalized);

    let uptimeStats = null;
    if (state.historicalMode) {
      uptimeStats = calculateHeartbeatUptime(activeSeries);
    }
    updateHeartbeat(latestNormalized.heartbeat || [], uptimeStats);

    const lastUpdated = document.getElementById('lastUpdated');
    if (lastUpdated) {
      lastUpdated.textContent = new Date().toLocaleTimeString();
    }

    state.previousMetrics = latestNormalized;
    state.retryCount = 0;

    if (filterPayload) {
      state.pendingFilter = null;
    }

    if (state.isInitialLoad) {
      setLoadingState('initial', false);
      state.isInitialLoad = false;
    }
  } catch (error) {
    console.error('Error fetching metrics:', error);
    state.retryCount += 1;

    if (state.isInitialLoad) {
      setLoadingState('initial', false);
      state.isInitialLoad = false;
    }

    if (state.retryCount > state.maxRetries) {
      updateConnectionStatus('offline', error.message);
      showAlert('error', `Connection failed: ${error.message}`);
      showErrorState();
    }
  }
}

export function scheduleNextFetch() {
  clearTimeout(state.refreshTimer);
  state.refreshTimer = setTimeout(async () => {
    try {
      await fetchServerConfig(state.selectedBaseUrl);
      await fetchMetrics();
    } catch (error) {
      console.error('Scheduled fetch failed:', error);
    } finally {
      scheduleNextFetch();
    }
  }, state.refreshInterval);
}

export async function fetchServerConfig(baseUrl = '') {
  try {
    const sanitizedBase = sanitizeBaseUrl(baseUrl);
    const endpoint = sanitizedBase ? `${sanitizedBase}/api/v1/server-config` : '/api/v1/server-config';
    const response = await fetch(endpoint, { cache: 'no-store' });
    if (!response.ok) {
      return null;
    }

    const config = await response.json();
    if (!sanitizedBase) {
      state.serverConfig = config;
      state.availableServers = Array.isArray(config.servers)
        ? config.servers
          .filter((server) => server && server.name && server.address)
          .map((server) => ({
            name: server.name,
            address: sanitizeBaseUrl(server.address)
          }))
        : [];

      renderServerButtons();
    }

    const candidate = Number(config.refresh_interval_seconds) * 1000;
    if (!Number.isNaN(candidate) && candidate > 0) {
      state.refreshInterval = candidate;
      updateRefreshDisplay();
    }

    return config;
  } catch (error) {
    const target = baseUrl ? baseUrl : 'local';
    console.debug(`Using existing server config for ${target}:`, error);
    return null;
  }
}

export function renderServerButtons() {
  const container = document.getElementById('serverSwitcher');
  if (!container) return;

  container.innerHTML = '';

  const servers = [LOCAL_SERVER_OPTION, ...state.availableServers];
  const activeAddress = sanitizeBaseUrl(state.selectedBaseUrl);

  servers.forEach((server) => {
    if (!server || !server.name) {
      return;
    }

    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'server-button';
    button.textContent = server.name;

    if (server === LOCAL_SERVER_OPTION || !server.address) {
      button.classList.add('local');
    } else {
      button.classList.add('remote');
    }

    const serverAddress = sanitizeBaseUrl(server.address || '');
    if (serverAddress === activeAddress) {
      button.classList.add('active');
    }

    button.addEventListener('click', () => handleServerSelection(server));
    container.appendChild(button);
  });
}

export async function handleServerSelection(server) {
  const address = sanitizeBaseUrl(server.address || '');
  if (address === sanitizeBaseUrl(state.selectedBaseUrl)) {
    return;
  }

  state.selectedServer = server.address ? { ...server, address } : null;
  state.selectedBaseUrl = address;
  state.pendingFilter = null;
  state.historicalMode = false;
  state.historicalSeries = [];
  state.previousMetrics = null;

  updateStatus(false);
  updateHeartbeat();
  const lastUpdated = document.getElementById('lastUpdated');
  if (lastUpdated) {
    lastUpdated.textContent = '--';
  }

  resetCharts();
  renderServerButtons();

  clearTimeout(state.refreshTimer);
  await fetchServerConfig(state.selectedBaseUrl);
  await fetchMetrics();
  scheduleNextFetch();
}

function calculateOverallDiskUsage(diskSpaces, container) {
  if (!diskSpaces || diskSpaces.length === 0) {
    const legacy = container.disk ?? {};
    return {
      percentage: container.disk?.percentage
        ?? legacy.used_pct
        ?? container.disk_used_percent
        ?? null,
      total_bytes: legacy.total_bytes ?? container.disk_total_bytes ?? null,
      used_bytes: legacy.used_bytes ?? container.disk_used_bytes ?? null,
      available_bytes: legacy.available_bytes ?? container.disk_available_bytes ?? null
    };
  }

  let totalBytes = 0;
  let usedBytes = 0;
  let availableBytes = 0;

  for (const disk of diskSpaces) {
    if (disk.total_bytes && typeof disk.total_bytes === 'number') {
      totalBytes += disk.total_bytes;
      usedBytes += disk.used_bytes || 0;
      availableBytes += disk.available_bytes || 0;
    }
  }

  const percentage = totalBytes > 0 ? (usedBytes / totalBytes) * 100 : 0;

  return {
    percentage: Math.round(percentage * 100) / 100,
    total_bytes: totalBytes,
    used_bytes: usedBytes,
    available_bytes: availableBytes
  };
}

function normalizeMetrics(raw) {
  if (!raw || typeof raw !== 'object') return {};

  const container = raw.body && typeof raw.body === 'object'
    ? raw.body
    : raw;

  const cpu = container.cpu ?? {};
  const ram = container.ram ?? {};
  const diskArray = container.disk_space ?? container.disk ?? [];
  const networkIO = container.network_io ?? container.network ?? {};
  const process = container.process ?? {};

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

  const diskSpaces = Array.isArray(diskArray) ? diskArray : [diskArray].filter((d) => d && Object.keys(d).length > 0);
  const diskInfo = calculateOverallDiskUsage(diskSpaces, container);

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

  const loadAverageSource = container.load_average ?? {};
  const loadAverage = {
    one_minute: loadAverageSource.one_minute
      ?? process.load_avg_1
      ?? container.process_load_avg_1
      ?? null,
    five_minutes: loadAverageSource.five_minutes
      ?? process.load_avg_5
      ?? container.process_load_avg_5
      ?? null,
    fifteen_minutes: loadAverageSource.fifteen_minutes
      ?? process.load_avg_15
      ?? container.process_load_avg_15
      ?? null
  };

  return {
    timestamp: raw.timestamp
      ?? container.timestamp
      ?? raw.time
      ?? container.time
      ?? null,
    cpu_usage: cpuUsage,
    memory,
    disk: diskInfo,
    disk_spaces: diskSpaces,
    network,
    load_average: loadAverage,
    heartbeat: container.heartbeat ?? raw.heartbeat ?? [],
    disk_io: container.disk_io ?? {},
    process,
    raw
  };
}

function buildEndpoint(path) {
  const base = sanitizeBaseUrl(state.selectedBaseUrl);
  if (!base) {
    return path;
  }
  if (!path.startsWith('/')) {
    return `${base}/${path}`;
  }
  return `${base}${path}`;
}
