import { state } from './state.js';
import { escapeHtml } from './utils.js';

export function showAlert(type, message, duration = 5000) {
  const alertsContainer = document.getElementById('alertsPanel');
  if (!alertsContainer) {
    return null;
  }

  // Suppress all alerts if muted
  if (state.muteAlerts) {
    return null;
  }

  const alertId = `alert_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  const alertEl = document.createElement('div');
  alertEl.className = `alert ${type}`;
  alertEl.setAttribute('data-alert-id', alertId);
  
  // Get appropriate icon for alert type
  const icons = {
    success: '✓',
    error: '✕',
    warning: '⚠',
    info: 'i'
  };
  
  // Get appropriate title for alert type
  const titles = {
    success: 'Success',
    error: 'Error',
    warning: 'Warning', 
    info: 'Information'
  };
  
  alertEl.innerHTML = `
    <div class="alert-content">
      <div class="alert-icon">${icons[type] || 'i'}</div>
      <div class="alert-body">
        <div class="alert-title">${titles[type] || 'Alert'}</div>
        <div class="alert-message">${escapeHtml(message)}</div>
      </div>
      <button class="alert-close" aria-label="Close alert">×</button>
    </div>
  `;

  const closeButton = alertEl.querySelector('.alert-close');
  if (closeButton) {
    closeButton.addEventListener('click', () => removeAlert(alertId));
  }

  alertsContainer.appendChild(alertEl);
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

  let alertEl;
  if (typeof storedValue === 'string') {
    alertEl = state.activeAlerts.get(storedValue);
    if (alertEl) {
      state.activeAlerts.delete(storedValue);
    }
  } else if (storedValue && typeof storedValue.remove === 'function') {
    alertEl = storedValue;
  }

  if (alertEl && typeof alertEl.remove === 'function') {
    // Add removing class for animation
    alertEl.classList.add('removing');
    
    // Wait for animation to complete before removing
    setTimeout(() => {
      if (alertEl.parentNode) {
        alertEl.remove();
      }
    }, 300);
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

function determinePhase(value, thresholds) {
  if (value == null || !Number.isFinite(value) || !thresholds) return 'ok';
  if (value >= thresholds.critical) return 'critical';
  if (value >= thresholds.warning) return 'warning';
  return 'ok';
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
    const phase = determinePhase(check.value, check.thresholds);
    const prevPhase = state.lastThresholdPhase.get(alertKey) || 'ok';

    if (phase === prevPhase) {
      // No phase change; do nothing to avoid spamming alerts.
      return;
    }

    // Remove existing alert UI if any for this metric
    if (state.activeAlerts.has(alertKey)) {
      removeAlert(alertKey);
    }

    // On phase change, raise appropriate alert
    if (phase === 'critical') {
      const alertId = showAlert('error', `${check.name} usage critical: ${check.value.toFixed(1)}${check.unit}`, 0);
      if (alertId) state.activeAlerts.set(alertKey, alertId);
    } else if (phase === 'warning') {
      const alertId = showAlert('warning', `${check.name} usage high: ${check.value.toFixed(1)}${check.unit}`, 10000);
      if (alertId) state.activeAlerts.set(alertKey, alertId);
    } else if (phase === 'ok') {
      // Optionally show a short recovery notice when exiting warning/critical
      if (prevPhase === 'critical' || prevPhase === 'warning') {
        showAlert('success', `${check.name} back to normal: ${check.value.toFixed(1)}${check.unit}`, 6000);
      }
      // No persistent alert to track when OK
    }

    state.lastThresholdPhase.set(alertKey, phase);
  });
}
