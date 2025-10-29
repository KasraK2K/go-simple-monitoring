'use client';

import { Card, CardBody } from '@chakra-ui/react';
import { Line } from 'react-chartjs-2';
import { useMemo } from 'react';
import { MetricPoint } from '@/lib/types';
import '@/lib/chart';

interface NetworkTrendsChartProps {
  series: MetricPoint[];
  colorMode: 'light' | 'dark';
}

export function NetworkTrendsChart({ series, colorMode }: NetworkTrendsChartProps) {
  const chartData = useMemo(() => {
    const labels = series.map(point => new Date(point.timestamp).toLocaleTimeString());
    return {
      labels,
      datasets: [
        {
          label: 'Received (MB/s)',
          data: series.map(point => point.networkRx ?? null),
          borderColor: 'rgba(52, 211, 153, 0.9)',
          backgroundColor: 'rgba(52, 211, 153, 0.2)',
          fill: true,
          tension: 0.35,
          pointRadius: 0,
          borderWidth: 2.5
        },
        {
          label: 'Sent (MB/s)',
          data: series.map(point => point.networkTx ?? null),
          borderColor: 'rgba(248, 113, 113, 0.9)',
          backgroundColor: 'rgba(248, 113, 113, 0.2)',
          fill: true,
          tension: 0.35,
          pointRadius: 0,
          borderWidth: 2.5
        }
      ]
    };
  }, [series]);

  const textColor = colorMode === 'dark' ? 'rgba(226,232,240,0.85)' : 'rgba(30,41,59,0.85)';
  const gridColor = colorMode === 'dark' ? 'rgba(148,163,184,0.16)' : 'rgba(148,163,184,0.22)';

  return (
    <Card variant="outline" borderRadius="xl" shadow="sm" height="100%">
      <CardBody p={{ base: 4, md: 6 }}>
        <Line
          data={chartData}
          options={{
            responsive: true,
            maintainAspectRatio: false,
          interaction: { mode: 'index', intersect: false },
          scales: {
            y: {
              beginAtZero: true,
              ticks: {
                color: textColor,
                callback: value => `${value} MB`
              },
              grid: {
                color: gridColor
              }
            },
            x: {
              ticks: {
                color: textColor,
                maxTicksLimit: 10
              },
              grid: { display: false }
            }
          },
          plugins: {
            legend: {
              labels: {
                color: textColor,
                usePointStyle: true
              }
            }
          }
        }}
        />
      </CardBody>
    </Card>
  );
}
