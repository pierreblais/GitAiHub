import Link from "next/link";
import { RepoIcon, HomeIcon, PulseIcon, PeopleIcon, CpuIcon, CodeReviewIcon } from "@primer/octicons-react";
import { api } from "@/lib/api";
import type { Agent, Changeset, Repository, User } from "@/lib/types";
import { AgentAvatar } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { ChangesetItem } from "@/components/ChangesetItem";
import { ReviewRow } from "@/components/ReviewRow";
import { Greeting } from "@/components/Greeting";
import { reviewsAcross } from "@/lib/reviews";
import { plural } from "@/lib/format";

export default async function Dashboard({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { q } = await searchParams;
  const [user, { repositories }, { changesets }, { agents }] = await Promise.all([
    api<User>("/v1/users/me"),
    api<{ repositories: Repository[] }>("/v1/repositories?pageSize=100"),
    api<{ changesets: Changeset[] }>("/v1/changesets?pageSize=10"),
    api<{ agents: Agent[] }>("/v1/agents"),
  ]);
  const reviews = await reviewsAcross(repositories);
  const filter = (q ?? "").trim().toLowerCase();
  const repos = filter ? repositories.filter((r) => r.repositoryId.includes(filter)) : repositories;
  const recentAgents = [...agents].sort((a, b) => (b.lastSeenTime ?? "").localeCompare(a.lastSeenTime ?? "")).slice(0, 5);

  return (
    <div className="gh-dashboard">
      <aside className="gh-dash-left" aria-label="Your account and repositories">
        <div className="d-flex flex-items-center mb-3" style={{ gap: 12 }}>
          <AgentAvatar agent={user.username} size={40} round />
          <div>
            <div className="text-bold">{user.username}</div>
            <div className="f6 color-fg-muted">Your account</div>
          </div>
        </div>
        <ul className="gh-side-nav mb-3">
          <li>
            <Link href="/" aria-current="page">
              <HomeIcon /> Home
            </Link>
          </li>
          <li>
            <Link href="/activity">
              <PulseIcon /> Activity
            </Link>
          </li>
          <li>
            <Link href="/reviews">
              <CodeReviewIcon /> Reviews
            </Link>
          </li>
          <li>
            <Link href="/agents">
              <PeopleIcon /> Agents
            </Link>
          </li>
        </ul>
        <div className="gh-menu-sep mb-3" />
        <div className="d-flex flex-items-center mb-2">
          <h2 className="f5 text-bold color-fg-muted flex-1">Top repositories</h2>
        </div>
        <form action="/" method="get" className="mb-2">
          <label htmlFor="dash-repo-filter" className="sr-only">
            Find a repository
          </label>
          <input id="dash-repo-filter" name="q" type="search" defaultValue={q} className="form-control input-sm width-full" placeholder="Find a repository…" />
        </form>
        <ul className="list-style-none" data-testid="top-repositories">
          {repos.map((r) => (
            <li key={r.name} className="py-1">
              <Link href={`/repositories/${r.repositoryId}`} className="d-flex flex-items-center color-fg-default text-bold f5" style={{ gap: 8 }}>
                <RepoIcon className="color-fg-muted" />
                <span className="gh-truncate">gitai/{r.repositoryId}</span>
              </Link>
            </li>
          ))}
          {repos.length === 0 ? <li className="f6 color-fg-muted">No repository matches.</li> : null}
        </ul>
      </aside>

      <section className="gh-dash-main" aria-labelledby="greeting">
        <h1 id="greeting" className="gh-greeting">
          <CpuIcon size={24} /> <Greeting name={user.username} />
        </h1>

        <div className="d-flex flex-items-center mb-2 mt-4">
          <h2 className="f5 color-fg-muted flex-1">Open reviews</h2>
          <Link href="/reviews" className="f6">
            View all
          </Link>
        </div>
        {reviews.length === 0 ? (
          <div className="Box p-4 f6 color-fg-muted">No review is waiting.</div>
        ) : (
          <ul className="Box list-style-none mb-4">
            {reviews.slice(0, 5).map((rv) => (
              <ReviewRow key={rv.name} repo={rv.repositoryId} review={rv} showRepo />
            ))}
          </ul>
        )}

        <div className="d-flex flex-items-center mb-3 mt-4">
          <h2 className="f5 color-fg-muted flex-1">Recent agent activity</h2>
          <Link href="/activity" className="f6">
            View all
          </Link>
        </div>
        {changesets.length === 0 ? (
          <p className="color-fg-muted f6">Nothing landed yet.</p>
        ) : (
          changesets.map((cs) => <ChangesetItem key={cs.name} cs={cs} />)
        )}
      </section>

      <aside className="gh-dash-right" aria-labelledby="agents-heading">
        <div className="Box p-3">
          <h2 id="agents-heading" className="f5 text-bold mb-3">
            Agents at work
          </h2>
          <ul className="gh-timeline list-style-none">
            {recentAgents.map((a) => (
              <li key={a.name}>
                <div className="f6 color-fg-muted">
                  <RelativeTime iso={a.lastSeenTime} />
                </div>
                <Link href={`/agents/${a.agentId}`} className="color-fg-default d-flex flex-items-center" style={{ gap: 6 }}>
                  <AgentAvatar agent={a.agentId} size={16} />
                  <strong>{a.agentId}</strong>
                </Link>
                <div className="f6 color-fg-muted">
                  {plural(a.landCount, "land")} · {plural(a.commitCount, "commit")}
                  {a.conflictsResolvedCount ? ` · ${plural(a.conflictsResolvedCount, "conflict")} resolved` : ""}
                </div>
              </li>
            ))}
          </ul>
          <Link href="/agents" className="f6 color-fg-muted">
            View all agents →
          </Link>
        </div>
      </aside>
    </div>
  );
}
