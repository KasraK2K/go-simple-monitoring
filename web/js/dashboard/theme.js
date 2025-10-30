import { state } from './state.js';
import { updateChartTheme } from './charts.js';

export function initializeTheme() {
  state.currentTheme = localStorage.getItem('theme') || 'dark';
  document.body.setAttribute('data-theme', state.currentTheme);
  const themeIcon = document.getElementById('themeIcon');
  if (themeIcon) {
    themeIcon.className = state.currentTheme === 'dark' ? 'fas fa-moon' : 'fas fa-sun';
  }
}

export function toggleTheme() {
  state.currentTheme = state.currentTheme === 'dark' ? 'light' : 'dark';
  document.body.setAttribute('data-theme', state.currentTheme);
  const themeIcon = document.getElementById('themeIcon');
  if (themeIcon) {
    themeIcon.className = state.currentTheme === 'dark' ? 'fas fa-moon' : 'fas fa-sun';
  }
  localStorage.setItem('theme', state.currentTheme);

  if (state.systemChart) {
    updateChartTheme(state.systemChart);
  }
  if (state.networkChart) {
    updateChartTheme(state.networkChart);
  }
  if (state.usageDonut) {
    updateChartTheme(state.usageDonut);
  }
}

export async function requestNotificationPermission() {
  if ('Notification' in window) {
    state.notificationPermission = await Notification.requestPermission();
  }
}
