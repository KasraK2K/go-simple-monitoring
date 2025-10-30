import { state } from './state.js';
import { escapeHtml } from './utils.js';

export function showAlert(type, message, duration = 5000) {
  const alertsPanel = document.getElementById('alertsPanel');
  if (!alertsPanel) {
    return null;
  }

  const alertId = `alert_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  const alertEl = document.createElement('div');
  alertEl.className = `alert-item ${type}`;
  alertEl.innerHTML = `
    <i class="fas fa-exclamation-triangle"></i>
    <span>${escapeHtml(message)}</span>
    <span class="alert-close" data-alert-id="${alertId}"><i class="fas fa-times"></i></span>
  `;

  const closeButton = alertEl.querySelector('.alert-close');
  if (closeButton) {
    closeButton.addEventListener('click', () => removeAlert(alertId));
  }

  alertsPanel.appendChild(alertEl);
  state.activeAlerts.set(alertId, alertEl);

  if (duration > 0) {
    setTimeout(() => removeAlert(alertId), duration);
  }

  if (state.notificationPermission === 'granted' && type === 'error') {
    new Notification('System Monitoring Alert', { body: message });
  }

  return alertId;
}

export function removeAlert(alertId) {
  const storedValue = state.activeAlerts.get(alertId);
  if (!storedValue) {
    return;
  }

  if (typeof storedValue === 'string') {
    const alertEl = state.activeAlerts.get(storedValue);
    if (alertEl && typeof alertEl.remove === 'function') {
      alertEl.remove();
      state.activeAlerts.delete(storedValue);
    }
  } else if (storedValue && typeof storedValue.remove === 'function') {
    storedValue.remove();
  }

  state.activeAlerts.delete(alertId);
}

export function clearAllAlerts() {
  state.activeAlerts.forEach((storedValue) => {
    if (storedValue && typeof storedValue.remove === 'function') {
      storedValue.remove();
    }
  });
  state.activeAlerts.clear();
}

export function checkThresholds(data) {
  if (!data) return;

  const checks = [
    { name: 'CPU', value: data.cpu_usage, thresholds: state.thresholds.cpu, unit: '%' },
    { name: 'Memory', value: data.memory?.percentage, thresholds: state.thresholds.memory, unit: '%' }
  ];

  if (Array.isArray(data.disk_spaces)) {
    data.disk_spaces.forEach((disk, index) => {
      if (disk?.used_pct != null && Number.isFinite(disk.used_pct)) {
        checks.push({
          name: `Disk ${disk.path || `#${index + 1}`}`,
          value: disk.used_pct,
          thresholds: state.thresholds.disk,
          unit: '%'
        });
      }
    });
  } else if (data.disk?.percentage != null) {
    checks.push({ name: 'Disk', value: data.disk.percentage, thresholds: state.thresholds.disk, unit: '%' });
  }

  checks.forEach((check) => {
    if (check.value == null || !Number.isFinite(check.value)) return;

    const alertKey = `threshold_${check.name.toLowerCase().replace(/[^a-z0-9]/g, '_')}`;

    if (check.value >= check.thresholds.critical) {
      if (!state.activeAlerts.has(alertKey)) {
        const alertId = showAlert('error', `${check.name} usage critical: ${check.value.toFixed(1)}${check.unit}`, 0);
        if (alertId) {
          state.activeAlerts.set(alertKey, alertId);
        }
      }
    } else if (check.value >= check.thresholds.warning) {
      if (!state.activeAlerts.has(alertKey)) {
        const alertId = showAlert('warning', `${check.name} usage high: ${check.value.toFixed(1)}${check.unit}`, 10000);
        if (alertId) {
          state.activeAlerts.set(alertKey, alertId);
        }
      }
    } else if (state.activeAlerts.has(alertKey)) {
      removeAlert(alertKey);
    }
  });
}
