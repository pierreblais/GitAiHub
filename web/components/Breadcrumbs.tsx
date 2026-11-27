import Link from "next/link";
import { pathParts, qs } from "@/lib/format";

export function Breadcrumbs({ repo, gitRef, path, leafIsFile }: { repo: string; gitRef: string; path: string; leafIsFile?: boolean }) {
  const parts = pathParts(path);
  return (
    <nav aria-label="Breadcrumb" className="f4 text-mono d-flex flex-wrap flex-items-center" style={{ gap: 4 }}>
      <Link href={`/repositories/${repo}/tree${qs({ ref: gitRef })}`} className="text-bold">
        {repo}
      </Link>
      {parts.map((p, i) => (
        <span key={p.path}>
          <span className="color-fg-muted mx-1">/</span>
          {i === parts.length - 1 ? (
            <strong aria-current="page">{p.name}</strong>
          ) : (
            <Link href={`/repositories/${repo}/tree${qs({ ref: gitRef, path: p.path })}`}>{p.name}</Link>
          )}
        </span>
      ))}
      {!leafIsFile && parts.length > 0 ? <span className="color-fg-muted">/</span> : null}
    </nav>
  );
}
