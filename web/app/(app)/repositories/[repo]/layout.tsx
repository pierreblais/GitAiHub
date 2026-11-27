import Link from "next/link";
import { RepoIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Repository, Review } from "@/lib/types";
import { RepoTabs } from "@/components/RepoTabs";

export default async function RepoLayout({ children, params }: { children: React.ReactNode; params: Promise<{ repo: string }> }) {
  const { repo } = await params;
  const [r, { reviews }] = await Promise.all([
    api<Repository>(`/v1/repositories/${enc(repo)}`),
    api<{ reviews: Review[] }>(`/v1/repositories/${enc(repo)}/reviews?state=OPEN`).catch(() => ({ reviews: [] as Review[] })),
  ]);
  return (
    <>
      <div className="gh-repohead">
        <div className="px-4 pb-3 d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
          <RepoIcon className="color-fg-muted" />
          <h1 className="f3 text-normal">
            <Link href={`/repositories/${r.repositoryId}`} className="text-bold">
              {r.repositoryId}
            </Link>
          </h1>
          <span className="Label">gitai</span>
          <span className="f6 color-fg-muted text-mono ml-2 gh-hide-sm">{r.relativePath}</span>
        </div>
        <RepoTabs repo={r.repositoryId} openReviews={reviews.length} />
      </div>
      {children}
    </>
  );
}
