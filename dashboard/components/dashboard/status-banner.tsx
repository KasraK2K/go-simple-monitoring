'use client';

import { Alert, AlertDescription, AlertIcon, AlertTitle, CloseButton } from '@chakra-ui/react';
import { useMemo } from 'react';

interface StatusBannerProps {
  status: 'online' | 'offline' | 'connecting';
  message?: string;
  onDismiss?: () => void;
}

export function StatusBanner({ status, message, onDismiss }: StatusBannerProps) {
  const alertProps = useMemo(() => {
    switch (status) {
      case 'connecting':
        return { status: 'info' as const, title: 'Connecting', description: message ?? 'Attempting to reach monitoring service.' };
      case 'offline':
        return { status: 'error' as const, title: 'Disconnected', description: message ?? 'Connection lost. Retrying shortly.' };
      default:
        return { status: 'success' as const, title: 'Online', description: message ?? 'Live metrics streaming from server.' };
    }
  }, [status, message]);

  if (status === 'online') {
    return null;
  }

  return (
    <Alert status={alertProps.status} borderRadius="2xl" variant="subtle" alignItems="center">
      <AlertIcon />
      <AlertTitle mr={2}>{alertProps.title}</AlertTitle>
      <AlertDescription>{alertProps.description}</AlertDescription>
      {onDismiss ? <CloseButton position="absolute" right={3} top={3} onClick={onDismiss} /> : null}
    </Alert>
  );
}
