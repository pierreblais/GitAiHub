import { describe, expect, it } from "vitest";
import { parsePatch } from "@/lib/diff";
import { relativeTime, shortSha, formatBytes, pathParts, qs, plural } from "@/lib/format";
import { agentColor, agentInitials } from "@/lib/agents";
import { languageFor, isMarkdown } from "@/lib/lang";

describe("parsePatch", () => {
  it("numbers lines on both sides", () => {
    const lines = parsePatch("@@ -3,3 +3,4 @@ func x() {\n a\n-b\n+c\n+d\n e\n\\ No newline at end of file\n");
    expect(lines.map((l) => l.kind)).toEqual(["hunk", "context", "del", "add", "add", "context", "meta"]);
    expect(lines[1]).toMatchObject({ oldNumber: 3, newNumber: 3, text: "a" });
    expect(lines[2]).toMatchObject({ oldNumber: 4, text: "b" });
    expect(lines[3]).toMatchObject({ newNumber: 4, text: "c" });
    expect(lines[5]).toMatchObject({ oldNumber: 5, newNumber: 6 });
  });
  it("handles an empty patch", () => {
    expect(parsePatch("")).toEqual([]);
  });
});

describe("format", () => {
  const now = new Date("2026-11-27T12:00:00Z");
  it("formats relative times", () => {
    expect(relativeTime("2026-11-27T11:59:50Z", now)).toBe("just now");
    expect(relativeTime("2026-11-27T11:57:00Z", now)).toBe("3 minutes ago");
    expect(relativeTime("2026-11-26T12:00:00Z", now)).toBe("yesterday");
    expect(relativeTime(undefined, now)).toBe("");
  });
  it("formats the rest", () => {
    expect(shortSha("abcdef1234")).toBe("abcdef1");
    expect(formatBytes("2048")).toBe("2.0 KB");
    expect(formatBytes(12)).toBe("12 Bytes");
    expect(pathParts("a/b/c")).toEqual([
      { name: "a", path: "a" },
      { name: "b", path: "a/b" },
      { name: "c", path: "a/b/c" },
    ]);
    expect(qs({ ref: "main", path: "", n: 2 })).toBe("?ref=main&n=2");
    expect(qs({})).toBe("");
    expect(plural(1, "file")).toBe("1 file");
    expect(plural(3, "file")).toBe("3 files");
  });
});

describe("agents", () => {
  it("gives stable colours and initials", () => {
    expect(agentColor("builder")).toBe(agentColor("builder"));
    expect(agentColor("builder")).toMatch(/^#[0-9a-f]{6}$/);
    expect(agentInitials("docs-bot")).toBe("DB");
    expect(agentInitials("planner")).toBe("PL");
  });
});

describe("lang", () => {
  it("maps file names to languages", () => {
    expect(languageFor("limiter/limiter.go")).toBe("go");
    expect(languageFor("src/App.tsx")).toBe("tsx");
    expect(languageFor("Dockerfile")).toBe("dockerfile");
    expect(languageFor("LICENSE")).toBe("text");
    expect(isMarkdown("README.md")).toBe(true);
  });
});
