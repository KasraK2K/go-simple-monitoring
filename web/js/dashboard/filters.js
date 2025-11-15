import { resetCharts } from './charts.js';
import { fetchMetrics, scheduleNextFetch } from './data-service.js';
import { state } from './state.js';
import { buildFilterFromRange, isValidRangePreset } from './ranges.js';

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

  // IMPORTANT: datetime-local values are parsed as local time by the browser.
  // new Date(value) already represents the correct absolute instant in UTC.
  // Do NOT apply timezoneOffset math here, or you will double-shift.
  if (endOfRange) {
    const hasSeconds = /:\d{2}$/.test(value);
    if (!hasSeconds) {
      // Set to end-of-minute to make the upper bound inclusive
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
  if (!isValidRangePreset(range)) {
    console.warn('Invalid range preset:', range);
    return;
  }

  const filter = buildFilterFromRange(range);
  if (!filter) {
    return;
  }

  // Format date for datetime-local input (YYYY-MM-DDTHH:mm)
  // Convert UTC timestamps back to local display for the input fields
  const formatDate = (date) => {
    // Convert UTC Date -> local wall time string for datetime-local input
    // Use subtraction because getTimezoneOffset is minutes to add to local to get UTC
    const localDate = new Date(date.getTime() - (date.getTimezoneOffset() * 60000));
    return localDate.toISOString().slice(0, 16);
  };
  const fromDate = new Date(filter.from);
  const toDate = new Date(filter.to);

  if (state.filterElements?.fromInput) {
    state.filterElements.fromInput.value = formatDate(fromDate);
  }
  if (state.filterElements?.toInput) {
    state.filterElements.toInput.value = formatDate(toDate);
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
