import Link from "next/link";
import { FileDirectoryFillIcon, FileIcon, FileSubmoduleIcon, HistoryIcon, BookIcon, GitBranchIcon, TagIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Commit, Ref, Repository, Tree } from "@/lib/types";
import { RefSelector } from "./RefSelector";
import { Breadcrumbs } from "./Breadcrumbs";
import { Markdown } from "./Markdown";
import { CodeView } from "./CodeView";
import { AgentLink } from "./Agent";
import { RelativeTime } from "./Time";
import { CommitBadges } from "./CommitList";
import { Blankslate } from "./Blankslate";
import { isMarkdown } from "@/lib/lang";
import { qs, shortSha } from "@/lib/format";

/** The Code tab: a directory with its last commits and README. */
export async function TreeView({ repo, gitRef, path }: { repo: string; gitRef?: string; path?: string }) {
  const r = await api<Repository>(`/v1/repositories/${enc(repo)}`);
  const current = gitRef || r.defaultBranch || "main";
  if (!r.defaultBranch && !gitRef) {
    return (
      <div className="gh-container">
        <div className="Box">
          <Blankslate icon={<BookIcon size={24} />} title="This repository is empty">
            Land a first commit with gitai and it shows up here.
          </Blankslate>
        </div>
      </div>
    );
  }
  const [tree, { branches }, { tags }, latest] = await Promise.all([
    api<Tree>(`/v1/repositories/${enc(repo)}/tree${qs({ revision: current, path })}`),
    api<{ branches: Ref[] }>(`/v1/repositories/${enc(repo)}/branches`),
    api<{ tags: Ref[] }>(`/v1/repositories/${enc(repo)}/tags`),
    api<{ commits: Commit[] }>(`/v1/repositories/${enc(repo)}/commits${qs({ revision: current, path, pageSize: 1 })}`),
  ]);
  const head = latest.commits[0];
  const parent = path && path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
  return (
    <div className="gh-container">
      <div className="d-flex flex-items-center flex-wrap mb-3" style={{ gap: 12 }}>
        <RefSelector current={current} branches={branches} tags={tags} hrefFor={(ref) => `/repositories/${repo}/tree${qs({ ref, path })}`} />
        {path ? (
          <Breadcrumbs repo={repo} gitRef={current} path={path} />
        ) : (
          <span className="f6 color-fg-muted d-flex" style={{ gap: 12 }}>
            <Link href={`/repositories/${repo}/branches`} className="color-fg-muted">
              <GitBranchIcon /> <strong className="color-fg-default">{branches.length}</strong> branches
            </Link>
            <span>
              <TagIcon /> <strong className="color-fg-default">{tags.length}</strong> tags
            </span>
          </span>
        )}
      </div>
      <div className="Box mb-4">
        {head ? (
          <div className="Box-header d-flex flex-items-center flex-wrap" style={{ gap: 8 }}>
            <AgentLink agent={head.agent} verified={head.agentVerified} />
            <Link href={`/repositories/${repo}/commit/${head.id}`} className="color-fg-default gh-truncate" style={{ maxWidth: 480 }}>
              {head.subject}
            </Link>
            <CommitBadges c={head} />
            <span className="ml-auto f6 color-fg-muted d-flex flex-items-center" style={{ gap: 8 }}>
              <Link href={`/repositories/${repo}/commit/${head.id}`} className="text-mono color-fg-muted">
                {shortSha(head.id)}
              </Link>
              <RelativeTime iso={head.committer.time} />
              <Link href={`/repositories/${repo}/commits${qs({ ref: current, path })}`} className="color-fg-default text-bold">
                <HistoryIcon /> History
              </Link>
            </span>
          </div>
        ) : null}
        <table className="width-full" style={{ borderCollapse: "collapse" }} aria-label="Files">
          <tbody>
            {path ? (
              <tr className="border-top">
                <td className="p-2 pl-3" colSpan={3}>
                  <Link href={`/repositories/${repo}/tree${qs({ ref: current, path: parent })}`} className="text-bold" aria-label="Parent directory">
                    ..
                  </Link>
                </td>
              </tr>
            ) : null}
            {tree.entries.map((e) => {
              const href =
                e.type === "TREE"
                  ? `/repositories/${repo}/tree${qs({ ref: current, path: e.path })}`
                  : `/repositories/${repo}/blob${qs({ ref: current, path: e.path })}`;
              return (
                <tr key={e.path} className="border-top" data-testid="tree-entry">
                  <td className="p-2 pl-3" style={{ width: "30%" }}>
                    <span className="d-inline-flex flex-items-center" style={{ gap: 8 }}>
                      {e.type === "TREE" ? (
                        <FileDirectoryFillIcon className="color-fg-accent" />
                      ) : e.type === "SUBMODULE" ? (
                        <FileSubmoduleIcon className="color-fg-muted" />
                      ) : (
                        <FileIcon className="color-fg-muted" />
                      )}
                      {e.type === "SUBMODULE" ? <span>{e.name}</span> : <Link href={href}>{e.name}</Link>}
                    </span>
                  </td>
                  <td className="p-2 f6 color-fg-muted gh-hide-sm">
                    {e.lastCommit ? (
                      <Link href={`/repositories/${repo}/commit/${e.lastCommit.id}`} className="color-fg-muted">
                        {e.lastCommit.subject}
                      </Link>
                    ) : null}
                  </td>
                  <td className="p-2 pr-3 f6 color-fg-muted text-right" style={{ whiteSpace: "nowrap" }}>
                    {e.lastCommit ? <RelativeTime iso={e.lastCommit.committer.time} /> : null}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {tree.readme ? (
        <div className="Box" id="readme">
          <div className="Box-header d-flex flex-items-center" style={{ gap: 8 }}>
            <BookIcon />
            <h2 className="Box-title">{tree.readme.path.split("/").pop()}</h2>
          </div>
          <div className="Box-body px-5 py-4">
            {isMarkdown(tree.readme.path) ? (
              <Markdown source={tree.readme.content} baseHref={`/repositories/${repo}/blob?ref=${encodeURIComponent(current)}&path=${path ? encodeURIComponent(path + "/") : ""}`} />
            ) : (
              <CodeView path={tree.readme.path} content={tree.readme.content} />
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}
