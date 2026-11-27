import "server-only";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import type { ApiErrorBody } from "./types";

/** Where the server side reaches the GitaiHub API (12-factor backing service URL). */
export const API_URL = process.env.GITAIHUB_API_URL ?? "http://localhost:8080";
export const SESSION_COOKIE = "gitaihub_session";

export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

/**
 * GET a resource of the API as the signed-in user. A missing session sends
 * the browser to the sign-in page, and a missing resource renders the 404 page.
 */
export async function api<T>(path: string, opts: { allowNotFound?: boolean } = {}): Promise<T> {
  const jar = await cookies();
  const token = jar.get(SESSION_COOKIE)?.value;
  if (!token) redirect("/login");
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
      cache: "no-store",
    });
  } catch {
    throw new ApiError(503, "UNAVAILABLE", "The GitaiHub API is unreachable.");
  }
  if (res.ok) return (await res.json()) as T;
  const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
  if (res.status === 401) redirect("/login");
  if (res.status === 404 && !opts.allowNotFound) notFound();
  throw new ApiError(res.status, body?.error.status ?? "UNKNOWN", body?.error.message ?? res.statusText);
}

/** Like api, but returns null for a 404 instead of rendering the 404 page. */
export async function apiOrNull<T>(path: string): Promise<T | null> {
  try {
    return await api<T>(path, { allowNotFound: true });
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null;
    throw e;
  }
}

export function enc(s: string): string {
  return encodeURIComponent(s);
}
