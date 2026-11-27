import { Header } from "@/components/Header";
import { api } from "@/lib/api";
import type { User } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const user = await api<User>("/v1/users/me");
  return (
    <>
      <a href="#main" className="sr-only">
        Skip to content
      </a>
      <Header user={user} />
      <main id="main">{children}</main>
      <footer className="gh-container f6 color-fg-muted border-top mt-6 d-flex flex-wrap" style={{ gap: 16 }}>
        <span>GitaiHub, a web front end for gitai</span>
        <a href="https://github.com/feder-cr/git-ai">gitai on GitHub</a>
      </footer>
    </>
  );
}
