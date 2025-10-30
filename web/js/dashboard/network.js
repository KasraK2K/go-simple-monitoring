import { parseTimestamp, toFiniteNumber } from './utils.js';

export function deriveCounterDelta(current, previous, hasPrevious) {
  if (!Number.isFinite(current)) return 0;
  if (!hasPrevious || !Number.isFinite(previous)) return 0;
  const diff = current - previous;
  if (diff < 0) {
    return Math.max(current, 0);
  }
  return diff;
}

export function calculateNetworkDelta(current, previous) {
  const hasPrevious = Boolean(previous && previous.network);
  const currentNetwork = current?.network || {};
  const previousNetwork = previous?.network || {};

  const currentRx = toFiniteNumber(currentNetwork.bytes_received);
  const currentTx = toFiniteNumber(currentNetwork.bytes_sent);
  const previousRx = toFiniteNumber(previousNetwork.bytes_received);
  const previousTx = toFiniteNumber(previousNetwork.bytes_sent);

  const currentTime = parseTimestamp(current?.timestamp);
  const previousTime = parseTimestamp(previous?.timestamp);
  let durationSeconds = null;
  if (currentTime && previousTime) {
    const diffMs = currentTime.getTime() - previousTime.getTime();
    if (Number.isFinite(diffMs) && diffMs > 0) {
      durationSeconds = diffMs / 1000;
    }
  }

  return {
    bytes_received: deriveCounterDelta(currentRx, previousRx, hasPrevious),
    bytes_sent: deriveCounterDelta(currentTx, previousTx, hasPrevious),
    durationSeconds
  };
}
