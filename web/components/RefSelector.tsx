import Link from "next/link";
import { GitBranchIcon, TagIcon, ChevronDownIcon, CheckIcon } from "@primer/octicons-react";
import type { Ref } from "@/lib/types";

/**
 * Branch and tag switcher. A <details> menu: it works without JavaScript and
 * keeps the page server-rendered.
 */
export function RefSelector({
  current,
  branches,
  tags,
  hrefFor,
}: {
  current: string;
  branches: Ref[];
  tags: Ref[];
  hrefFor: (ref: string) => string;
}) {
  const isTag = tags.some((t) => t.refId === current);
  return (
    <details className="gh-ref-menu">
      <summary className="btn btn-sm" aria-haspopup="menu">
        {isTag ? <TagIcon /> : <GitBranchIcon />} <span className="text-bold ml-1">{current}</span> <ChevronDownIcon />
      </summary>
      <div className="gh-ref-list" role="menu">
        <div className="px-3 py-2 text-bold f6">Branches</div>
        {branches.map((b) => (
          <Link key={b.refId} href={hrefFor(b.refId)} role="menuitem">
            <span className="d-inline-block" style={{ width: 20 }}>
              {b.refId === current ? <CheckIcon /> : null}
            </span>
            {b.refId}
          </Link>
        ))}
        {tags.length > 0 ? <div className="px-3 py-2 text-bold f6 border-top">Tags</div> : null}
        {tags.map((t) => (
          <Link key={t.refId} href={hrefFor(t.refId)} role="menuitem">
            <span className="d-inline-block" style={{ width: 20 }}>
              {t.refId === current ? <CheckIcon /> : null}
            </span>
            {t.refId}
          </Link>
        ))}
      </div>
    </details>
  );
}
