const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 24 * 3600],
  ["month", 30 * 24 * 3600],
  ["week", 7 * 24 * 3600],
  ["day", 24 * 3600],
  ["hour", 3600],
  ["minute", 60],
  ["second", 1],
];

const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

/** "3 minutes ago", relative to now. */
export function relativeTime(iso: string | undefined, now: Date = new Date()): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const diff = Math.round((t - now.getTime()) / 1000);
  const abs = Math.abs(diff);
  if (abs < 30) return "just now";
  for (const [unit, secs] of UNITS) {
    if (abs >= secs) return rtf.format(Math.round(diff / secs), unit);
  }
  return "just now";
}

export function absoluteTime(iso: string | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  return d.toLocaleString("en-GB", { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }) + " UTC";
}

export function shortSha(id: string | undefined): string {
  return (id ?? "").slice(0, 7);
}

export function formatBytes(n: number | string): string {
  const v = typeof n === "string" ? Number(n) : n;
  if (!Number.isFinite(v)) return "";
  if (v < 1024) return `${v} Bytes`;
  if (v < 1024 * 1024) return `${(v / 1024).toFixed(1)} KB`;
  return `${(v / 1024 / 1024).toFixed(1)} MB`;
}

export function plural(n: number, one: string, many = one + "s"): string {
  return `${n} ${n === 1 ? one : many}`;
}

/** Day heading for commit lists: "Commits on Nov 27, 2026". */
export function dayKey(iso: string | undefined): string {
  if (!iso) return "Unknown date";
  return new Date(iso).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
}

/** Splits "a/b/c" into breadcrumb parts with their cumulative paths. */
export function pathParts(path: string): { name: string; path: string }[] {
  const parts = path.split("/").filter(Boolean);
  return parts.map((name, i) => ({ name, path: parts.slice(0, i + 1).join("/") }));
}

/** Builds a query string, leaving out empty values. */
export function qs(params: Record<string, string | number | undefined | null>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
  }
  const s = sp.toString();
  return s ? `?${s}` : "";
}
