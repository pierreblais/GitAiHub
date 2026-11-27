import type { Metadata } from "next";
import Link from "next/link";
import { RepoIcon } from "@primer/octicons-react";
import { api } from "@/lib/api";
import type { Repository } from "@/lib/types";
import { AgentAvatar } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { Blankslate } from "@/components/Blankslate";
import { plural } from "@/lib/format";

export const metadata: Metadata = { title: "Repositories" };

export default async function RepositoriesPage({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { q } = await searchParams;
  const { repositories } = await api<{ repositories: Repository[] }>("/v1/repositories?pageSize=100");
  const filter = (q ?? "").trim().toLowerCase();
  const repos = filter ? repositories.filter((r) => r.repositoryId.includes(filter) || r.relativePath.toLowerCase().includes(filter)) : repositories;
  return (
    <div className="gh-container" style={{ maxWidth: 1012 }}>
      <h1 className="f3 text-normal mb-3">Repositories</h1>
      <form action="/repositories" method="get" className="mb-3 d-flex" style={{ gap: 8 }}>
        <label htmlFor="repo-filter" className="sr-only">
          Find a repository
        </label>
        <input id="repo-filter" name="q" type="search" defaultValue={q} className="form-control flex-1" placeholder="Find a repository…" />
        <button type="submit" className="btn">
          Search
        </button>
      </form>
      {filter ? (
        <p className="f6 color-fg-muted mb-3">
          {plural(repos.length, "result")} for “{filter}” · <Link href="/repositories">Clear filter</Link>
        </p>
      ) : null}
      {repos.length === 0 ? (
        <div className="Box">
          <Blankslate icon={<RepoIcon size={24} />} title={filter ? "No repository matches" : "No repositories yet"} />
        </div>
      ) : (
        <ul className="list-style-none border-top">
          {repos.map((r) => (
            <li key={r.name} className="py-4 border-bottom" data-testid="repository">
              <div className="d-flex flex-items-center" style={{ gap: 8 }}>
                <Link href={`/repositories/${r.repositoryId}`} className="f3 text-bold">
                  {r.repositoryId}
                </Link>
                <span className="Label">gitai</span>
              </div>
              <p className="f6 color-fg-muted mt-1 mb-2 text-mono">{r.relativePath}</p>
              <div className="d-flex flex-items-center flex-wrap f6 color-fg-muted" style={{ gap: 16 }}>
                {r.topAgents.length > 0 ? (
                  <span className="d-inline-flex flex-items-center" style={{ gap: 2 }} aria-label="Most active agents">
                    {r.topAgents.map((a) => (
                      <AgentAvatar key={a.agent} agent={a.agent} size={20} />
                    ))}
                  </span>
                ) : null}
                <span>{plural(r.changesetCount, "changeset")}</span>
                {r.lastActivityTime ? (
                  <span>
                    Updated <RelativeTime iso={r.lastActivityTime} />
                  </span>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
