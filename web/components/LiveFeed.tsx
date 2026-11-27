"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import type { Changeset } from "@/lib/types";
import { ChangesetItem } from "./ChangesetItem";

/**
 * The live activity feed: server-rendered history, then new changesets pushed
 * by the API over server-sent events (fed by gitai's WatchChangesets).
 */
export function LiveFeed({
  initial,
  repository,
  agent,
  showRepo = true,
  nextPageToken,
  moreHref,
}: {
  initial: Changeset[];
  repository?: string;
  agent?: string;
  showRepo?: boolean;
  nextPageToken?: string;
  moreHref?: string;
}) {
  const [items, setItems] = useState(initial);
  const [fresh, setFresh] = useState<Set<string>>(new Set());
  const [live, setLive] = useState(false);
  const seen = useRef(new Set(initial.map((c) => c.name)));

  useEffect(() => {
    const params = new URLSearchParams();
    if (repository) params.set("repository", repository);
    if (agent) params.set("agent", agent);
    const es = new EventSource(`/api/v1/changesets:watch?${params}`);
    es.onopen = () => setLive(true);
    es.onerror = () => setLive(false);
    es.addEventListener("changeset", (ev) => {
      const cs = JSON.parse((ev as MessageEvent).data) as Changeset;
      if (seen.current.has(cs.name)) return;
      seen.current.add(cs.name);
      setItems((prev) => [cs, ...prev].slice(0, 200));
      setFresh((prev) => new Set(prev).add(cs.name));
    });
    return () => es.close();
  }, [repository, agent]);

  return (
    <div>
      <div className="d-flex flex-items-center mb-3 f6 color-fg-muted" style={{ gap: 6 }} aria-live="polite">
        <span className={`gh-live-dot ${live ? "" : "off"}`} aria-hidden="true" />
        {live ? "Live: new lands appear here as gitai records them" : "Connecting to the live feed…"}
      </div>
      {items.length === 0 ? (
        <p className="color-fg-muted">No activity yet. When an agent lands a change, it shows up here.</p>
      ) : (
        <div className="pt-1">
          {items.map((cs) => (
            <ChangesetItem key={cs.name} cs={cs} showRepo={showRepo} isNew={fresh.has(cs.name)} />
          ))}
        </div>
      )}
      {nextPageToken && moreHref ? (
        <div className="text-center mt-2">
          <Link className="btn btn-sm" href={`${moreHref}${moreHref.includes("?") ? "&" : "?"}pageToken=${encodeURIComponent(nextPageToken)}`}>
            Older activity
          </Link>
        </div>
      ) : null}
    </div>
  );
}
