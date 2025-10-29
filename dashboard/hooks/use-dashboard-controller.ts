'use client';

import { useCallback, useEffect, useRef } from 'react';
import { fetchMetrics, fetchServerConfig, mapServerConfig } from '@/lib/api';
import { DEFAULT_REFRESH_INTERVAL, DEFAULT_THRESHOLDS } from '@/lib/constants';
import { calculateNetworkDelta, createMetricPoint, evaluateThresholds, normalizeMetrics } from '@/lib/transform';
import { AlertMessage, NormalizedMetrics, ThresholdConfig } from '@/lib/types';
import { useDashboardStore } from './use-dashboard-store';

interface ControllerOptions {
  thresholds?: Partial<Record<'cpu' | 'memory' | 'disk', ThresholdConfig>>;
}

export function useDashboardController(options: ControllerOptions = {}) {
  const {
    filter,
    activeServer,
    refreshInterval,
    setRefreshInterval,
    setMetrics,
    appendSeries,
    setSeries,
    resetSeries,
    setHeartbeats,
    setStatus,
    setLastUpdated,
    setAlerts,
    clearAlert,
    setServers
  } = useDashboardStore(state => ({
    filter: state.filter,
    activeServer: state.activeServer,
    refreshInterval: state.refreshInterval,
    setRefreshInterval: state.setRefreshInterval,
    setMetrics: state.setMetrics,
    appendSeries: state.appendSeries,
    setSeries: state.setSeries,
    resetSeries: state.resetSeries,
    setHeartbeats: state.setHeartbeats,
    setStatus: state.setStatus,
    setLastUpdated: state.setLastUpdated,
    setAlerts: state.setAlerts,
    clearAlert: state.clearAlert,
    setServers: state.setServers
  }));

  const thresholdsRef = useRef(options.thresholds ?? DEFAULT_THRESHOLDS);
  const previousMetricsRef = useRef<NormalizedMetrics | null>(null);
  const pollingTimerRef = useRef<NodeJS.Timeout | null>(null);
  const isFetchingRef = useRef(false);
  const scheduleNextPollRef = useRef<() => void>(() => {});

  const syncConnectionAlert = useCallback(
    (status: 'online' | 'offline', message?: string) => {
      const id = 'connection_offline';
      if (status === 'offline') {
        const alert: AlertMessage = {
          id,
          type: 'error',
          message: message ?? 'Connection lost to monitoring service.',
          createdAt: Date.now(),
          persistent: true
        };
        setAlerts(prev => {
          const existing = new Map(prev.map(item => [item.id, item]));
          existing.set(id, alert);
          return Array.from(existing.values()).sort((a, b) => a.createdAt - b.createdAt);
        });
      } else {
        clearAlert(id);
      }
    },
    [setAlerts, clearAlert]
  );

  const applyThresholdAlerts = useCallback(
    (latest: NormalizedMetrics | null) => {
      const alerts = evaluateThresholds(latest, thresholdsRef.current);
      setAlerts(prev => {
        const existing = new Map(prev.map(item => [item.id, item]));
        // Remove previous threshold alerts
        prev.forEach(alert => {
          if (alert.id.startsWith('warning_') || alert.id.startsWith('critical_')) {
            existing.delete(alert.id);
          }
        });
        alerts.forEach(alert => existing.set(alert.id, alert));
        return Array.from(existing.values()).sort((a, b) => a.createdAt - b.createdAt);
      });
    },
    [setAlerts]
  );

  const processMetricsResponse = useCallback(
    (payload: unknown) => {
      const normalizedArray = Array.isArray(payload)
        ? payload.map(item => normalizeMetrics(item))
        : [normalizeMetrics(payload)];

      const filtered = normalizedArray.filter(item => Object.keys(item).length > 0);
      if (!filtered.length) {
        return;
      }

      let previous = previousMetricsRef.current;
      const points = filtered.map(item => {
        const delta = calculateNetworkDelta(previous, item);
        const point = createMetricPoint(item, delta);
        previous = item;
        return { point, delta, item };
      });

      const latestEntry = points[points.length - 1];
      previousMetricsRef.current = latestEntry.item;

      const latestMetrics = latestEntry.item;
      const latestDelta = latestEntry.delta;

      setMetrics(latestMetrics, latestDelta ?? null);
      setHeartbeats(latestMetrics.heartbeat ?? []);
      setLastUpdated(latestMetrics.timestamp ?? new Date().toISOString());
      applyThresholdAlerts(latestMetrics);

      if (filter && (filter.from || filter.to)) {
        setSeries(points.map(entry => entry.point));
      } else {
        const finalPoint = latestEntry.point;
        appendSeries(finalPoint);
      }
    },
    [appendSeries, setMetrics, setHeartbeats, setLastUpdated, applyThresholdAlerts, setSeries, filter]
  );

  const fetchAndUpdate = useCallback(async () => {
    if (isFetchingRef.current) {
      return;
    }
    isFetchingRef.current = true;
    setStatus('connecting');

    try {
      const result = await fetchMetrics(filter, activeServer?.address);
      processMetricsResponse(result);
      setStatus('online');
      syncConnectionAlert('online');
    } catch (error) {
      console.error('Failed to fetch metrics', error);
      setStatus('offline');
      syncConnectionAlert('offline', error instanceof Error ? error.message : 'Unknown error');
    } finally {
      isFetchingRef.current = false;
      if (!filter || (!filter.from && !filter.to)) {
        scheduleNextPollRef.current();
      }
    }
  }, [filter, activeServer?.address, processMetricsResponse, setStatus, syncConnectionAlert]);

  const scheduleNextPoll = useCallback(() => {
    if (pollingTimerRef.current) {
      clearTimeout(pollingTimerRef.current);
    }
    if (filter && (filter.from || filter.to)) {
      return; // Do not auto-poll when viewing historical data
    }
    pollingTimerRef.current = setTimeout(() => {
      void fetchAndUpdate();
    }, refreshInterval || DEFAULT_REFRESH_INTERVAL);
  }, [fetchAndUpdate, filter, refreshInterval]);

  scheduleNextPollRef.current = scheduleNextPoll;

  const initialize = useCallback(async () => {
    const baseAddress = activeServer?.address ?? null;
    previousMetricsRef.current = null;
    resetSeries();
    if (pollingTimerRef.current) {
      clearTimeout(pollingTimerRef.current);
      pollingTimerRef.current = null;
    }

    const config = await fetchServerConfig(baseAddress);
    if (config?.refresh_interval_seconds) {
      setRefreshInterval(config.refresh_interval_seconds * 1000);
    } else {
      setRefreshInterval(DEFAULT_REFRESH_INTERVAL);
    }
    if (config?.thresholds) {
      thresholdsRef.current = {
        cpu: config.thresholds.cpu ?? DEFAULT_THRESHOLDS.cpu,
        memory: config.thresholds.memory ?? DEFAULT_THRESHOLDS.memory,
        disk: config.thresholds.disk ?? DEFAULT_THRESHOLDS.disk
      };
    }
    if (!baseAddress) {
      const serverOptions = mapServerConfig(config);
      setServers(serverOptions);
    }
    await fetchAndUpdate();
  }, [activeServer?.address, fetchAndUpdate, setRefreshInterval, setServers, resetSeries]);

  const refresh = useCallback(async () => {
    await fetchAndUpdate();
  }, [fetchAndUpdate]);

  useEffect(() => {
    initialize();
    return () => {
      if (pollingTimerRef.current) {
        clearTimeout(pollingTimerRef.current);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeServer?.address]);

  useEffect(() => {
    if (pollingTimerRef.current) {
      clearTimeout(pollingTimerRef.current);
      pollingTimerRef.current = null;
    }

    if (filter && (filter.from || filter.to)) {
      fetchAndUpdate();
    } else {
      // filter cleared, resume real-time series
      resetSeries();
      fetchAndUpdate();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter?.from, filter?.to]);

  useEffect(() => {
    if (!filter || (!filter.from && !filter.to)) {
      scheduleNextPoll();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshInterval]);

  return { refresh };
}
