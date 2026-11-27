import Link from "next/link";
import { FileDirectoryFillIcon, FileIcon, FileSubmoduleIcon, HistoryIcon, BookIcon, GitBranchIcon, TagIcon, RepoIcon, CodeIcon, TriangleDownIcon, SearchIcon, PulseIcon, CodeReviewIcon, ServerIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Commit, Ref, Repository, Tree } from "@/lib/types";
import { RefSelector } from "./RefSelector";
import { Breadcrumbs } from "./Breadcrumbs";
import { Markdown } from "./Markdown";
import { CodeView } from "./CodeView";
import { AgentAvatar, AgentLink } from "./Agent";
import { RelativeTime } from "./Time";
import { CommitBadges } from "./CommitList";
import { Blankslate } from "./Blankslate";
import { isMarkdown } from "@/lib/lang";
import { plural, qs, shortSha } from "@/lib/format";

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
    api<{ commits: Commit[]; totalSize?: number }>(`/v1/repositories/${enc(repo)}/commits${qs({ revision: current, path, pageSize: 1 })}`),
  ]);
  const head = latest.commits[0];
  const parent = path && path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
  const home = !path;
  const files = (
    <div style={{ minWidth: 0 }}>
      <div className="d-flex flex-items-center flex-wrap mb-3" style={{ gap: 8 }}>
        <RefSelector current={current} branches={branches} tags={tags} hrefFor={(ref) => `/repositories/${repo}/tree${qs({ ref, path })}`} />
        {path ? (
          <Breadcrumbs repo={repo} gitRef={current} path={path} />
        ) : (
          <span className="d-flex flex-items-center gh-hide-sm" style={{ gap: 16 }}>
            <Link href={`/repositories/${repo}/branches`} className="color-fg-muted no-underline">
              <GitBranchIcon /> <strong className="color-fg-default">{branches.length}</strong> {branches.length === 1 ? "Branch" : "Branches"}
            </Link>
            <span className="color-fg-muted">
              <TagIcon /> <strong className="color-fg-default">{tags.length}</strong> {tags.length === 1 ? "Tag" : "Tags"}
            </span>
          </span>
        )}
        <span className="flex-1" />
        <form action={`/repositories/${repo}/search`} method="get" className="gh-search gh-hide-sm" role="search">
          <label htmlFor="go-to-file" className="sr-only">
            Search code
          </label>
          <SearchIcon className="gh-search-icon" />
          <input id="go-to-file" name="q" type="search" placeholder="Search code" style={{ width: 200 }} />
          <input type="hidden" name="ref" value={current} />
        </form>
        <details className="gh-ref-menu">
          <summary className="btn btn-sm btn-primary-gh" aria-haspopup="menu">
            <CodeIcon /> <span className="ml-1">Code</span> <TriangleDownIcon />
          </summary>
          <div className="gh-ref-list p-3" role="menu" style={{ right: 0, left: "auto", width: 340 }}>
            <div className="d-flex flex-items-center mb-2 text-bold" style={{ gap: 6 }}>
              <ServerIcon /> gitai repository
            </div>
            <input className="form-control input-sm width-full text-mono" readOnly value={r.relativePath} aria-label="Relative path on the gitai server" />
            <p className="f6 color-fg-muted mt-2 mb-0">Agents read and land changes on this repository through the gitai SDK or the graft CLI.</p>
          </div>
        </details>
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
              <span>·</span>
              <RelativeTime iso={head.committer.time} />
              <Link href={`/repositories/${repo}/commits${qs({ ref: current, path })}`} className="color-fg-default text-bold ml-2 no-underline" style={{ whiteSpace: "nowrap" }}>
                <HistoryIcon /> {latest.totalSize !== undefined ? `${latest.totalSize.toLocaleString("en-US")} ${latest.totalSize === 1 ? "Commit" : "Commits"}` : "History"}
              </Link>
            </span>
          </div>
        ) : null}
        <table className="width-full gh-file-table" style={{ borderCollapse: "collapse" }} aria-label="Files">
          <tbody>
            {path ? (
              <tr>
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
                <tr key={e.path} data-testid="tree-entry">
                  <td className="p-2 pl-3" style={{ width: "30%" }}>
                    <span className="d-inline-flex flex-items-center" style={{ gap: 8 }}>
                      {e.type === "TREE" ? (
                        <FileDirectoryFillIcon className="color-fg-accent" />
                      ) : e.type === "SUBMODULE" ? (
                        <FileSubmoduleIcon className="color-fg-muted" />
                      ) : (
                        <FileIcon className="color-fg-muted" />
                      )}
                      {e.type === "SUBMODULE" ? <span>{e.name}</span> : <Link href={href} className="color-fg-default">{e.name}</Link>}
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
          <div className="gh-readme-head">
            <span className="gh-readme-tab">
              <BookIcon /> <h2 className="Box-title d-inline">{tree.readme.path.split("/").pop()}</h2>
            </span>
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
  if (!home) return <div className="gh-container">{files}</div>;
  const description = readmeSummary(tree.readme?.content);
  return (
    <div className="gh-container">
      <div className="gh-repo-title">
        <RepoIcon className="color-fg-muted" size={20} />
        <h1 className="f3 text-bold">
          <Link href={`/repositories/${repo}`} className="color-fg-default">
            {r.repositoryId}
          </Link>
        </h1>
        <span className="Label">gitai</span>
      </div>
      <div className="gh-repo-layout">
        {files}
        <aside className="gh-about" aria-labelledby="about-heading">
          <section>
            <h2 id="about-heading">About</h2>
            {description ? <p className="f5 mb-3">{description}</p> : <p className="f5 color-fg-muted mb-3">No description, website, or topics provided.</p>}
            <ul className="gh-about-list">
              {tree.readme ? (
                <li>
                  <BookIcon /> <a href="#readme">Readme</a>
                </li>
              ) : null}
              <li>
                <PulseIcon /> <Link href={`/repositories/${repo}/activity`}>Activity</Link>
              </li>
              <li>
                <ServerIcon /> <span className="text-mono">{r.relativePath}</span>
              </li>
              <li>
                <GitBranchIcon /> <Link href={`/repositories/${repo}/branches`}><strong>{branches.length}</strong> {branches.length === 1 ? "branch" : "branches"}</Link>
              </li>
              <li>
                <CodeReviewIcon /> <Link href={`/repositories/${repo}/reviews`}>Reviews</Link>
              </li>
            </ul>
          </section>
          <section>
            <h2>
              <Link href={`/repositories/${repo}/activity`} className="color-fg-default">
                Changesets
              </Link>{" "}
              <span className="Counter">{r.changesetCount}</span>
            </h2>
            <p className="f6 color-fg-muted">
              {r.lastActivityTime ? (
                <>
                  Last land <RelativeTime iso={r.lastActivityTime} />
                </>
              ) : (
                "Nothing landed yet"
              )}
            </p>
          </section>
          {r.topAgents.length > 0 ? (
            <section>
              <h2>
                Agents <span className="Counter">{r.topAgents.length}</span>
              </h2>
              <ul className="gh-about-list">
                {r.topAgents.map((a) => (
                  <li key={a.agent}>
                    <Link href={`/agents/${a.agent}`} className="d-flex flex-items-center color-fg-default" style={{ gap: 8 }}>
                      <AgentAvatar agent={a.agent} size={24} round />
                      <span className="text-bold">{a.agent}</span>
                    </Link>
                    <span className="f6">{plural(a.count, "land")}</span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </aside>
      </div>
    </div>
  );
}

/** The first plain paragraph of a README, used as the repository description. */
function readmeSummary(md?: string): string {
  if (!md) return "";
  for (const block of md.split(/\n\s*\n/)) {
    const t = block.trim();
    if (!t || /^(#|```|[-*>|]|!\[|<)/.test(t)) continue;
    return t.replace(/\s+/g, " ").replace(/[*_`]/g, "").slice(0, 200);
  }
  return "";
}
