import Link from "next/link";
import type { Metadata } from "next";
import { api, enc } from "@/lib/api";
import type { Blob, Commit } from "@/lib/types";
import { FileHeader } from "@/components/FileHeader";
import { CodeView } from "@/components/CodeView";
import { Markdown } from "@/components/Markdown";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { CommitBadges } from "@/components/CommitList";
import { formatBytes, plural, qs, shortSha } from "@/lib/format";
import { isMarkdown } from "@/lib/lang";

type Search = { ref?: string; path?: string; plain?: string };

export async function generateMetadata({ searchParams }: { searchParams: Promise<Search> }): Promise<Metadata> {
  const { path } = await searchParams;
  return { title: path ?? "File" };
}

export default async function BlobPage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<Search> }) {
  const { repo } = await params;
  const { ref, path = "", plain } = await searchParams;
  const header = await FileHeader({ repo, gitRef: ref, path, mode: "blob" });
  const { blob, lastCommit } = await api<{ blob: Blob; lastCommit?: Commit; commitId: string }>(
    `/v1/repositories/${enc(repo)}/blob${qs({ revision: header.current, path })}`,
  );
  const md = isMarkdown(path) && !plain;
  const lines = blob.content ? blob.content.split("\n").length - (blob.content.endsWith("\n") ? 1 : 0) : 0;
  return (
    <div className="gh-container">
      {header.node}
      {lastCommit ? (
        <div className="Box mb-3">
          <div className="Box-header d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
            <AgentLink agent={lastCommit.agent} verified={lastCommit.agentVerified} />
            <Link href={`/repositories/${repo}/commit/${lastCommit.id}`} className="color-fg-default gh-truncate">
              {lastCommit.subject}
            </Link>
            <CommitBadges c={lastCommit} />
            <span className="ml-auto f6 color-fg-muted">
              <Link href={`/repositories/${repo}/commit/${lastCommit.id}`} className="text-mono color-fg-muted mr-2">
                {shortSha(lastCommit.id)}
              </Link>
              <RelativeTime iso={lastCommit.committer.time} />
            </span>
          </div>
        </div>
      ) : null}
      <div className="Box">
        <div className="Box-header py-2 d-flex flex-items-center f6 color-fg-muted" style={{ gap: 12 }}>
          {blob.binary ? null : <span>{plural(lines, "line")}</span>}
          <span>{formatBytes(blob.size)}</span>
          {isMarkdown(path) ? (
            <div className="BtnGroup ml-auto">
              <Link className={`btn btn-sm BtnGroup-item ${md ? "selected" : ""}`} href={`/repositories/${repo}/blob${qs({ ref: header.current, path })}`}>
                Preview
              </Link>
              <Link className={`btn btn-sm BtnGroup-item ${md ? "" : "selected"}`} href={`/repositories/${repo}/blob${qs({ ref: header.current, path, plain: 1 })}`}>
                Code
              </Link>
            </div>
          ) : null}
        </div>
        {blob.binary ? (
          <div className="p-4 color-fg-muted text-center">Binary file not shown.</div>
        ) : md ? (
          <div className="Box-body px-5 py-4">
            <Markdown source={blob.content} />
          </div>
        ) : (
          <CodeView path={path} content={blob.content} />
        )}
        {blob.truncated ? <div className="Box-footer f6 color-fg-muted">This file is larger than 1 MB; only the beginning is shown.</div> : null}
      </div>
    </div>
  );
}
