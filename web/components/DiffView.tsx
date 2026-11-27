import { parsePatch } from "@/lib/diff";
import type { FileDiff } from "@/lib/types";
import { plural } from "@/lib/format";

const STATUS_LABEL: Record<FileDiff["status"], string> = {
  ADDED: "added",
  MODIFIED: "modified",
  DELETED: "deleted",
  RENAMED: "renamed",
};

export function DiffStat({ additions, deletions }: { additions: number; deletions: number }) {
  return (
    <span className="f6 text-mono">
      <span className="color-fg-success">+{additions}</span> <span className="color-fg-danger">−{deletions}</span>
    </span>
  );
}

export function FileDiffView({ file, id }: { file: FileDiff; id: string }) {
  const path = file.status === "RENAMED" ? `${file.oldPath} → ${file.newPath}` : file.newPath || file.oldPath;
  const lines = parsePatch(file.patch);
  return (
    <div className="Box mb-3" id={id}>
      <div className="Box-header py-2 d-flex flex-items-center" style={{ gap: 8 }}>
        <DiffStat additions={file.additions} deletions={file.deletions} />
        <span className="text-mono f6 text-bold gh-truncate">{path}</span>
        <span className="Label ml-auto">{STATUS_LABEL[file.status]}</span>
      </div>
      {file.binary ? (
        <div className="p-3 color-fg-muted f6">Binary file not shown.</div>
      ) : file.tooLarge || (lines.length === 0 && file.additions + file.deletions > 0) ? (
        <div className="p-3 color-fg-muted f6">Large diff not rendered ({plural(file.additions + file.deletions, "changed line")}).</div>
      ) : lines.length === 0 ? (
        <div className="p-3 color-fg-muted f6">No content changes.</div>
      ) : (
        <div className="gh-code gh-diff">
          <table>
            <tbody>
              {lines.map((l, i) =>
                l.kind === "hunk" || l.kind === "meta" ? (
                  <tr key={i}>
                    <td className="gh-ln gh-hunk" />
                    <td className="gh-ln gh-hunk" />
                    <td className="gh-hunk" colSpan={2}>
                      {l.text}
                    </td>
                  </tr>
                ) : (
                  <tr key={i}>
                    <td className={`gh-ln ${l.kind === "add" ? "gh-add" : l.kind === "del" ? "gh-del" : ""}`}>{l.oldNumber ?? ""}</td>
                    <td className={`gh-ln ${l.kind === "add" ? "gh-add" : l.kind === "del" ? "gh-del" : ""}`}>{l.newNumber ?? ""}</td>
                    <td className={`gh-sign ${l.kind === "add" ? "gh-add" : l.kind === "del" ? "gh-del" : ""}`}>
                      {l.kind === "add" ? "+" : l.kind === "del" ? "−" : " "}
                    </td>
                    <td className={l.kind === "add" ? "gh-add" : l.kind === "del" ? "gh-del" : ""}>{l.text || " "}</td>
                  </tr>
                ),
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
