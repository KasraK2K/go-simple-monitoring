import { extendTheme, ThemeConfig } from '@chakra-ui/react';

const config: ThemeConfig = {
  initialColorMode: 'light',
  useSystemColorMode: false
};

const fonts = {
  heading: 'DM Sans, Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  body: 'DM Sans, Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'
};

const colors = {
  brand: {
    50: '#f0f9ff',
    100: '#e0f2fe',
    200: '#bae6fd',
    300: '#7dd3fc',
    400: '#38bdf8',
    500: '#0ea5e9',
    600: '#0284c7',
    700: '#0369a1',
    800: '#075985',
    900: '#0c4a6e',
  },
  navy: {
    50: '#f8fafc',
    100: '#f1f5f9',
    200: '#e2e8f0',
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748b',
    600: '#475569',
    700: '#334155',
    800: '#1e293b',
    900: '#0f172a',
  },
  gray: {
    50: '#f9fafb',
    100: '#f3f4f6',
    200: '#e5e7eb',
    300: '#d1d5db',
    400: '#9ca3af',
    500: '#6b7280',
    600: '#4b5563',
    700: '#374151',
    800: '#1f2937',
    900: '#111827',
  }
};

const shadows = {
  brand: '0px 18px 40px rgba(112, 144, 176, 0.12)',
  cardLight: '0px 18px 40px rgba(112, 144, 176, 0.06)',
  cardDark: '0px 18px 40px rgba(8, 23, 53, 0.16)',
  glass: '0 8px 32px 0 rgba(31, 38, 135, 0.37)',
  glow: '0 0 20px rgba(14, 165, 233, 0.3)',
};

const components = {
  Card: {
    baseStyle: {
      container: {
        borderRadius: '20px',
        boxShadow: 'cardLight',
        border: '1px solid',
        borderColor: 'gray.100',
        _dark: {
          bg: 'navy.800',
          borderColor: 'navy.700',
          boxShadow: 'cardDark',
        },
      },
    },
    variants: {
      glass: {
        container: {
          bg: 'rgba(255, 255, 255, 0.1)',
          backdropFilter: 'blur(10px)',
          border: '1px solid rgba(255, 255, 255, 0.2)',
          boxShadow: 'glass',
          _dark: {
            bg: 'rgba(30, 41, 59, 0.8)',
            border: '1px solid rgba(51, 65, 85, 0.3)',
          },
        },
      },
      gradient: {
        container: {
          bgGradient: 'linear(135deg, brand.400, brand.600)',
          color: 'white',
          border: 'none',
          boxShadow: 'brand',
        },
      },
    },
  },
  Button: {
    baseStyle: {
      borderRadius: '12px',
      fontWeight: '600',
      _focus: {
        boxShadow: 'none',
      },
    },
    variants: {
      brand: {
        bg: 'brand.500',
        color: 'white',
        _hover: {
          bg: 'brand.600',
          transform: 'translateY(-2px)',
          boxShadow: 'brand',
        },
        _active: {
          bg: 'brand.700',
          transform: 'translateY(0)',
        },
      },
      glass: {
        bg: 'rgba(255, 255, 255, 0.1)',
        backdropFilter: 'blur(10px)',
        border: '1px solid rgba(255, 255, 255, 0.2)',
        color: 'white',
        _hover: {
          bg: 'rgba(255, 255, 255, 0.2)',
        },
      },
    },
  },
  Text: {
    baseStyle: {
      color: 'gray.700',
      _dark: {
        color: 'gray.200',
      },
    },
  },
  Heading: {
    baseStyle: {
      color: 'gray.900',
      _dark: {
        color: 'white',
      },
    },
  },
};

const styles = {
  global: (props: any) => ({
    body: {
      bg: props.colorMode === 'dark' ? 'navy.900' : 'gray.50',
      color: props.colorMode === 'dark' ? 'gray.200' : 'gray.700',
    },
  }),
};

export const theme = extendTheme({
  config,
  fonts,
  colors,
  shadows,
  components,
  styles,
});
