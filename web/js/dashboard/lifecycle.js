import { clearAllAlerts, showAlert } from './alerts.js';
import { scheduleNextFetch } from './data-service.js';
import { state } from './state.js';

export function registerLifecycleHandlers() {
  if (state.lifecycleRegistered) {
    return;
  }
  state.lifecycleRegistered = true;

  document.addEventListener('visibilitychange', handleVisibilityChange);
  window.addEventListener('beforeunload', cleanup);
  window.addEventListener('error', handleGlobalError);
  window.addEventListener('unhandledrejection', handleUnhandledRejection);
}

function handleVisibilityChange() {
  if (document.hidden) {
    clearTimeout(state.refreshTimer);
  } else {
    scheduleNextFetch();
  }
}

function handleGlobalError(event) {
  console.error('Global error:', event.error);
  showAlert('error', 'An unexpected error occurred');
}

function handleUnhandledRejection(event) {
  console.error('Unhandled promise rejection:', event.reason);
  showAlert('error', 'A network or processing error occurred');
}

export function cleanup() {
  clearTimeout(state.refreshTimer);
  state.chartUpdateQueue.length = 0;
  clearAllAlerts();

  if (state.systemChart) {
    state.systemChart.destroy();
    state.systemChart = null;
  }
  if (state.networkChart) {
    state.networkChart.destroy();
    state.networkChart = null;
  }
  if (state.usageDonut) {
    state.usageDonut.destroy();
    state.usageDonut = null;
  }
}
