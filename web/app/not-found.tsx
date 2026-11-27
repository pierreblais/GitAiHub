import Link from "next/link";
import { LogoMark } from "@/components/Logo";

export default function NotFound() {
  return (
    <main className="gh-container text-center" style={{ paddingTop: 96 }}>
      <LogoMark size={56} />
      <h1 className="f1 text-normal mt-3">Not found</h1>
      <p className="color-fg-muted">This repository, revision or path does not exist on the gitai server.</p>
      <Link href="/" className="btn mt-3">
        Back to repositories
      </Link>
    </main>
  );
}
