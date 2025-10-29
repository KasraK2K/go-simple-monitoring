'use client';

import { Badge, Card, CardBody, Flex, Grid, GridItem, Progress, Text } from '@chakra-ui/react';
import { DiskSpace } from '@/lib/types';
import { formatBytes, formatPercent } from '@/lib/format';

interface StorageGridProps {
  disks?: DiskSpace[];
}

export function StorageGrid({ disks }: StorageGridProps) {
  const items = Array.isArray(disks) ? disks : [];

  if (items.length === 0) {
    return (
      <Card variant="outline" borderRadius="xl" shadow="sm">
        <CardBody color="gray.500">
          <Text>No storage data available.</Text>
        </CardBody>
      </Card>
    );
  }

  return (
    <Grid templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, minmax(0, 1fr))' }} gap={5}>
      {items.map((disk, index) => {
        const usage = disk.used_pct ?? 0;
        const badgeColor = usage >= 90 ? 'red' : usage >= 75 ? 'yellow' : 'green';
        return (
          <GridItem key={`${disk.path || disk.filesystem || 'disk'}-${index}`}>
            <Card variant="outline" borderRadius="xl" shadow="sm">
              <CardBody>
                <Flex justify="space-between" align="center" mb={3}>
                  <Text fontWeight="semibold">{disk.path || disk.filesystem || `Disk ${index + 1}`}</Text>
                  <Badge colorScheme={badgeColor}>{formatPercent(usage)}</Badge>
                </Flex>
                <Text fontSize="sm" color="gray.500">
                  {disk.filesystem || 'Filesystem'}
                </Text>
                <Progress
                  value={usage}
                  mt={4}
                  size="sm"
                  bg="gray.100"
                  rounded="full"
                  colorScheme={badgeColor === 'red' ? 'red' : badgeColor === 'yellow' ? 'yellow' : 'green'}
                />
                <Flex justify="space-between" mt={4} fontSize="sm" color="gray.500">
                  <Text>Used: {formatBytes(disk.used_bytes)}</Text>
                  <Text>Free: {formatBytes(disk.available_bytes)}</Text>
                </Flex>
                <Text fontSize="sm" color="gray.500" mt={2}>
                  Total: {formatBytes(disk.total_bytes)}
                </Text>
              </CardBody>
            </Card>
          </GridItem>
        );
      })}
    </Grid>
  );
}
