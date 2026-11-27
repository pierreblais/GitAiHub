import Link from "next/link";
import {
  GitMergeIcon, GitCommitIcon, UndoIcon, SparkleFillIcon, GitBranchIcon, XIcon, RepoPushIcon, TagIcon,
} from "@primer/octicons-react";
import type { Changeset, ChangesetKind } from "@/lib/types";
import { AgentLink } from "./Agent";
import { RelativeTime } from "./Time";
import { shortSha } from "@/lib/format";

const KIND: Record<ChangesetKind, { label: string; icon: React.ElementType; className: string }> = {
  LAND: { label: "landed", icon: GitMergeIcon, className: "color-fg-done" },
  BATCH_LAND: { label: "batch-landed", icon: GitMergeIcon, className: "color-fg-done" },
  CONFLICT_RESOLVED: { label: "resolved a conflict and landed", icon: SparkleFillIcon, className: "color-fg-attention" },
  AUTO_REVERT: { label: "auto-reverted", icon: UndoIcon, className: "color-fg-danger" },
  BRANCH_CREATED: { label: "created branch", icon: GitBranchIcon, className: "color-fg-muted" },
  BRANCH_DELETED: { label: "deleted branch", icon: XIcon, className: "color-fg-muted" },
  UPDATE: { label: "updated", icon: RepoPushIcon, className: "color-fg-muted" },
};

export function kindLabel(kind: ChangesetKind): string {
  return KIND[kind]?.label ?? "updated";
}

export function ChangesetItem({ cs, showRepo = true, isNew = false }: { cs: Changeset; showRepo?: boolean; isNew?: boolean }) {
  const repo = cs.repository.replace(/^repositories\//, "");
  const k = KIND[cs.kind] ?? KIND.UPDATE;
  const Icon = cs.reference.startsWith("refs/tags/") ? TagIcon : k.icon;
  const target = cs.branch ?? cs.reference;
  const agents = cs.agents.length ? cs.agents : [cs.actor];
  return (
    <div className={`gh-feed-item ${isNew ? "gh-feed-new" : ""}`} data-testid="changeset">
      <span className={`gh-feed-dot ${k.className}`}>
        <Icon size={14} />
      </span>
      <div className="d-flex flex-wrap flex-items-center" style={{ gap: 6 }}>
        {agents.map((a) => (
          <AgentLink key={a} agent={a} verified />
        ))}
        <span className="color-fg-muted">{k.label}</span>
        {cs.kind === "BRANCH_CREATED" || cs.kind === "BRANCH_DELETED" ? null : <span className="color-fg-muted">on</span>}
        <Link href={`/repositories/${repo}/commits?ref=${encodeURIComponent(target)}`} className="branch-name">
          {target}
        </Link>
        {showRepo ? (
          <>
            <span className="color-fg-muted">in</span>
            <Link href={`/repositories/${repo}`} className="text-bold">
              {repo}
            </Link>
          </>
        ) : null}
        <span className="color-fg-muted f6 ml-1">
          <RelativeTime iso={cs.recordTime} /> · #{cs.sequence}
          {cs.rewritten ? " · rebased by gitai" : ""}
        </span>
      </div>
      {cs.commits.length > 0 ? (
        <ul className="list-style-none mt-2">
          {cs.commits.slice(0, 5).map((c) => (
            <li key={c.commitId} className="d-flex flex-items-center f6 py-1" style={{ gap: 8 }}>
              <GitCommitIcon size={14} className="color-fg-muted" />
              <Link href={`/repositories/${repo}/commit/${c.commitId}`} className="text-mono color-fg-muted">
                {shortSha(c.commitId)}
              </Link>
              <span className="gh-truncate">{c.subject}</span>
              {c.resolvedBy ? <span className="Label Label--attention">AI-resolved</span> : null}
              {c.autoRevertOf ? <span className="Label Label--danger">auto-revert of {shortSha(c.autoRevertOf)}</span> : null}
            </li>
          ))}
          {cs.commits.length > 5 ? <li className="f6 color-fg-muted py-1">and {cs.commits.length - 5} more commits</li> : null}
        </ul>
      ) : null}
    </div>
  );
}
