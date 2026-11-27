import { Header } from "@/components/Header";
import { api } from "@/lib/api";
import type { Repository, User } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const [user, { repositories }] = await Promise.all([
    api<User>("/v1/users/me"),
    api<{ repositories: Repository[] }>("/v1/repositories?pageSize=100"),
  ]);
  return (
    <>
      <a href="#main" className="sr-only">
        Skip to content
      </a>
      <Header user={user} repositories={repositories} />
      <main id="main">{children}</main>
      <footer className="gh-footer">
        <span className="d-inline-flex flex-items-center" style={{ gap: 8 }}>
          GitaiHub · a web front end for gitai
        </span>
        <a href="https://github.com/feder-cr/git-ai">gitai</a>
      </footer>
    </>
  );
}
