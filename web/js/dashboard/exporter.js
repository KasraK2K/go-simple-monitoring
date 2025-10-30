import { showAlert } from './alerts.js';
import { state } from './state.js';
import { bytesToMb } from './utils.js';

export function showExportPanel() {
  document.getElementById('exportOverlay')?.classList.add('visible');
  document.getElementById('exportPanel')?.classList.add('visible');
}

export function hideExportPanel() {
  document.getElementById('exportOverlay')?.classList.remove('visible');
  document.getElementById('exportPanel')?.classList.remove('visible');
}

export function exportData(format) {
  try {
    let data;
    let filename;
    let mimeType;
    const timestamp = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-');

    if (format === 'csv') {
      data = generateCSV();
      filename = `monitoring-data-${timestamp}.csv`;
      mimeType = 'text/csv';
    } else if (format === 'json') {
      data = JSON.stringify({
        exported_at: new Date().toISOString(),
        server: state.selectedServer?.name || 'Local',
        historical_mode: state.historicalMode,
        data: state.historicalMode ? state.historicalSeries : [state.previousMetrics]
      }, null, 2);
      filename = `monitoring-data-${timestamp}.json`;
      mimeType = 'application/json';
    } else {
      throw new Error('Unsupported export format');
    }

    const blob = new Blob([data], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);

    showAlert('success', `Data exported as ${format.toUpperCase()}`);
    hideExportPanel();
  } catch (error) {
    showAlert('error', `Export failed: ${error.message}`);
  }
}

export function generateCSV() {
  const series = state.historicalMode ? state.historicalSeries : [state.previousMetrics];
  if (!series || series.length === 0) return 'No data available';

  const headers = ['timestamp', 'cpu_usage', 'memory_percentage', 'disk_percentage', 'network_rx_mb', 'network_tx_mb', 'load_average'];
  const rows = [headers.join(',')];

  series.forEach((item) => {
    if (!item) return;
    const row = [
      item.timestamp || '',
      item.cpu_usage || '',
      item.memory?.percentage || '',
      item.disk?.percentage || '',
      bytesToMb(item.network_delta?.bytes_received) || '',
      bytesToMb(item.network_delta?.bytes_sent) || '',
      item.load_average?.one_minute || ''
    ];
    rows.push(row.join(','));
  });

  return rows.join('\n');
}
