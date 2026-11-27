// Same-origin proxy from the browser to the GitaiHub API: the session cookie
// stays first-party, and the live feed's server-sent events stream through.
import { NextRequest } from "next/server";

const API_URL = process.env.GITAIHUB_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

const HOP_BY_HOP = new Set(["connection", "keep-alive", "transfer-encoding", "upgrade", "content-length", "content-encoding", "host"]);

async function proxy(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const { path } = await ctx.params;
  const url = `${API_URL}/v1/${path.map(encodeURIComponent).join("/").replace(/%3A/gi, ":")}${req.nextUrl.search}`;
  const headers = new Headers();
  for (const [k, v] of req.headers) {
    if (!HOP_BY_HOP.has(k.toLowerCase())) headers.set(k, v);
  }
  const init: RequestInit & { duplex?: "half" } = { method: req.method, headers, redirect: "manual", signal: req.signal };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.arrayBuffer();
  }
  let upstream: Response;
  try {
    upstream = await fetch(url, init);
  } catch {
    return Response.json({ error: { code: 503, message: "The GitaiHub API is unreachable.", status: "UNAVAILABLE" } }, { status: 503 });
  }
  const out = new Headers();
  for (const [k, v] of upstream.headers) {
    if (!HOP_BY_HOP.has(k.toLowerCase()) && k.toLowerCase() !== "set-cookie") out.set(k, v);
  }
  for (const c of upstream.headers.getSetCookie()) out.append("set-cookie", c);
  if (upstream.headers.get("content-type")?.startsWith("text/event-stream")) {
    out.set("Cache-Control", "no-cache, no-transform");
  }
  return new Response(upstream.body, { status: upstream.status, headers: out });
}

export { proxy as GET, proxy as POST, proxy as DELETE, proxy as PATCH, proxy as PUT };
