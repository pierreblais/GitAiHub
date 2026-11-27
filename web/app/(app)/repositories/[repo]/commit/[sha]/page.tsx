import Link from "next/link";
import type { Metadata } from "next";
import { api, enc } from "@/lib/api";
import type { CommitDetail } from "@/lib/types";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { CommitBadges } from "@/components/CommitList";
import { FileDiffView, DiffStat } from "@/components/DiffView";
import { plural, qs, shortSha } from "@/lib/format";

export async function generateMetadata({ params }: { params: Promise<{ sha: string }> }): Promise<Metadata> {
  const { sha } = await params;
  return { title: `Commit ${sha.slice(0, 7)}` };
}

export default async function CommitPage({ params }: { params: Promise<{ repo: string; sha: string }> }) {
  const { repo, sha } = await params;
  const { commit: c, files, stats } = await api<CommitDetail>(`/v1/repositories/${enc(repo)}/commits/${enc(sha)}`);
  const body = c.body.trim();
  return (
    <div className="gh-container">
      <div className="Box mb-4">
        <div className="Box-header">
          <div className="d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
            <h1 className="f3 text-normal">{c.subject}</h1>
            <CommitBadges c={c} />
            <Link href={`/repositories/${repo}/tree${qs({ ref: c.id })}`} className="btn btn-sm ml-auto">
              Browse files
            </Link>
          </div>
          {body ? <pre className="mt-2 f6 color-fg-muted" style={{ whiteSpace: "pre-wrap" }}>{body}</pre> : null}
        </div>
        <div className="Box-body d-flex flex-items-center flex-wrap f6" style={{ gap: 12 }}>
          <AgentLink agent={c.agent} verified={c.agentVerified} />
          {c.landing ? (
            <span className="color-fg-muted">
              landed on <span className="branch-name">{c.landing.reference.replace(/^refs\/heads\//, "")}</span> by {c.landing.operation} (changeset #{c.landing.sequence}){" "}
              <RelativeTime iso={c.landing.recordTime} />
            </span>
          ) : (
            <span className="color-fg-muted">
              authored <RelativeTime iso={c.author.time} /> (no landing in the changeset log; the name comes from the commit author)
            </span>
          )}
          <span className="ml-auto color-fg-muted">
            {c.parentIds.length > 0 ? (
              <>
                {plural(c.parentIds.length, "parent")}{" "}
                {c.parentIds.map((p) => (
                  <Link key={p} href={`/repositories/${repo}/commit/${p}`} className="text-mono mr-1">
                    {shortSha(p)}
                  </Link>
                ))}
              </>
            ) : (
              "root commit"
            )}
            · commit <span className="text-mono color-fg-default">{c.id}</span>
          </span>
        </div>
        {c.trailers.length > 0 ? (
          <div className="Box-footer f6">
            <div className="text-bold mb-1">Trailers</div>
            <dl className="m-0">
              {c.trailers.map((t, i) => (
                <div key={i} className="d-flex" style={{ gap: 8 }}>
                  <dt className="text-mono color-fg-muted">{t.key}:</dt>
                  <dd className="text-mono m-0">
                    {t.key === "Graft-Auto-Revert" ? <Link href={`/repositories/${repo}/commit/${t.value.split(" ")[0]}`}>{t.value}</Link> : t.value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        ) : null}
      </div>
      <div className="d-flex flex-items-center mb-3 f6" style={{ gap: 12 }}>
        <span>
          Showing <strong>{plural(stats.files, "changed file")}</strong>
        </span>
        <DiffStat additions={stats.additions} deletions={stats.deletions} />
      </div>
      {files.map((f, i) => (
        <FileDiffView key={i} file={f} id={`diff-${i}`} />
      ))}
    </div>
  );
}
