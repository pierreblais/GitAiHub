// GitaiHub's own mark: three agent nodes landing on one trunk, in a rounded
// hexagon. Original artwork, not derived from any other product's logo.
export function LogoMark({ size = 32, title = "GitaiHub" }: { size?: number; title?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" role="img" aria-label={title}>
      <title>{title}</title>
      <path d="M16 1.8 28.3 8.9v14.2L16 30.2 3.7 23.1V8.9Z" fill="url(#gh-g)" />
      <path d="M16 8v16M16 13l-5.5-3.2M16 18l5.5-3.2" stroke="#fff" strokeWidth="2" strokeLinecap="round" fill="none" />
      <circle cx="16" cy="24" r="2.6" fill="#fff" />
      <circle cx="10" cy="9.5" r="2.2" fill="#fff" />
      <circle cx="22" cy="14.5" r="2.2" fill="#fff" />
      <circle cx="16" cy="8" r="2.2" fill="#fff" />
      <defs>
        <linearGradient id="gh-g" x1="4" y1="2" x2="28" y2="30" gradientUnits="userSpaceOnUse">
          <stop stopColor="#a78bfa" />
          <stop offset="1" stopColor="#6d28d9" />
        </linearGradient>
      </defs>
    </svg>
  );
}

export function Logo() {
  return (
    <span className="d-inline-flex flex-items-center gh-logo">
      <LogoMark size={32} />
      <span className="ml-2 f4 text-bold gh-logo-text">
        Gitai<span className="gh-logo-hub">Hub</span>
      </span>
    </span>
  );
}
