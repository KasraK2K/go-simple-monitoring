// Core system variables
let systemChart;
let networkChart;
let usageDonut;
let refreshInterval = 2000;
let refreshTimer;
let previousMetrics = null;
let serverConfig = null;
let availableServers = [];
let selectedServer = null;
let selectedBaseUrl = '';
let pendingFilter = null;
let historicalSeries = [];
let historicalMode = false;
let filterElements = null;
const LOCAL_SERVER_OPTION = { name: 'Local', address: '' };

// New enhancement variables
let connectionState = 'online';
let currentTheme = 'dark';
let alertsQueue = [];
let activeAlerts = new Map();
let notificationPermission = 'default';
let lastMetricValues = {};
let thresholds = {
    cpu: { warning: 70, critical: 90 },
    memory: { warning: 80, critical: 95 },
    disk: { warning: 85, critical: 95 }
};
let isLoading = false;
let retryCount = 0;
let maxRetries = 3;
let filteredHeartbeats = [];
let searchTerm = '';
let isInitialLoad = true;

// Performance optimization
let chartUpdateQueue = [];
let isUpdatingCharts = false;

const REQUIRED_ELEMENT_IDS = [
    'themeToggle',
    'themeIcon',
    'connectionStatus',
    'connectionText',
    'alertsPanel',
    'exportOverlay',
    'exportPanel',
    'exportCSV',
    'exportJSON',
    'exportClose',
    'exportTrigger',
    'serverSwitcher',
    'filterFrom',
    'filterTo',
    'applyFilter',
    'clearFilter',
    'initialLoading',
    'systemChart',
    'networkChart',
    'usageDonut',
    'heartbeatList',
    'heartbeatSearch',
    'heartbeatSummary',
    'cpu',
    'memory',
    'networkRx',
    'networkTx',
    'loadAvg',
    'cpuBar',
    'memoryBar',
    'cpuTrend',
    'memoryTrend',
    'loadTrend'
];

let initialized = false;
let initInProgress = false;
let bootstrapIntervalId = null;
let watchersRegistered = false;

function sanitizeBaseUrl(url) {
    if (!url) return '';
    return String(url).replace(/\/+$/, '');
}

// Theme Management
function toggleTheme() {
    currentTheme = currentTheme === 'dark' ? 'light' : 'dark';
    document.body.setAttribute('data-theme', currentTheme);
    const themeIcon = document.getElementById('themeIcon');
    themeIcon.className = currentTheme === 'dark' ? 'fas fa-moon' : 'fas fa-sun';
    localStorage.setItem('theme', currentTheme);

    // Reinitialize charts with new theme colors
    if (systemChart) {
        updateChartTheme(systemChart);
    }
    if (networkChart) {
        updateChartTheme(networkChart);
    }
    if (usageDonut) {
        updateChartTheme(usageDonut);
    }
}

function updateChartTheme(chart) {
    const textColor = currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)';
    const gridColor = currentTheme === 'dark' ? 'rgba(148, 163, 184, 0.18)' : 'rgba(100, 116, 139, 0.2)';
    const borderColor = currentTheme === 'dark' ? 'rgba(148, 163, 184, 0.25)' : 'rgba(100, 116, 139, 0.3)';

    if (chart.options.plugins?.legend?.labels) {
        chart.options.plugins.legend.labels.color = textColor;
    }
    if (chart.options.scales?.y?.ticks) {
        chart.options.scales.y.ticks.color = textColor;
    }
    if (chart.options.scales?.x?.ticks) {
        chart.options.scales.x.ticks.color = textColor;
    }
    if (chart.options.scales?.y?.grid) {
        chart.options.scales.y.grid.color = gridColor;
        chart.options.scales.y.grid.borderColor = borderColor;
    }

    chart.update('none');
}

// Connection Status Management
function updateConnectionStatus(status, message = '') {
    const statusEl = document.getElementById('connectionStatus');
    const textEl = document.getElementById('connectionText');

    connectionState = status;

    statusEl.className = `connection-status ${status}`;

    switch (status) {
        case 'online':
            textEl.textContent = 'Connected';
            statusEl.classList.add('hidden');
            break;
        case 'offline':
            textEl.textContent = message || 'Connection Lost';
            statusEl.classList.remove('hidden');
            break;
    }
}

// Alert System
function showAlert(type, message, duration = 5000) {
    const alertsPanel = document.getElementById('alertsPanel');
    const alertId = 'alert_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);

    const alertEl = document.createElement('div');
    alertEl.className = `alert-item ${type}`;
    alertEl.innerHTML = `
        <i class="fas fa-exclamation-triangle"></i>
        <span>${escapeHtml(message)}</span>
        <span class="alert-close" onclick="removeAlert('${alertId}')"><i class="fas fa-times"></i></span>
    `;

    alertsPanel.appendChild(alertEl);
    activeAlerts.set(alertId, alertEl);

    // Auto-remove after duration
    if (duration > 0) {
        setTimeout(() => removeAlert(alertId), duration);
    }

    // Browser notification if permission granted
    if (notificationPermission === 'granted' && type === 'error') {
        new Notification('System Monitoring Alert', {
            body: message
        });
    }

    return alertId;
}

function removeAlert(alertId) {
    const storedValue = activeAlerts.get(alertId);
    if (storedValue) {
        // If it's a string, it's an alert ID - find the actual element
        if (typeof storedValue === 'string') {
            const alertEl = activeAlerts.get(storedValue);
            if (alertEl && alertEl.remove) {
                alertEl.remove();
                activeAlerts.delete(storedValue);
            }
        }
        // If it has a remove method, it's a DOM element
        else if (storedValue && typeof storedValue.remove === 'function') {
            storedValue.remove();
        }
        activeAlerts.delete(alertId);
    }
}

function clearAllAlerts() {
    activeAlerts.forEach((storedValue) => {
        if (storedValue && typeof storedValue.remove === 'function') {
            storedValue.remove();
        }
    });
    activeAlerts.clear();
}

// Threshold Monitoring
function checkThresholds(data) {
    const checks = [
        { name: 'CPU', value: data.cpu_usage, thresholds: thresholds.cpu, unit: '%' },
        { name: 'Memory', value: data.memory?.percentage, thresholds: thresholds.memory, unit: '%' }
    ];

    // Add disk checks for each individual disk
    if (data.disk_spaces && Array.isArray(data.disk_spaces)) {
        data.disk_spaces.forEach((disk, index) => {
            if (disk.used_pct != null && Number.isFinite(disk.used_pct)) {
                checks.push({
                    name: `Disk ${disk.path || `#${index + 1}`}`,
                    value: disk.used_pct,
                    thresholds: thresholds.disk,
                    unit: '%'
                });
            }
        });
    } else if (data.disk?.percentage != null) {
        // Fallback to legacy single disk
        checks.push({ name: 'Disk', value: data.disk.percentage, thresholds: thresholds.disk, unit: '%' });
    }

    checks.forEach(check => {
        if (check.value == null || !Number.isFinite(check.value)) return;

        const alertKey = `threshold_${check.name.toLowerCase().replace(/[^a-z0-9]/g, '_')}`;

        if (check.value >= check.thresholds.critical) {
            if (!activeAlerts.has(alertKey)) {
                const alertId = showAlert('error', `${check.name} usage critical: ${check.value.toFixed(1)}${check.unit}`, 0);
                activeAlerts.set(alertKey, alertId);
            }
        } else if (check.value >= check.thresholds.warning) {
            if (!activeAlerts.has(alertKey)) {
                const alertId = showAlert('warning', `${check.name} usage high: ${check.value.toFixed(1)}${check.unit}`, 10000);
                activeAlerts.set(alertKey, alertId);
            }
        } else {
            // Remove threshold alert if it exists and value is now normal
            if (activeAlerts.has(alertKey)) {
                removeAlert(alertKey);
            }
        }
    });
}

// Loading State Management
function setLoadingState(section, loading) {
    // Only show loading for initial load, not subsequent refreshes
    if (section === 'initial') {
        const loadingEl = document.getElementById('initialLoading');
        if (loadingEl) {
            loadingEl.style.display = loading ? 'flex' : 'none';
        }
        isLoading = loading;
    }
}

// Export Functionality
function showExportPanel() {
    document.getElementById('exportOverlay').classList.add('visible');
    document.getElementById('exportPanel').classList.add('visible');
}

function hideExportPanel() {
    document.getElementById('exportOverlay').classList.remove('visible');
    document.getElementById('exportPanel').classList.remove('visible');
}

function exportData(format) {
    try {
        let data, filename, mimeType;
        const timestamp = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-');

        if (format === 'csv') {
            data = generateCSV();
            filename = `monitoring-data-${timestamp}.csv`;
            mimeType = 'text/csv';
        } else if (format === 'json') {
            data = JSON.stringify({
                exported_at: new Date().toISOString(),
                server: selectedServer?.name || 'Local',
                historical_mode: historicalMode,
                data: historicalMode ? historicalSeries : [previousMetrics]
            }, null, 2);
            filename = `monitoring-data-${timestamp}.json`;
            mimeType = 'application/json';
        }

        const blob = new Blob([data], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);

        showAlert('success', `Data exported as ${format.toUpperCase()}`);
        hideExportPanel();
    } catch (error) {
        showAlert('error', `Export failed: ${error.message}`);
    }
}

function generateCSV() {
    const series = historicalMode ? historicalSeries : [previousMetrics];
    if (!series || series.length === 0) return 'No data available';

    const headers = ['timestamp', 'cpu_usage', 'memory_percentage', 'disk_percentage', 'network_rx_mb', 'network_tx_mb', 'load_average'];
    const rows = [headers.join(',')];

    series.forEach(item => {
        if (!item) return;
        const row = [
            item.timestamp || '',
            item.cpu_usage || '',
            item.memory?.percentage || '',
            item.disk?.percentage || '',
            bytesToMb(item.network_delta?.bytes_received) || '',
            bytesToMb(item.network_delta?.bytes_sent) || '',
            item.load_average?.one_minute || ''
        ];
        rows.push(row.join(','));
    });

    return rows.join('\n');
}

// Range Preset Functionality
function applyRangePreset(range) {
    const now = new Date();
    let fromDate;

    switch (range) {
        case '1h':
            fromDate = new Date(now.getTime() - 60 * 60 * 1000);
            break;
        case '6h':
            fromDate = new Date(now.getTime() - 6 * 60 * 60 * 1000);
            break;
        case '24h':
            fromDate = new Date(now.getTime() - 24 * 60 * 60 * 1000);
            break;
        case '7d':
            fromDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
            break;
        case '30d':
            fromDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
            break;
        default:
            return;
    }

    // Format for datetime-local input
    const formatDate = (date) => {
        return date.toISOString().slice(0, 16);
    };

    if (filterElements.fromInput) {
        filterElements.fromInput.value = formatDate(fromDate);
    }
    if (filterElements.toInput) {
        filterElements.toInput.value = formatDate(now);
    }

    // Update active state
    document.querySelectorAll('.range-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.range === range);
    });

    // Auto-apply the filter
    applyDateFilter();
}

// Heartbeat Search and Filter
function filterHeartbeats(searchTerm = '') {
    const heartbeatList = document.getElementById('heartbeatList');
    const cards = heartbeatList.querySelectorAll('.heartbeat-card');

    let visibleCount = 0;
    cards.forEach(card => {
        const name = card.querySelector('.name')?.textContent || '';
        const url = card.querySelector('.url')?.textContent || '';
        const matches = name.toLowerCase().includes(searchTerm.toLowerCase()) ||
            url.toLowerCase().includes(searchTerm.toLowerCase());

        card.style.display = matches ? 'flex' : 'none';
        if (matches) visibleCount++;
    });

    const existingEmpty = document.getElementById('searchEmptyState');

    // Show empty state if no matches without duplicating placeholder
    if (visibleCount === 0 && cards.length > 0) {
        if (existingEmpty) {
            existingEmpty.textContent = `No servers match "${searchTerm}"`;
        } else {
            const emptyState = document.createElement('div');
            emptyState.className = 'heartbeat-empty';
            emptyState.textContent = `No servers match "${searchTerm}"`;
            emptyState.id = 'searchEmptyState';
            heartbeatList.appendChild(emptyState);
        }
    } else if (existingEmpty) {
        existingEmpty.remove();
    }
}

function buildEndpoint(path) {
    const base = sanitizeBaseUrl(selectedBaseUrl);
    if (!base) {
        return path;
    }
    if (!path.startsWith('/')) {
        return `${base}/${path}`;
    }
    return `${base}${path}`;
}

function getHeartbeatKey(server) {
    if (!server) return '';
    return server.url || server.name || '';
}

function parseTimestamp(value) {
    if (!value) return null;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? null : date;
}

function initializeCharts() {
    const systemCtx = document.getElementById('systemChart').getContext('2d');
    const cpuGradient = systemCtx.createLinearGradient(0, 0, 0, 320);
    cpuGradient.addColorStop(0, 'rgba(56, 189, 248, 0.35)');
    cpuGradient.addColorStop(1, 'rgba(56, 189, 248, 0)');
    const memoryGradient = systemCtx.createLinearGradient(0, 0, 0, 320);
    memoryGradient.addColorStop(0, 'rgba(96, 165, 250, 0.28)');
    memoryGradient.addColorStop(1, 'rgba(96, 165, 250, 0)');

    systemChart = new Chart(systemCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'CPU Usage (%)',
                    data: [],
                    borderColor: 'rgba(56, 189, 248, 0.9)',
                    backgroundColor: cpuGradient,
                    borderWidth: 2.5,
                    fill: true,
                    tension: 0.35,
                    pointRadius: 0
                },
                {
                    label: 'Memory Usage (%)',
                    data: [],
                    borderColor: 'rgba(96, 165, 250, 0.9)',
                    backgroundColor: memoryGradient,
                    borderWidth: 2.5,
                    fill: true,
                    tension: 0.35,
                    pointRadius: 0
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: { mode: 'index', intersect: false },
            plugins: {
                legend: {
                    labels: {
                        color: currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)',
                        usePointStyle: true,
                        boxHeight: 8,
                        padding: 18
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    ticks: {
                        color: 'rgba(148, 163, 184, 0.85)',
                        callback: value => value + '%'
                    },
                    grid: {
                        color: 'rgba(148, 163, 184, 0.18)',
                        borderColor: 'rgba(148, 163, 184, 0.25)'
                    }
                },
                x: {
                    ticks: { color: 'rgba(148, 163, 184, 0.65)', maxTicksLimit: 10 },
                    grid: { display: false }
                }
            }
        }
    });

    const networkCtx = document.getElementById('networkChart').getContext('2d');
    const rxGradient = networkCtx.createLinearGradient(0, 0, 0, 320);
    rxGradient.addColorStop(0, 'rgba(52, 211, 153, 0.32)');
    rxGradient.addColorStop(1, 'rgba(52, 211, 153, 0)');
    const txGradient = networkCtx.createLinearGradient(0, 0, 0, 320);
    txGradient.addColorStop(0, 'rgba(249, 168, 212, 0.3)');
    txGradient.addColorStop(1, 'rgba(249, 168, 212, 0)');

    networkChart = new Chart(networkCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Received (MB/s)',
                    data: [],
                    borderColor: 'rgba(52, 211, 153, 0.9)',
                    backgroundColor: rxGradient,
                    fill: true,
                    tension: 0.3,
                    borderWidth: 2.5,
                    pointRadius: 0
                },
                {
                    label: 'Sent (MB/s)',
                    data: [],
                    borderColor: 'rgba(248, 113, 113, 0.9)',
                    backgroundColor: txGradient,
                    fill: true,
                    tension: 0.3,
                    borderWidth: 2.5,
                    pointRadius: 0
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: { mode: 'index', intersect: false },
            plugins: {
                legend: {
                    labels: {
                        color: currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)',
                        usePointStyle: true,
                        boxHeight: 8,
                        padding: 18
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        color: 'rgba(148, 163, 184, 0.85)',
                        callback: value => value + ' MB'
                    },
                    grid: {
                        color: 'rgba(148, 163, 184, 0.18)',
                        borderColor: 'rgba(148, 163, 184, 0.25)'
                    }
                },
                x: {
                    ticks: { color: 'rgba(148, 163, 184, 0.65)', maxTicksLimit: 10 },
                    grid: { display: false }
                }
            }
        }
    });

    const donutCtx = document.getElementById('usageDonut').getContext('2d');
    usageDonut = new Chart(donutCtx, {
        type: 'doughnut',
        data: {
            labels: ['CPU', 'Memory', 'Disk'],
            datasets: [
                {
                    data: [0, 0, 0],
                    backgroundColor: [
                        'rgba(56, 189, 248, 0.85)',
                        'rgba(96, 165, 250, 0.85)',
                        'rgba(165, 180, 252, 0.85)'
                    ],
                    borderColor: ['rgba(15, 23, 42, 0.6)'],
                    borderWidth: 1,
                    hoverOffset: 6
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '70%',
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        color: currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.85)' : 'rgba(30, 41, 59, 0.9)',
                        usePointStyle: true,
                        padding: 18
                    }
                }
            }
        }
    });
}

// Request Notification Permission
async function requestNotificationPermission() {
    if ('Notification' in window) {
        notificationPermission = await Notification.requestPermission();
    }
}

function calculateOverallDiskUsage(diskSpaces, container) {
    if (!diskSpaces || diskSpaces.length === 0) {
        // Fallback to legacy single disk data
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

    // Calculate weighted average based on total size
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

    // Process disk array - ensure it's an array
    const diskSpaces = Array.isArray(diskArray) ? diskArray : [diskArray].filter(d => d && Object.keys(d).length > 0);

    // For backwards compatibility, calculate overall disk usage
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

    const normalized = {
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
        process: process,
        raw
    };

    return normalized;
}

async function fetchMetrics() {
    try {
        // Only show loading overlay on initial load
        if (isInitialLoad) {
            setLoadingState('initial', true);
        }
        // Don't show reconnecting status for normal requests

        const monitoringUrl = buildEndpoint('/monitoring');
        const filterPayload = pendingFilter;
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
            .map(entry => normalizeMetrics(entry))
            .filter(item => item && typeof item === 'object');

        if (normalizedList.length === 0) {
            throw new Error('No monitoring data available');
        }

        if (filterPayload) {
            historicalSeries = normalizedList;
        } else if (historicalMode) {
            const latest = normalizedList[0];
            const latestTimestamp = latest.timestamp;
            historicalSeries = [latest, ...historicalSeries.filter(item => item.timestamp !== latestTimestamp)];
        }

        const activeSeries = historicalMode && historicalSeries.length > 0
            ? historicalSeries
            : normalizedList;

        const latestNormalized = activeSeries[0];
        const comparisonBaseline = activeSeries[1] || previousMetrics;
        const networkDelta = calculateNetworkDelta(latestNormalized, comparisonBaseline);

        latestNormalized.network_delta = networkDelta;

        updateMetrics(latestNormalized, networkDelta);

        if (historicalMode) {
            renderHistoricalCharts(activeSeries);
        } else {
            updateCharts(latestNormalized, networkDelta);
        }

        updateTrends(latestNormalized, networkDelta, comparisonBaseline);
        updateStatus(true);
        updateConnectionStatus('online');

        // Check thresholds and show alerts
        checkThresholds(latestNormalized);

        let uptimeStats = null;
        if (historicalMode) {
            uptimeStats = calculateHeartbeatUptime(activeSeries);
        }
        updateHeartbeat(latestNormalized.heartbeat || [], uptimeStats);

        const lastUpdated = document.getElementById('lastUpdated');
        if (lastUpdated) {
            lastUpdated.textContent = new Date().toLocaleTimeString();
        }

        previousMetrics = latestNormalized;
        retryCount = 0; // Reset retry count on success

        if (filterPayload) {
            pendingFilter = null;
        }

        // Hide initial loading overlay after first successful load
        if (isInitialLoad) {
            setLoadingState('initial', false);
            isInitialLoad = false;
        }
    } catch (error) {
        console.error('Error fetching metrics:', error);
        retryCount++;

        // Connection status will be set based on retry logic below

        // Hide loading even on error
        if (isInitialLoad) {
            setLoadingState('initial', false);
            isInitialLoad = false;
        }

        if (retryCount <= maxRetries) {
            // Retry silently without showing reconnecting status
        } else {
            updateConnectionStatus('offline', error.message);
            showAlert('error', `Connection failed: ${error.message}`);
            showErrorState(error.message);
        }
    }
}

function updateMetrics(data, networkDelta = { bytes_received: 0, bytes_sent: 0, durationSeconds: null }) {
    setMetricValue('cpu', data.cpu_usage, 1);
    setMetricValue('memory', data.memory?.percentage, 1);
    setMetricValue('diskOverall', data.disk?.percentage, 1);
    setMetricValue('networkRx', bytesToMbPerSecond(networkDelta.bytes_received, networkDelta.durationSeconds), 2);
    setMetricValue('networkTx', bytesToMbPerSecond(networkDelta.bytes_sent, networkDelta.durationSeconds), 2);
    setMetricValue('loadAvg', data.load_average?.one_minute, 2);

    updateProgress('cpuBar', data.cpu_usage);
    updateProgress('memoryBar', data.memory?.percentage);
    updateDiskSpaces(data.disk_spaces, data.disk?.percentage);

    // Update progress bar ARIA attributes
    updateProgressAria('cpuBar', data.cpu_usage);
    updateProgressAria('memoryBar', data.memory?.percentage);
}

function updateProgressAria(barId, value) {
    const progressBar = document.getElementById(barId)?.closest('[role="progressbar"]');
    if (progressBar && value != null) {
        progressBar.setAttribute('aria-valuenow', Math.round(value));
    }
}

function setMetricValue(id, value, digits = 1) {
    const element = document.getElementById(id);
    if (!element) return;
    if (value === undefined || value === null || Number.isNaN(Number(value))) {
        element.textContent = '--';
        return;
    }
    element.textContent = Number(value).toFixed(digits);
}

function updateDiskSpaces(diskSpaces, overallPercentage) {
    const storageGrid = document.getElementById('storageGrid');
    if (!storageGrid) return;

    // Clear existing storage cards
    storageGrid.innerHTML = '';

    if (!diskSpaces || diskSpaces.length === 0) {
        storageGrid.innerHTML = `
            <article class="glass-panel metric-card storage-card">
                <div class="storage-header">
                    <div class="storage-info">
                        <div class="storage-path">No storage data available</div>
                        <div class="storage-device">Please check your system configuration</div>
                    </div>
                </div>
            </article>
        `;
        return;
    }

    // Create storage cards for each disk space
    diskSpaces.forEach(disk => {
        const storageCard = document.createElement('article');
        storageCard.className = 'glass-panel metric-card storage-card';

        const usagePercent = disk.used_pct || 0;
        const totalGB = disk.total_bytes ? (disk.total_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';
        const usedGB = disk.used_bytes ? (disk.used_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';
        const availableGB = disk.available_bytes ? (disk.available_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';

        // Determine color class based on usage
        let badgeClass = '';
        let progressClass = '';
        if (usagePercent >= thresholds.disk.critical) {
            badgeClass = 'critical';
            progressClass = 'critical';
        } else if (usagePercent >= thresholds.disk.warning) {
            badgeClass = 'warning';
            progressClass = 'warning';
        }

        storageCard.innerHTML = `
            <div class="storage-header">
                <div class="storage-info">
                    <div class="storage-path">${escapeHtml(disk.path || '/')}</div>
                    <div class="storage-device">${escapeHtml(disk.device || 'Unknown device')}</div>
                    <div class="storage-filesystem">${escapeHtml(disk.filesystem || 'Unknown')}</div>
                </div>
                <div class="storage-usage-badge ${badgeClass}">
                    <i class="fas fa-hdd"></i>
                    ${usagePercent.toFixed(1)}%
                </div>
            </div>
            <div class="storage-progress">
                <div class="storage-progress-fill ${progressClass}" style="width: ${usagePercent}%"></div>
            </div>
            <div class="storage-stats">
                <div class="storage-stat">
                    <span class="storage-stat-value">${usedGB} GB</span>
                    <span class="storage-stat-label">Used</span>
                </div>
                <div class="storage-stat">
                    <span class="storage-stat-value">${availableGB} GB</span>
                    <span class="storage-stat-label">Free</span>
                </div>
                <div class="storage-stat">
                    <span class="storage-stat-value">${totalGB} GB</span>
                    <span class="storage-stat-label">Total</span>
                </div>
            </div>
        `;

        storageGrid.appendChild(storageCard);
    });
}

function bytesToMb(bytes) {
    if (bytes === undefined || bytes === null) return undefined;
    return bytes / 1024 / 1024;
}

function bytesToMbPerSecond(bytes, seconds = null) {
    if (bytes === undefined || bytes === null) return undefined;
    const intervalSeconds = Number.isFinite(seconds) && seconds > 0
        ? seconds
        : refreshInterval / 1000;
    if (!Number.isFinite(intervalSeconds) || intervalSeconds <= 0) {
        return bytesToMb(bytes);
    }
    return (bytes / 1024 / 1024) / intervalSeconds;
}

function toFiniteNumber(value) {
    if (value === undefined || value === null) return NaN;
    const numeric = Number(value);
    return Number.isFinite(numeric) ? numeric : NaN;
}

function deriveCounterDelta(current, previous, hasPrevious) {
    if (!Number.isFinite(current)) return 0;
    if (!hasPrevious || !Number.isFinite(previous)) return 0;
    const diff = current - previous;
    if (diff < 0) {
        return Math.max(current, 0);
    }
    return diff;
}

function calculateNetworkDelta(current, previous) {
    const hasPrevious = Boolean(previous && previous.network);
    const currentNetwork = current?.network || {};
    const previousNetwork = previous?.network || {};

    const currentRx = toFiniteNumber(currentNetwork.bytes_received);
    const currentTx = toFiniteNumber(currentNetwork.bytes_sent);
    const previousRx = toFiniteNumber(previousNetwork.bytes_received);
    const previousTx = toFiniteNumber(previousNetwork.bytes_sent);

    const currentTime = parseTimestamp(current?.timestamp);
    const previousTime = parseTimestamp(previous?.timestamp);
    let durationSeconds = null;
    if (currentTime && previousTime) {
        const diffMs = currentTime.getTime() - previousTime.getTime();
        if (Number.isFinite(diffMs) && diffMs > 0) {
            durationSeconds = diffMs / 1000;
        }
    }

    return {
        bytes_received: deriveCounterDelta(currentRx, previousRx, hasPrevious),
        bytes_sent: deriveCounterDelta(currentTx, previousTx, hasPrevious),
        durationSeconds
    };
}

function calculateHeartbeatUptime(series) {
    if (!Array.isArray(series) || series.length === 0) return null;

    const statsMap = new Map();

    series.forEach(entry => {
        const heartbeatList = entry?.heartbeat;
        if (!Array.isArray(heartbeatList)) return;

        heartbeatList.forEach(server => {
            const key = getHeartbeatKey(server);
            if (!key) return;

            if (!statsMap.has(key)) {
                statsMap.set(key, {
                    name: server.name || 'Unknown',
                    url: server.url || '',
                    up: 0,
                    total: 0
                });
            }

            const entryStats = statsMap.get(key);
            entryStats.total += 1;
            if ((server.status || '').toLowerCase() === 'up') {
                entryStats.up += 1;
            }
        });
    });

    const result = {};
    statsMap.forEach((value, key) => {
        const uptime = value.total > 0 ? (value.up / value.total) * 100 : null;
        result[key] = {
            name: value.name,
            url: value.url,
            upCount: value.up,
            totalCount: value.total,
            uptime
        };
    });

    return Object.keys(result).length > 0 ? result : null;
}

function updateProgress(id, value) {
    const bar = document.getElementById(id);
    if (!bar) return;
    if (value === undefined || value === null || Number.isNaN(Number(value))) {
        bar.style.width = '0%';
        return;
    }
    const clamped = Math.min(100, Math.max(0, Number(value)));
    bar.style.width = `${clamped}%`;
}

function updateTrends(data, networkDelta = { bytes_received: 0, bytes_sent: 0, durationSeconds: null }, referenceMetrics = previousMetrics) {
    if (!referenceMetrics || referenceMetrics.cpu_usage === undefined || referenceMetrics.cpu_usage === null) return;

    const previousNetworkDelta = referenceMetrics.network_delta || {};

    const trends = [
        { id: 'cpuTrend', current: data.cpu_usage, previous: referenceMetrics.cpu_usage },
        { id: 'memoryTrend', current: data.memory?.percentage, previous: referenceMetrics.memory?.percentage },
        { id: 'diskTrend', current: data.disk?.percentage, previous: referenceMetrics.disk?.percentage },
        {
            id: 'networkRxTrend',
            current: bytesToMbPerSecond(networkDelta.bytes_received, networkDelta.durationSeconds),
            previous: bytesToMbPerSecond(previousNetworkDelta.bytes_received, previousNetworkDelta.durationSeconds)
        },
        {
            id: 'networkTxTrend',
            current: bytesToMbPerSecond(networkDelta.bytes_sent, networkDelta.durationSeconds),
            previous: bytesToMbPerSecond(previousNetworkDelta.bytes_sent, previousNetworkDelta.durationSeconds)
        },
        { id: 'loadTrend', current: data.load_average?.one_minute, previous: referenceMetrics.load_average?.one_minute }
    ];

    trends.forEach(({ id, current, previous }) => {
        const element = document.getElementById(id);
        if (!element) return;

        const currentVal = Number.isFinite(Number(current)) ? Number(current) : 0;
        const previousVal = Number.isFinite(Number(previous)) ? Number(previous) : 0;

        if (currentVal > previousVal) {
            element.innerHTML = '<i class="fas fa-arrow-up"></i>';
        } else if (currentVal < previousVal) {
            element.innerHTML = '<i class="fas fa-arrow-down"></i>';
        } else {
            element.innerHTML = '<i class="fas fa-minus"></i>';
        }
    });
}

function updateCharts(data, networkDelta = { bytes_received: 0, bytes_sent: 0 }) {
    const timestamp = new Date().toLocaleTimeString();
    const maxPoints = 20;

    if (systemChart) {
        systemChart.data.labels.push(timestamp);
        systemChart.data.datasets[0].data.push(data.cpu_usage || 0);
        systemChart.data.datasets[1].data.push(data.memory?.percentage || 0);

        if (systemChart.data.labels.length > maxPoints) {
            systemChart.data.labels.shift();
            systemChart.data.datasets.forEach(dataset => dataset.data.shift());
        }

        systemChart.update('none');
    }

    if (networkChart) {
        networkChart.data.labels.push(timestamp);
        networkChart.data.datasets[0].data.push(bytesToMbPerSecond(networkDelta.bytes_received) || 0);
        networkChart.data.datasets[1].data.push(bytesToMbPerSecond(networkDelta.bytes_sent) || 0);

        if (networkChart.data.labels.length > maxPoints) {
            networkChart.data.labels.shift();
            networkChart.data.datasets.forEach(dataset => dataset.data.shift());
        }

        networkChart.update('none');
    }

    updateUsageDonut(data);
}

function renderHistoricalCharts(series) {
    if (!Array.isArray(series) || series.length === 0) return;
    if (!systemChart || !networkChart) return;

    const chronological = [...series].reverse();

    systemChart.data.labels = [];
    systemChart.data.datasets[0].data = [];
    systemChart.data.datasets[1].data = [];

    networkChart.data.labels = [];
    networkChart.data.datasets[0].data = [];
    networkChart.data.datasets[1].data = [];

    let previous = null;
    chronological.forEach(item => {
        const timestamp = parseTimestamp(item.timestamp);
        const label = timestamp
            ? timestamp.toLocaleString()
            : new Date().toLocaleString();

        systemChart.data.labels.push(label);
        systemChart.data.datasets[0].data.push(item.cpu_usage || 0);
        systemChart.data.datasets[1].data.push(item.memory?.percentage || 0);

        const delta = calculateNetworkDelta(item, previous);
        networkChart.data.labels.push(label);
        networkChart.data.datasets[0].data.push(bytesToMbPerSecond(delta.bytes_received, delta.durationSeconds) || 0);
        networkChart.data.datasets[1].data.push(bytesToMbPerSecond(delta.bytes_sent, delta.durationSeconds) || 0);

        previous = item;
    });

    systemChart.update('default');
    networkChart.update('default');

    updateUsageDonut(series[0]);
}

function resetCharts() {
    if (systemChart) {
        systemChart.data.labels = [];
        systemChart.data.datasets.forEach(dataset => {
            dataset.data = [];
        });
        systemChart.update('none');
    }
    if (networkChart) {
        networkChart.data.labels = [];
        networkChart.data.datasets.forEach(dataset => {
            dataset.data = [];
        });
        networkChart.update('none');
    }

    if (usageDonut) {
        usageDonut.data.datasets[0].data = [0, 0, 0];
        usageDonut.update('none');
    }
}

function updateUsageDonut(data) {
    if (!usageDonut || !data) return;
    usageDonut.data.datasets[0].data = [
        data.cpu_usage || data.cpu?.usage_percent || 0,
        data.memory?.percentage
        ?? data.memory?.used_pct
        ?? data.ram_used_percent
        ?? 0,
        data.disk?.percentage
        ?? data.disk_space?.used_pct
        ?? data.disk_used_percent
        ?? 0
    ];
    usageDonut.update('none');
}

function renderServerButtons() {
    const container = document.getElementById('serverSwitcher');
    if (!container) return;

    container.innerHTML = '';

    const servers = [LOCAL_SERVER_OPTION, ...availableServers];
    const activeAddress = sanitizeBaseUrl(selectedBaseUrl);

    servers.forEach(server => {
        if (!server || !server.name) {
            return;
        }

        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'server-button';
        button.textContent = server.name;

        // Add local or remote class based on server type
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

async function handleServerSelection(server) {
    const address = sanitizeBaseUrl(server.address || '');
    if (address === sanitizeBaseUrl(selectedBaseUrl)) {
        return;
    }

    selectedServer = server.address ? { ...server, address } : null;
    selectedBaseUrl = address;
    pendingFilter = null;
    historicalMode = false;
    historicalSeries = [];
    previousMetrics = null;

    updateStatus(false);
    updateHeartbeat();
    const lastUpdated = document.getElementById('lastUpdated');
    if (lastUpdated) {
        lastUpdated.textContent = '--';
    }

    resetCharts();
    renderServerButtons();

    clearTimeout(refreshTimer);
    await fetchServerConfig(selectedBaseUrl);
    await fetchMetrics();
    scheduleNextFetch();
}

async function fetchServerConfig(baseUrl = '') {
    try {
        const sanitizedBase = sanitizeBaseUrl(baseUrl);
        const endpoint = sanitizedBase ? `${sanitizedBase}/api/v1/server-config` : '/api/v1/server-config';
        const response = await fetch(endpoint, { cache: 'no-store' });
        if (!response.ok) {
            return null;
        }

        const config = await response.json();
        if (!sanitizedBase) {
            serverConfig = config;
            availableServers = Array.isArray(config.servers)
                ? config.servers
                    .filter(server => server && server.name && server.address)
                    .map(server => ({
                        name: server.name,
                        address: sanitizeBaseUrl(server.address)
                    }))
                : [];

            renderServerButtons();
        }

        const candidate = Number(config.refresh_interval_seconds) * 1000;
        if (!Number.isNaN(candidate) && candidate > 0) {
            refreshInterval = candidate;
            updateRefreshDisplay();
        }

        return config;
    } catch (error) {
        const target = baseUrl ? baseUrl : 'local';
        console.debug(`Using existing server config for ${target}:`, error);
        return null;
    }
}

function getFilterInputs() {
    if (!filterElements) return { from: null, to: null };
    return {
        from: filterElements.fromInput ? filterElements.fromInput.value : '',
        to: filterElements.toInput ? filterElements.toInput.value : ''
    };
}

function formatDateInputToISO(value, endOfRange = false) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return '';
    }
    if (endOfRange) {
        const hasSeconds = /:\d{2}$/.test(value);
        if (!hasSeconds) {
            date.setSeconds(59, 999);
        }
    }
    return date.toISOString();
}

async function applyDateFilter() {
    const { from, to } = getFilterInputs();
    if (!from && !to) {
        pendingFilter = null;
        historicalMode = false;
        historicalSeries = [];
        clearTimeout(refreshTimer);
        resetCharts();
        document.querySelectorAll('.range-btn.active').forEach(btn => btn.classList.remove('active'));
        await fetchMetrics();
        scheduleNextFetch();
        return;
    }

    const fromISO = formatDateInputToISO(from);
    const toISO = formatDateInputToISO(to, true);

    if (fromISO && toISO) {
        const fromDate = new Date(fromISO);
        const toDate = new Date(toISO);
        if (fromDate > toDate) {
            console.warn('Invalid range: From must be earlier than To');
            return;
        }
    }

    pendingFilter = {
        from: fromISO || null,
        to: toISO || null
    };
    historicalMode = true;
    historicalSeries = [];
    previousMetrics = null;

    clearTimeout(refreshTimer);
    resetCharts();
    await fetchMetrics();
    scheduleNextFetch();
}

async function clearDateFilter() {
    pendingFilter = null;
    historicalMode = false;
    historicalSeries = [];
    previousMetrics = null;
    if (filterElements) {
        if (filterElements.fromInput) filterElements.fromInput.value = '';
        if (filterElements.toInput) filterElements.toInput.value = '';
    }

    document.querySelectorAll('.range-btn.active').forEach(btn => btn.classList.remove('active'));

    clearTimeout(refreshTimer);
    resetCharts();
    await fetchMetrics();
    scheduleNextFetch();
}

function updateRefreshDisplay() {
    const display = document.getElementById('refreshDisplay');
    if (!display) return;
    display.textContent = `${(refreshInterval / 1000).toFixed(1)}s`;
}

function updateStatus(isOnline) {
    const status = document.getElementById('systemStatus');
    if (!status) return;
    status.classList.toggle('offline', !isOnline);
    status.classList.toggle('online', isOnline);
    const text = status.querySelector('.status-text');
    if (text) {
        text.textContent = isOnline ? 'System Online' : 'Connection Error';
    }
}

function showErrorState() {
    updateStatus(false);
    const lastUpdated = document.getElementById('lastUpdated');
    if (lastUpdated) {
        lastUpdated.textContent = '--';
    }
    updateHeartbeat();
}

function updateHeartbeat(servers = [], uptimeStats = null) {
    const list = document.getElementById('heartbeatList');
    const summary = document.getElementById('heartbeatSummary');
    if (!list) return;

    list.innerHTML = '';

    if (!servers || servers.length === 0) {
        list.innerHTML = '<div class="heartbeat-empty">No heartbeat targets configured</div>';
        if (summary) {
            summary.textContent = '0 online / 0 total';
        }
        return;
    }

    let onlineCount = 0;
    let uptimeAccumulator = 0;
    let uptimeSamples = 0;

    servers.forEach(server => {
        const status = (server.status || '').toLowerCase();
        if (status === 'up') onlineCount += 1;

        const card = document.createElement('div');
        card.className = 'heartbeat-card';

        const responseTime = formatResponseTime(server.response_ms);
        const lastChecked = formatLastChecked(server.last_checked);
        const key = getHeartbeatKey(server);
        const uptimeInfo = uptimeStats && key ? uptimeStats[key] : null;
        const uptimeText = uptimeInfo && Number.isFinite(uptimeInfo.uptime)
            ? `${uptimeInfo.uptime.toFixed(1)}%`
            : null;
        if (uptimeText) {
            uptimeAccumulator += uptimeInfo.uptime;
            uptimeSamples += 1;
        }

        card.innerHTML = `
            <div class="heartbeat-info">
                <div class="name">${escapeHtml(server.name || 'Unnamed Server')}</div>
                <div class="url">${escapeHtml(server.url || '—')}</div>
            </div>
            <span class="server-status ${statusClass(status)}">${statusLabel(status)}</span>
            <div class="heartbeat-detail">
                <span class="label">Latency</span>
                <span class="value">${escapeHtml(responseTime)}</span>
            </div>
            <div class="heartbeat-detail">
                <span class="label">Last Seen</span>
                <span class="value">${escapeHtml(lastChecked)}</span>
            </div>
        `;

        if (uptimeText) {
            const uptimeDetail = document.createElement('div');
            uptimeDetail.className = 'heartbeat-detail';
            uptimeDetail.innerHTML = `
                <span class="label">Uptime</span>
                <span class="value">${escapeHtml(uptimeText)}</span>
            `;
            card.appendChild(uptimeDetail);
        }

        list.appendChild(card);
    });

    if (summary) {
        let summaryText = `${onlineCount} online / ${servers.length} total`;
        if (uptimeSamples > 0) {
            const avgUptime = uptimeAccumulator / uptimeSamples;
            summaryText += ` • Avg ${avgUptime.toFixed(1)}% uptime`;
        }
        summary.textContent = summaryText;
    }
}

function statusClass(status) {
    if (status === 'up') return 'status-up';
    if (status === 'down') return 'status-down';
    return 'status-warning';
}

function statusLabel(status) {
    if (status === 'up') return 'Online';
    if (status === 'down') return 'Offline';
    return status ? status.charAt(0).toUpperCase() + status.slice(1) : 'Unknown';
}

function formatResponseTime(ms) {
    const value = Number(ms);
    if (!Number.isFinite(value) || value < 0) return '—';
    if (value < 1000) return `${Math.round(value)} ms`;
    return `${(value / 1000).toFixed(2)} s`;
}

function formatLastChecked(timestamp) {
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

function escapeHtml(value) {
    if (value === undefined || value === null) return '';
    const content = String(value);
    return content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function scheduleNextFetch() {
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(async () => {
        try {
            await fetchServerConfig(selectedBaseUrl);
            await fetchMetrics();
        } catch (error) {
            console.error('Scheduled fetch failed:', error);
        } finally {
            scheduleNextFetch();
        }
    }, refreshInterval);
}

// Cleanup function to prevent memory leaks
function cleanup() {
    clearTimeout(refreshTimer);
    chartUpdateQueue.length = 0;
    activeAlerts.clear();
    clearAllAlerts();

    // Destroy charts
    if (systemChart) {
        systemChart.destroy();
        systemChart = null;
    }
    if (networkChart) {
        networkChart.destroy();
        networkChart = null;
    }
    if (usageDonut) {
        usageDonut.destroy();
        usageDonut = null;
    }
}

// Handle page visibility changes for performance
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        clearTimeout(refreshTimer);
    } else {
        scheduleNextFetch();
    }
});

// Handle page unload
window.addEventListener('beforeunload', cleanup);

// Handle errors globally
window.addEventListener('error', (event) => {
    console.error('Global error:', event.error);
    showAlert('error', 'An unexpected error occurred');
});

window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
    showAlert('error', 'A network or processing error occurred');
});

async function init() {
    // Initialize theme
    currentTheme = localStorage.getItem('theme') || 'dark';
    document.body.setAttribute('data-theme', currentTheme);
    const themeIcon = document.getElementById('themeIcon');
    themeIcon.className = currentTheme === 'dark' ? 'fas fa-moon' : 'fas fa-sun';

    // Request notification permission
    await requestNotificationPermission();

    initializeCharts();
    renderServerButtons();
    filterElements = {
        fromInput: document.getElementById('filterFrom'),
        toInput: document.getElementById('filterTo'),
        applyButton: document.getElementById('applyFilter'),
        clearButton: document.getElementById('clearFilter')
    };

    // Event listeners for existing functionality
    if (filterElements.applyButton) {
        filterElements.applyButton.addEventListener('click', applyDateFilter);
    }
    if (filterElements.clearButton) {
        filterElements.clearButton.addEventListener('click', clearDateFilter);
    }

    // New event listeners
    document.getElementById('themeToggle')?.addEventListener('click', toggleTheme);
    document.getElementById('exportTrigger')?.addEventListener('click', showExportPanel);
    document.getElementById('exportClose')?.addEventListener('click', hideExportPanel);
    document.getElementById('exportOverlay')?.addEventListener('click', hideExportPanel);
    document.getElementById('exportCSV')?.addEventListener('click', () => exportData('csv'));
    document.getElementById('exportJSON')?.addEventListener('click', () => exportData('json'));

    // Range preset buttons
    document.querySelectorAll('.range-btn').forEach(btn => {
        btn.addEventListener('click', () => applyRangePreset(btn.dataset.range));
    });

    // Heartbeat search
    const searchInput = document.getElementById('heartbeatSearch');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            searchTerm = e.target.value;
            filterHeartbeats(searchTerm);
        });
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey || e.metaKey) {
            switch (e.key) {
                case 'e':
                    e.preventDefault();
                    showExportPanel();
                    break;
                case 'r':
                    e.preventDefault();
                    fetchMetrics();
                    break;
                case 't':
                    e.preventDefault();
                    toggleTheme();
                    break;
            }
        }
        if (e.key === 'Escape') {
            hideExportPanel();
        }
    });

    await fetchServerConfig(selectedBaseUrl);
    await fetchMetrics();
    scheduleNextFetch();
}

function hasRequiredElements() {
    return REQUIRED_ELEMENT_IDS.every((id) => document.getElementById(id));
}

async function attemptBootstrap() {
    if (initialized || initInProgress) {
        return;
    }

    if (!hasRequiredElements()) {
        return;
    }

    initInProgress = true;
    try {
        await init();
        initialized = true;
        if (bootstrapIntervalId) {
            clearInterval(bootstrapIntervalId);
            bootstrapIntervalId = null;
        }
    } catch (error) {
        console.error('Dashboard initialization failed:', error);
    } finally {
        initInProgress = false;
    }
}

function startBootstrapWatchers() {
    if (watchersRegistered) {
        return;
    }
    watchersRegistered = true;

    if (window.htmx && document.body) {
        document.body.addEventListener('htmx:afterSwap', attemptBootstrap);
        document.body.addEventListener('htmx:afterSettle', attemptBootstrap);
    }

    bootstrapIntervalId = setInterval(() => {
        if (initialized) {
            clearInterval(bootstrapIntervalId);
            bootstrapIntervalId = null;
            return;
        }
        attemptBootstrap();
    }, 100);
}

function onDomReady() {
    startBootstrapWatchers();
    attemptBootstrap();
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', onDomReady);
} else {
    onDomReady();
}

window.removeAlert = removeAlert;
