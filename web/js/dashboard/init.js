import { removeAlert } from './alerts.js';
import { initializeCharts } from './charts.js';
import { fetchMetrics, fetchServerConfig, renderServerButtons, scheduleNextFetch } from './data-service.js';
import { registerEventHandlers } from './events.js';
import { captureFilterElements } from './filters.js';
import { registerLifecycleHandlers } from './lifecycle.js';
import { state } from './state.js';
import { initializeTheme, requestNotificationPermission } from './theme.js';
import { updateRefreshDisplay } from './ui.js';

export async function initDashboard() {
  initializeTheme();
  await requestNotificationPermission();
  initializeCharts();

  const elements = captureFilterElements();
  state.filterElements = elements;

  renderServerButtons();
  registerEventHandlers();
  registerLifecycleHandlers();

  window.removeAlert = removeAlert;

  await fetchServerConfig(state.selectedBaseUrl);
  updateRefreshDisplay();
  await fetchMetrics();
  scheduleNextFetch();
}
