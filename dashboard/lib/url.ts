export function sanitizeBaseUrl(url?: string | null): string {
  if (!url) return '';
  try {
    const trimmed = url.trim();
    if (!trimmed) return '';
    return trimmed.replace(/\/+$/, '');
  } catch {
    return '';
  }
}
