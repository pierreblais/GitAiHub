"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

/**
 * Comment on a review, or give a verdict on its current head. These are the
 * only writes in the MVP, and neither changes code. gitai records them under
 * the user's own gitai identity.
 */
export function ReviewActions({ repo, reviewId, headCommitId, canReview, open }: { repo: string; reviewId: string; headCommitId: string; canReview: boolean; open: boolean }) {
  const router = useRouter();
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return <p className="color-fg-muted f6">This review is closed.</p>;
  if (!canReview) {
    return (
      <div className="flash">
        Your account has no gitai identity, so you can read reviews but not comment or approve. Ask the operator to set
        <code className="ml-1">GITAIHUB_USER_GITAI_TOKENS</code>.
      </div>
    );
  }

  async function send(path: string, payload: object) {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/repositories/${repo}/reviews/${path}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => null);
        setError(j?.error?.message ?? `Request failed (${res.status})`);
        return;
      }
      setBody("");
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="Box" onSubmit={(e) => e.preventDefault()}>
      <div className="Box-body">
        <label htmlFor="review-body" className="text-bold d-block mb-2">
          Add your review
        </label>
        <textarea
          id="review-body"
          className="form-control width-full"
          rows={4}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Leave a comment"
        />
        {error ? <div className="flash flash-error mt-2" role="alert">{error}</div> : null}
      </div>
      <div className="Box-footer d-flex flex-justify-end flex-wrap" style={{ gap: 8 }}>
        <button type="button" className="btn" disabled={busy || !body.trim()} onClick={() => send(`${reviewId}/comments`, { body })}>
          Comment
        </button>
        <button type="button" className="btn btn-danger" disabled={busy} onClick={() => send(`${reviewId}:submitVerdict`, { kind: "REQUEST_CHANGES", body, headCommitId })}>
          Request changes
        </button>
        <button type="button" className="btn btn-primary" disabled={busy} onClick={() => send(`${reviewId}:submitVerdict`, { kind: "APPROVE", body, headCommitId })}>
          Approve
        </button>
      </div>
    </form>
  );
}
