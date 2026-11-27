import Link from "next/link";
import type { Metadata } from "next";
import { api, enc } from "@/lib/api";
import type { Commit, Ref, Repository } from "@/lib/types";
import { CommitList } from "@/components/CommitList";
import { RefSelector } from "@/components/RefSelector";
import { Blankslate } from "@/components/Blankslate";
import { qs } from "@/lib/format";

type Search = { ref?: string; path?: string; pageToken?: string };

export const metadata: Metadata = { title: "Commits" };

export default async function CommitsPage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<Search> }) {
  const { repo } = await params;
  const { ref, path, pageToken } = await searchParams;
  const r = await api<Repository>(`/v1/repositories/${enc(repo)}`);
  const current = ref || r.defaultBranch || "main";
  const [data, { branches }, { tags }] = await Promise.all([
    api<{ commits: Commit[]; nextPageToken?: string }>(`/v1/repositories/${enc(repo)}/commits${qs({ revision: current, path, pageSize: 30, pageToken })}`),
    api<{ branches: Ref[] }>(`/v1/repositories/${enc(repo)}/branches`),
    api<{ tags: Ref[] }>(`/v1/repositories/${enc(repo)}/tags`),
  ]);
  return (
    <div className="gh-container">
      <div className="d-flex flex-items-center flex-wrap mb-4" style={{ gap: 12 }}>
        <RefSelector current={current} branches={branches} tags={tags} hrefFor={(gitRef) => `/repositories/${repo}/commits${qs({ ref: gitRef, path })}`} />
        <h1 className="f4 text-normal">
          History{path ? <> for <span className="text-mono text-bold">{path}</span></> : null}
        </h1>
        {path ? (
          <Link href={`/repositories/${repo}/commits${qs({ ref: current })}`} className="f6">
            Show all commits
          </Link>
        ) : null}
      </div>
      {data.commits.length === 0 ? (
        <div className="Box">
          <Blankslate title="No commits" />
        </div>
      ) : (
        <CommitList repo={repo} commits={data.commits} />
      )}
      <nav className="d-flex flex-justify-center" style={{ gap: 8 }} aria-label="Pagination">
        {pageToken ? (
          <Link className="btn btn-sm" href={`/repositories/${repo}/commits${qs({ ref: current, path })}`}>
            Newest
          </Link>
        ) : null}
        {data.nextPageToken ? (
          <Link className="btn btn-sm" href={`/repositories/${repo}/commits${qs({ ref: current, path, pageToken: data.nextPageToken })}`}>
            Older
          </Link>
        ) : null}
      </nav>
    </div>
  );
}
