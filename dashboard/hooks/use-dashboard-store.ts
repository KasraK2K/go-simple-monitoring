'use client';

import { create } from 'zustand';
import { AlertMessage, DateRangeFilter, MetricPoint, MonitorServerOption, NetworkDelta, NormalizedMetrics } from '@/lib/types';

export type ConnectionState = 'online' | 'offline' | 'connecting';

interface DashboardStore {
  metrics: NormalizedMetrics | null;
  series: MetricPoint[];
  networkDelta: NetworkDelta | null;
  heartbeats: NormalizedMetrics['heartbeat'];
  alerts: AlertMessage[];
  servers: MonitorServerOption[];
  activeServer: MonitorServerOption | null;
  status: ConnectionState;
  lastUpdated: string | null;
  refreshInterval: number;
  filter: DateRangeFilter | null;
  setMetrics: (metrics: NormalizedMetrics | null, networkDelta?: NetworkDelta | null) => void;
  appendSeries: (point: MetricPoint) => void;
  setSeries: (points: MetricPoint[]) => void;
  resetSeries: () => void;
  setHeartbeats: (heartbeats: NormalizedMetrics['heartbeat']) => void;
  setAlerts: (alerts: AlertMessage[] | ((previous: AlertMessage[]) => AlertMessage[])) => void;
  clearAlert: (id: string) => void;
  clearAllAlerts: () => void;
  setServers: (servers: MonitorServerOption[]) => void;
  setActiveServer: (server: MonitorServerOption | null) => void;
  setStatus: (status: ConnectionState) => void;
  setLastUpdated: (value: string | null) => void;
  setRefreshInterval: (ms: number) => void;
  setFilter: (filter: DateRangeFilter | null) => void;
}

export const useDashboardStore = create<DashboardStore>((set, get) => ({
  metrics: null,
  series: [],
  networkDelta: null,
  heartbeats: [],
  alerts: [],
  servers: [],
  activeServer: null,
  status: 'connecting',
  lastUpdated: null,
  refreshInterval: 10_000,
  filter: null,
  setMetrics: (metrics, networkDelta) => {
    set({ metrics, networkDelta: networkDelta ?? null });
  },
  appendSeries: point => {
    const next = [...get().series, point].slice(-90);
    set({ series: next });
  },
  setSeries: points => set({ series: points.slice(-90) }),
  resetSeries: () => set({ series: [] }),
  setHeartbeats: heartbeats => set({ heartbeats: heartbeats ?? [] }),
  setAlerts: alerts => {
    set(state => ({
      alerts: typeof alerts === 'function' ? alerts(state.alerts) : alerts
    }));
  },
  clearAlert: id => set(state => ({ alerts: state.alerts.filter(alert => alert.id !== id) })),
  clearAllAlerts: () => set({ alerts: [] }),
  setServers: servers => set({ servers }),
  setActiveServer: server => set({ activeServer: server }),
  setStatus: status => set({ status }),
  setLastUpdated: value => set({ lastUpdated: value }),
  setRefreshInterval: ms => set({ refreshInterval: ms }),
  setFilter: filter => set({ filter })
}));
