export function Blankslate({ icon, title, children }: { icon?: React.ReactNode; title: string; children?: React.ReactNode }) {
  return (
    <div className="blankslate">
      {icon ? <div className="color-fg-muted mb-2">{icon}</div> : null}
      <h3 className="blankslate-heading">{title}</h3>
      {children ? <p>{children}</p> : null}
    </div>
  );
}
