"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export function LoginForm({ next }: { next: string }) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/sessions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: data.get("username"), password: data.get("password") }),
      });
      if (!res.ok) {
        const j = await res.json().catch(() => null);
        setError(j?.error?.message ?? "Sign in failed");
        return;
      }
      router.push(next);
      router.refresh();
    } catch {
      setError("The GitaiHub API is unreachable.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="Box p-4 color-bg-subtle" onSubmit={onSubmit}>
      {error ? (
        <div className="flash flash-error mb-3" role="alert">
          {error}
        </div>
      ) : null}
      <label htmlFor="username" className="d-block mb-2 text-bold">
        Username
      </label>
      <input id="username" name="username" className="form-control width-full mb-3" autoComplete="username" autoCapitalize="none" required autoFocus />
      <label htmlFor="password" className="d-block mb-2 text-bold">
        Password
      </label>
      <input id="password" name="password" type="password" className="form-control width-full mb-3" autoComplete="current-password" required />
      <button type="submit" className="btn btn-block gh-brand-btn" disabled={busy}>
        {busy ? "Signing in…" : "Sign in"}
      </button>
    </form>
  );
}
