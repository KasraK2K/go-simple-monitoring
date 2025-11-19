import { state } from "./state.js";
import { updateChartTheme } from "./charts.js";
import { updateCompactView } from "./compact.js";

export function initializeTheme() {
  state.currentTheme = localStorage.getItem("theme") || "dark";
  document.body.setAttribute("data-theme", state.currentTheme);
  const select = document.getElementById("themeSelect");
  if (select) {
    select.value = state.currentTheme;
  }
  try {
    // Ensure compact view visibility is synced on load
    updateCompactView();
  } catch {}
}

function applyTheme(targetTheme, originEl = null) {
  const prefersReducedMotion = window.matchMedia
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
    : false;

  // Prevent concurrent animations
  if (document.body.dataset.animatingTheme === "true") {
    return;
  }

  if (targetTheme === state.currentTheme) {
    return;
  }

  const oldTheme = state.currentTheme;

  // Compute the animation origin at the center of the toggle button
  const rectOld = originEl ? originEl.getBoundingClientRect() : null;
  const x1 = rectOld ? rectOld.left + rectOld.width / 2 : window.innerWidth / 2;
  const y1 = rectOld ? rectOld.top + rectOld.height / 2 : window.innerHeight / 2;

  // If reduced motion or missing API support, do an instant toggle
  const canAnimate =
    typeof document.body.animate === "function" &&
    "clipPath" in document.body.style;
  if (prefersReducedMotion || !canAnimate) {
    state.currentTheme = targetTheme;
    document.body.setAttribute("data-theme", state.currentTheme);
    try {
      localStorage.setItem("theme", state.currentTheme);
    } catch {}
    if (state.systemChart) updateChartTheme(state.systemChart);
    if (state.networkChart) updateChartTheme(state.networkChart);
    if (state.usageDonut) updateChartTheme(state.usageDonut);
    const select = document.getElementById("themeSelect");
    if (select) {
      select.value = state.currentTheme;
    }
    updateCompactView();
    return;
  }

  // Begin animated transition: switch base theme underneath, overlay the old theme and shrink it
  document.body.dataset.animatingTheme = "true";
  document.body.setAttribute("data-theme", targetTheme);

  const overlay = document.createElement("div");
  overlay.className = "theme-transition-overlay";
  overlay.setAttribute("data-theme", oldTheme);
  document.body.appendChild(overlay);

  // Calculate radius to cover the furthest viewport corner from both origins (start and end)
  const vw = Math.max(
    document.documentElement.clientWidth,
    window.innerWidth || 0
  );
  const vh = Math.max(
    document.documentElement.clientHeight,
    window.innerHeight || 0
  );
  const rectNew = originEl ? originEl.getBoundingClientRect() : null;
  // After theme switch, the control may have moved; recompute new origin
  const x2 = rectNew ? rectNew.left + rectNew.width / 2 : window.innerWidth / 2;
  const y2 = rectNew ? rectNew.top + rectNew.height / 2 : window.innerHeight / 2;

  const maxX1 = Math.max(x1, vw - x1);
  const maxY1 = Math.max(y1, vh - y1);
  const r1 = Math.hypot(maxX1, maxY1);
  const maxX2 = Math.max(x2, vw - x2);
  const maxY2 = Math.max(y2, vh - y2);
  const r2 = Math.hypot(maxX2, maxY2);
  const r = Math.max(r1, r2);

  // Ensure initial state applied before animation
  overlay.style.clipPath = `circle(${r}px at ${x1}px ${y1}px)`;
  // Force style flush
  overlay.getBoundingClientRect();

  const duration = 1000; // slower, smoother reveal
  const easing = "cubic-bezier(0.22, 1, 0.36, 1)";

  const animation = overlay.animate(
    [
      { clipPath: `circle(${r}px at ${x1}px ${y1}px)`, opacity: 1 },
      { clipPath: `circle(0px at ${x2}px ${y2}px)`, opacity: 0.9 },
    ],
    { duration, easing, fill: "forwards" }
  );

  const finalize = () => {
    overlay.remove();
    state.currentTheme = targetTheme;
    try {
      localStorage.setItem("theme", state.currentTheme);
    } catch {}
    if (state.systemChart) updateChartTheme(state.systemChart);
    if (state.networkChart) updateChartTheme(state.networkChart);
    if (state.usageDonut) updateChartTheme(state.usageDonut);
    const select = document.getElementById("themeSelect");
    if (select) {
      select.value = state.currentTheme;
    }
    delete document.body.dataset.animatingTheme;
    updateCompactView();
  };

  animation.addEventListener("finish", finalize, { once: true });
  animation.addEventListener("cancel", finalize, { once: true });
}

export function toggleTheme(event) {
  const targetTheme = state.currentTheme === "dark" ? "light" : "dark";
  const origin = document.getElementById("themeSelect");
  applyTheme(targetTheme, origin || null);
}

export function handleThemeSelectChange(event) {
  const target = event?.target || null;
  const value = target ? String(target.value) : "";
  if (value === "dark" || value === "light" || value === "compact") {
    applyTheme(value, target);
  }
}

export async function requestNotificationPermission() {
  if ("Notification" in window) {
    state.notificationPermission = await Notification.requestPermission();
  }
}
