import Link from "next/link";
import type { Metadata } from "next";
import { api, enc } from "@/lib/api";
import type { Blame, Commit } from "@/lib/types";
import { FileHeader } from "@/components/FileHeader";
import { AgentLink } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { agentColor } from "@/lib/agents";
import { highlightLines } from "@/lib/highlight";
import { languageFor } from "@/lib/lang";
import { qs, shortSha } from "@/lib/format";

type Search = { ref?: string; path?: string };

export async function generateMetadata({ searchParams }: { searchParams: Promise<Search> }): Promise<Metadata> {
  const { path } = await searchParams;
  return { title: `Blame ${path ?? ""}` };
}

export default async function BlamePage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<Search> }) {
  const { repo } = await params;
  const { ref, path = "" } = await searchParams;
  const header = await FileHeader({ repo, gitRef: ref, path, mode: "blame" });
  const blame = await api<Blame>(`/v1/repositories/${enc(repo)}/blame${qs({ revision: header.current, path })}`);
  const html = await highlightLines(blame.lines.map((l) => l.content).join("\n"), languageFor(path));
  const agents = new Map<string, number>();
  for (const l of blame.lines) {
    const a = blame.commits[l.commitId]?.agent ?? "unknown";
    agents.set(a, (agents.get(a) ?? 0) + 1);
  }
  return (
    <div className="gh-container">
      {header.node}
      <div className="Box">
        <div className="Box-header py-2 d-flex flex-items-center flex-wrap f6" style={{ gap: 16 }}>
          <span className="color-fg-muted">Lines by agent:</span>
          {[...agents.entries()]
            .sort((a, b) => b[1] - a[1])
            .map(([a, n]) => (
              <span key={a} className="d-inline-flex flex-items-center" style={{ gap: 6 }}>
                <span style={{ width: 10, height: 10, borderRadius: 2, background: agentColor(a), display: "inline-block" }} />
                <AgentLink agent={a} size={16} /> <span className="color-fg-muted">{Math.round((n / blame.lines.length) * 100)}%</span>
              </span>
            ))}
        </div>
        <div className="gh-code gh-highlight shiki">
          <table>
            <tbody>
              {blame.lines.map((l, i) => {
                const c: Commit | undefined = blame.commits[l.commitId];
                const first = i === 0 || blame.lines[i - 1].commitId !== l.commitId;
                const agent = c?.agent ?? "unknown";
                return (
                  <tr key={i} id={`L${l.lineNumber}`} className={first ? "gh-blame-row" : ""} data-testid="blame-line">
                    <td className="gh-blame-bar" style={{ background: agentColor(agent) }} aria-hidden="true" />
                    <td className="gh-blame-meta px-2 py-0 color-fg-muted">
                      {first && c ? (
                        <div className="d-flex flex-items-center py-1" style={{ gap: 8 }}>
                          <span style={{ minWidth: 70 }} className="f6">
                            <RelativeTime iso={c.committer.time} />
                          </span>
                          <AgentLink agent={agent} size={16} verified={c.agentVerified} />
                          <Link href={`/repositories/${repo}/commit/${c.id}`} className="color-fg-default gh-truncate flex-1" title={c.subject}>
                            {c.subject}
                          </Link>
                          <span className="text-mono f6">{shortSha(c.id)}</span>
                        </div>
                      ) : null}
                    </td>
                    <td className="gh-ln">
                      <a href={`#L${l.lineNumber}`}>{l.lineNumber}</a>
                    </td>
                    <td dangerouslySetInnerHTML={{ __html: html[i] || " " }} />
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
