import Link from "next/link";
import { agentColor, agentInitials } from "@/lib/agents";
import { VerifiedIcon } from "@primer/octicons-react";

export function AgentAvatar({ agent, size = 20 }: { agent: string; size?: number }) {
  return (
    <span
      className="gh-avatar"
      aria-hidden="true"
      style={{ width: size, height: size, fontSize: Math.max(8, Math.round(size * 0.42)), background: agentColor(agent) }}
    >
      {agentInitials(agent)}
    </span>
  );
}

/**
 * An agent's name linked to its page. "verified" means the name comes from
 * gitai's changeset log (the authenticated landing identity), not from the
 * commit author, which clients can set freely.
 */
export function AgentLink({ agent, size = 20, verified }: { agent: string; size?: number; verified?: boolean }) {
  return (
    <Link href={`/agents/${encodeURIComponent(agent)}`} className="gh-agent">
      <AgentAvatar agent={agent} size={size} />
      <span>{agent}</span>
      {verified ? (
        <span className="color-fg-success" title="Landed by this agent, according to gitai's changeset log">
          <VerifiedIcon size={14} />
        </span>
      ) : null}
    </Link>
  );
}
