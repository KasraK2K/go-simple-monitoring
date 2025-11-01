import { state } from './state.js';
import {
  escapeHtml,
  formatLastChecked,
  formatResponseTime,
  getHeartbeatKey,
  statusClass,
  statusLabel
} from './utils.js';

export function updateHeartbeat(servers = [], uptimeStats = null) {
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

  servers.forEach((server) => {
    const status = (server.status || '').toLowerCase();
    if (status === 'up') onlineCount += 1;

    const statusLabelText = statusLabel(status);

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
      <span class="server-status ${statusClass(status)}"><span class="status-dot"></span><span class="status-text">${escapeHtml(statusLabelText)}</span></span>
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

  filterHeartbeats(state.searchTerm);
}

export function filterHeartbeats(searchTerm = '') {
  state.searchTerm = searchTerm;
  const heartbeatList = document.getElementById('heartbeatList');
  if (!heartbeatList) return;

  const cards = heartbeatList.querySelectorAll('.heartbeat-card');
  let visibleCount = 0;

  cards.forEach((card) => {
    const name = card.querySelector('.name')?.textContent || '';
    const url = card.querySelector('.url')?.textContent || '';
    const matches = name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      url.toLowerCase().includes(searchTerm.toLowerCase());

    card.style.display = matches ? 'flex' : 'none';
    if (matches) visibleCount += 1;
  });

  const existingEmpty = document.getElementById('searchEmptyState');

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

export function calculateHeartbeatUptime(series) {
  if (!Array.isArray(series) || series.length === 0) return null;

  const statsMap = new Map();

  series.forEach((entry) => {
    const heartbeatList = entry?.heartbeat;
    if (!Array.isArray(heartbeatList)) return;

    heartbeatList.forEach((server) => {
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
