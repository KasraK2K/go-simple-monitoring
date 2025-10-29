'use client';

import { Alert, AlertDescription, AlertIcon, AlertTitle, Badge, Button, Card, CardBody, Stack, Text } from '@chakra-ui/react';
import { AlertMessage } from '@/lib/types';

interface AlertsPanelProps {
  alerts: AlertMessage[];
  onClearAlert?: (id: string) => void;
  onClearAll?: () => void;
}

export function AlertsPanel({ alerts, onClearAlert, onClearAll }: AlertsPanelProps) {
  if (!alerts.length) {
    return (
      <Card variant="outline" borderRadius="xl" shadow="sm">
        <CardBody color="gray.500">
          <Text>No active alerts.</Text>
        </CardBody>
      </Card>
    );
  }

  return (
    <Card variant="outline" borderRadius="xl" shadow="sm">
      <CardBody>
        <Stack spacing={4}>
          {alerts.map(alert => (
            <Alert
              key={alert.id}
              status={alert.type === 'error' ? 'error' : alert.type === 'warning' ? 'warning' : 'info'}
              borderRadius="lg"
              variant="left-accent"
              alignItems="flex-start"
            >
              <AlertIcon />
              <Stack spacing={1} flex="1">
                <Badge alignSelf="flex-start" colorScheme={alert.type === 'error' ? 'red' : alert.type === 'warning' ? 'yellow' : 'blue'}>
                  {alert.type.toUpperCase()}
                </Badge>
                <AlertTitle>{alert.message}</AlertTitle>
                {alert.persistent ? (
                  <AlertDescription fontSize="sm">This alert stays visible until conditions recover.</AlertDescription>
                ) : null}
              </Stack>
              {onClearAlert ? (
                <Button variant="ghost" size="sm" onClick={() => onClearAlert(alert.id)}>
                  Dismiss
                </Button>
              ) : null}
            </Alert>
          ))}
          {onClearAll ? (
            <Button onClick={onClearAll} variant="outline" size="sm" alignSelf="flex-end">
              Clear all
            </Button>
          ) : null}
        </Stack>
      </CardBody>
    </Card>
  );
}
