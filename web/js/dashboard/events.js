import { exportData, hideExportPanel, showExportPanel } from './exporter.js';
import { applyDateFilter, applyRangePreset, clearDateFilter } from './filters.js';
import { filterHeartbeats } from './heartbeat.js';
import { fetchMetrics, handleServerSelection } from './data-service.js';
import { LOCAL_SERVER_OPTION } from './constants.js';
import { state } from './state.js';
import { toggleTheme, handleThemeSelectChange } from './theme.js';

const HERO_COLLAPSE_STORAGE_KEY = 'dashboardHeroCollapsed';

export function registerEventHandlers() {
  if (state.eventsRegistered) {
    return;
  }
  state.eventsRegistered = true;

  document.getElementById('themeSelect')?.addEventListener('change', handleThemeSelectChange);

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

  const heroToggle = document.getElementById('heroToggle');
  const heroSection = document.getElementById('heroSection');
  const heroCopy = heroSection?.querySelector('.hero-copy');
  const prefersReducedMotion = window.matchMedia
    ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
    : false;

  const animateHeroCopy = (firstRect) => {
    if (!heroCopy || prefersReducedMotion || !firstRect) {
      return;
    }

    const lastRect = heroCopy.getBoundingClientRect();
    const deltaX = firstRect.left - lastRect.left;
    const deltaY = firstRect.top - lastRect.top;

    if (Math.abs(deltaX) < 1 && Math.abs(deltaY) < 1) {
      return;
    }

    heroCopy.animate(
      [
        {
          transform: `translate(${deltaX}px, ${deltaY}px)`
        },
        {
          transform: 'translate(0, 0)'
        }
      ],
      {
        duration: 500,
        easing: 'cubic-bezier(0.4, 0, 0.2, 1)'
      }
    );
  };

  if (heroToggle && heroSection) {
    const applyHeroCollapseState = (collapsed, animate = false) => {
      let firstRect = null;
      if (animate && heroCopy && !prefersReducedMotion) {
        firstRect = heroCopy.getBoundingClientRect();
      }

      heroSection.classList.toggle('hero-collapsed', collapsed);
      heroToggle.setAttribute('aria-expanded', String(!collapsed));
      heroToggle.setAttribute(
        'aria-label',
        collapsed ? 'Expand hero panel' : 'Collapse hero panel'
      );

      if (animate) {
        animateHeroCopy(firstRect);
      }
    };

    let initialCollapsed = true;
    try {
      const storedValue = localStorage.getItem(HERO_COLLAPSE_STORAGE_KEY);
      if (storedValue !== null) {
        initialCollapsed = storedValue === 'true';
      }
    } catch (error) {
      console.debug('Unable to read hero collapse preference:', error);
    }
    applyHeroCollapseState(initialCollapsed);

    heroToggle.addEventListener('click', () => {
      const collapsed = !heroSection.classList.contains('hero-collapsed');
      applyHeroCollapseState(collapsed, true);
      try {
        localStorage.setItem(HERO_COLLAPSE_STORAGE_KEY, String(collapsed));
      } catch (error) {
        console.debug('Unable to persist hero collapse preference:', error);
      }
    });
  }

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
