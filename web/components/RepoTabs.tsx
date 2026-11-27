"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { CodeIcon, GitCommitIcon, CodeReviewIcon, PulseIcon, GitBranchIcon, SearchIcon } from "@primer/octicons-react";

export function RepoTabs({ repo, openReviews, branches }: { repo: string; openReviews: number; branches: number }) {
  const pathname = usePathname();
  const base = `/repositories/${repo}`;
  const tabs = [
    { href: base, label: "Code", icon: CodeIcon, match: (p: string) => p === base || /\/(tree|blob|blame)$/.test(p) },
    { href: `${base}/reviews`, label: "Reviews", icon: CodeReviewIcon, count: openReviews, match: (p: string) => p.startsWith(`${base}/reviews`) },
    { href: `${base}/commits`, label: "Commits", icon: GitCommitIcon, match: (p: string) => p.startsWith(`${base}/commit`) },
    { href: `${base}/activity`, label: "Activity", icon: PulseIcon, match: (p: string) => p.startsWith(`${base}/activity`) },
    { href: `${base}/branches`, label: "Branches", icon: GitBranchIcon, count: branches, match: (p: string) => p.startsWith(`${base}/branches`) },
    { href: `${base}/search`, label: "Search", icon: SearchIcon, match: (p: string) => p.startsWith(`${base}/search`) },
  ];
  return (
    <nav className="UnderlineNav" aria-label="Repository">
      <div className="UnderlineNav-body">
        {tabs.map((t) => {
          const active = t.match(pathname);
          const Icon = t.icon;
          return (
            <Link key={t.href} href={t.href} className="UnderlineNav-item" aria-current={active ? "page" : undefined}>
              <Icon className="UnderlineNav-octicon" />
              <span>{t.label}</span>
              {t.count ? <span className="Counter">{t.count}</span> : null}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
