import { state } from "./state.js";
import { updateChartTheme } from "./charts.js";

export function initializeTheme() {
  state.currentTheme = localStorage.getItem("theme") || "dark";
  document.body.setAttribute("data-theme", state.currentTheme);
  const themeIcon = document.getElementById("themeIcon");
  if (themeIcon) {
    themeIcon.className =
      state.currentTheme === "dark" ? "fas fa-moon" : "fas fa-sun";
  }
}

export function toggleTheme(event) {
  const prefersReducedMotion = window.matchMedia
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
    : false;

  // Prevent concurrent animations
  if (document.body.dataset.animatingTheme === "true") {
    return;
  }

  const oldTheme = state.currentTheme;
  const targetTheme = oldTheme === "dark" ? "light" : "dark";

  // Compute the animation origin at the center of the toggle button
  const toggleEl = document.getElementById("themeToggle");
  const rect = toggleEl ? toggleEl.getBoundingClientRect() : null;
  const x = rect ? rect.left + rect.width / 2 : window.innerWidth / 2;
  const y = rect ? rect.top + rect.height / 2 : window.innerHeight / 2;

  // If reduced motion or missing API support, do an instant toggle
  const canAnimate =
    typeof document.body.animate === "function" &&
    "clipPath" in document.body.style;
  if (prefersReducedMotion || !canAnimate) {
    state.currentTheme = targetTheme;
    document.body.setAttribute("data-theme", state.currentTheme);
    const themeIcon = document.getElementById("themeIcon");
    if (themeIcon) {
      themeIcon.className =
        state.currentTheme === "dark" ? "fas fa-moon" : "fas fa-sun";
    }
    try {
      localStorage.setItem("theme", state.currentTheme);
    } catch {}
    if (state.systemChart) updateChartTheme(state.systemChart);
    if (state.networkChart) updateChartTheme(state.networkChart);
    if (state.usageDonut) updateChartTheme(state.usageDonut);
    return;
  }

  // Begin animated transition: switch base theme underneath, overlay the old theme and shrink it
  document.body.dataset.animatingTheme = "true";
  document.body.setAttribute("data-theme", targetTheme);

  const overlay = document.createElement("div");
  overlay.className = "theme-transition-overlay";
  overlay.setAttribute("data-theme", oldTheme);
  document.body.appendChild(overlay);

  // Calculate radius to cover the furthest viewport corner from the origin
  const vw = Math.max(
    document.documentElement.clientWidth,
    window.innerWidth || 0
  );
  const vh = Math.max(
    document.documentElement.clientHeight,
    window.innerHeight || 0
  );
  const maxX = Math.max(x, vw - x);
  const maxY = Math.max(y, vh - y);
  const r = Math.hypot(maxX, maxY);

  // Ensure initial state applied before animation
  overlay.style.clipPath = `circle(${r}px at ${x}px ${y}px)`;
  // Force style flush
  overlay.getBoundingClientRect();

  const duration = 1000; // slower, smoother reveal
  const easing = "cubic-bezier(0.22, 1, 0.36, 1)";

  const animation = overlay.animate(
    [
      { clipPath: `circle(${r}px at ${x}px ${y}px)`, opacity: 1 },
      { clipPath: `circle(0px at ${x}px ${y}px)`, opacity: 0.9 },
    ],
    { duration, easing, fill: "forwards" }
  );

  const finalize = () => {
    overlay.remove();
    state.currentTheme = targetTheme;
    const themeIcon = document.getElementById("themeIcon");
    if (themeIcon) {
      themeIcon.className =
        state.currentTheme === "dark" ? "fas fa-moon" : "fas fa-sun";
    }
    try {
      localStorage.setItem("theme", state.currentTheme);
    } catch {}
    if (state.systemChart) updateChartTheme(state.systemChart);
    if (state.networkChart) updateChartTheme(state.networkChart);
    if (state.usageDonut) updateChartTheme(state.usageDonut);
    delete document.body.dataset.animatingTheme;
  };

  animation.addEventListener("finish", finalize, { once: true });
  animation.addEventListener("cancel", finalize, { once: true });
}

export async function requestNotificationPermission() {
  if ("Notification" in window) {
    state.notificationPermission = await Notification.requestPermission();
  }
}
