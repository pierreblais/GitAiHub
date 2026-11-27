import { absoluteTime, relativeTime } from "@/lib/format";

export function RelativeTime({ iso }: { iso?: string }) {
  if (!iso) return null;
  return (
    <time dateTime={iso} title={absoluteTime(iso)} suppressHydrationWarning>
      {relativeTime(iso)}
    </time>
  );
}
