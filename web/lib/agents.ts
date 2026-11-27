// Every agent gets a stable colour, used in blame, badges and the feed.
const PALETTE = [
  "#8250df", "#0969da", "#1a7f37", "#bf3989", "#bc4c00", "#0a7f8c", "#9a6700", "#cf222e", "#6639ba", "#116329",
];

export function hashString(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

export function agentColor(agent: string): string {
  return PALETTE[hashString(agent) % PALETTE.length];
}

export function agentInitials(agent: string): string {
  const parts = agent.split(/[-_./\s]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return agent.slice(0, 2).toUpperCase();
}
