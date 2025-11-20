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
import { initSectionCollapsibles } from './sections.js';

export async function initDashboard() {
  initializeTheme();
  // Initialize alerts mute state early to ensure no alerts during startup if muted
  try {
    state.muteAlerts = localStorage.getItem('dashboardMuteAlerts') === 'true';
  } catch {}
  await requestNotificationPermission();
  initializeCharts();

  const elements = captureFilterElements();
  state.filterElements = elements;

  renderServerButtons();
  const isCompactTheme = document.body.getAttribute('data-theme') === 'compact';
  const defaultRangePreset = detectDefaultRangePreset();
  state.defaultRangePreset = isCompactTheme ? '' : defaultRangePreset;
  state.autoFilter = isCompactTheme ? null : (defaultRangePreset ? buildFilterFromRange(defaultRangePreset) : null);
  applyAutoFilterUI(state.autoFilter, isCompactTheme ? null : (defaultRangePreset || null));
  updateRemoteContext();
  registerEventHandlers();
  initLayoutDragAndDrop();
  initSectionCollapsibles();
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
