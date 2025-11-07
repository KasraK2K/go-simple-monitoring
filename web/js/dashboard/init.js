import { removeAlert } from './alerts.js';
import { initializeCharts } from './charts.js';
import { applyAutoFilterUI, fetchMetrics, fetchServerConfig, renderServerButtons, scheduleNextFetch } from './data-service.js';
import { registerEventHandlers } from './events.js';
import { captureFilterElements } from './filters.js';
import { registerLifecycleHandlers } from './lifecycle.js';
import { initLayoutDragAndDrop } from './layout.js';
import { state } from './state.js';
import { initializeTheme, requestNotificationPermission } from './theme.js';
import { updateRefreshDisplay, updateRemoteContext } from './ui.js';
import { buildFilterFromRange, isValidRangePreset } from './ranges.js';

export async function initDashboard() {
  initializeTheme();
  await requestNotificationPermission();
  initializeCharts();

  const elements = captureFilterElements();
  state.filterElements = elements;

  renderServerButtons();
  const defaultRangePreset = detectDefaultRangePreset();
  state.defaultRangePreset = defaultRangePreset;
  state.autoFilter = defaultRangePreset ? buildFilterFromRange(defaultRangePreset) : null;
  applyAutoFilterUI(state.autoFilter, defaultRangePreset || null);
  updateRemoteContext();
  registerEventHandlers();
  initLayoutDragAndDrop();
  registerLifecycleHandlers();

  window.removeAlert = removeAlert;

  await fetchServerConfig("");
  updateRefreshDisplay();
  await fetchMetrics();
  scheduleNextFetch();
}

function detectDefaultRangePreset() {
  const container = document.querySelector('.range-presets');
  if (!container || !container.dataset) {
    return '';
  }
  const value = (container.dataset.defaultRange || '').trim().toLowerCase();
  if (!value) {
    return '';
  }
  return isValidRangePreset(value) ? value : '';
}
