import { state } from './state.js';
import { bytesToMbPerSecond, escapeHtml } from './utils.js';

const PROGRESS_NEAR_DANGER_FRACTION = 0.35;

function getProgressSeverity(value, type) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || !type || !state.thresholds[type]) {
    return '';
  }
  const { warning, critical } = state.thresholds[type];
  if (numeric >= critical) {
    return 'danger';
  }
  const buffer = Math.max(2, (critical - warning) * PROGRESS_NEAR_DANGER_FRACTION);
  if (numeric >= critical - buffer) {
    return 'caution';
  }
  if (numeric >= warning) {
    return 'warning';
  }
  return 'safe';
}

export function updateMetrics(data, networkDelta = { bytes_received: 0, bytes_sent: 0, durationSeconds: null }) {
  setMetricValue('cpu', data.cpu_usage, 1);
  setMetricValue('memory', data.memory?.percentage, 1);
  setMetricValue('diskOverall', data.disk?.percentage, 1);
  setMetricValue('networkRx', bytesToMbPerSecond(networkDelta.bytes_received, networkDelta.durationSeconds), 2);
  setMetricValue('networkTx', bytesToMbPerSecond(networkDelta.bytes_sent, networkDelta.durationSeconds), 2);
  setMetricValue('loadAvg', data.load_average?.one_minute, 2);

  updateProgress('cpuBar', data.cpu_usage, 'cpu');
  updateProgress('memoryBar', data.memory?.percentage, 'memory');
  updateDiskSpaces(data.disk_spaces, data.disk?.percentage);

  updateProgressAria('cpuBar', data.cpu_usage);
  updateProgressAria('memoryBar', data.memory?.percentage);
}

export function updateProgressAria(barId, value) {
  const progressBar = document.getElementById(barId)?.closest('[role="progressbar"]');
  if (progressBar && value != null) {
    progressBar.setAttribute('aria-valuenow', Math.round(value));
  }
}

export function setMetricValue(id, value, digits = 1) {
  const element = document.getElementById(id);
  if (!element) return;
  if (value === undefined || value === null || Number.isNaN(Number(value))) {
    element.textContent = '--';
    return;
  }
  element.textContent = Number(value).toFixed(digits);
}

export function updateDiskSpaces(diskSpaces, overallPercentage) {
  const storageGrid = document.getElementById('storageGrid');
  if (!storageGrid) return;

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

  diskSpaces.forEach((disk) => {
    const storageCard = document.createElement('article');
    storageCard.className = 'glass-panel metric-card storage-card';

    const usagePercent = disk.used_pct || 0;
    const totalGB = disk.total_bytes ? (disk.total_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';
    const usedGB = disk.used_bytes ? (disk.used_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';
    const availableGB = disk.available_bytes ? (disk.available_bytes / 1024 / 1024 / 1024).toFixed(1) : '0';

    let badgeClass = '';
    let progressClass = '';
    if (usagePercent >= state.thresholds.disk.critical) {
      badgeClass = 'critical';
      progressClass = 'critical';
    } else if (usagePercent >= state.thresholds.disk.warning) {
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

export function updateProgress(id, value, severityType = null) {
  const bar = document.getElementById(id);
  if (!bar) return;
  if (value === undefined || value === null || Number.isNaN(Number(value))) {
    bar.style.width = '0%';
    bar.classList.remove(
      'progress-fill--safe',
      'progress-fill--warning',
      'progress-fill--caution',
      'progress-fill--danger'
    );
    return;
  }
  const clamped = Math.min(100, Math.max(0, Number(value)));
  bar.style.width = `${clamped}%`;
  if (severityType) {
    const severity = getProgressSeverity(clamped, severityType);
    bar.classList.remove(
      'progress-fill--safe',
      'progress-fill--warning',
      'progress-fill--caution',
      'progress-fill--danger'
    );
    if (severity) {
      bar.classList.add(`progress-fill--${severity}`);
    }
  }
}

export function updateTrends(data, networkDelta = { bytes_received: 0, bytes_sent: 0, durationSeconds: null }, referenceMetrics = state.previousMetrics) {
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
