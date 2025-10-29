'use client';

import { CacheProvider } from '@chakra-ui/next-js';
import { ChakraProvider, ColorModeScript } from '@chakra-ui/react';
import { ReactNode } from 'react';
import { theme } from '@/lib/theme';

interface ProvidersProps {
  children: ReactNode;
}

export function Providers({ children }: ProvidersProps) {
  return (
    <CacheProvider>
      <ChakraProvider theme={theme}>
        <ColorModeScript initialColorMode={theme.config.initialColorMode} />
        {children}
      </ChakraProvider>
    </CacheProvider>
  );
}
