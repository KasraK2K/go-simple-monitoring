import { REQUIRED_ELEMENT_IDS } from './constants.js';
import { initDashboard } from './init.js';
import { state } from './state.js';

function hasRequiredElements() {
  return REQUIRED_ELEMENT_IDS.every((id) => document.getElementById(id));
}

async function attemptBootstrap() {
  if (state.initialized || state.initInProgress) {
    return;
  }

  if (!hasRequiredElements()) {
    return;
  }

  state.initInProgress = true;
  try {
    await initDashboard();
    state.initialized = true;
    if (state.bootstrapIntervalId) {
      clearInterval(state.bootstrapIntervalId);
      state.bootstrapIntervalId = null;
    }
  } catch (error) {
    console.error('Dashboard initialization failed:', error);
  } finally {
    state.initInProgress = false;
  }
}

function startBootstrapWatchers() {
  if (state.watchersRegistered) {
    return;
  }
  state.watchersRegistered = true;

  if (window.htmx && document.body) {
    document.body.addEventListener('htmx:afterSwap', attemptBootstrap);
    document.body.addEventListener('htmx:afterSettle', attemptBootstrap);
  }

  state.bootstrapIntervalId = setInterval(() => {
    if (state.initialized) {
      clearInterval(state.bootstrapIntervalId);
      state.bootstrapIntervalId = null;
      return;
    }
    attemptBootstrap();
  }, 100);
}

export function bootstrapDashboard() {
  startBootstrapWatchers();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', attemptBootstrap);
  } else {
    attemptBootstrap();
  }
}
