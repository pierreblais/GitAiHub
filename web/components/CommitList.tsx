import Link from "next/link";
import { GitCommitIcon, SparkleFillIcon, UndoIcon } from "@primer/octicons-react";
import type { Commit } from "@/lib/types";
import { AgentLink } from "./Agent";
import { RelativeTime } from "./Time";
import { dayKey, shortSha } from "@/lib/format";

export function CommitBadges({ c }: { c: Commit }) {
  return (
    <>
      {c.resolvedBy ? (
        <span className="Label Label--attention" title={`Graft-Resolved-By: ${c.resolvedBy}`}>
          <SparkleFillIcon size={12} /> AI-resolved conflict
        </span>
      ) : null}
      {c.autoRevertOf ? (
        <span className="Label Label--danger" title={`Graft-Auto-Revert: ${c.autoRevertOf}`}>
          <UndoIcon size={12} /> auto-revert of {shortSha(c.autoRevertOf)}
        </span>
      ) : null}
    </>
  );
}

export function CommitList({ repo, commits }: { repo: string; commits: Commit[] }) {
  const groups: { day: string; commits: Commit[] }[] = [];
  for (const c of commits) {
    const day = dayKey(c.committer.time ?? c.author.time);
    const last = groups[groups.length - 1];
    if (last && last.day === day) last.commits.push(c);
    else groups.push({ day, commits: [c] });
  }
  return (
    <div>
      {groups.map((g) => (
        <section key={g.day} className="mb-4">
          <h2 className="f5 text-normal color-fg-muted mb-2 d-flex flex-items-center" style={{ gap: 6 }}>
            <GitCommitIcon /> Commits on {g.day}
          </h2>
          <ul className="Box list-style-none">
            {g.commits.map((c) => (
              <li key={c.id} className="Box-row d-flex flex-items-center" style={{ gap: 12 }} data-testid="commit">
                <div className="flex-1" style={{ minWidth: 0 }}>
                  <div className="d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
                    <Link href={`/repositories/${repo}/commit/${c.id}`} className="text-bold color-fg-default gh-truncate">
                      {c.subject}
                    </Link>
                    <CommitBadges c={c} />
                  </div>
                  <div className="f6 color-fg-muted mt-1 d-flex flex-items-center flex-wrap" style={{ gap: 6 }}>
                    <AgentLink agent={c.agent} size={16} verified={c.agentVerified} />
                    <span>{c.agentVerified ? "landed" : "authored"}</span>
                    <RelativeTime iso={c.landing?.recordTime ?? c.committer.time} />
                    {c.agentVerified && c.author.name !== c.agent ? <span>· authored as {c.author.name}</span> : null}
                  </div>
                </div>
                <Link href={`/repositories/${repo}/commit/${c.id}`} className="btn btn-sm text-mono">
                  {shortSha(c.id)}
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
