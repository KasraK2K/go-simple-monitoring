'use client';

import {
  Box,
  Flex,
  HStack,
  Text,
  Badge,
  Tooltip,
  useColorMode,
} from '@chakra-ui/react';
import { keyframes } from '@emotion/react';
import { FiWifi, FiWifiOff, FiClock } from 'react-icons/fi';

interface ConnectionStatusProps {
  isConnected: boolean;
  lastUpdated?: string | null;
  refreshInterval?: number; // in seconds
  nextRefreshIn?: number; // in seconds
}

const pulse = keyframes`
  0% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.1); opacity: 0.7; }
  100% { transform: scale(1); opacity: 1; }
`;

const spin = keyframes`
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
`;

export function ConnectionStatus({
  isConnected,
  lastUpdated,
  refreshInterval = 2,
  nextRefreshIn = 0,
}: ConnectionStatusProps) {
  const { colorMode } = useColorMode();

  const formatTime = (timestamp?: string | null) => {
    if (!timestamp) return '--';
    
    try {
      const date = new Date(timestamp);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffSecs = Math.floor(diffMs / 1000);
      
      if (diffSecs < 60) return `${diffSecs}s ago`;
      if (diffSecs < 3600) return `${Math.floor(diffSecs / 60)}m ago`;
      if (diffSecs < 86400) return `${Math.floor(diffSecs / 3600)}h ago`;
      
      return date.toLocaleDateString();
    } catch {
      return '--';
    }
  };

  const getStatusProps = () => {
    if (isConnected) {
      return {
        color: 'green.500',
        bg: 'green.50',
        borderColor: 'green.200',
        icon: FiWifi,
        label: 'Connected',
        animation: `${pulse} 2s infinite`,
        _dark: {
          bg: 'green.900',
          borderColor: 'green.700',
        }
      };
    } else {
      return {
        color: 'red.500',
        bg: 'red.50',
        borderColor: 'red.200',
        icon: FiWifiOff,
        label: 'Disconnected',
        _dark: {
          bg: 'red.900',
          borderColor: 'red.700',
        }
      };
    }
  };

  const statusProps = getStatusProps();
  const StatusIcon = statusProps.icon;

  return (
    <Flex
      align="center"
      gap={3}
      p={3}
      borderRadius="12px"
      bg={statusProps.bg}
      border="1px solid"
      borderColor={statusProps.borderColor}
      {...(colorMode === 'dark' && statusProps._dark)}
      transition="all 0.3s ease"
    >
      <HStack spacing={2}>
        <Box
          position="relative"
          animation={isConnected ? statusProps.animation : undefined}
        >
          <StatusIcon 
            size={16} 
            color={statusProps.color}
          />
          {isConnected && (
            <Box
              position="absolute"
              top="0"
              right="0"
              w="6px"
              h="6px"
              bg="green.400"
              borderRadius="full"
              animation={`${pulse} 1.5s infinite`}
            />
          )}
        </Box>
        
        <VStack spacing={0} align="flex-start">
          <Text
            fontSize="xs"
            fontWeight="600"
            color={statusProps.color}
            lineHeight="1"
          >
            {statusProps.label}
          </Text>
          
          {isConnected && (
            <Text
              fontSize="xs"
              color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
              lineHeight="1"
            >
              {formatTime(lastUpdated)}
            </Text>
          )}
        </VStack>
      </HStack>

      {isConnected && (
        <Tooltip 
          label={`Auto-refresh every ${refreshInterval}s`}
          placement="top"
        >
          <Badge
            size="sm"
            colorScheme="blue"
            variant="subtle"
            borderRadius="full"
            px={2}
            py={1}
            fontSize="xs"
            fontWeight="500"
          >
            <HStack spacing={1}>
              <Box
                animation={nextRefreshIn <= 5 ? `${spin} 1s linear infinite` : undefined}
              >
                <FiClock size={10} />
              </Box>
              <Text>{refreshInterval}s</Text>
            </HStack>
          </Badge>
        </Tooltip>
      )}
    </Flex>
  );
}

// Compact version for header
export function ConnectionStatusCompact({
  isConnected,
  refreshInterval = 2,
}: Pick<ConnectionStatusProps, 'isConnected' | 'refreshInterval'>) {
  const statusProps = isConnected
    ? { color: 'green.500', icon: FiWifi, bg: 'green.50' }
    : { color: 'red.500', icon: FiWifiOff, bg: 'red.50' };
  
  const StatusIcon = statusProps.icon;

  return (
    <Tooltip 
      label={`${isConnected ? 'Connected' : 'Disconnected'} • Refresh: ${refreshInterval}s`}
      placement="bottom"
    >
      <Flex
        align="center"
        gap={1}
        px={2}
        py={1}
        borderRadius="full"
        bg={statusProps.bg}
        _dark={{
          bg: isConnected ? 'green.900' : 'red.900',
        }}
        transition="all 0.3s ease"
      >
        <Box
          animation={isConnected ? `${pulse} 2s infinite` : undefined}
        >
          <StatusIcon size={12} color={statusProps.color} />
        </Box>
        <Text
          fontSize="xs"
          fontWeight="600"
          color={statusProps.color}
        >
          {refreshInterval}s
        </Text>
      </Flex>
    </Tooltip>
  );
}

// Import VStack for the main component
import { VStack } from '@chakra-ui/react';