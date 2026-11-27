import Link from "next/link";
import { Logo } from "./Logo";
import { AgentAvatar } from "./Agent";
import { SignOutButton } from "./SignOutButton";
import type { User } from "@/lib/types";

export function Header({ user }: { user: User }) {
  return (
    <header className="gh-header" role="banner">
      <Link href="/" aria-label="GitaiHub home">
        <Logo />
      </Link>
      <form action="/" method="get" role="search" className="flex-1" style={{ maxWidth: 320 }}>
        <label htmlFor="gh-repo-filter" className="sr-only">
          Find a repository
        </label>
        <input id="gh-repo-filter" name="q" type="search" className="form-control input-sm" placeholder="Find a repository…" />
      </form>
      <nav className="gh-nav d-flex flex-items-center" aria-label="Global">
        <Link href="/">Repositories</Link>
        <Link href="/activity">Activity</Link>
        <Link href="/agents">Agents</Link>
      </nav>
      <div className="d-flex flex-items-center ml-auto" style={{ gap: 12 }}>
        <span className="d-inline-flex flex-items-center" style={{ gap: 6 }} title={user.canReview ? "Can comment and give verdicts on reviews" : "Read-only"}>
          <AgentAvatar agent={user.username} size={24} />
          <span className="gh-muted f6 gh-hide-sm">{user.displayName}</span>
        </span>
        <SignOutButton />
      </div>
    </header>
  );
}
