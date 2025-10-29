'use client';

import { Card, CardBody } from '@chakra-ui/react';
import { Doughnut } from 'react-chartjs-2';
import { useMemo } from 'react';
import { NormalizedMetrics } from '@/lib/types';
import '@/lib/chart';

interface ResourceDonutProps {
  metrics?: NormalizedMetrics | null;
  colorMode: 'light' | 'dark';
}

export function ResourceDonut({ metrics, colorMode }: ResourceDonutProps) {
  const chartData = useMemo(() => {
    const cpu = metrics?.cpu_usage ?? 0;
    const memory = metrics?.memory?.percentage ?? 0;
    const disk = metrics?.disk?.percentage ?? 0;

    return {
      labels: ['CPU', 'Memory', 'Disk'],
      datasets: [
        {
          data: [cpu, memory, disk],
          backgroundColor: [
            'rgba(56, 189, 248, 0.85)',
            'rgba(129, 140, 248, 0.85)',
            'rgba(139, 92, 246, 0.85)'
          ],
          borderWidth: 1,
          borderColor: colorMode === 'dark' ? 'rgba(15, 23, 42, 0.65)' : 'rgba(226, 232, 240, 0.85)'
        }
      ]
    };
  }, [metrics, colorMode]);

  const textColor = colorMode === 'dark' ? 'rgba(226,232,240,0.85)' : 'rgba(30,41,59,0.8)';

  return (
    <Card variant="outline" borderRadius="xl" shadow="sm" height="100%">
      <CardBody p={{ base: 4, md: 6 }}>
        <Doughnut
          data={chartData}
          options={{
            cutout: '70%',
            plugins: {
              legend: {
                position: 'bottom',
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
