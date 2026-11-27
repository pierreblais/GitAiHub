const BY_EXT: Record<string, string> = {
  go: "go", ts: "typescript", tsx: "tsx", js: "javascript", jsx: "jsx", mjs: "javascript", cjs: "javascript",
  json: "json", md: "markdown", markdown: "markdown", py: "python", rs: "rust", rb: "ruby", java: "java",
  kt: "kotlin", c: "c", h: "c", cc: "cpp", cpp: "cpp", hpp: "cpp", cs: "csharp", sh: "shellscript", bash: "shellscript",
  zsh: "shellscript", yml: "yaml", yaml: "yaml", toml: "toml", sql: "sql", html: "html", css: "css", scss: "scss",
  proto: "proto", mod: "go", dockerfile: "dockerfile", xml: "xml", swift: "swift", php: "php", lua: "lua",
};

const BY_NAME: Record<string, string> = {
  Dockerfile: "dockerfile", Makefile: "makefile", "go.mod": "go", ".gitignore": "text",
};

/** The highlighting language of a file, by name; "text" when unknown. */
export function languageFor(path: string): string {
  const name = path.split("/").pop() ?? path;
  if (BY_NAME[name]) return BY_NAME[name];
  const dot = name.lastIndexOf(".");
  if (dot < 0) return "text";
  return BY_EXT[name.slice(dot + 1).toLowerCase()] ?? "text";
}

export function isMarkdown(path: string): boolean {
  return /\.(md|markdown)$/i.test(path);
}
