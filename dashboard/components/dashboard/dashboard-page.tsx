'use client';

import {
  Box,
  Container,
  Flex,
  Grid,
  GridItem,
  SimpleGrid,
  Stack,
  useColorMode
} from '@chakra-ui/react';
import { DashboardHeader } from './dashboard-header';
import { StatusBanner } from './status-banner';
import { DateRangeFilterControl } from './date-range-filter';
import { MetricsOverview } from './metrics-overview';
import { MetricTrendsChart } from './metric-trends-chart';
import { NetworkTrendsChart } from './network-trends-chart';
import { ResourceDonut } from './resource-donut';
import { StorageGrid } from './storage-grid';
import { HeartbeatList } from './heartbeat-list';
import { AlertsPanel } from './alerts-panel';
import { useDashboardStore } from '@/hooks/use-dashboard-store';
import { useDashboardController } from '@/hooks/use-dashboard-controller';

export function DashboardPage() {
  const {
    metrics,
    series,
    networkDelta,
    heartbeats,
    alerts,
    servers,
    activeServer,
    status,
    lastUpdated,
    filter,
    setFilter,
    clearAlert,
    clearAllAlerts,
    setActiveServer,
    resetSeries
  } = useDashboardStore(state => ({
    metrics: state.metrics,
    series: state.series,
    networkDelta: state.networkDelta,
    heartbeats: state.heartbeats,
    alerts: state.alerts,
    servers: state.servers,
    activeServer: state.activeServer,
    status: state.status,
    lastUpdated: state.lastUpdated,
    filter: state.filter,
    setFilter: state.setFilter,
    clearAlert: state.clearAlert,
    clearAllAlerts: state.clearAllAlerts,
    setActiveServer: state.setActiveServer,
    resetSeries: state.resetSeries
  }));

  const { refresh } = useDashboardController();
  const { colorMode } = useColorMode();

  return (
    <Box 
      minH="100vh" 
      bgGradient={colorMode === 'dark' 
        ? 'linear(180deg, navy.900 0%, navy.800 100%)' 
        : 'linear(180deg, gray.50 0%, white 100%)'
      }
      py={{ base: 6, md: 8 }}
      position="relative"
      _before={{
        content: '""',
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        height: '400px',
        bgGradient: colorMode === 'dark'
          ? 'radial(ellipse at top, brand.900, transparent 70%)'
          : 'radial(ellipse at top, brand.50, transparent 70%)',
        opacity: 0.6,
        zIndex: 0,
      }}
    >
      <Container maxW="7xl" position="relative" zIndex={1}>
        <Stack spacing={{ base: 8, md: 10 }}>
          <DashboardHeader
            lastUpdated={lastUpdated}
            servers={servers}
            activeServer={activeServer ?? undefined}
            onSelectServer={server => {
              setActiveServer(server);
              resetSeries();
            }}
            onManualRefresh={refresh}
            isConnected={status === 'online'}
            refreshInterval={2}
            metrics={metrics}
            heartbeats={heartbeats}
            alerts={alerts}
            series={series}
          />

          <StatusBanner status={status} />

          <Flex justify="space-between" align="center" direction={{ base: 'column', lg: 'row' }} gap={6}>
            <DateRangeFilterControl value={filter} onChange={setFilter} />
          </Flex>

          <MetricsOverview metrics={metrics} networkDelta={networkDelta} />

          <Grid templateColumns={{ base: 'repeat(1, 1fr)', xl: 'repeat(12, 1fr)' }} gap={8}>
            <GridItem colSpan={{ base: 12, xl: 8 }} minH="400px">
              <MetricTrendsChart series={series} colorMode={colorMode} />
            </GridItem>
            <GridItem colSpan={{ base: 12, xl: 4 }}>
              <ResourceDonut metrics={metrics} colorMode={colorMode} />
            </GridItem>
          </Grid>

          <Grid templateColumns={{ base: 'repeat(1, 1fr)', xl: 'repeat(12, 1fr)' }} gap={8}>
            <GridItem colSpan={{ base: 12, xl: 8 }} minH="360px">
              <NetworkTrendsChart series={series} colorMode={colorMode} />
            </GridItem>
            <GridItem colSpan={{ base: 12, xl: 4 }}>
              <AlertsPanel alerts={alerts} onClearAlert={clearAlert} onClearAll={clearAllAlerts} />
            </GridItem>
          </Grid>

          <SimpleGrid columns={{ base: 1, xl: 2 }} spacing={8}>
            <Box>
              <StorageGrid disks={metrics?.disk_spaces} />
            </Box>
            <Box>
              <HeartbeatList heartbeats={heartbeats ?? []} />
            </Box>
          </SimpleGrid>
        </Stack>
      </Container>
    </Box>
  );
}
