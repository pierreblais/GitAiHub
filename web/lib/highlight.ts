import "server-only";
import { createHighlighter, type BundledLanguage, type Highlighter } from "shiki";

const LANGS = [
  "go", "typescript", "tsx", "javascript", "jsx", "json", "markdown", "python", "rust", "yaml", "toml",
  "shellscript", "css", "html", "sql", "dockerfile", "makefile", "proto", "java", "c", "cpp", "ruby",
];

let highlighter: Promise<Highlighter> | null = null;

function getHighlighter() {
  highlighter ??= createHighlighter({ themes: ["github-light", "github-dark"], langs: LANGS });
  return highlighter;
}

/**
 * Highlights code into one HTML string per line (so the page can number and
 * anchor lines). Unknown languages and very large files fall back to plain
 * escaped text.
 */
export async function highlightLines(code: string, lang: string): Promise<string[]> {
  const plain = () => code.split("\n").map(escapeHtml);
  if (lang === "text" || code.length > 400_000) return plain();
  try {
    const h = await getHighlighter();
    if (!h.getLoadedLanguages().includes(lang)) return plain();
    const tokens = h.codeToTokens(code, { lang: lang as BundledLanguage, themes: { light: "github-light", dark: "github-dark" }, defaultColor: false });
    return tokens.tokens.map((line) =>
      line
        .map((t) => {
          const style = Object.entries(t.htmlStyle ?? {})
            .map(([k, v]) => `${k}:${v}`)
            .join(";");
          return `<span style="${style}">${escapeHtml(t.content)}</span>`;
        })
        .join(""),
    );
  } catch {
    return plain();
  }
}

export function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
