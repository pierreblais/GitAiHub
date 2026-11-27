import { api, enc } from "@/lib/api";
import type { Ref, Repository, Review } from "@/lib/types";
import { RepoTabs } from "@/components/RepoTabs";

export default async function RepoLayout({ children, params }: { children: React.ReactNode; params: Promise<{ repo: string }> }) {
  const { repo } = await params;
  const [r, { reviews }, { branches }] = await Promise.all([
    api<Repository>(`/v1/repositories/${enc(repo)}`),
    api<{ reviews: Review[] }>(`/v1/repositories/${enc(repo)}/reviews?state=OPEN`).catch(() => ({ reviews: [] as Review[] })),
    api<{ branches: Ref[] }>(`/v1/repositories/${enc(repo)}/branches`).catch(() => ({ branches: [] as Ref[] })),
  ]);
  return (
    <>
      <div className="gh-repohead">
        <RepoTabs repo={r.repositoryId} openReviews={reviews.length} branches={branches.length} />
      </div>
      {children}
    </>
  );
}
