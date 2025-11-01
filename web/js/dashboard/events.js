import { exportData, hideExportPanel, showExportPanel } from './exporter.js';
import { applyDateFilter, applyRangePreset, clearDateFilter } from './filters.js';
import { filterHeartbeats } from './heartbeat.js';
import { fetchMetrics, handleServerSelection } from './data-service.js';
import { LOCAL_SERVER_OPTION } from './constants.js';
import { state } from './state.js';
import { toggleTheme } from './theme.js';

export function registerEventHandlers() {
  if (state.eventsRegistered) {
    return;
  }
  state.eventsRegistered = true;

  document.getElementById('themeToggle')?.addEventListener('click', toggleTheme);

  document.getElementById('exportTrigger')?.addEventListener('click', showExportPanel);
  document.getElementById('exportClose')?.addEventListener('click', hideExportPanel);
  document.getElementById('exportOverlay')?.addEventListener('click', hideExportPanel);
  document.getElementById('exportCSV')?.addEventListener('click', () => exportData('csv'));
  document.getElementById('exportJSON')?.addEventListener('click', () => exportData('json'));

  document.querySelectorAll('.range-btn').forEach((btn) => {
    btn.addEventListener('click', () => applyRangePreset(btn.dataset.range));
  });

  if (state.filterElements?.applyButton) {
    state.filterElements.applyButton.addEventListener('click', (event) => {
      event.preventDefault();
      applyDateFilter();
    });
  }

  if (state.filterElements?.clearButton) {
    state.filterElements.clearButton.addEventListener('click', (event) => {
      event.preventDefault();
      clearDateFilter();
    });
  }

  const searchInput = document.getElementById('heartbeatSearch');
  if (searchInput) {
    searchInput.addEventListener('input', (event) => {
      filterHeartbeats(event.target.value);
    });
  }

  document.getElementById('remoteContextReset')?.addEventListener('click', () => {
    handleServerSelection(LOCAL_SERVER_OPTION);
  });

  const keydownHandler = (event) => {
    if (event.ctrlKey || event.metaKey) {
      switch (event.key.toLowerCase()) {
        case 'e':
          event.preventDefault();
          showExportPanel();
          break;
        case 'r':
          event.preventDefault();
          fetchMetrics();
          break;
        case 't':
          event.preventDefault();
          toggleTheme();
          break;
        default:
          break;
      }
    }

    if (event.key === 'Escape') {
      hideExportPanel();
    }
  };

  document.addEventListener('keydown', keydownHandler);
}
