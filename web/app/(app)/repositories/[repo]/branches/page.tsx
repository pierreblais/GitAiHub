import Link from "next/link";
import type { Metadata } from "next";
import { GitBranchIcon, TagIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Ref, Repository } from "@/lib/types";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { qs, shortSha } from "@/lib/format";

export const metadata: Metadata = { title: "Branches" };

function RefRows({ repo, refs, defaultBranch }: { repo: string; refs: Ref[]; defaultBranch?: string }) {
  return (
    <ul className="list-style-none">
      {refs.map((b) => (
        <li key={b.name} className="Box-row d-flex flex-items-center flex-wrap" style={{ gap: 12 }} data-testid="branch">
          <Link href={`/repositories/${repo}/tree${qs({ ref: b.refId })}`} className="branch-name">
            {b.refId}
          </Link>
          {b.refId === defaultBranch ? <span className="Label">default</span> : null}
          {b.target ? (
            <span className="f6 color-fg-muted d-flex flex-items-center flex-wrap" style={{ gap: 6 }}>
              <AgentLink agent={b.target.agent} size={16} verified={b.target.agentVerified} />
              <Link href={`/repositories/${repo}/commit/${b.target.id}`} className="color-fg-muted">
                {b.target.subject}
              </Link>
              <RelativeTime iso={b.target.committer.time} />
            </span>
          ) : null}
          {b.target ? <span className="ml-auto text-mono f6 color-fg-muted">{shortSha(b.target.id)}</span> : null}
        </li>
      ))}
    </ul>
  );
}

export default async function BranchesPage({ params }: { params: Promise<{ repo: string }> }) {
  const { repo } = await params;
  const [r, { branches }, { tags }] = await Promise.all([
    api<Repository>(`/v1/repositories/${enc(repo)}`),
    api<{ branches: Ref[] }>(`/v1/repositories/${enc(repo)}/branches`),
    api<{ tags: Ref[] }>(`/v1/repositories/${enc(repo)}/tags`),
  ]);
  return (
    <div className="gh-container">
      <div className="Box mb-4">
        <div className="Box-header">
          <h2 className="Box-title">
            <GitBranchIcon /> Branches <span className="Counter">{branches.length}</span>
          </h2>
        </div>
        <RefRows repo={repo} refs={branches} defaultBranch={r.defaultBranch} />
      </div>
      <div className="Box">
        <div className="Box-header">
          <h2 className="Box-title">
            <TagIcon /> Tags <span className="Counter">{tags.length}</span>
          </h2>
        </div>
        {tags.length ? <RefRows repo={repo} refs={tags} /> : <p className="p-3 f6 color-fg-muted">No tags.</p>}
      </div>
    </div>
  );
}
