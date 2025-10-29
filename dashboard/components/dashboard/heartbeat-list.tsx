'use client';

import { 
  Badge, 
  Card, 
  CardBody, 
  CardHeader, 
  Flex, 
  HStack, 
  Icon, 
  Input, 
  InputGroup,
  InputLeftElement,
  Stack, 
  Text,
  Box,
  Heading,
  useColorMode,
  VStack
} from '@chakra-ui/react';
import { FiAlertTriangle, FiCheckCircle, FiClock, FiTrendingUp, FiXCircle, FiSearch } from 'react-icons/fi';
import { HeartbeatEntry } from '@/lib/types';
import { formatDuration, formatRelativeDate } from '@/lib/format';
import { useState, useMemo } from 'react';

interface HeartbeatListProps {
  heartbeats?: HeartbeatEntry[];
}

const STATUS_ICON_MAP = {
  healthy: FiCheckCircle,
  degraded: FiAlertTriangle,
  offline: FiXCircle,
  unknown: FiClock
} as const;

const STATUS_COLOR_MAP = {
  healthy: 'green',
  degraded: 'yellow',
  offline: 'red',
  unknown: 'gray'
} as const;

export function HeartbeatList({ heartbeats }: HeartbeatListProps) {
  const { colorMode } = useColorMode();
  const [searchTerm, setSearchTerm] = useState('');
  const items = heartbeats ?? [];

  const filteredItems = useMemo(() => {
    if (!searchTerm.trim()) return items;
    
    const query = searchTerm.toLowerCase();
    return items.filter(item => 
      item.name?.toLowerCase().includes(query) ||
      item.url?.toLowerCase().includes(query) ||
      item.region?.toLowerCase().includes(query) ||
      item.tags?.some(tag => tag.toLowerCase().includes(query))
    );
  }, [items, searchTerm]);

  return (
    <Card 
      bg={colorMode === 'dark' ? 'navy.800' : 'white'}
      borderRadius="20px"
      border="1px solid"
      borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
      boxShadow={colorMode === 'dark' ? 'cardDark' : 'cardLight'}
    >
      <CardHeader>
        <VStack spacing={4} align="stretch">
          <Heading size="md" color={colorMode === 'dark' ? 'white' : 'gray.900'}>
            Server Heartbeats
          </Heading>
          
          <InputGroup>
            <InputLeftElement pointerEvents="none">
              <FiSearch color={colorMode === 'dark' ? '#94A3B8' : '#6B7280'} />
            </InputLeftElement>
            <Input
              placeholder="Search heartbeats..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              bg={colorMode === 'dark' ? 'navy.700' : 'white'}
              borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.200'}
              _hover={{
                borderColor: colorMode === 'dark' ? 'brand.400' : 'brand.300',
              }}
              _focus={{
                borderColor: 'brand.500',
                boxShadow: '0 0 0 1px var(--chakra-colors-brand-500)',
              }}
            />
          </InputGroup>
        </VStack>
      </CardHeader>
      
      <CardBody>
        {!items.length ? (
          <Text color="gray.500" textAlign="center" py={8}>
            No heartbeat data available.
          </Text>
        ) : !filteredItems.length ? (
          <Text color="gray.500" textAlign="center" py={8}>
            No heartbeats match your search.
          </Text>
        ) : (
          <Stack spacing={4}>
            {filteredItems.map(item => {
              const icon = STATUS_ICON_MAP[item.status ?? 'unknown'];
              const colorScheme = STATUS_COLOR_MAP[item.status ?? 'unknown'];
              return (
                <Box
                  key={item.url || item.name}
                  p={4}
                  borderRadius="16px"
                  border="1px solid"
                  borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.100'}
                  bg={colorMode === 'dark' ? 'navy.700' : 'gray.50'}
                  _hover={{
                    borderColor: colorMode === 'dark' ? 'brand.400' : 'brand.200',
                    transform: 'translateY(-2px)',
                    boxShadow: colorMode === 'dark' ? 'cardDark' : 'brand',
                  }}
                  transition="all 0.2s ease"
                >
                  <Flex 
                    justify="space-between" 
                    align={{ base: 'flex-start', md: 'center' }} 
                    direction={{ base: 'column', md: 'row' }} 
                    gap={4}
                  >
                    <HStack spacing={4} align="center">
                      <Flex 
                        align="center" 
                        justify="center" 
                        w={12}
                        h={12}
                        bg={`${colorScheme}.50`}
                        rounded="16px"
                        border="1px solid"
                        borderColor={`${colorScheme}.200`}
                        _dark={{ 
                          bg: `${colorScheme}.900`,
                          borderColor: `${colorScheme}.700`
                        }}
                      >
                        <Icon as={icon} boxSize={6} color={`${colorScheme}.500`} />
                      </Flex>
                      <Stack spacing={1}>
                        <Text 
                          fontWeight="bold" 
                          fontSize="md"
                          color={colorMode === 'dark' ? 'white' : 'gray.900'}
                        >
                          {item.name}
                        </Text>
                        <Text 
                          fontSize="sm" 
                          color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                        >
                          {item.url ?? '—'}
                        </Text>
                        {item.region && (
                          <Text 
                            fontSize="xs" 
                            color={colorMode === 'dark' ? 'gray.500' : 'gray.500'}
                          >
                            Region: {item.region}
                          </Text>
                        )}
                      </Stack>
                    </HStack>
                    
                    <Stack 
                      direction={{ base: 'column', md: 'row' }} 
                      spacing={3} 
                      align={{ base: 'flex-start', md: 'center' }}
                    >
                      <Badge 
                        colorScheme={colorScheme} 
                        variant="solid"
                        borderRadius="full"
                        px={3}
                        py={1}
                        fontSize="xs"
                        fontWeight="600"
                      >
                        {item.status?.toUpperCase() ?? 'UNKNOWN'}
                      </Badge>
                      
                      <HStack spacing={1} color="gray.500" fontSize="sm">
                        <Icon as={FiClock} size={14} />
                        <Text>{formatRelativeDate(item.last_beat)}</Text>
                      </HStack>
                      
                      {item.uptime_percentage != null && (
                        <HStack spacing={1} color="gray.500" fontSize="sm">
                          <Icon as={FiTrendingUp} size={14} />
                          <Text>{item.uptime_percentage.toFixed(1)}%</Text>
                        </HStack>
                      )}
                      
                      {item.last_duration_ms != null && (
                        <Text fontSize="sm" color="gray.500">
                          {formatDuration(item.last_duration_ms)}
                        </Text>
                      )}
                    </Stack>
                  </Flex>
                  
                  {item.tags && item.tags.length > 0 && (
                    <HStack spacing={2} mt={4} flexWrap="wrap">
                      {item.tags.map(tag => (
                        <Badge 
                          key={tag} 
                          variant="subtle" 
                          colorScheme="blue"
                          borderRadius="full"
                          fontSize="xs"
                        >
                          {tag}
                        </Badge>
                      ))}
                    </HStack>
                  )}
                </Box>
              );
            })}
          </Stack>
        )}
      </CardBody>
    </Card>
  );
}
