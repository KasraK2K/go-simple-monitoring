import { ThresholdConfig } from '@/lib/types';

export const DEFAULT_REFRESH_INTERVAL = 10_000;

export const DEFAULT_THRESHOLDS: Record<'cpu' | 'memory' | 'disk', ThresholdConfig> = {
  cpu: { warning: 70, critical: 90 },
  memory: { warning: 80, critical: 95 },
  disk: { warning: 85, critical: 95 }
};

export const MAX_SERIES_POINTS = 120;
