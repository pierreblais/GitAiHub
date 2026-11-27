export type DiffLineKind = "context" | "add" | "del" | "hunk" | "meta";

export interface DiffLine {
  kind: DiffLineKind;
  oldNumber?: number;
  newNumber?: number;
  text: string;
}

const HUNK = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$/;

/** Parses a unified diff patch (as gitai's CommitDiff returns it) into lines. */
export function parsePatch(patch: string): DiffLine[] {
  const out: DiffLine[] = [];
  let oldN = 0;
  let newN = 0;
  const lines = patch.split("\n");
  if (lines.length && lines[lines.length - 1] === "") lines.pop();
  for (const line of lines) {
    const m = HUNK.exec(line);
    if (m) {
      oldN = Number(m[1]);
      newN = Number(m[3]);
      out.push({ kind: "hunk", text: line });
      continue;
    }
    if (line.startsWith("\\")) {
      out.push({ kind: "meta", text: line });
    } else if (line.startsWith("+")) {
      out.push({ kind: "add", newNumber: newN++, text: line.slice(1) });
    } else if (line.startsWith("-")) {
      out.push({ kind: "del", oldNumber: oldN++, text: line.slice(1) });
    } else {
      out.push({ kind: "context", oldNumber: oldN++, newNumber: newN++, text: line.startsWith(" ") ? line.slice(1) : line });
    }
  }
  return out;
}
