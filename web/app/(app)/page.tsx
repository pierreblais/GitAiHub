import Link from "next/link";
import { RepoIcon, PulseIcon } from "@primer/octicons-react";
import { api } from "@/lib/api";
import type { Changeset, Repository } from "@/lib/types";
import { AgentAvatar } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { ChangesetItem } from "@/components/ChangesetItem";
import { Blankslate } from "@/components/Blankslate";
import { plural } from "@/lib/format";

export default async function Dashboard({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { q } = await searchParams;
  const [{ repositories }, { changesets }] = await Promise.all([
    api<{ repositories: Repository[] }>("/v1/repositories?pageSize=100"),
    api<{ changesets: Changeset[] }>("/v1/changesets?pageSize=8"),
  ]);
  const filter = (q ?? "").trim().toLowerCase();
  const repos = filter ? repositories.filter((r) => r.repositoryId.includes(filter)) : repositories;
  return (
    <div className="gh-container">
      <div className="d-flex flex-wrap" style={{ gap: 24 }}>
        <section className="flex-1" style={{ minWidth: 300 }} aria-labelledby="repos-heading">
          <div className="d-flex flex-items-center mb-3">
            <h1 id="repos-heading" className="f3 text-normal">
              Repositories <span className="Counter">{repositories.length}</span>
            </h1>
            {filter ? (
              <span className="ml-3 f6 color-fg-muted">
                matching “{filter}” · <Link href="/">clear</Link>
              </span>
            ) : null}
          </div>
          {repos.length === 0 ? (
            <div className="Box">
              <Blankslate icon={<RepoIcon size={24} />} title={filter ? "No repository matches" : "No repositories yet"}>
                {filter ? "Try another name." : "GitaiHub lists the repositories of your gitai server. Create one with `graft repo create`."}
              </Blankslate>
            </div>
          ) : (
            <ul className="Box list-style-none">
              {repos.map((r) => (
                <li key={r.name} className="Box-row" data-testid="repository">
                  <div className="d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
                    <RepoIcon className="color-fg-muted" />
                    <Link href={`/repositories/${r.repositoryId}`} className="f4 text-bold">
                      {r.repositoryId}
                    </Link>
                    <span className="Label">gitai</span>
                  </div>
                  <div className="d-flex flex-items-center flex-wrap mt-2 f6 color-fg-muted" style={{ gap: 16 }}>
                    <span className="text-mono">{r.relativePath}</span>
                    <span>{plural(r.changesetCount, "changeset")}</span>
                    {r.lastActivityTime ? (
                      <span>
                        Updated <RelativeTime iso={r.lastActivityTime} />
                      </span>
                    ) : (
                      <span>No activity yet</span>
                    )}
                    {r.topAgents.length > 0 ? (
                      <span className="d-inline-flex flex-items-center" style={{ gap: 4 }} aria-label="Most active agents">
                        {r.topAgents.map((a) => (
                          <Link key={a.agent} href={`/agents/${a.agent}`} title={`${a.agent}: ${plural(a.count, "land")}`}>
                            <AgentAvatar agent={a.agent} size={20} />
                          </Link>
                        ))}
                      </span>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
        <aside style={{ width: 400, maxWidth: "100%" }} aria-labelledby="recent-heading">
          <div className="d-flex flex-items-center mb-3">
            <h2 id="recent-heading" className="f4">
              <PulseIcon /> Recent agent activity
            </h2>
            <Link href="/activity" className="ml-auto f6">
              Live feed →
            </Link>
          </div>
          {changesets.length === 0 ? (
            <p className="color-fg-muted f6">Nothing landed yet.</p>
          ) : (
            changesets.map((cs) => <ChangesetItem key={cs.name} cs={cs} />)
          )}
        </aside>
      </div>
    </div>
  );
}
