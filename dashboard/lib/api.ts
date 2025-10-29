import { DateRangeFilter, MonitorServerOption, ServerConfig } from '@/lib/types';
import { sanitizeBaseUrl } from '@/lib/url';

const ENV_BASE = process.env.NEXT_PUBLIC_MONITORING_BASE_URL ?? '';

export function buildEndpoint(path: string, overrideBase?: string | null) {
  const base = sanitizeBaseUrl(overrideBase) || sanitizeBaseUrl(ENV_BASE);
  if (!base) {
    return path.startsWith('/') ? path : `/${path}`;
  }
  if (!path.startsWith('/')) {
    return `${base}/${path}`;
  }
  return `${base}${path}`;
}

export async function fetchMetrics(filter?: DateRangeFilter | null, baseUrl?: string | null) {
  const endpoint = buildEndpoint('/monitoring', baseUrl);
  const payload = {
    from: filter?.from ?? undefined,
    to: filter?.to ?? undefined
  };

  const response = await fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
    cache: 'no-store'
  });

  if (!response.ok) {
    const text = await response.text().catch(() => '');
    throw new Error(`Metrics request failed (${response.status}): ${text}`);
  }

  return response.json() as Promise<unknown>;
}

export async function fetchServerConfig(baseUrl?: string | null): Promise<ServerConfig | null> {
  const endpoint = buildEndpoint('/api/v1/server-config', baseUrl);
  try {
    const response = await fetch(endpoint, { cache: 'no-store' });
    if (!response.ok) {
      return null;
    }
    return (await response.json()) as ServerConfig;
  } catch (error) {
    console.warn('Failed to fetch server config', error);
    return null;
  }
}

export function mapServerConfig(config?: ServerConfig | null): MonitorServerOption[] {
  if (!config?.servers || !Array.isArray(config.servers)) {
    return [];
  }
  return config.servers
    .filter(server => server && server.name)
    .map(server => ({
      name: server.name,
      address: sanitizeBaseUrl(server.address)
    }));
}
