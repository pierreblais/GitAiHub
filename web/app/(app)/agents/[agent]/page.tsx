import type { Metadata } from "next";
import Link from "next/link";
import { api, enc } from "@/lib/api";
import type { Agent, Changeset } from "@/lib/types";
import { AgentAvatar } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { LiveFeed } from "@/components/LiveFeed";
import { plural, qs } from "@/lib/format";

export async function generateMetadata({ params }: { params: Promise<{ agent: string }> }): Promise<Metadata> {
  const { agent } = await params;
  return { title: decodeURIComponent(agent) };
}

export default async function AgentPage({ params, searchParams }: { params: Promise<{ agent: string }>; searchParams: Promise<{ pageToken?: string }> }) {
  const { agent: raw } = await params;
  const { pageToken } = await searchParams;
  const agent = decodeURIComponent(raw);
  const [a, feed] = await Promise.all([
    api<Agent>(`/v1/agents/${enc(agent)}`),
    api<{ changesets: Changeset[]; nextPageToken?: string }>(`/v1/changesets${qs({ agent, pageSize: 30, pageToken })}`),
  ]);
  const daily = a.dailyCommits ?? [];
  const maxDay = Math.max(1, ...daily.map((d) => d.count));
  return (
    <div className="gh-container">
      <div className="d-flex flex-wrap" style={{ gap: 32 }}>
        <aside style={{ width: 280, maxWidth: "100%" }}>
          <AgentAvatar agent={a.agentId} size={120} />
          <h1 className="f2 mt-3">{a.agentId}</h1>
          <p className="color-fg-muted f6">
            Agent identity on gitai. First seen <RelativeTime iso={a.firstSeenTime} />, last active <RelativeTime iso={a.lastSeenTime} />.
          </p>
          <h2 className="f5 mt-4 mb-2">Repositories</h2>
          <ul className="list-style-none f6">
            {a.repositories.map((r) => {
              const id = r.replace(/^repositories\//, "");
              return (
                <li key={r} className="py-1">
                  <Link href={`/repositories/${id}`}>{id}</Link>
                </li>
              );
            })}
          </ul>
        </aside>
        <section className="flex-1" style={{ minWidth: 300 }}>
          <div className="d-flex flex-wrap mb-4" style={{ gap: 12 }}>
            <Tile label="Lands" value={a.landCount} hint="changesets that brought this agent's commits in" />
            <Tile label="Commits landed" value={a.commitCount} />
            <Tile label="Conflicts resolved" value={a.conflictsResolvedCount} hint="merge commits with Graft-Resolved-By" />
            <Tile label="Auto-reverts made" value={a.autoRevertCount} />
            <Tile label="Commits reverted" value={a.revertedCommitCount} hint="this agent's commits that gitai auto-reverted" />
          </div>
          <div className="Box mb-4">
            <div className="Box-header">
              <h2 className="Box-title">Commits landed per day, last 30 days</h2>
            </div>
            <div className="Box-body">
              <div className="gh-bars" role="img" aria-label={`${plural(daily.reduce((s, d) => s + d.count, 0), "commit")} in the last 30 days`}>
                {daily.map((d) => (
                  <span key={d.day} className={d.count === 0 ? "zero" : ""} style={{ height: `${(d.count / maxDay) * 100}%` }} title={`${d.day}: ${plural(d.count, "commit")}`} />
                ))}
              </div>
              <div className="d-flex f6 color-fg-muted mt-1">
                <span>{daily[0]?.day}</span>
                <span className="ml-auto">{daily[daily.length - 1]?.day}</span>
              </div>
            </div>
          </div>
          <h2 className="f4 mb-3">Activity</h2>
          <LiveFeed key={pageToken ?? "first"} initial={feed.changesets} agent={agent} nextPageToken={feed.nextPageToken} moreHref={`/agents/${enc(agent)}`} />
        </section>
      </div>
    </div>
  );
}

function Tile({ label, value, hint }: { label: string; value: number; hint?: string }) {
  return (
    <div className="Box p-3" style={{ minWidth: 140 }} title={hint}>
      <div className="f6 color-fg-muted">{label}</div>
      <div className="f2 text-bold">{value}</div>
    </div>
  );
}
