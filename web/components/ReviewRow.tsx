import Link from "next/link";
import { CodeReviewIcon, GitMergeIcon, CommentIcon, CheckIcon, XCircleIcon } from "@primer/octicons-react";
import type { Review } from "@/lib/types";
import { RelativeTime } from "./Time";

/** One review in a list, laid out like a pull request row. */
export function ReviewRow({ repo, review, showRepo }: { repo: string; review: Review; showRepo?: boolean }) {
  const icon =
    review.state === "LANDED" ? (
      <GitMergeIcon className="color-fg-done" />
    ) : review.state === "CLOSED" ? (
      <CodeReviewIcon className="color-fg-muted" />
    ) : (
      <CodeReviewIcon className="color-fg-open" />
    );
  const comments = review.comments?.length ?? 0;
  return (
    <li className="Box-row gh-list-row" data-testid="review-row">
      <span className="gh-list-icon">{icon}</span>
      <div className="flex-1" style={{ minWidth: 0 }}>
        <Link href={`/repositories/${repo}/reviews/${review.id}`} className="gh-list-title">
          {review.title || `Review ${review.id}`}
        </Link>
        {review.state === "OPEN" ? (
          review.approved ? (
            <span className="Label Label--success ml-2">
              <CheckIcon size={12} /> Approved
            </span>
          ) : review.currentVerdicts?.some((v) => v.kind === "REQUEST_CHANGES") ? (
            <span className="Label Label--danger ml-2">
              <XCircleIcon size={12} /> Changes requested
            </span>
          ) : null
        ) : null}
        <div className="f6 color-fg-muted mt-1">
          {showRepo ? `gitai/${repo}#${review.id}` : `#${review.id}`} · Opened by {review.author} · into{" "}
          <span className="text-mono">{review.targetBranch}</span> · Updated <RelativeTime iso={review.updateTime} />
        </div>
      </div>
      {comments > 0 ? (
        <Link href={`/repositories/${repo}/reviews/${review.id}`} className="f6 color-fg-muted no-underline d-flex flex-items-center" style={{ gap: 4 }}>
          <CommentIcon /> {comments}
        </Link>
      ) : null}
    </li>
  );
}
