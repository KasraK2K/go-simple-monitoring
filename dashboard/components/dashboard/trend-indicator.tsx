'use client';

import {
  Box,
  HStack,
  Text,
  useColorMode,
  Tooltip,
} from '@chakra-ui/react';
import { FiTrendingUp, FiTrendingDown, FiMinus } from 'react-icons/fi';

export type TrendDirection = 'up' | 'down' | 'neutral';

interface TrendIndicatorProps {
  direction: TrendDirection;
  value?: number;
  percentage?: boolean;
  size?: 'sm' | 'md' | 'lg';
  showValue?: boolean;
  tooltip?: string;
}

export function TrendIndicator({
  direction,
  value,
  percentage = true,
  size = 'sm',
  showValue = true,
  tooltip,
}: TrendIndicatorProps) {
  const { colorMode } = useColorMode();

  const getTrendProps = () => {
    switch (direction) {
      case 'up':
        return {
          color: 'red.500',
          bg: 'red.50',
          icon: FiTrendingUp,
          label: 'Increasing',
          _dark: { bg: 'red.900', color: 'red.300' }
        };
      case 'down':
        return {
          color: 'green.500',
          bg: 'green.50',
          icon: FiTrendingDown,
          label: 'Decreasing',
          _dark: { bg: 'green.900', color: 'green.300' }
        };
      default:
        return {
          color: 'gray.500',
          bg: 'gray.50',
          icon: FiMinus,
          label: 'Stable',
          _dark: { bg: 'gray.800', color: 'gray.400' }
        };
    }
  };

  const getSizeProps = () => {
    switch (size) {
      case 'lg':
        return {
          iconSize: 16,
          fontSize: 'sm',
          px: 3,
          py: 1.5,
          borderRadius: '8px',
        };
      case 'md':
        return {
          iconSize: 14,
          fontSize: 'xs',
          px: 2.5,
          py: 1,
          borderRadius: '6px',
        };
      default:
        return {
          iconSize: 12,
          fontSize: 'xs',
          px: 2,
          py: 0.5,
          borderRadius: '4px',
        };
    }
  };

  const trendProps = getTrendProps();
  const sizeProps = getSizeProps();
  const TrendIcon = trendProps.icon;

  const formatValue = (val?: number) => {
    if (val === undefined || val === null) return '';
    const formatted = Math.abs(val).toFixed(1);
    const prefix = val > 0 ? '+' : val < 0 ? '-' : '';
    const suffix = percentage ? '%' : '';
    return `${prefix}${formatted}${suffix}`;
  };

  const component = (
    <HStack
      spacing={1}
      bg={trendProps.bg}
      color={trendProps.color}
      px={sizeProps.px}
      py={sizeProps.py}
      borderRadius={sizeProps.borderRadius}
      border="1px solid"
      borderColor={trendProps.color + '20'}
      {...(colorMode === 'dark' && trendProps._dark)}
      transition="all 0.2s ease"
    >
      <TrendIcon size={sizeProps.iconSize} />
      {showValue && value !== undefined && (
        <Text
          fontSize={sizeProps.fontSize}
          fontWeight="600"
          lineHeight="1"
        >
          {formatValue(value)}
        </Text>
      )}
    </HStack>
  );

  if (tooltip) {
    return (
      <Tooltip label={tooltip} placement="top">
        {component}
      </Tooltip>
    );
  }

  return component;
}

// Helper function to calculate trend from historical data
export function calculateTrend(
  currentValue: number, 
  previousValue?: number
): { direction: TrendDirection; percentage: number } {
  if (previousValue === undefined || previousValue === 0) {
    return { direction: 'neutral', percentage: 0 };
  }

  const change = currentValue - previousValue;
  const percentageChange = (change / previousValue) * 100;
  
  let direction: TrendDirection = 'neutral';
  if (Math.abs(percentageChange) > 0.1) { // Only show trend if change > 0.1%
    direction = percentageChange > 0 ? 'up' : 'down';
  }

  return {
    direction,
    percentage: percentageChange,
  };
}

// Compact trend for inline use
interface CompactTrendProps {
  current: number;
  previous?: number;
  unit?: string;
  reverseColors?: boolean; // For metrics where down is good (like CPU usage)
}

export function CompactTrend({ 
  current, 
  previous, 
  unit = '%',
  reverseColors = false 
}: CompactTrendProps) {
  const trend = calculateTrend(current, previous);
  
  // Reverse colors for metrics where lower is better
  let direction = trend.direction;
  if (reverseColors && direction !== 'neutral') {
    direction = direction === 'up' ? 'down' : 'up';
  }

  if (trend.direction === 'neutral') return null;

  return (
    <TrendIndicator
      direction={direction}
      value={trend.percentage}
      percentage={unit === '%'}
      size="sm"
      tooltip={`${trend.percentage > 0 ? '+' : ''}${trend.percentage.toFixed(1)}${unit} from previous period`}
    />
  );
}