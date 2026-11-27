import Link from "next/link";
import type { Metadata } from "next";
import { SearchIcon } from "@primer/octicons-react";
import { api, enc } from "@/lib/api";
import type { Repository, SearchResult } from "@/lib/types";
import { Blankslate } from "@/components/Blankslate";
import { plural, qs } from "@/lib/format";

export const metadata: Metadata = { title: "Search" };

export default async function SearchPage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<{ q?: string; ref?: string }> }) {
  const { repo } = await params;
  const { q = "", ref } = await searchParams;
  const r = await api<Repository>(`/v1/repositories/${enc(repo)}`);
  const current = ref || r.defaultBranch || "main";
  const result = q.trim() ? await api<SearchResult>(`/v1/repositories/${enc(repo)}/search${qs({ query: q, revision: current })}`) : null;
  const byFile = new Map<string, SearchResult["matches"]>();
  for (const m of result?.matches ?? []) byFile.set(m.path, [...(byFile.get(m.path) ?? []), m]);
  return (
    <div className="gh-container">
      <form method="get" className="d-flex mb-4" style={{ gap: 8 }} role="search">
        <label htmlFor="code-q" className="sr-only">
          Search code
        </label>
        <input id="code-q" name="q" defaultValue={q} className="form-control flex-1" placeholder={`Search code on ${current}`} />
        {ref ? <input type="hidden" name="ref" value={ref} /> : null}
        <button className="btn" type="submit">
          <SearchIcon /> Search
        </button>
      </form>
      {!result ? (
        <div className="Box">
          <Blankslate icon={<SearchIcon size={24} />} title="Search this repository">
            gitai searches the files of {current} on the server.
          </Blankslate>
        </div>
      ) : result.matches.length === 0 ? (
        <div className="Box">
          <Blankslate title={`No results for “${q}”`} />
        </div>
      ) : (
        <>
          <p className="f6 color-fg-muted mb-3">
            {plural(result.matches.length, "result")} in {plural(byFile.size, "file")}
            {result.truncated ? " (truncated)" : ""}
          </p>
          {[...byFile.entries()].map(([path, ms]) => (
            <div key={path} className="Box mb-3" data-testid="search-file">
              <div className="Box-header py-2">
                <Link href={`/repositories/${repo}/blob${qs({ ref: current, path })}`} className="text-mono text-bold">
                  {path}
                </Link>
              </div>
              <div className="gh-code">
                <table>
                  <tbody>
                    {ms.map((m) => (
                      <tr key={m.lineNumber}>
                        <td className="gh-ln">
                          <Link href={`/repositories/${repo}/blob${qs({ ref: current, path })}#L${m.lineNumber}`}>{m.lineNumber}</Link>
                        </td>
                        <td>{m.line}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );
}
