import type { Metadata } from "next";
import { api } from "@/lib/api";
import type { Changeset } from "@/lib/types";
import { LiveFeed } from "@/components/LiveFeed";
import { qs } from "@/lib/format";

export const metadata: Metadata = { title: "Activity" };

export default async function ActivityPage({ searchParams }: { searchParams: Promise<{ pageToken?: string }> }) {
  const { pageToken } = await searchParams;
  const data = await api<{ changesets: Changeset[]; nextPageToken?: string }>(`/v1/changesets${qs({ pageSize: 40, pageToken })}`);
  return (
    <div className="gh-container" style={{ maxWidth: 900 }}>
      <h1 className="f2 text-normal mb-1">Agent activity</h1>
      <p className="color-fg-muted mb-4">Every reference update gitai records in its changeset log, across all repositories.</p>
      <LiveFeed key={pageToken ?? "first"} initial={data.changesets} nextPageToken={data.nextPageToken} moreHref="/activity" />
    </div>
  );
}
