'use client';

import { 
  Alert, 
  AlertDescription, 
  AlertIcon, 
  AlertTitle, 
  Badge, 
  Button, 
  Card, 
  CardBody, 
  CardHeader,
  Stack, 
  Text,
  Heading,
  HStack,
  IconButton,
  useColorMode,
  Box,
  Flex
} from '@chakra-ui/react';
import { FiAlertTriangle, FiX, FiTrash2 } from 'react-icons/fi';
import { AlertMessage } from '@/lib/types';

interface AlertsPanelProps {
  alerts: AlertMessage[];
  onClearAlert?: (id: string) => void;
  onClearAll?: () => void;
}

export function AlertsPanel({ alerts, onClearAlert, onClearAll }: AlertsPanelProps) {
  const { colorMode } = useColorMode();

  return (
    <Card 
      bg={colorMode === 'dark' ? 'navy.800' : 'white'}
      borderRadius="20px"
      border="1px solid"
      borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
      boxShadow={colorMode === 'dark' ? 'cardDark' : 'cardLight'}
      overflow="hidden"
    >
      <CardHeader pb={3}>
        <Flex justify="space-between" align="center">
          <HStack spacing={3}>
            <Flex
              align="center"
              justify="center"
              w={10}
              h={10}
              rounded="full"
              bg={alerts.length > 0 ? 'red.50' : 'green.50'}
              _dark={{
                bg: alerts.length > 0 ? 'red.900' : 'green.900'
              }}
              color={alerts.length > 0 ? 'red.500' : 'green.500'}
            >
              <FiAlertTriangle size={20} />
            </Flex>
            <Box>
              <Heading size="md" color={colorMode === 'dark' ? 'white' : 'gray.900'}>
                System Alerts
              </Heading>
              <Text fontSize="sm" color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}>
                {alerts.length === 0 ? 'All systems operational' : `${alerts.length} active alert${alerts.length > 1 ? 's' : ''}`}
              </Text>
            </Box>
          </HStack>
          
          {alerts.length > 0 && onClearAll && (
            <IconButton
              aria-label="Clear all alerts"
              icon={<FiTrash2 />}
              onClick={onClearAll}
              variant="ghost"
              size="sm"
              colorScheme="red"
              _hover={{
                bg: 'red.50',
                _dark: { bg: 'red.900' }
              }}
            />
          )}
        </Flex>
      </CardHeader>

      <CardBody pt={0}>
        {!alerts.length ? (
          <Box
            textAlign="center"
            py={8}
            color={colorMode === 'dark' ? 'gray.400' : 'gray.500'}
          >
            <Box
              w={16}
              h={16}
              mx="auto"
              mb={4}
              rounded="full"
              bg="green.50"
              _dark={{ bg: 'green.900' }}
              display="flex"
              alignItems="center"
              justifyContent="center"
            >
              <FiAlertTriangle size={24} color="var(--chakra-colors-green-500)" />
            </Box>
            <Text fontWeight="500">No active alerts</Text>
            <Text fontSize="sm" mt={1}>
              Your system is running smoothly
            </Text>
          </Box>
        ) : (
          <Stack spacing={4}>
            {alerts.map(alert => {
              const getAlertStyles = () => {
                switch (alert.type) {
                  case 'error':
                    return {
                      bg: colorMode === 'dark' ? 'red.900' : 'red.50',
                      borderColor: 'red.200',
                      iconColor: 'red.500',
                      badgeScheme: 'red'
                    };
                  case 'warning':
                    return {
                      bg: colorMode === 'dark' ? 'yellow.900' : 'yellow.50',
                      borderColor: 'yellow.200',
                      iconColor: 'yellow.500',
                      badgeScheme: 'yellow'
                    };
                  default:
                    return {
                      bg: colorMode === 'dark' ? 'blue.900' : 'blue.50',
                      borderColor: 'blue.200',
                      iconColor: 'blue.500',
                      badgeScheme: 'blue'
                    };
                }
              };

              const styles = getAlertStyles();

              return (
                <Box
                  key={alert.id}
                  p={4}
                  borderRadius="16px"
                  bg={styles.bg}
                  border="1px solid"
                  borderColor={styles.borderColor}
                  _dark={{ borderColor: `${styles.badgeScheme}.700` }}
                  position="relative"
                  transition="all 0.2s ease"
                  _hover={{
                    transform: 'translateY(-1px)',
                    boxShadow: 'md'
                  }}
                >
                  <HStack spacing={4} align="flex-start">
                    <Flex
                      align="center"
                      justify="center"
                      w={8}
                      h={8}
                      rounded="full"
                      bg={`${styles.badgeScheme}.100`}
                      _dark={{ bg: `${styles.badgeScheme}.800` }}
                      color={styles.iconColor}
                      flexShrink={0}
                    >
                      <FiAlertTriangle size={16} />
                    </Flex>
                    
                    <Stack spacing={2} flex="1">
                      <HStack justify="space-between" align="flex-start">
                        <Badge 
                          colorScheme={styles.badgeScheme} 
                          variant="solid"
                          borderRadius="full"
                          px={3}
                          py={1}
                          fontSize="xs"
                          fontWeight="600"
                        >
                          {alert.type.toUpperCase()}
                        </Badge>
                        
                        {onClearAlert && (
                          <IconButton
                            aria-label="Dismiss alert"
                            icon={<FiX />}
                            onClick={() => onClearAlert(alert.id)}
                            variant="ghost"
                            size="xs"
                            color="gray.500"
                            _hover={{
                              color: 'gray.700',
                              bg: 'gray.100',
                              _dark: {
                                color: 'gray.300',
                                bg: 'gray.700'
                              }
                            }}
                          />
                        )}
                      </HStack>
                      
                      <Text 
                        fontWeight="600" 
                        fontSize="sm"
                        color={colorMode === 'dark' ? 'white' : 'gray.900'}
                        lineHeight="1.4"
                      >
                        {alert.message}
                      </Text>
                      
                      {alert.persistent && (
                        <Text 
                          fontSize="xs" 
                          color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                          fontStyle="italic"
                        >
                          This alert persists until conditions recover
                        </Text>
                      )}
                      
                      <Text 
                        fontSize="xs" 
                        color={colorMode === 'dark' ? 'gray.500' : 'gray.500'}
                      >
                        {new Date(alert.createdAt).toLocaleString()}
                      </Text>
                    </Stack>
                  </HStack>
                </Box>
              );
            })}
          </Stack>
        )}
      </CardBody>
    </Card>
  );
}
