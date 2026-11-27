"use client";

import { SignOutIcon } from "@primer/octicons-react";
import { useRouter } from "next/navigation";

export function SignOutButton() {
  const router = useRouter();
  return (
    <button
      type="button"
      className="btn-octicon"
      style={{ color: "var(--gh-header-muted)" }}
      aria-label="Sign out"
      title="Sign out"
      onClick={async () => {
        await fetch("/api/v1/sessions/current", { method: "DELETE" });
        router.push("/login");
        router.refresh();
      }}
    >
      <SignOutIcon />
    </button>
  );
}
