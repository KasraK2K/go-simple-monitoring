'use client';

import { 
  Box, 
  Card, 
  CardBody, 
  CardHeader, 
  Skeleton, 
  SkeletonCircle, 
  SkeletonText, 
  Stack, 
  HStack, 
  Grid, 
  GridItem,
  useColorMode 
} from '@chakra-ui/react';

interface LoadingSkeletonProps {
  variant?: 'card' | 'metrics' | 'chart' | 'list' | 'storage' | 'alerts';
  count?: number;
}

export function LoadingSkeleton({ variant = 'card', count = 1 }: LoadingSkeletonProps) {
  const { colorMode } = useColorMode();

  const renderSkeleton = () => {
    switch (variant) {
      case 'metrics':
        return (
          <Grid templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)', lg: 'repeat(4, 1fr)' }} gap={6}>
            {Array.from({ length: 4 }).map((_, i) => (
              <GridItem key={i}>
                <Card 
                  bg={colorMode === 'dark' ? 'navy.800' : 'white'}
                  borderRadius="20px"
                  border="1px solid"
                  borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
                  overflow="hidden"
                >
                  <CardBody p={6}>
                    <HStack spacing={4} mb={4}>
                      <SkeletonCircle size="12" />
                      <Stack spacing={2} flex="1">
                        <Skeleton height="4" width="60%" />
                        <Skeleton height="3" width="40%" />
                      </Stack>
                    </HStack>
                    <Skeleton height="8" width="80%" mb={2} />
                    <Skeleton height="3" width="50%" />
                  </CardBody>
                </Card>
              </GridItem>
            ))}
          </Grid>
        );

      case 'chart':
        return (
          <Card 
            bg={colorMode === 'dark' ? 'navy.800' : 'white'}
            borderRadius="20px"
            border="1px solid"
            borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
            overflow="hidden"
          >
            <CardHeader>
              <HStack spacing={3}>
                <SkeletonCircle size="10" />
                <Stack spacing={2}>
                  <Skeleton height="5" width="150px" />
                  <Skeleton height="3" width="100px" />
                </Stack>
              </HStack>
            </CardHeader>
            <CardBody>
              <Skeleton height="300px" borderRadius="16px" />
            </CardBody>
          </Card>
        );

      case 'storage':
        return (
          <Card 
            bg={colorMode === 'dark' ? 'navy.800' : 'white'}
            borderRadius="20px"
            border="1px solid"
            borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
            overflow="hidden"
          >
            <CardHeader>
              <HStack spacing={3}>
                <SkeletonCircle size="10" />
                <Stack spacing={2}>
                  <Skeleton height="5" width="140px" />
                  <Skeleton height="3" width="100px" />
                </Stack>
              </HStack>
            </CardHeader>
            <CardBody>
              <Grid templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)' }} gap={5}>
                {Array.from({ length: 2 }).map((_, i) => (
                  <GridItem key={i}>
                    <Box
                      p={5}
                      borderRadius="16px"
                      border="1px solid"
                      borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.100'}
                      bg={colorMode === 'dark' ? 'navy.700' : 'gray.50'}
                    >
                      <HStack spacing={3} mb={4}>
                        <SkeletonCircle size="10" />
                        <Stack spacing={2} flex="1">
                          <Skeleton height="4" width="70%" />
                          <Skeleton height="3" width="50%" />
                        </Stack>
                        <Skeleton height="6" width="50px" borderRadius="full" />
                      </HStack>
                      <Skeleton height="2" mb={4} borderRadius="full" />
                      <Stack spacing={3}>
                        <HStack justify="space-between">
                          <Stack spacing={1}>
                            <Skeleton height="2" width="30px" />
                            <Skeleton height="4" width="60px" />
                          </Stack>
                          <Stack spacing={1} align="flex-end">
                            <Skeleton height="2" width="30px" />
                            <Skeleton height="4" width="60px" />
                          </Stack>
                        </HStack>
                        <Box pt={3} borderTop="1px solid" borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.200'}>
                          <Skeleton height="2" width="80px" mb={1} />
                          <Skeleton height="5" width="100px" />
                        </Box>
                      </Stack>
                    </Box>
                  </GridItem>
                ))}
              </Grid>
            </CardBody>
          </Card>
        );

      case 'alerts':
        return (
          <Card 
            bg={colorMode === 'dark' ? 'navy.800' : 'white'}
            borderRadius="20px"
            border="1px solid"
            borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
            overflow="hidden"
          >
            <CardHeader>
              <HStack spacing={3}>
                <SkeletonCircle size="10" />
                <Stack spacing={2}>
                  <Skeleton height="5" width="120px" />
                  <Skeleton height="3" width="80px" />
                </Stack>
              </HStack>
            </CardHeader>
            <CardBody>
              <Stack spacing={4}>
                {Array.from({ length: 3 }).map((_, i) => (
                  <Box
                    key={i}
                    p={4}
                    borderRadius="16px"
                    border="1px solid"
                    borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.100'}
                    bg={colorMode === 'dark' ? 'navy.700' : 'gray.50'}
                  >
                    <HStack spacing={4} align="flex-start">
                      <SkeletonCircle size="8" />
                      <Stack spacing={2} flex="1">
                        <HStack justify="space-between">
                          <Skeleton height="5" width="60px" borderRadius="full" />
                          <SkeletonCircle size="4" />
                        </HStack>
                        <Skeleton height="4" width="90%" />
                        <Skeleton height="3" width="60%" />
                      </Stack>
                    </HStack>
                  </Box>
                ))}
              </Stack>
            </CardBody>
          </Card>
        );

      case 'list':
        return (
          <Card 
            bg={colorMode === 'dark' ? 'navy.800' : 'white'}
            borderRadius="20px"
            border="1px solid"
            borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
            overflow="hidden"
          >
            <CardHeader>
              <Stack spacing={4}>
                <HStack spacing={3}>
                  <SkeletonCircle size="10" />
                  <Skeleton height="5" width="150px" />
                </HStack>
                <Skeleton height="10" borderRadius="8px" />
              </Stack>
            </CardHeader>
            <CardBody>
              <Stack spacing={4}>
                {Array.from({ length: count }).map((_, i) => (
                  <Box
                    key={i}
                    p={4}
                    borderRadius="16px"
                    border="1px solid"
                    borderColor={colorMode === 'dark' ? 'navy.600' : 'gray.100'}
                    bg={colorMode === 'dark' ? 'navy.700' : 'gray.50'}
                  >
                    <HStack spacing={4} align="center">
                      <SkeletonCircle size="12" />
                      <Stack spacing={2} flex="1">
                        <Skeleton height="4" width="70%" />
                        <Skeleton height="3" width="50%" />
                        <Skeleton height="3" width="40%" />
                      </Stack>
                      <Stack spacing={2} align="flex-end">
                        <Skeleton height="5" width="60px" borderRadius="full" />
                        <Skeleton height="3" width="80px" />
                      </Stack>
                    </HStack>
                  </Box>
                ))}
              </Stack>
            </CardBody>
          </Card>
        );

      default:
        return (
          <Card 
            bg={colorMode === 'dark' ? 'navy.800' : 'white'}
            borderRadius="20px"
            border="1px solid"
            borderColor={colorMode === 'dark' ? 'navy.700' : 'gray.100'}
            overflow="hidden"
          >
            <CardBody>
              <Stack spacing={3}>
                <HStack spacing={3}>
                  <SkeletonCircle size="10" />
                  <Stack spacing={2} flex="1">
                    <Skeleton height="4" width="60%" />
                    <Skeleton height="3" width="40%" />
                  </Stack>
                </HStack>
                <SkeletonText mt="4" noOfLines={4} spacing="4" skeletonHeight="2" />
              </Stack>
            </CardBody>
          </Card>
        );
    }
  };

  return <>{renderSkeleton()}</>;
}