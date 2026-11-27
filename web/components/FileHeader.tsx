import Link from "next/link";
import type { Ref, Repository } from "@/lib/types";
import { api, enc } from "@/lib/api";
import { RefSelector } from "./RefSelector";
import { Breadcrumbs } from "./Breadcrumbs";
import { qs } from "@/lib/format";

/** Ref switcher, breadcrumbs and the Code / Blame / History switch of a file. */
export async function FileHeader({ repo, gitRef, path, mode }: { repo: string; gitRef?: string; path: string; mode: "blob" | "blame" }) {
  const [r, { branches }, { tags }] = await Promise.all([
    api<Repository>(`/v1/repositories/${enc(repo)}`),
    api<{ branches: Ref[] }>(`/v1/repositories/${enc(repo)}/branches`),
    api<{ tags: Ref[] }>(`/v1/repositories/${enc(repo)}/tags`),
  ]);
  const current = gitRef || r.defaultBranch || "main";
  return {
    current,
    node: (
      <div className="d-flex flex-items-center flex-wrap mb-3" style={{ gap: 12 }}>
        <RefSelector current={current} branches={branches} tags={tags} hrefFor={(ref) => `/repositories/${repo}/${mode}${qs({ ref, path })}`} />
        <Breadcrumbs repo={repo} gitRef={current} path={path} leafIsFile />
        <div className="BtnGroup ml-auto">
          <Link className={`btn btn-sm BtnGroup-item ${mode === "blob" ? "selected" : ""}`} href={`/repositories/${repo}/blob${qs({ ref: current, path })}`} aria-current={mode === "blob" ? "page" : undefined}>
            Code
          </Link>
          <Link className={`btn btn-sm BtnGroup-item ${mode === "blame" ? "selected" : ""}`} href={`/repositories/${repo}/blame${qs({ ref: current, path })}`} aria-current={mode === "blame" ? "page" : undefined}>
            Blame
          </Link>
          <Link className="btn btn-sm BtnGroup-item" href={`/repositories/${repo}/commits${qs({ ref: current, path })}`}>
            History
          </Link>
        </div>
      </div>
    ),
  };
}
