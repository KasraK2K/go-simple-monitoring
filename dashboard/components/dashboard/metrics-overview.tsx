'use client';

import { Grid, GridItem, Progress, Text, VStack } from '@chakra-ui/react';
import { FiCpu, FiHardDrive, FiActivity } from 'react-icons/fi';
import { MdOutlineMemory } from 'react-icons/md';
import { StatCard } from './stat-card';
import { formatPercent } from '@/lib/format';
import { NormalizedMetrics, NetworkDelta } from '@/lib/types';
import { TrendDirection } from './trend-indicator';

interface MetricsOverviewProps {
  metrics?: NormalizedMetrics | null;
  networkDelta?: NetworkDelta | null;
}

export function MetricsOverview({ metrics, networkDelta }: MetricsOverviewProps) {
  const cpu = formatPercent(metrics?.cpu_usage);
  const memory = formatPercent(metrics?.memory?.percentage);
  const disk = formatPercent(metrics?.disk?.percentage);
  const load = metrics?.load_average?.one_minute ?? null;

  const rxRate = networkDelta?.bytesReceived != null && networkDelta.durationSeconds
    ? networkDelta.bytesReceived / 1024 / 1024 / Math.max(networkDelta.durationSeconds, 1)
    : null;
  const txRate = networkDelta?.bytesSent != null && networkDelta.durationSeconds
    ? networkDelta.bytesSent / 1024 / 1024 / Math.max(networkDelta.durationSeconds, 1)
    : null;

  const networkRx = rxRate != null ? `${rxRate.toFixed(2)} MB/s` : '--';
  const networkTx = txRate != null ? `${txRate.toFixed(2)} MB/s` : '--';

  const loadHelper = load != null && Number.isFinite(load)
    ? `Load avg (1m): ${load.toFixed(2)}`
    : 'Load avg (1m): --';

  // Mock trend data (in production, this would come from historical metrics)
  const getTrend = (value?: number): { direction: TrendDirection; value?: number; tooltip?: string } => {
    if (!value) return { direction: 'neutral' };
    
    // Generate mock trend based on current value
    const mockChange = ((value * 0.1) - 5) + (Math.random() * 6 - 3); // Simulate trend
    
    if (Math.abs(mockChange) < 0.5) {
      return { direction: 'neutral' };
    }
    
    return {
      direction: mockChange > 0 ? 'up' : 'down',
      value: mockChange,
      tooltip: `${mockChange > 0 ? '+' : ''}${mockChange.toFixed(1)}% from previous period`
    };
  };

  return (
    <Grid 
      templateColumns={{ 
        base: 'repeat(1, 1fr)', 
        md: 'repeat(2, minmax(0, 1fr))', 
        xl: 'repeat(4, minmax(0, 1fr))' 
      }} 
      gap={6}
      className="fade-in-up"
    >
      <GridItem>
        <StatCard 
          label="CPU Usage" 
          value={cpu} 
          helper="Real-time utilization" 
          icon={FiCpu} 
          accentColor="brand.500"
          variant="gradient"
          trend={getTrend(metrics?.cpu_usage)}
        >
          <Progress 
            value={metrics?.cpu_usage ?? 0} 
            rounded="full" 
            size="md" 
            bg="rgba(255, 255, 255, 0.2)"
            sx={{
              '& > div': {
                bg: 'rgba(255, 255, 255, 0.8)',
              }
            }}
          />
        </StatCard>
      </GridItem>
      
      <GridItem>
        <StatCard 
          label="Memory" 
          value={memory} 
          helper="Active usage vs total" 
          icon={MdOutlineMemory} 
          accentColor="purple.500"
          variant="default"
          trend={getTrend(metrics?.memory?.percentage)}
        >
          <Progress 
            value={metrics?.memory?.percentage ?? 0} 
            rounded="full" 
            size="md" 
            colorScheme="purple" 
            bg="purple.50"
            _dark={{ bg: 'purple.900' }}
          />
        </StatCard>
      </GridItem>
      
      <GridItem>
        <StatCard 
          label="Storage" 
          value={disk} 
          helper="Aggregate disk usage" 
          icon={FiHardDrive} 
          accentColor="teal.500"
          variant="default"
          trend={getTrend(metrics?.disk?.percentage)}
        >
          <Progress 
            value={metrics?.disk?.percentage ?? 0} 
            rounded="full" 
            size="md" 
            colorScheme="teal" 
            bg="teal.50"
            _dark={{ bg: 'teal.900' }}
          />
        </StatCard>
      </GridItem>
      
      <GridItem>
        <StatCard 
          label="Network I/O" 
          value={networkRx} 
          helper={`Transmit: ${networkTx}`} 
          icon={FiActivity} 
          accentColor="orange.500"
          variant="glass"
          trend={getTrend(rxRate || 0)}
        >
          <VStack align="flex-start" spacing={3} w="full">
            <VStack align="flex-start" spacing={1} w="full">
              <Text fontSize="xs" fontWeight="600" opacity={0.8}>
                Download
              </Text>
              <Progress 
                value={Math.min(100, (rxRate ?? 0) * 10)} 
                size="sm" 
                colorScheme="orange" 
                bg="rgba(255, 255, 255, 0.2)"
                rounded="full" 
                w="full"
                sx={{
                  '& > div': {
                    bg: 'orange.400',
                  }
                }}
              />
            </VStack>
            
            <VStack align="flex-start" spacing={1} w="full">
              <Text fontSize="xs" fontWeight="600" opacity={0.8}>
                Upload
              </Text>
              <Progress 
                value={Math.min(100, (txRate ?? 0) * 10)} 
                size="sm" 
                colorScheme="yellow" 
                bg="rgba(255, 255, 255, 0.2)"
                rounded="full" 
                w="full"
                sx={{
                  '& > div': {
                    bg: 'yellow.400',
                  }
                }}
              />
            </VStack>
          </VStack>
        </StatCard>
      </GridItem>
    </Grid>
  );
}
