'use client';

import {
  Box,
  Container,
  Flex,
  Grid,
  GridItem,
  SimpleGrid,
  Stack,
  Heading,
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
import { LoadingSkeleton } from '../ui/loading-skeleton';
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

          <Stack spacing={{ base: 10, md: 12 }}>
            {/* Filters Section */}
            <Box>
              <Heading 
                size="lg" 
                color={colorMode === 'dark' ? 'white' : 'gray.900'}
                mb={6}
                fontWeight="800"
              >
                Dashboard Controls
              </Heading>
              <Flex justify="space-between" align="center" direction={{ base: 'column', lg: 'row' }} gap={6}>
                <DateRangeFilterControl value={filter} onChange={setFilter} />
              </Flex>
            </Box>

            {/* Metrics Overview Section */}
            <Box>
              <Heading 
                size="lg" 
                color={colorMode === 'dark' ? 'white' : 'gray.900'}
                mb={6}
                fontWeight="800"
              >
                System Metrics Overview
              </Heading>
              {!metrics ? (
                <LoadingSkeleton variant="metrics" />
              ) : (
                <MetricsOverview metrics={metrics} networkDelta={networkDelta} />
              )}
            </Box>

            {/* Performance Analytics Section */}
            <Box>
              <Heading 
                size="lg" 
                color={colorMode === 'dark' ? 'white' : 'gray.900'}
                mb={6}
                fontWeight="800"
              >
                Performance Analytics
              </Heading>
              <Grid templateColumns={{ base: 'repeat(1, 1fr)', xl: 'repeat(12, 1fr)' }} gap={8}>
                <GridItem colSpan={{ base: 12, xl: 8 }} minH="400px">
                  {!series || series.length === 0 ? (
                    <LoadingSkeleton variant="chart" />
                  ) : (
                    <MetricTrendsChart series={series} colorMode={colorMode} />
                  )}
                </GridItem>
                <GridItem colSpan={{ base: 12, xl: 4 }}>
                  {!metrics ? (
                    <LoadingSkeleton variant="chart" />
                  ) : (
                    <ResourceDonut metrics={metrics} colorMode={colorMode} />
                  )}
                </GridItem>
              </Grid>
            </Box>

            {/* Network & Alerts Section */}
            <Box>
              <Heading 
                size="lg" 
                color={colorMode === 'dark' ? 'white' : 'gray.900'}
                mb={6}
                fontWeight="800"
              >
                Network Monitoring & Alerts
              </Heading>
              <Grid templateColumns={{ base: 'repeat(1, 1fr)', xl: 'repeat(12, 1fr)' }} gap={8}>
                <GridItem colSpan={{ base: 12, xl: 8 }} minH="360px">
                  {!series || series.length === 0 ? (
                    <LoadingSkeleton variant="chart" />
                  ) : (
                    <NetworkTrendsChart series={series} colorMode={colorMode} />
                  )}
                </GridItem>
                <GridItem colSpan={{ base: 12, xl: 4 }}>
                  {!alerts ? (
                    <LoadingSkeleton variant="alerts" />
                  ) : (
                    <AlertsPanel alerts={alerts} onClearAlert={clearAlert} onClearAll={clearAllAlerts} />
                  )}
                </GridItem>
              </Grid>
            </Box>

            {/* Infrastructure Status Section */}
            <Box>
              <Heading 
                size="lg" 
                color={colorMode === 'dark' ? 'white' : 'gray.900'}
                mb={6}
                fontWeight="800"
              >
                Infrastructure Status
              </Heading>
              <SimpleGrid columns={{ base: 1, xl: 2 }} spacing={8}>
                <Box>
                  {!metrics?.disk_spaces ? (
                    <LoadingSkeleton variant="storage" />
                  ) : (
                    <StorageGrid disks={metrics.disk_spaces} />
                  )}
                </Box>
                <Box>
                  {!heartbeats ? (
                    <LoadingSkeleton variant="list" count={5} />
                  ) : (
                    <HeartbeatList heartbeats={heartbeats} />
                  )}
                </Box>
              </SimpleGrid>
            </Box>
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}
