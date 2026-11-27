"use client";

export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="gh-container">
      <div className="flash flash-error" role="alert">
        <strong>Something went wrong.</strong> {error.message || "The page could not be loaded."}
        {error.digest ? <div className="f6 mt-1 color-fg-muted">Reference: {error.digest}</div> : null}
      </div>
      <button className="btn mt-3" onClick={reset}>
        Try again
      </button>
    </div>
  );
}
