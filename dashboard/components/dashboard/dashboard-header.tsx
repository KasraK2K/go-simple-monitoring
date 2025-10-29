'use client';

import { Box, Flex, Heading, HStack, IconButton, Text, Badge, useColorMode, useDisclosure } from '@chakra-ui/react';
import { FiMoon, FiRefreshCw, FiSun, FiActivity } from 'react-icons/fi';
import { MonitorServerOption, NormalizedMetrics, HeartbeatEntry, AlertMessage } from '@/lib/types';
import { ServerSwitcher } from './server-switcher';
import { formatRelativeDate } from '@/lib/format';
import { ExportPanel, ExportTrigger } from './export-panel';
import { ConnectionStatusCompact } from './connection-status';

interface DashboardHeaderProps {
  lastUpdated?: string | null;
  servers: MonitorServerOption[];
  activeServer?: MonitorServerOption | null;
  onSelectServer: (server: MonitorServerOption | null) => void;
  onManualRefresh?: () => void;
  isConnected?: boolean;
  refreshInterval?: number;
  metrics?: NormalizedMetrics | null;
  heartbeats?: HeartbeatEntry[];
  alerts?: AlertMessage[];
  series?: any[];
}

export function DashboardHeader({ 
  lastUpdated, 
  servers, 
  activeServer, 
  onSelectServer, 
  onManualRefresh,
  isConnected = true,
  refreshInterval = 2,
  metrics,
  heartbeats = [],
  alerts = [],
  series = []
}: DashboardHeaderProps) {
  const { colorMode, toggleColorMode } = useColorMode();
  const { isOpen: isExportOpen, onOpen: onExportOpen, onClose: onExportClose } = useDisclosure();

  return (
    <Flex 
      direction={{ base: 'column', lg: 'row' }} 
      justify="space-between" 
      align={{ base: 'flex-start', lg: 'center' }} 
      gap={6}
      p={6}
      borderRadius="24px"
      bg={colorMode === 'dark' 
        ? 'rgba(30, 41, 59, 0.8)' 
        : 'rgba(255, 255, 255, 0.9)'
      }
      backdropFilter="blur(20px)"
      border="1px solid"
      borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
      boxShadow={colorMode === 'dark' ? 'cardDark' : 'cardLight'}
    >
      <Box>
        <HStack spacing={3} mb={2}>
          <Flex
            align="center"
            justify="center"
            w={10}
            h={10}
            rounded="full"
            bgGradient="linear(135deg, brand.400, brand.600)"
            color="white"
          >
            <FiActivity size={20} />
          </Flex>
          <Heading 
            size="xl" 
            fontWeight="bold" 
            bgGradient={colorMode === 'dark' 
              ? 'linear(90deg, white, gray.300)' 
              : 'linear(90deg, gray.800, gray.600)'
            }
            bgClip="text"
          >
            Infrastructure Monitoring
          </Heading>
          <Badge 
            colorScheme="brand" 
            variant="subtle" 
            borderRadius="full" 
            px={3} 
            py={1}
            fontSize="xs"
            fontWeight="600"
          >
            LIVE
          </Badge>
        </HStack>
        
        <Text 
          color={colorMode === 'dark' ? 'gray.300' : 'gray.600'} 
          fontSize="md"
          fontWeight="500"
          mb={1}
        >
          Real-time observability across your infrastructure powered by Horizon UI
        </Text>
        
        <HStack spacing={2}>
          <Text 
            fontSize="sm" 
            color={colorMode === 'dark' ? 'gray.400' : 'gray.500'}
            fontWeight="500"
          >
            Last updated:
          </Text>
          <Badge 
            colorScheme="green" 
            variant="subtle" 
            borderRadius="full"
            fontSize="xs"
          >
            {formatRelativeDate(lastUpdated)}
          </Badge>
        </HStack>
      </Box>
      
      <HStack spacing={4} align="center">
        <ServerSwitcher servers={servers} activeServer={activeServer ?? null} onSelect={onSelectServer} />
        
        <ConnectionStatusCompact 
          isConnected={isConnected}
          refreshInterval={refreshInterval}
        />
        
        <ExportTrigger onOpen={onExportOpen} />
        
        <IconButton
          aria-label="Toggle color mode"
          icon={colorMode === 'dark' ? <FiSun /> : <FiMoon />}
          onClick={toggleColorMode}
          variant="ghost"
          size="lg"
          borderRadius="full"
          bg={colorMode === 'dark' 
            ? 'rgba(255, 255, 255, 0.1)' 
            : 'rgba(0, 0, 0, 0.05)'
          }
          _hover={{
            bg: colorMode === 'dark' 
              ? 'rgba(255, 255, 255, 0.2)' 
              : 'rgba(0, 0, 0, 0.1)',
            transform: 'scale(1.05)',
          }}
          transition="all 0.2s"
        />
        
        <IconButton
          aria-label="Refresh metrics"
          icon={<FiRefreshCw />}
          onClick={onManualRefresh}
          variant="brand"
          size="lg"
          borderRadius="full"
          _hover={{
            transform: 'scale(1.05) rotate(180deg)',
          }}
          transition="all 0.3s"
        />
      </HStack>
      
      <ExportPanel
        isOpen={isExportOpen}
        onClose={onExportClose}
        metrics={metrics}
        heartbeats={heartbeats}
        alerts={alerts}
        series={series}
      />
    </Flex>
  );
}
