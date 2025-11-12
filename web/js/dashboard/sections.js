const SECTION_COLLAPSE_STORAGE_KEY = 'dashboardSectionCollapseState';
let cachedCollapseState = null;
const sectionCache = new Map();

function loadStoredCollapseState() {
  if (cachedCollapseState) {
    return cachedCollapseState;
  }

  try {
    const raw = localStorage.getItem(SECTION_COLLAPSE_STORAGE_KEY);
    if (!raw) {
      cachedCollapseState = {};
      return cachedCollapseState;
    }
    const parsed = JSON.parse(raw);
    cachedCollapseState = parsed && typeof parsed === 'object' ? parsed : {};
  } catch (error) {
    console.debug('Unable to read section collapse state:', error);
    cachedCollapseState = {};
  }
  return cachedCollapseState;
}

function persistCollapseState(sectionId, collapsed) {
  const state = { ...loadStoredCollapseState() };
  if (collapsed) {
    state[sectionId] = true;
  } else {
    delete state[sectionId];
  }
  cachedCollapseState = state;
  try {
    localStorage.setItem(SECTION_COLLAPSE_STORAGE_KEY, JSON.stringify(state));
  } catch (error) {
    console.debug('Unable to persist section collapse state:', error);
  }
}

function updateToggleAccessibility(toggle, label, collapsed) {
  const action = collapsed ? 'Expand' : 'Collapse';
  const message = `${action} ${label} section`;
  toggle.setAttribute('aria-expanded', String(!collapsed));
  toggle.setAttribute('aria-label', message);
  const srLabel = toggle.querySelector('[data-section-toggle-label]');
  if (srLabel) {
    srLabel.textContent = message;
  }
}

function animateContentHeight(content, collapsed) {
  const transitionEndHandler = (event) => {
    if (event.propertyName !== 'height') {
      return;
    }
    content.removeEventListener('transitionend', transitionEndHandler);
    if (!collapsed) {
      content.style.height = 'auto';
    }
  };

  content.addEventListener('transitionend', transitionEndHandler);

  if (collapsed) {
    const startHeight = content.scrollHeight;
    content.style.height = `${startHeight}px`;
    // Force layout so height is applied before collapsing.
    content.getBoundingClientRect();
    requestAnimationFrame(() => {
      content.style.height = '0px';
    });
    return;
  }

  const targetHeight = content.scrollHeight;
  content.style.height = '0px';
  content.getBoundingClientRect();
  requestAnimationFrame(() => {
    content.style.height = `${targetHeight}px`;
  });
}

function setHeightWithoutAnimation(content, collapsed) {
  const previousTransition = content.style.transition;
  content.style.transition = 'none';
  content.style.height = collapsed ? '0px' : 'auto';
  content.getBoundingClientRect();
  content.style.transition = previousTransition;
}

function setSectionState({ section, toggle, content, label, collapsed, animate }) {
  section.classList.toggle('is-collapsed', collapsed);
  updateToggleAccessibility(toggle, label, collapsed);

  if (!content) {
    return;
  }

  if (!animate) {
    setHeightWithoutAnimation(content, collapsed);
    return;
  }

  animateContentHeight(content, collapsed);
}

export function initSectionCollapsibles() {
  const sections = document.querySelectorAll('[data-dashboard-section][data-section-id]');
  if (!sections.length) {
    return;
  }

  const collapseState = loadStoredCollapseState();
  const prefersReducedMotion = window.matchMedia
    ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
    : false;

  sections.forEach((section) => {
    const sectionId = section.dataset.sectionId;
    if (!sectionId) {
      return;
    }

    const toggle = section.querySelector('[data-section-collapse]');
    const content = section.querySelector('[data-section-content]');
    if (!toggle || !content) {
      return;
    }

    const label = toggle.dataset.sectionLabel || sectionId;
    const collapsed = Boolean(collapseState[sectionId]);

    setSectionState({ section, toggle, content, label, collapsed, animate: false });

    toggle.addEventListener('click', () => {
      const nextCollapsed = !section.classList.contains('is-collapsed');
      setSectionState({
        section,
        toggle,
        content,
        label,
        collapsed: nextCollapsed,
        animate: !prefersReducedMotion,
      });
      persistCollapseState(sectionId, nextCollapsed);
    });
  });
}

function findSectionElement(sectionId) {
  if (!sectionId) {
    return null;
  }
  if (sectionCache.has(sectionId)) {
    const cached = sectionCache.get(sectionId);
    if (cached?.isConnected) {
      return cached;
    }
    sectionCache.delete(sectionId);
  }
  const element = document.querySelector(`[data-dashboard-section][data-section-id="${sectionId}"]`);
  if (element) {
    sectionCache.set(sectionId, element);
  }
  return element;
}

export function setSectionVisibility(sectionId, isVisible) {
  const section = findSectionElement(sectionId);
  if (!section) {
    return;
  }
  const shouldHide = !isVisible;
  section.classList.toggle('is-hidden', shouldHide);
  section.setAttribute('aria-hidden', shouldHide ? 'true' : 'false');
}
