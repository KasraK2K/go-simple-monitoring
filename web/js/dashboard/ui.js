import { updateHeartbeat } from './heartbeat.js';
import { state } from './state.js';

export function setLoadingState(section, loading) {
  if (section === 'initial') {
    const loadingEl = document.getElementById('initialLoading');
    if (loadingEl) {
      loadingEl.style.display = loading ? 'flex' : 'none';
    }
    state.isLoading = loading;
  }
}

export function updateConnectionStatus(status, message = '') {
  const statusEl = document.getElementById('connectionStatus');
  const textEl = document.getElementById('connectionText');
  if (!statusEl || !textEl) return;

  state.connectionState = status;
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
    default:
      textEl.textContent = message || status;
      statusEl.classList.remove('hidden');
  }
}

export function updateStatus(isOnline) {
  const status = document.getElementById('systemStatus');
  if (!status) return;
  status.classList.toggle('offline', !isOnline);
  status.classList.toggle('online', isOnline);
  const text = status.querySelector('.status-text');
  if (text) {
    text.textContent = isOnline ? 'System Online' : 'Connection Error';
  }
}

export function showErrorState() {
  updateStatus(false);
  const lastUpdated = document.getElementById('lastUpdated');
  if (lastUpdated) {
    lastUpdated.textContent = '--';
  }
  updateHeartbeat();
}

export function updateRefreshDisplay() {
  const display = document.getElementById('refreshDisplay');
  if (!display) return;
  display.textContent = `${(state.refreshInterval / 1000).toFixed(1)}s`;
}
