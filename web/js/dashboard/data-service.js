import { checkThresholds, showAlert } from "./alerts.js";
import { LOCAL_SERVER_OPTION } from "./constants.js";
import { renderHistoricalCharts, resetCharts, updateCharts } from "./charts.js";
import { calculateHeartbeatUptime, updateHeartbeat } from "./heartbeat.js";
import { calculateNetworkDelta } from "./network.js";
import { state } from "./state.js";
import { updateServerMetricsSection } from "./servers.js";
import {
  setLoadingState,
  showErrorState,
  updateConnectionStatus,
  updateRefreshDisplay,
  updateRemoteContext,
  updateStatus,
} from "./ui.js";
import { sanitizeBaseUrl } from "./utils.js";
import { updateMetrics, updateTrends } from "./metrics.js";
import { buildFilterFromRange } from "./ranges.js";

export async function fetchMetrics() {
  try {
    if (state.isInitialLoad) {
      setLoadingState("initial", true);
    }

    const usingAutoFilter = Boolean(state.autoFilter);
    const filterPayload = state.autoFilter || state.pendingFilter;
    
    // For remote servers with historical data requests, we need to query local database first
    const isRemoteServer = Boolean(state.selectedBaseUrl);
    
    let monitoringUrl, requestBody;
    
    if (isRemoteServer) {
      // For remote servers, ALWAYS use local server with table_name parameter
      const tableName = findTableNameForServer(state.selectedBaseUrl);
      monitoringUrl = "/monitoring"; // Local server endpoint
      requestBody = JSON.stringify({
        table_name: tableName,
        from: filterPayload?.from || undefined,
        to: filterPayload?.to || undefined,
      });
    } else {
      // Local server requests: use local endpoint
      monitoringUrl = "/monitoring";
      requestBody = filterPayload
        ? JSON.stringify({
            from: filterPayload.from || undefined,
            to: filterPayload.to || undefined,
          })
        : null;
    }

    const response = await fetch(monitoringUrl, {
      method: "POST",
      cache: "no-store",
      headers: requestBody ? { "Content-Type": "application/json" } : undefined,
      body: requestBody,
    });

    if (!response.ok) {
      throw new Error(`Request failed with status ${response.status}`);
    }

    const payload = await response.json();
    let entries = payload;

    if (payload && typeof payload === "object" && !Array.isArray(payload)) {
      entries = payload.data ?? [];
    }

    if (!Array.isArray(entries)) {
      entries = entries ? [entries] : [];
    }

    const data = entries[0] || null;

    if (!data) {
      if (filterPayload) {
        if (usingAutoFilter) {
          state.autoFilter = null;
          return fetchMetrics();
        }
        // For date range filters with no data, show empty state instead of error
        if (state.historicalMode) {
          state.historicalSeries = [];
          renderHistoricalCharts([]);
          updateStatus(true);
          updateConnectionStatus("online");
          
          // Clear other UI elements
          updateHeartbeat([], null);
          updateServerMetricsSection([], []);
          updateMetrics({
            cpu_usage: 0,
            memory: { percentage: 0 },
            disk: { percentage: 0 },
            network: { bytes_received: 0, bytes_sent: 0 }
          }, { rx_rate: 0, tx_rate: 0 });
          
          const lastUpdated = document.getElementById("lastUpdated");
          if (lastUpdated) {
            lastUpdated.textContent = "No data in selected range";
          }
          
          if (state.isInitialLoad) {
            setLoadingState("initial", false);
            state.isInitialLoad = false;
          }
          return;
        }
      }
      throw new Error("No monitoring data available");
    }

    const normalizedList = entries
      .map((entry) => normalizeMetrics(entry))
      .filter((item) => item && typeof item === "object");

    normalizedList.sort((a, b) => getTimestampMs(b) - getTimestampMs(a));

    if (normalizedList.length === 0) {
      // For date range filters with no valid data, show empty state instead of error
      if (filterPayload && state.historicalMode) {
        state.historicalSeries = [];
        renderHistoricalCharts([]);
        updateStatus(true);
        updateConnectionStatus("online");
        
        // Clear other UI elements
        updateHeartbeat([], null);
        updateServerMetricsSection([], []);
        updateMetrics({
          cpu_usage: 0,
          memory: { percentage: 0 },
          disk: { percentage: 0 },
          network: { bytes_received: 0, bytes_sent: 0 }
        }, { rx_rate: 0, tx_rate: 0 });
        
        const lastUpdated = document.getElementById("lastUpdated");
        if (lastUpdated) {
          lastUpdated.textContent = "No data in selected range";
        }
        
        if (state.isInitialLoad) {
          setLoadingState("initial", false);
          state.isInitialLoad = false;
        }
        return;
      }
      throw new Error("No monitoring data available");
    }

    if (filterPayload) {
      state.historicalSeries = normalizedList;
      // Set historical mode when we have filter data (including autoFilter) to ensure proper chart rendering
      if (!state.historicalMode) {
        state.historicalMode = true;
      }
    } else if (state.historicalMode) {
      const latest = normalizedList[0];
      const latestTimestamp = latest.timestamp;
      state.historicalSeries = [
        latest,
        ...state.historicalSeries.filter(
          (item) => item.timestamp !== latestTimestamp
        ),
      ];
    }

    const activeSeries =
      state.historicalMode && state.historicalSeries.length > 0
        ? state.historicalSeries
        : normalizedList;

    const latestNormalized = activeSeries[0];
    const comparisonBaseline = activeSeries[1] || state.previousMetrics;
    const networkDelta = calculateNetworkDelta(
      latestNormalized,
      comparisonBaseline
    );

    latestNormalized.network_delta = networkDelta;

    updateMetrics(latestNormalized, networkDelta);

    if (state.historicalMode) {
      renderHistoricalCharts(activeSeries);
    } else {
      updateCharts(latestNormalized, networkDelta);
    }

    updateTrends(latestNormalized, networkDelta, comparisonBaseline);
    updateStatus(true);
    updateConnectionStatus("online");

    if (usingAutoFilter) {
      state.autoFilter = null;
    }

    checkThresholds(latestNormalized);

    let uptimeStats = null;
    if (state.historicalMode) {
      uptimeStats = calculateHeartbeatUptime(activeSeries);
    }
    updateHeartbeat(latestNormalized.heartbeat || [], uptimeStats);
    updateServerMetricsSection(
      latestNormalized.server_metrics || [],
      state.serverConfig?.servers || []
    );

    const lastUpdated = document.getElementById("lastUpdated");
    if (lastUpdated) {
      lastUpdated.textContent = new Date().toLocaleTimeString();
    }

    state.previousMetrics = latestNormalized;
    state.retryCount = 0;

    if (filterPayload) {
      state.pendingFilter = null;
    }

    if (state.isInitialLoad) {
      setLoadingState("initial", false);
      state.isInitialLoad = false;
    }

    // After processing autoFilter, transition to live mode for subsequent requests
    if (usingAutoFilter && !state.pendingFilter) {
      state.historicalMode = false;
      state.historicalSeries = [];
      
    }
  } catch (error) {
    console.error("Error fetching metrics:", error);
    state.retryCount += 1;

    if (state.isInitialLoad) {
      setLoadingState("initial", false);
      state.isInitialLoad = false;
    }

    if (state.retryCount > state.maxRetries) {
      updateConnectionStatus("offline", error.message);
      showAlert("error", `Connection failed: ${error.message}`);
      showErrorState();
    }
  }
}

export function scheduleNextFetch() {
  clearTimeout(state.refreshTimer);
  state.refreshTimer = setTimeout(async () => {
    try {
      // Always fetch server config from local server to get complete server list
      await fetchServerConfig("");
      await fetchMetrics();
    } catch (error) {
      console.error("Scheduled fetch failed:", error);
    } finally {
      scheduleNextFetch();
    }
  }, state.refreshInterval);
}

export async function fetchServerConfig(baseUrl = "") {
  try {
    const sanitizedBase = sanitizeBaseUrl(baseUrl);
    const endpoint = sanitizedBase
      ? `${sanitizedBase}/api/v1/server-config`
      : "/api/v1/server-config";
    const response = await fetch(endpoint, { cache: "no-store" });
    if (!response.ok) {
      return null;
    }

    const config = await response.json();
    const activeBase = sanitizeBaseUrl(state.selectedBaseUrl);
    if (sanitizedBase !== activeBase) {
      return config;
    }

    const normalizedServers = normalizeConfiguredServers(config.servers || []);

    state.serverConfig = config;
    state.availableServers = normalizedServers;

    if (!activeBase) {
      state.selectedServer = null;
    } else {
      const matchedServer = normalizedServers.find(
        (server) => server.address === activeBase
      );
      if (matchedServer) {
        state.selectedServer = { ...matchedServer };
      }
    }
    renderServerButtons();
    updateRemoteContext();

    updateServerMetricsSection(state.serverMetrics || [], config.servers || []);

    const candidate = Number(config.refresh_interval_seconds) * 1000;
    if (!Number.isNaN(candidate) && candidate > 0) {
      state.refreshInterval = candidate;
      updateRefreshDisplay();
    }

    return config;
  } catch (error) {
    const target = baseUrl ? baseUrl : "local";
    console.debug(`Using existing server config for ${target}:`, error);
    return null;
  }
}

export function renderServerButtons() {
  const container = document.getElementById("serverSwitcher");
  if (!container) return;

  container.innerHTML = "";

  const activeAddress = sanitizeBaseUrl(state.selectedBaseUrl);
  const servers = [LOCAL_SERVER_OPTION, ...state.availableServers];

  if (activeAddress) {
    const hasActive = servers.some(
      (server) => sanitizeBaseUrl(server.address || "") === activeAddress
    );
    if (!hasActive) {
      const label =
        state.selectedServer?.name ||
        (state.serverConfig?.name && String(state.serverConfig.name).trim()) ||
        activeAddress;
      servers.push({ name: label, address: activeAddress });
    }
  }

  const seenAddresses = new Set();

  servers.forEach((server) => {
    if (!server || !server.name) {
      return;
    }

    const button = document.createElement("button");
    button.type = "button";
    button.className = "server-button";
    button.textContent = server.name;

    if (server === LOCAL_SERVER_OPTION || !server.address) {
      button.classList.add("local");
    } else {
      button.classList.add("remote");
    }

    const serverAddress = sanitizeBaseUrl(server.address || "");
    if (serverAddress) {
      if (seenAddresses.has(serverAddress)) {
        return;
      }
      seenAddresses.add(serverAddress);
    }

    if (serverAddress === activeAddress) {
      button.classList.add("active");
    }

    button.addEventListener("click", () => handleServerSelection(server));
    container.appendChild(button);
  });
}

export async function handleServerSelection(server) {
  const address = sanitizeBaseUrl(server.address || "");
  if (address === sanitizeBaseUrl(state.selectedBaseUrl)) {
    return;
  }

  state.selectedServer = server.address ? { ...server, address } : null;
  state.selectedBaseUrl = address;
  
  // Fetch server config from local server to ensure state.serverConfig has the complete server list
  await fetchServerConfig("");
  
  state.autoFilter = state.defaultRangePreset ? buildFilterFromRange(state.defaultRangePreset) : null;
  state.pendingFilter = null;
  applyAutoFilterUI(state.autoFilter, state.defaultRangePreset || null);
  state.historicalMode = false;
  state.historicalSeries = [];
  state.previousMetrics = null;

  updateStatus(false);
  updateHeartbeat();
  updateServerMetricsSection([], []);
  const lastUpdated = document.getElementById("lastUpdated");
  if (lastUpdated) {
    lastUpdated.textContent = "--";
  }

  resetCharts();
  renderServerButtons();
  updateRemoteContext();

  clearTimeout(state.refreshTimer);
  await fetchServerConfig("");
  await fetchMetrics();
  scheduleNextFetch();
}

function calculateOverallDiskUsage(diskSpaces, container) {
  if (!diskSpaces || diskSpaces.length === 0) {
    const legacy = container.disk ?? {};
    return {
      percentage:
        container.disk?.percentage ??
        legacy.used_pct ??
        container.disk_used_percent ??
        null,
      total_bytes: legacy.total_bytes ?? container.disk_total_bytes ?? null,
      used_bytes: legacy.used_bytes ?? container.disk_used_bytes ?? null,
      available_bytes:
        legacy.available_bytes ?? container.disk_available_bytes ?? null,
    };
  }

  let totalBytes = 0;
  let usedBytes = 0;
  let availableBytes = 0;

  for (const disk of diskSpaces) {
    if (disk.total_bytes && typeof disk.total_bytes === "number") {
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
    available_bytes: availableBytes,
  };
}

function normalizeMetrics(raw) {
  if (!raw || typeof raw !== "object") return {};

  const container = raw.body && typeof raw.body === "object" ? raw.body : raw;

  const cpu = container.cpu ?? {};
  const ram = container.ram ?? {};
  const diskArray = container.disk_space ?? container.disk ?? [];
  const networkIO = container.network_io ?? container.network ?? {};
  const process = container.process ?? {};

  const cpuUsage =
    container.cpu_usage ??
    cpu.usage_percent ??
    container.cpu_usage_percent ??
    null;

  const memory = {
    percentage:
      container.memory?.percentage ??
      ram.used_pct ??
      container.ram_used_percent ??
      null,
    total_bytes: ram.total_bytes ?? container.ram_total_bytes ?? null,
    used_bytes: ram.used_bytes ?? container.ram_used_bytes ?? null,
    available_bytes:
      ram.available_bytes ?? container.ram_available_bytes ?? null,
    buffer_bytes: ram.buffer_bytes ?? container.ram_buffer_bytes ?? null,
  };

  const diskSpaces = Array.isArray(diskArray)
    ? diskArray
    : [diskArray].filter((d) => d && Object.keys(d).length > 0);
  const diskInfo = calculateOverallDiskUsage(diskSpaces, container);

  const network = {
    bytes_received:
      networkIO.bytes_recv ??
      networkIO.bytes_received ??
      container.network_bytes_recv ??
      null,
    bytes_sent: networkIO.bytes_sent ?? container.network_bytes_sent ?? null,
    packets_received:
      networkIO.packets_recv ??
      networkIO.packets_received ??
      container.network_packets_recv ??
      null,
    packets_sent:
      networkIO.packets_sent ?? container.network_packets_sent ?? null,
    errors_in: networkIO.errors_in ?? container.network_errors_in ?? null,
    errors_out: networkIO.errors_out ?? container.network_errors_out ?? null,
    drops_in: networkIO.drops_in ?? container.network_drops_in ?? null,
    drops_out: networkIO.drops_out ?? container.network_drops_out ?? null,
  };

  const loadAverageSource = container.load_average ?? {};
  const loadAverage = {
    one_minute:
      loadAverageSource.one_minute ??
      process.load_avg_1 ??
      container.process_load_avg_1 ??
      null,
    five_minutes:
      loadAverageSource.five_minutes ??
      process.load_avg_5 ??
      container.process_load_avg_5 ??
      null,
    fifteen_minutes:
      loadAverageSource.fifteen_minutes ??
      process.load_avg_15 ??
      container.process_load_avg_15 ??
      null,
  };

  return {
    timestamp:
      raw.timestamp ??
      container.timestamp ??
      raw.time ??
      container.time ??
      null,
    cpu_usage: cpuUsage,
    memory,
    disk: diskInfo,
    disk_spaces: diskSpaces,
    network,
    load_average: loadAverage,
    heartbeat: container.heartbeat ?? raw.heartbeat ?? [],
    server_metrics: container.server_metrics ?? raw.server_metrics ?? [],
    disk_io: container.disk_io ?? {},
    process,
    raw,
  };
}

function buildEndpoint(path) {
  const base = sanitizeBaseUrl(state.selectedBaseUrl);
  if (!base) {
    return path;
  }
  if (!path.startsWith("/")) {
    return `${base}/${path}`;
  }
  return `${base}${path}`;
}

function normalizeConfiguredServers(servers) {
  if (!Array.isArray(servers)) {
    return [];
  }

  const normalized = [];
  const seen = new Set();

  servers.forEach((server) => {
    if (!server) return;

    const address = sanitizeBaseUrl(server.address || "");
    if (!address || seen.has(address)) {
      return;
    }

    const name =
      typeof server.name === "string" && server.name.trim().length > 0
        ? server.name.trim()
        : address;

    normalized.push({ name, address });
    seen.add(address);
  });

  return normalized;
}

export function applyAutoFilterUI(filter, activeRange) {
  const inputs = state.filterElements;

  if (!filter) {
    if (inputs?.fromInput) {
      inputs.fromInput.value = "";
    }
    if (inputs?.toInput) {
      inputs.toInput.value = "";
    }
    setActiveRangeButton(null);
    return;
  }

  if (inputs?.fromInput) {
    const fromDate = new Date(filter.from);
    if (!Number.isNaN(fromDate.getTime())) {
      inputs.fromInput.value = formatDateForInput(fromDate);
    }
  }
  if (inputs?.toInput) {
    const toDate = new Date(filter.to);
    if (!Number.isNaN(toDate.getTime())) {
      inputs.toInput.value = formatDateForInput(toDate);
    }
  }

  setActiveRangeButton(activeRange || null);
}

function setActiveRangeButton(range) {
  document.querySelectorAll(".range-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.range === range);
  });
}

function formatDateForInput(date) {
  const pad = (value) => String(value).padStart(2, "0");
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  );
}

function findTableNameForServer(serverAddress) {
  const sanitizedAddress = sanitizeBaseUrl(serverAddress);
  
  // Look for the server in the configuration to find its table_name
  const servers = state.serverConfig?.servers || [];
  
  for (const server of servers) {
    const serverSanitized = sanitizeBaseUrl(server.address);
    if (serverSanitized === sanitizedAddress) {
      return server.table_name || `local_monitoring_${extractPortFromAddress(sanitizedAddress)}`;
    }
  }
  
  // Fallback: generate table name from port
  return `local_monitoring_${extractPortFromAddress(sanitizedAddress)}`;
}

function extractPortFromAddress(address) {
  try {
    const url = new URL(address.startsWith('http') ? address : `http://${address}`);
    return url.port || '80';
  } catch {
    // Extract port from address string manually if URL parsing fails
    const match = address.match(/:(\d+)$/);
    return match ? match[1] : '80';
  }
}

state.handleServerSelection = handleServerSelection;

function getTimestampMs(entry) {
  const value = entry?.timestamp || entry?.time || entry?.raw?.timestamp || entry?.raw?.time;
  if (!value) {
    return 0;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 0 : date.getTime();
}
