const SECTION_STORAGE_KEY = 'dashboardSectionOrder';
let draggingSection = null;
let layoutContainer = null;

function getSections() {
  return Array.from(
    document.querySelectorAll('[data-dashboard-section][data-section-id]')
  );
}

function loadStoredOrder(defaultOrder) {
  try {
    const raw = localStorage.getItem(SECTION_STORAGE_KEY);
    if (!raw) {
      return defaultOrder;
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return defaultOrder;
    }
    const filtered = parsed.filter((id) => defaultOrder.includes(id));
    const missing = defaultOrder.filter((id) => !filtered.includes(id));
    return [...filtered, ...missing];
  } catch (error) {
    console.debug('Unable to read dashboard layout preference:', error);
    return defaultOrder;
  }
}

function saveOrder() {
  if (!layoutContainer) return;
  const order = getSections().map((section) => section.dataset.sectionId);
  try {
    localStorage.setItem(SECTION_STORAGE_KEY, JSON.stringify(order));
  } catch (error) {
    console.debug('Unable to persist dashboard layout preference:', error);
  }
}

function applyOrder(order) {
  if (!layoutContainer) return;
  const lookup = new Map(
    getSections().map((section) => [section.dataset.sectionId, section])
  );
  order.forEach((id) => {
    const section = lookup.get(id);
    if (section) {
      layoutContainer.appendChild(section);
    }
  });
}

function handleDragStart(event) {
  const handle = event.currentTarget;
  const section = handle.closest('[data-dashboard-section]');
  if (!section || !event.dataTransfer) {
    return;
  }
  draggingSection = section;
  section.classList.add('is-dragging');
  event.dataTransfer.effectAllowed = 'move';
  event.dataTransfer.setData('text/plain', section.dataset.sectionId || '');
  if (event.dataTransfer.setDragImage) {
    event.dataTransfer.setDragImage(section, section.offsetWidth / 2, 30);
  }
}

function handleDragEnd() {
  if (!draggingSection) {
    return;
  }
  draggingSection.classList.remove('is-dragging');
  draggingSection = null;
  saveOrder();
}

function handleContainerDragOver(event) {
  if (!draggingSection || !layoutContainer) {
    return;
  }
  event.preventDefault();
  const eventTarget = event.target instanceof Element ? event.target : null;
  const target = eventTarget?.closest('[data-dashboard-section]');
  if (!target) {
    layoutContainer.appendChild(draggingSection);
    return;
  }
  if (target === draggingSection) {
    return;
  }
  const rect = target.getBoundingClientRect();
  const shouldInsertBefore = event.clientY < rect.top + rect.height / 2;
  if (shouldInsertBefore) {
    layoutContainer.insertBefore(draggingSection, target);
  } else {
    layoutContainer.insertBefore(draggingSection, target.nextSibling);
  }
}

function handleContainerDrop(event) {
  if (!draggingSection) {
    return;
  }
  event.preventDefault();
  saveOrder();
}

export function initLayoutDragAndDrop() {
  layoutContainer = document.querySelector('[data-dashboard-layout]');
  if (!layoutContainer) {
    return;
  }
  const sections = getSections();
  if (sections.length === 0) {
    return;
  }

  const defaultOrder = sections.map((section) => section.dataset.sectionId);
  const storedOrder = loadStoredOrder(defaultOrder);
  applyOrder(storedOrder);

  getSections().forEach((section) => {
    const handle = section.querySelector('[data-section-handle]');
    if (!handle) {
      return;
    }
    handle.addEventListener('dragstart', handleDragStart);
    handle.addEventListener('dragend', handleDragEnd);
  });

  layoutContainer.addEventListener('dragover', handleContainerDragOver);
  layoutContainer.addEventListener('drop', handleContainerDrop);
}
