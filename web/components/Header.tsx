"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import {
  ThreeBarsIcon,
  XIcon,
  HomeIcon,
  PulseIcon,
  PeopleIcon,
  RepoIcon,
  SearchIcon,
  SignOutIcon,
  CodeReviewIcon,
} from "@primer/octicons-react";
import { LogoMark } from "./Logo";
import { AgentAvatar } from "./Agent";
import type { Repository, User } from "@/lib/types";

/** The page context shown next to the logo, like "Dashboard" or "gitai / orbit". */
function context(pathname: string): { owner?: string; title: string; href: string } {
  const repo = pathname.match(/^\/repositories\/([^/]+)/);
  if (repo) return { owner: "gitai", title: decodeURIComponent(repo[1]), href: `/repositories/${repo[1]}` };
  if (pathname.startsWith("/activity")) return { title: "Activity", href: "/activity" };
  if (pathname.startsWith("/agents")) return { title: "Agents", href: "/agents" };
  return { title: "Dashboard", href: "/" };
}

export function Header({ user, repositories }: { user: User; repositories: Repository[] }) {
  const pathname = usePathname();
  const router = useRouter();
  const [drawer, setDrawer] = useState(false);
  const [menu, setMenu] = useState(false);
  const [showAll, setShowAll] = useState(false);
  const search = useRef<HTMLInputElement>(null);
  const ctx = context(pathname);

  // "/" focuses the search box, as on GitHub.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (e.key === "/" && !["INPUT", "TEXTAREA"].includes(t.tagName) && !t.isContentEditable) {
        e.preventDefault();
        search.current?.focus();
      }
      if (e.key === "Escape") {
        setDrawer(false);
        setMenu(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  useEffect(() => {
    setDrawer(false);
    setMenu(false);
  }, [pathname]);

  const top = showAll ? repositories : repositories.slice(0, 5);
  const nav = [
    { href: "/", label: "Home", icon: HomeIcon },
    { href: "/activity", label: "Activity", icon: PulseIcon },
    { href: "/agents", label: "Agents", icon: PeopleIcon },
    { href: "/repositories", label: "All repositories", icon: RepoIcon },
  ];

  return (
    <header className="gh-header" role="banner">
      <div className="gh-header-left">
        <button type="button" className="gh-icon-btn" aria-label="Open global navigation menu" aria-expanded={drawer} onClick={() => setDrawer(true)}>
          <ThreeBarsIcon />
        </button>
        <Link href="/" aria-label="GitaiHub home" className="gh-header-logo">
          <LogoMark size={32} />
        </Link>
        <nav aria-label="Page context" className="gh-context">
          {ctx.owner ? (
            <>
              <span className="gh-context-owner">{ctx.owner}</span>
              <span className="gh-context-sep">/</span>
            </>
          ) : null}
          <Link href={ctx.href} className="gh-context-title">
            {ctx.title}
          </Link>
        </nav>
      </div>

      <div className="gh-header-right">
        <form action="/repositories" method="get" role="search" className="gh-search">
          <label htmlFor="gh-global-search" className="sr-only">
            Search repositories
          </label>
          <SearchIcon className="gh-search-icon" />
          <input ref={search} id="gh-global-search" name="q" type="search" placeholder="Type / to search" autoComplete="off" />
        </form>
        <span className="gh-header-divider gh-hide-sm" aria-hidden="true" />
        <Link href="/activity" className="gh-icon-btn gh-hide-sm" aria-label="Activity" title="Activity">
          <PulseIcon />
        </Link>
        <Link href="/reviews" className="gh-icon-btn gh-hide-sm" aria-label="Reviews" title="Reviews">
          <CodeReviewIcon />
        </Link>
        <Link href="/repositories" className="gh-icon-btn gh-hide-sm" aria-label="Repositories" title="Repositories">
          <RepoIcon />
        </Link>
        <div className="gh-user-menu">
          <button type="button" className="gh-avatar-btn" aria-label="Open user navigation menu" aria-expanded={menu} onClick={() => setMenu((m) => !m)}>
            <AgentAvatar agent={user.username} size={32} round />
          </button>
          {menu ? (
            <div className="gh-overlay-menu" role="menu">
              <div className="d-flex flex-items-center px-3 py-2" style={{ gap: 8 }}>
                <AgentAvatar agent={user.username} size={32} round />
                <div>
                  <div className="text-bold">{user.username}</div>
                  <div className="f6 color-fg-muted">{user.canReview ? "Can review as " + user.username : "Read-only"}</div>
                </div>
              </div>
              <div className="gh-menu-sep" />
              <button
                type="button"
                role="menuitem"
                className="gh-menu-item"
                onClick={async () => {
                  await fetch("/api/v1/sessions/current", { method: "DELETE" });
                  router.push("/login");
                  router.refresh();
                }}
              >
                <SignOutIcon /> Sign out
              </button>
            </div>
          ) : null}
        </div>
      </div>

      {drawer ? (
        <>
          <div className="gh-backdrop" onClick={() => setDrawer(false)} aria-hidden="true" />
          <aside className="gh-drawer" aria-label="Global navigation">
            <div className="d-flex flex-items-center flex-justify-between mb-3">
              <LogoMark size={32} />
              <button type="button" className="gh-icon-btn gh-icon-btn-plain" aria-label="Close navigation" onClick={() => setDrawer(false)}>
                <XIcon />
              </button>
            </div>
            <ul className="gh-side-nav">
              {nav.map((n) => {
                const Icon = n.icon;
                const active = n.href === "/" ? pathname === "/" : pathname.startsWith(n.href);
                return (
                  <li key={n.href}>
                    <Link href={n.href} aria-current={active ? "page" : undefined}>
                      <Icon /> {n.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
            <div className="gh-menu-sep my-3" />
            <h2 className="f6 color-fg-muted px-2 mb-2">Top repositories</h2>
            <ul className="gh-side-nav">
              {top.map((r) => (
                <li key={r.name}>
                  <Link href={`/repositories/${r.repositoryId}`}>
                    <RepoIcon /> gitai/{r.repositoryId}
                  </Link>
                </li>
              ))}
            </ul>
            {repositories.length > 5 && !showAll ? (
              <button type="button" className="btn-link f6 color-fg-muted px-2 mt-2" onClick={() => setShowAll(true)}>
                Show more
              </button>
            ) : null}
          </aside>
        </>
      ) : null}
    </header>
  );
}
