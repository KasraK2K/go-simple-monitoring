const RANGE_PRESETS_MS = Object.freeze({
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000
});

export function isValidRangePreset(range) {
  if (!range) {
    return false;
  }
  return Object.prototype.hasOwnProperty.call(RANGE_PRESETS_MS, range);
}

export function getRangeDurationMs(range) {
  if (!isValidRangePreset(range)) {
    return null;
  }
  return RANGE_PRESETS_MS[range];
}

export function buildFilterFromRange(range) {
  const durationMs = getRangeDurationMs(range);
  if (!durationMs) {
    return null;
  }
  
  // Force UTC calculation to ensure consistency across timezones
  const now = new Date();
  const nowUtc = new Date(now.getTime() - (now.getTimezoneOffset() * 60000));
  const fromUtc = new Date(nowUtc.getTime() - durationMs);
  
  return {
    from: fromUtc.toISOString(),
    to: nowUtc.toISOString()
  };
}

export function getAllowedRangePresets() {
  return Object.keys(RANGE_PRESETS_MS);
}
