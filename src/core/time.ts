/**
 * Source map in this file:
 * - stepTimestamp: adapted from rajiko/modules/timeshift.js (seek)
 * - parseTimestamp/formatTimestamp: new CLI helpers
 */
/**
 * Source: New helper for timestamp parsing in CLI.
 */
export function parseTimestamp(ts: string): Date {
  const m = /^(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})$/.exec(ts);
  if (!m) {
    throw new Error(`Invalid timestamp: ${ts}`);
  }
  const [, y, mo, d, h, mi, s] = m;
  return new Date(
    Number(y),
    Number(mo) - 1,
    Number(d),
    Number(h),
    Number(mi),
    Number(s),
    0,
  );
}

/**
 * Source: New helper for timestamp formatting in CLI.
 */
export function formatTimestamp(date: Date): string {
  const y = date.getFullYear().toString().padStart(4, "0");
  const m = (date.getMonth() + 1).toString().padStart(2, "0");
  const d = date.getDate().toString().padStart(2, "0");
  const h = date.getHours().toString().padStart(2, "0");
  const mi = date.getMinutes().toString().padStart(2, "0");
  const s = date.getSeconds().toString().padStart(2, "0");
  return `${y}${m}${d}${h}${mi}${s}`;
}

/**
 * Source: rajiko/modules/timeshift.js -> seek (adapted)
 */
export function stepTimestamp(ts: string, seconds: number): string {
  const date = parseTimestamp(ts);
  date.setSeconds(date.getSeconds() + seconds);
  return formatTimestamp(date);
}
