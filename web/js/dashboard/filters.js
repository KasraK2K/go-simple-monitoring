import { resetCharts } from './charts.js';
import { fetchMetrics, scheduleNextFetch } from './data-service.js';
import { state } from './state.js';

export function captureFilterElements() {
  const elements = {
    fromInput: document.getElementById('filterFrom'),
    toInput: document.getElementById('filterTo'),
    applyButton: document.getElementById('applyFilter'),
    clearButton: document.getElementById('clearFilter')
  };
  state.filterElements = elements;
  return elements;
}

export function getFilterInputs() {
  if (!state.filterElements) {
    return { from: null, to: null };
  }
  return {
    from: state.filterElements.fromInput ? state.filterElements.fromInput.value : '',
    to: state.filterElements.toInput ? state.filterElements.toInput.value : ''
  };
}

export function formatDateInputToISO(value, endOfRange = false) {
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

export async function applyDateFilter() {
  const { from, to } = getFilterInputs();
  if (!from && !to) {
    state.autoFilter = null;
    state.pendingFilter = null;
    state.historicalMode = false;
    state.historicalSeries = [];
    await restartMetrics();
    document.querySelectorAll('.range-btn.active').forEach((btn) => btn.classList.remove('active'));
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

  state.pendingFilter = {
    from: fromISO || null,
    to: toISO || null
  };
  state.autoFilter = null;
  state.historicalMode = true;
  state.historicalSeries = [];
  state.previousMetrics = null;

  await restartMetrics();
}

export async function clearDateFilter() {
  state.autoFilter = null;
  state.pendingFilter = null;
  state.historicalMode = false;
  state.historicalSeries = [];
  state.previousMetrics = null;

  if (state.filterElements) {
    if (state.filterElements.fromInput) state.filterElements.fromInput.value = '';
    if (state.filterElements.toInput) state.filterElements.toInput.value = '';
  }

  document.querySelectorAll('.range-btn.active').forEach((btn) => btn.classList.remove('active'));

  await restartMetrics();
}

export async function applyRangePreset(range) {
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

  const formatDate = (date) => date.toISOString().slice(0, 16);

  if (state.filterElements?.fromInput) {
    state.filterElements.fromInput.value = formatDate(fromDate);
  }
  if (state.filterElements?.toInput) {
    state.filterElements.toInput.value = formatDate(now);
  }

  document.querySelectorAll('.range-btn').forEach((btn) => {
    btn.classList.toggle('active', btn.dataset.range === range);
  });

  await applyDateFilter();
}

async function restartMetrics() {
  clearTimeout(state.refreshTimer);
  resetCharts();
  await fetchMetrics();
  scheduleNextFetch();
}
