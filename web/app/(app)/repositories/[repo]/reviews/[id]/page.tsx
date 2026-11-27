import Link from "next/link";
import type { Metadata } from "next";
import { CheckCircleFillIcon, XCircleFillIcon, CommentIcon, FileIcon, CircleIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Review, User } from "@/lib/types";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { Markdown } from "@/components/Markdown";
import { ReviewStateLabel, CLASS_LABEL } from "@/components/ReviewState";
import { ReviewActions } from "@/components/ReviewActions";
import { qs, shortSha } from "@/lib/format";

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `Review #${id}` };
}

export default async function ReviewPage({ params }: { params: Promise<{ repo: string; id: string }> }) {
  const { repo, id } = await params;
  const [r, me] = await Promise.all([api<Review>(`/v1/repositories/${enc(repo)}/reviews/${enc(id)}`), api<User>("/v1/users/me")]);
  const timeline = [
    ...r.comments.map((c) => ({ kind: "comment" as const, time: c.createTime, c })),
    ...r.verdicts.map((v) => ({ kind: "verdict" as const, time: v.createTime, v })),
  ].sort((a, b) => a.time.localeCompare(b.time));
  return (
    <div className="gh-container">
      <h1 className="f2 text-normal">
        {r.title || `Review ${r.id}`} <span className="color-fg-muted">#{r.id}</span>
      </h1>
      <div className="d-flex flex-items-center flex-wrap mt-2 mb-4 pb-3 border-bottom" style={{ gap: 8 }}>
        <ReviewStateLabel state={r.state} large />
        <AgentLink agent={r.author} />
        <span className="color-fg-muted">
          wants to land <Link href={`/repositories/${repo}/commit/${r.headCommitId}`} className="text-mono">{shortSha(r.headCommitId)}</Link> into{" "}
          <span className="branch-name">{r.targetBranch}</span> from base <span className="text-mono">{shortSha(r.baseCommitId)}</span>, opened{" "}
          <RelativeTime iso={r.createTime} />
        </span>
      </div>
      <div className="d-flex flex-wrap" style={{ gap: 24 }}>
        <div className="flex-1" style={{ minWidth: 300 }}>
          <div className="Box mb-4">
            <div className="Box-header d-flex flex-items-center" style={{ gap: 8 }}>
              <AgentLink agent={r.author} size={20} />
              <span className="color-fg-muted f6">
                opened <RelativeTime iso={r.createTime} />
              </span>
            </div>
            <div className="Box-body">{r.description ? <Markdown source={r.description} /> : <span className="color-fg-muted">No description.</span>}</div>
          </div>

          <h2 className="f4 mb-2">Changed files, by class</h2>
          <div className="Box mb-4">
            {r.fileGroups.map((g) => (
              <div key={g.class}>
                <div className="Box-header py-2 d-flex flex-items-center" style={{ gap: 8 }}>
                  <span className={`Label ${CLASS_LABEL[g.class]?.className ?? ""}`}>{CLASS_LABEL[g.class]?.label ?? g.class}</span>
                  <span className="f6 color-fg-muted">{g.files.length} files</span>
                </div>
                <ul className="list-style-none">
                  {g.files.map((f) => (
                    <li key={f.path} className="Box-row py-2 d-flex flex-items-center f6" style={{ gap: 8 }}>
                      <FileIcon className="color-fg-muted" />
                      <Link href={`/repositories/${repo}/blob${qs({ ref: r.headCommitId, path: f.path })}`} className="text-mono">
                        {f.path}
                      </Link>
                      <span className="color-fg-muted ml-auto">{f.reason}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>

          <h2 className="f4 mb-2">Conversation</h2>
          {timeline.length === 0 ? <p className="color-fg-muted f6 mb-3">No comments or verdicts yet.</p> : null}
          {timeline.map((t, i) =>
            t.kind === "comment" ? (
              <div key={i} className="Box mb-3" data-testid="review-comment">
                <div className="Box-header py-2 d-flex flex-items-center flex-wrap f6" style={{ gap: 8 }}>
                  <CommentIcon className="color-fg-muted" />
                  <AgentLink agent={t.c.actor} size={16} />
                  <span className="color-fg-muted">
                    commented <RelativeTime iso={t.c.createTime} />
                  </span>
                  {t.c.path ? (
                    <Link href={`/repositories/${repo}/blob${qs({ ref: t.c.commitId, path: t.c.path })}#L${t.c.line ?? 1}`} className="text-mono ml-auto">
                      {t.c.path}
                      {t.c.line ? `:${t.c.line}` : ""}
                    </Link>
                  ) : null}
                </div>
                <div className="Box-body">
                  <Markdown source={t.c.body} />
                </div>
              </div>
            ) : (
              <div key={i} className="d-flex flex-items-center flex-wrap mb-3 f6" style={{ gap: 8 }}>
                {t.v.kind === "APPROVE" ? <CheckCircleFillIcon className="color-fg-success" /> : <XCircleFillIcon className="color-fg-danger" />}
                <AgentLink agent={t.v.actor} size={16} />
                <span>{t.v.kind === "APPROVE" ? "approved" : "requested changes"}</span>
                <span className="text-mono color-fg-muted">on {shortSha(t.v.commitId)}</span>
                <RelativeTime iso={t.v.createTime} />
                {!t.v.current ? <span className="Label">outdated</span> : null}
                {t.v.body ? <div className="width-full pl-4 color-fg-muted">“{t.v.body}”</div> : null}
              </div>
            ),
          )}
          <ReviewActions repo={repo} reviewId={r.id} headCommitId={r.headCommitId} canReview={me.canReview} open={r.state === "OPEN"} />
        </div>

        <aside style={{ width: 300, maxWidth: "100%" }}>
          <div className={`flash mb-3 ${r.approved ? "flash-success" : "flash-warn"}`} data-testid="review-approval">
            {r.approved ? <CheckCircleFillIcon /> : <CircleIcon />} <strong>{r.approved ? "Approved" : "Not approved yet"}</strong>
            <div className="f6 mt-1">gitai evaluates its approval rules on every read. Landing does not enforce them yet.</div>
          </div>
          <h3 className="f5 mb-2">Requirements</h3>
          {r.requirements.length === 0 ? (
            <p className="f6 color-fg-muted">No rule applies to these files, so no approval is needed.</p>
          ) : (
            <ul className="list-style-none mb-3">
              {r.requirements.map((q) => (
                <li key={q.rule} className="d-flex flex-items-start py-1 f6" style={{ gap: 8 }}>
                  {q.satisfied ? <CheckCircleFillIcon className="color-fg-success" /> : <CircleIcon className="color-fg-muted" />}
                  <div>
                    <strong>{q.rule}</strong>: {q.approvals}/{q.requiredApprovals} approvals
                    {q.description ? <div className="color-fg-muted">{q.description}</div> : null}
                  </div>
                </li>
              ))}
            </ul>
          )}
          <h3 className="f5 mb-2">Current verdicts</h3>
          {r.currentVerdicts.length === 0 ? (
            <p className="f6 color-fg-muted">None yet.</p>
          ) : (
            <ul className="list-style-none">
              {r.currentVerdicts.map((v) => (
                <li key={v.actor} className="d-flex flex-items-center py-1 f6" style={{ gap: 8 }}>
                  <AgentLink agent={v.actor} size={16} />
                  {v.kind === "APPROVE" ? <span className="Label Label--success">approved</span> : <span className="Label Label--danger">changes requested</span>}
                </li>
              ))}
            </ul>
          )}
        </aside>
      </div>
    </div>
  );
}
