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
  
  // Get current time in user's local timezone
  const now = new Date();
  
  // Calculate from time by subtracting duration
  const fromTime = new Date(now.getTime() - durationMs);
  
  // JavaScript's toISOString() automatically converts to UTC
  return {
    from: fromTime.toISOString(),
    to: now.toISOString()
  };
}

export function getAllowedRangePresets() {
  return Object.keys(RANGE_PRESETS_MS);
}
