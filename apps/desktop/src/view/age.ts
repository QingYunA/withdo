const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** Compact duration: 5m, 3h, 2d, 6w. */
function compact(ms: number): string {
  if (ms < MINUTE) return "now";
  if (ms < HOUR) return `${Math.floor(ms / MINUTE)}m`;
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h`;
  if (ms < 14 * DAY) return `${Math.floor(ms / DAY)}d`;
  return `${Math.floor(ms / (7 * DAY))}w`;
}

/** How long ago `iso` was. Times in the future (clock skew) read "now". */
export function formatAge(iso: string, now: Date): string {
  return compact(Math.max(0, now.getTime() - Date.parse(iso)));
}

export interface StaleInput {
  readonly stale: boolean;
  /** When the task last saw activity (latest report, else last update). */
  readonly since: string;
}

export function staleBadge({ stale, since }: StaleInput, now: Date): string | null {
  return stale ? `Stale ${formatAge(since, now)}` : null;
}

/** A moment relative to now: "in 2h" or "3h ago". Overdue means clearly in the past. */
export function formatWhen(iso: string, now: Date): { text: string; overdue: boolean } {
  const diff = Date.parse(iso) - now.getTime();
  const span = compact(Math.abs(diff));
  if (span === "now") return { text: "now", overdue: false };
  return diff > 0 ? { text: `in ${span}`, overdue: false } : { text: `${span} ago`, overdue: true };
}
