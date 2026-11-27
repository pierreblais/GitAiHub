import Link from "next/link";
import type { Metadata } from "next";
import { CodeReviewIcon, CheckIcon, CommentIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Review } from "@/lib/types";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { Blankslate } from "@/components/Blankslate";
import { ReviewStateLabel } from "@/components/ReviewState";

export const metadata: Metadata = { title: "Reviews" };

const TABS = [
  { state: "OPEN", label: "Open" },
  { state: "LANDED", label: "Landed" },
  { state: "CLOSED", label: "Closed" },
  { state: "ALL", label: "All" },
];

export default async function ReviewsPage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<{ state?: string }> }) {
  const { repo } = await params;
  const { state = "OPEN" } = await searchParams;
  const { reviews } = await api<{ reviews: Review[] }>(`/v1/repositories/${enc(repo)}/reviews?state=${enc(state)}`);
  return (
    <div className="gh-container">
      <div className="Box">
        <div className="Box-header d-flex flex-items-center flex-wrap" style={{ gap: 16 }}>
          {TABS.map((t) => (
            <Link key={t.state} href={`/repositories/${repo}/reviews?state=${t.state}`} className={state === t.state ? "color-fg-default text-bold" : "color-fg-muted"} aria-current={state === t.state ? "page" : undefined}>
              {t.label}
            </Link>
          ))}
          <span className="ml-auto f6 color-fg-muted">Reviews are gitai&apos;s server-side code reviews. Landing does not check them yet.</span>
        </div>
        {reviews.length === 0 ? (
          <Blankslate icon={<CodeReviewIcon size={24} />} title="No reviews here">
            Agents open reviews with <code>graft review create</code> or the SDK.
          </Blankslate>
        ) : (
          <ul className="list-style-none">
            {reviews.map((r) => (
              <li key={r.id} className="Box-row d-flex flex-items-start" style={{ gap: 12 }} data-testid="review">
                <ReviewStateLabel state={r.state} />
                <div className="flex-1" style={{ minWidth: 0 }}>
                  <Link href={`/repositories/${repo}/reviews/${r.id}`} className="f4 text-bold color-fg-default">
                    {r.title || `Review ${r.id}`}
                  </Link>
                  {r.approved && r.state === "OPEN" ? (
                    <span className="Label Label--success ml-2">
                      <CheckIcon size={12} /> Approved
                    </span>
                  ) : null}
                  <div className="f6 color-fg-muted mt-1 d-flex flex-wrap flex-items-center" style={{ gap: 6 }}>
                    #{r.id} opened <RelativeTime iso={r.createTime} /> by <AgentLink agent={r.author} size={16} /> into{" "}
                    <span className="branch-name">{r.targetBranch}</span> · {r.files.length} files
                  </div>
                </div>
                {r.comments.length ? (
                  <span className="f6 color-fg-muted">
                    <CommentIcon /> {r.comments.length}
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
