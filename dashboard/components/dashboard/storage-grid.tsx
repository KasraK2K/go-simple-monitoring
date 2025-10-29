'use client';

import { 
  Badge, 
  Card, 
  CardBody, 
  CardHeader,
  Flex, 
  Grid, 
  GridItem, 
  Progress, 
  Text, 
  Box,
  HStack,
  Stack,
  Heading,
  useColorMode,
  Icon
} from '@chakra-ui/react';
import { FiHardDrive, FiPieChart } from 'react-icons/fi';
import { DiskSpace } from '@/lib/types';
import { formatBytes, formatPercent } from '@/lib/format';

interface StorageGridProps {
  disks?: DiskSpace[];
}

export function StorageGrid({ disks }: StorageGridProps) {
  const { colorMode } = useColorMode();
  const items = Array.isArray(disks) ? disks : [];

  if (items.length === 0) {
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
          <HStack spacing={3}>
            <Flex
              align="center"
              justify="center"
              w={10}
              h={10}
              rounded="full"
              bg="gray.50"
              _dark={{ bg: 'gray.900' }}
              color="gray.500"
            >
              <FiHardDrive size={20} />
            </Flex>
            <Box>
              <Heading size="md" color={colorMode === 'dark' ? 'white' : 'gray.900'}>
                Storage Overview
              </Heading>
              <Text fontSize="sm" color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}>
                No storage data available
              </Text>
            </Box>
          </HStack>
        </CardHeader>
        
        <CardBody pt={0}>
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
              bg="gray.50"
              _dark={{ bg: 'gray.900' }}
              display="flex"
              alignItems="center"
              justifyContent="center"
            >
              <FiHardDrive size={24} color="var(--chakra-colors-gray-500)" />
            </Box>
            <Text fontWeight="500">No storage devices found</Text>
            <Text fontSize="sm" mt={1}>
              Storage monitoring is currently unavailable
            </Text>
          </Box>
        </CardBody>
      </Card>
    );
  }

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
        <HStack spacing={3}>
          <Flex
            align="center"
            justify="center"
            w={10}
            h={10}
            rounded="full"
            bg="brand.50"
            _dark={{ bg: 'brand.900' }}
            color="brand.500"
          >
            <FiHardDrive size={20} />
          </Flex>
          <Box>
            <Heading size="md" color={colorMode === 'dark' ? 'white' : 'gray.900'}>
              Storage Overview
            </Heading>
            <Text fontSize="sm" color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}>
              {items.length} storage device{items.length > 1 ? 's' : ''} monitored
            </Text>
          </Box>
        </HStack>
      </CardHeader>

      <CardBody pt={0}>
        <Grid templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, minmax(0, 1fr))' }} gap={5}>
          {items.map((disk, index) => {
            const usage = disk.used_pct ?? 0;
            const badgeColor = usage >= 90 ? 'red' : usage >= 75 ? 'yellow' : 'green';
            
            return (
              <GridItem key={`${disk.path || disk.filesystem || 'disk'}-${index}`}>
                <Box
                  p={5}
                  borderRadius="16px"
                  bg={colorMode === 'dark' 
                    ? 'linear(135deg, navy.700 0%, navy.600 100%)' 
                    : 'linear(135deg, white 0%, gray.50 100%)'
                  }
                  border="1px solid"
                  borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.100'}
                  position="relative"
                  overflow="hidden"
                  transition="all 0.3s ease"
                  _hover={{
                    transform: 'translateY(-4px)',
                    boxShadow: colorMode === 'dark' ? 'brand' : 'xl',
                    borderColor: 'brand.200',
                  }}
                  _before={{
                    content: '""',
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    background: colorMode === 'dark' 
                      ? 'linear(135deg, rgba(76, 110, 245, 0.1) 0%, rgba(159, 122, 234, 0.05) 100%)'
                      : 'linear(135deg, rgba(76, 110, 245, 0.02) 0%, rgba(159, 122, 234, 0.01) 100%)',
                    borderRadius: '16px',
                    zIndex: 0,
                  }}
                >
                  <Box position="relative" zIndex={1}>
                    <Flex justify="space-between" align="center" mb={4}>
                      <HStack spacing={3}>
                        <Flex
                          align="center"
                          justify="center"
                          w={10}
                          h={10}
                          rounded="12px"
                          bg={`${badgeColor}.50`}
                          _dark={{ bg: `${badgeColor}.900` }}
                          color={`${badgeColor}.500`}
                        >
                          <Icon as={FiPieChart} boxSize={5} />
                        </Flex>
                        <Stack spacing={0}>
                          <Text 
                            fontWeight="700" 
                            fontSize="sm"
                            color={colorMode === 'dark' ? 'white' : 'gray.900'}
                          >
                            {disk.path || disk.filesystem || `Disk ${index + 1}`}
                          </Text>
                          <Text 
                            fontSize="xs" 
                            color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                          >
                            {disk.filesystem || 'Unknown filesystem'}
                          </Text>
                        </Stack>
                      </HStack>
                      <Badge 
                        colorScheme={badgeColor} 
                        variant="solid"
                        borderRadius="full"
                        px={3}
                        py={1}
                        fontSize="xs"
                        fontWeight="700"
                      >
                        {formatPercent(usage)}
                      </Badge>
                    </Flex>

                    <Box mb={4}>
                      <Flex justify="space-between" align="center" mb={2}>
                        <Text 
                          fontSize="xs" 
                          fontWeight="600"
                          color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                        >
                          USAGE
                        </Text>
                        <Text 
                          fontSize="xs" 
                          fontWeight="600"
                          color={`${badgeColor}.500`}
                        >
                          {formatPercent(usage)}
                        </Text>
                      </Flex>
                      <Progress
                        value={usage}
                        size="lg"
                        bg={colorMode === 'dark' ? 'navy.900' : 'gray.100'}
                        borderRadius="full"
                        colorScheme={badgeColor}
                        sx={{
                          '& > div': {
                            background: usage >= 90 
                              ? 'linear(90deg, red.400 0%, red.500 100%)'
                              : usage >= 75
                              ? 'linear(90deg, yellow.400 0%, yellow.500 100%)'
                              : 'linear(90deg, green.400 0%, green.500 100%)'
                          }
                        }}
                      />
                    </Box>

                    <Stack spacing={3}>
                      <Flex justify="space-between">
                        <Box>
                          <Text 
                            fontSize="xs" 
                            fontWeight="600"
                            color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                            mb={1}
                          >
                            USED
                          </Text>
                          <Text 
                            fontSize="sm" 
                            fontWeight="700"
                            color={colorMode === 'dark' ? 'white' : 'gray.900'}
                          >
                            {formatBytes(disk.used_bytes)}
                          </Text>
                        </Box>
                        <Box textAlign="right">
                          <Text 
                            fontSize="xs" 
                            fontWeight="600"
                            color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                            mb={1}
                          >
                            FREE
                          </Text>
                          <Text 
                            fontSize="sm" 
                            fontWeight="700"
                            color={colorMode === 'dark' ? 'white' : 'gray.900'}
                          >
                            {formatBytes(disk.available_bytes)}
                          </Text>
                        </Box>
                      </Flex>
                      
                      <Box 
                        pt={3} 
                        borderTop="1px solid" 
                        borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.200'}
                      >
                        <Text 
                          fontSize="xs" 
                          fontWeight="600"
                          color={colorMode === 'dark' ? 'gray.400' : 'gray.600'}
                          mb={1}
                        >
                          TOTAL CAPACITY
                        </Text>
                        <Text 
                          fontSize="lg" 
                          fontWeight="800"
                          color={colorMode === 'dark' ? 'white' : 'gray.900'}
                        >
                          {formatBytes(disk.total_bytes)}
                        </Text>
                      </Box>
                    </Stack>
                  </Box>
                </Box>
              </GridItem>
            );
          })}
        </Grid>
      </CardBody>
    </Card>
  );
}
