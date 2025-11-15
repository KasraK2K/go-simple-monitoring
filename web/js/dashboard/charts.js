import { state } from './state.js';
import { bytesToMbPerSecond, parseTimestamp } from './utils.js';
import { calculateNetworkDelta } from './network.js';

export function initializeCharts() {
  const systemCanvas = document.getElementById('systemChart');
  const networkCanvas = document.getElementById('networkChart');
  const usageCanvas = document.getElementById('usageDonut');

  if (!systemCanvas || !networkCanvas || !usageCanvas || typeof Chart === 'undefined') {
    return;
  }

  const systemCtx = systemCanvas.getContext('2d');
  const cpuGradient = systemCtx.createLinearGradient(0, 0, 0, 320);
  cpuGradient.addColorStop(0, 'rgba(56, 189, 248, 0.35)');
  cpuGradient.addColorStop(1, 'rgba(56, 189, 248, 0)');
  const memoryGradient = systemCtx.createLinearGradient(0, 0, 0, 320);
  memoryGradient.addColorStop(0, 'rgba(96, 165, 250, 0.28)');
  memoryGradient.addColorStop(1, 'rgba(96, 165, 250, 0)');

  state.systemChart = new Chart(systemCtx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        {
          label: 'CPU Usage (%)',
          data: [],
          borderColor: 'rgba(56, 189, 248, 0.9)',
          backgroundColor: cpuGradient,
          borderWidth: 2.5,
          fill: true,
          tension: 0.35,
          pointRadius: 0
        },
        {
          label: 'Memory Usage (%)',
          data: [],
          borderColor: 'rgba(96, 165, 250, 0.9)',
          backgroundColor: memoryGradient,
          borderWidth: 2.5,
          fill: true,
          tension: 0.35,
          pointRadius: 0
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          labels: {
            color: state.currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)',
            usePointStyle: true,
            boxHeight: 8,
            padding: 18
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          max: 100,
          ticks: {
            color: 'rgba(148, 163, 184, 0.85)',
            callback: (value) => `${value}%`
          },
          grid: {
            color: 'rgba(148, 163, 184, 0.18)',
            borderColor: 'rgba(148, 163, 184, 0.25)'
          }
        },
        x: {
          ticks: { color: 'rgba(148, 163, 184, 0.65)', maxTicksLimit: 10 },
          grid: { display: false }
        }
      }
    }
  });

  const networkCtx = networkCanvas.getContext('2d');
  const rxGradient = networkCtx.createLinearGradient(0, 0, 0, 320);
  rxGradient.addColorStop(0, 'rgba(52, 211, 153, 0.32)');
  rxGradient.addColorStop(1, 'rgba(52, 211, 153, 0)');
  const txGradient = networkCtx.createLinearGradient(0, 0, 0, 320);
  txGradient.addColorStop(0, 'rgba(249, 168, 212, 0.3)');
  txGradient.addColorStop(1, 'rgba(249, 168, 212, 0)');

  state.networkChart = new Chart(networkCtx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        {
          label: 'Received (MB/s)',
          data: [],
          borderColor: 'rgba(52, 211, 153, 0.9)',
          backgroundColor: rxGradient,
          fill: true,
          tension: 0.3,
          borderWidth: 2.5,
          pointRadius: 0
        },
        {
          label: 'Sent (MB/s)',
          data: [],
          borderColor: 'rgba(248, 113, 113, 0.9)',
          backgroundColor: txGradient,
          fill: true,
          tension: 0.3,
          borderWidth: 2.5,
          pointRadius: 0
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          labels: {
            color: state.currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)',
            usePointStyle: true,
            boxHeight: 8,
            padding: 18
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          ticks: {
            color: 'rgba(148, 163, 184, 0.85)',
            callback: (value) => `${value} MB`
          },
          grid: {
            color: 'rgba(148, 163, 184, 0.18)',
            borderColor: 'rgba(148, 163, 184, 0.25)'
          }
        },
        x: {
          ticks: { color: 'rgba(148, 163, 184, 0.65)', maxTicksLimit: 10 },
          grid: { display: false }
        }
      }
    }
  });

  const donutCtx = usageCanvas.getContext('2d');
  state.usageDonut = new Chart(donutCtx, {
    type: 'doughnut',
    data: {
      labels: ['CPU', 'Memory', 'Disk'],
      datasets: [
        {
          data: [0, 0, 0],
          backgroundColor: [
            'rgba(56, 189, 248, 0.85)',
            'rgba(96, 165, 250, 0.85)',
            'rgba(165, 180, 252, 0.85)'
          ],
          borderColor: ['rgba(15, 23, 42, 0.6)'],
          borderWidth: 1,
          hoverOffset: 6
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '70%',
      plugins: {
        legend: {
          position: 'bottom',
          labels: {
            color: state.currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.85)' : 'rgba(30, 41, 59, 0.9)',
            usePointStyle: true,
            padding: 18
          }
        }
      }
    }
  });
}

export function updateChartTheme(chart) {
  const textColor = state.currentTheme === 'dark' ? 'rgba(226, 232, 240, 0.9)' : 'rgba(30, 41, 59, 0.9)';
  const gridColor = state.currentTheme === 'dark' ? 'rgba(148, 163, 184, 0.18)' : 'rgba(100, 116, 139, 0.2)';
  const borderColor = state.currentTheme === 'dark' ? 'rgba(148, 163, 184, 0.25)' : 'rgba(100, 116, 139, 0.3)';

  if (chart.options.plugins?.legend?.labels) {
    chart.options.plugins.legend.labels.color = textColor;
  }
  if (chart.options.scales?.y?.ticks) {
    chart.options.scales.y.ticks.color = textColor;
  }
  if (chart.options.scales?.x?.ticks) {
    chart.options.scales.x.ticks.color = textColor;
  }
  if (chart.options.scales?.y?.grid) {
    chart.options.scales.y.grid.color = gridColor;
    chart.options.scales.y.grid.borderColor = borderColor;
  }

  chart.update('none');
}

export function updateCharts(data, networkDelta = { bytes_received: 0, bytes_sent: 0 }) {
  const timestamp = new Date().toLocaleTimeString();
  const maxPoints = 20;

  if (state.systemChart) {
    state.systemChart.data.labels.push(timestamp);
    state.systemChart.data.datasets[0].data.push(data.cpu_usage || 0);
    state.systemChart.data.datasets[1].data.push(data.memory?.percentage || 0);

    if (state.systemChart.data.labels.length > maxPoints) {
      state.systemChart.data.labels.shift();
      state.systemChart.data.datasets.forEach((dataset) => dataset.data.shift());
    }

    state.systemChart.update('none');
  }

  if (state.networkChart) {
    state.networkChart.data.labels.push(timestamp);
    state.networkChart.data.datasets[0].data.push(bytesToMbPerSecond(networkDelta.bytes_received, networkDelta.durationSeconds) || 0);
    state.networkChart.data.datasets[1].data.push(bytesToMbPerSecond(networkDelta.bytes_sent, networkDelta.durationSeconds) || 0);

    if (state.networkChart.data.labels.length > maxPoints) {
      state.networkChart.data.labels.shift();
      state.networkChart.data.datasets.forEach((dataset) => dataset.data.shift());
    }

    state.networkChart.update('none');
  }

  updateUsageDonut(data);
}

export function renderHistoricalCharts(series) {
  if (!state.systemChart || !state.networkChart) return;
  
  // Handle empty series by clearing the charts
  if (!Array.isArray(series) || series.length === 0) {
    state.systemChart.data.labels = [];
    state.systemChart.data.datasets[0].data = [];
    state.systemChart.data.datasets[1].data = [];
    
    state.networkChart.data.labels = [];
    state.networkChart.data.datasets[0].data = [];
    state.networkChart.data.datasets[1].data = [];
    
    // Clear usage donut as well
    if (state.usageDonut) {
      state.usageDonut.data.datasets[0].data = [0, 0, 0];
      state.usageDonut.update('none');
    }
    
    state.systemChart.update();
    state.networkChart.update();
    return;
  }

  const chronological = [...series].reverse();

  state.systemChart.data.labels = [];
  state.systemChart.data.datasets[0].data = [];
  state.systemChart.data.datasets[1].data = [];

  state.networkChart.data.labels = [];
  state.networkChart.data.datasets[0].data = [];
  state.networkChart.data.datasets[1].data = [];

  let previous = null;
  chronological.forEach((item) => {
    const timestamp = parseTimestamp(item.timestamp);
    const label = timestamp ? timestamp.toLocaleString() : new Date().toLocaleString();

    state.systemChart.data.labels.push(label);
    state.systemChart.data.datasets[0].data.push(item.cpu_usage || 0);
    state.systemChart.data.datasets[1].data.push(item.memory?.percentage || 0);

    const delta = calculateNetworkDelta(item, previous);
    state.networkChart.data.labels.push(label);
    state.networkChart.data.datasets[0].data.push(bytesToMbPerSecond(delta.bytes_received, delta.durationSeconds) || 0);
    state.networkChart.data.datasets[1].data.push(bytesToMbPerSecond(delta.bytes_sent, delta.durationSeconds) || 0);

    previous = item;
  });

  state.systemChart.update('default');
  state.networkChart.update('default');

  updateUsageDonut(series[0]);
}

export function resetCharts() {
  if (state.systemChart) {
    state.systemChart.data.labels = [];
    state.systemChart.data.datasets.forEach((dataset) => {
      dataset.data = [];
    });
    state.systemChart.update('none');
  }

  if (state.networkChart) {
    state.networkChart.data.labels = [];
    state.networkChart.data.datasets.forEach((dataset) => {
      dataset.data = [];
    });
    state.networkChart.update('none');
  }

  if (state.usageDonut) {
    state.usageDonut.data.datasets[0].data = [0, 0, 0];
    state.usageDonut.update('none');
  }
}

export function updateUsageDonut(data) {
  if (!state.usageDonut || !data) return;
  state.usageDonut.data.datasets[0].data = [
    data.cpu_usage || data.cpu?.usage_percent || 0,
    data.memory?.percentage
      ?? data.memory?.used_pct
      ?? data.ram_used_percent
      ?? 0,
    data.disk?.percentage
      ?? data.disk_space?.used_pct
      ?? data.disk_used_percent
      ?? 0
  ];
  state.usageDonut.update('none');
}
