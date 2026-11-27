import type { Metadata } from "next";
import { api, enc } from "@/lib/api";
import type { Changeset } from "@/lib/types";
import { LiveFeed } from "@/components/LiveFeed";
import { qs } from "@/lib/format";

export const metadata: Metadata = { title: "Activity" };

export default async function RepoActivityPage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<{ pageToken?: string }> }) {
  const { repo } = await params;
  const { pageToken } = await searchParams;
  const data = await api<{ changesets: Changeset[]; nextPageToken?: string }>(`/v1/repositories/${enc(repo)}/changesets${qs({ pageSize: 40, pageToken })}`);
  return (
    <div className="gh-container" style={{ maxWidth: 900 }}>
      <h2 className="f3 text-normal mb-1">Changeset log</h2>
      <p className="color-fg-muted mb-4 f6">Every reference update of this repository, in gitai&apos;s order, with the agent that made it.</p>
      <LiveFeed key={pageToken ?? "first"} initial={data.changesets} repository={repo} showRepo={false} nextPageToken={data.nextPageToken} moreHref={`/repositories/${repo}/activity`} />
    </div>
  );
}
