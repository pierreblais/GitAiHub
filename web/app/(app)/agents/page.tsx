import type { Metadata } from "next";
import Link from "next/link";
import { CpuIcon } from "@primer/octicons-react";
import { api } from "@/lib/api";
import type { Agent } from "@/lib/types";
import { AgentAvatar } from "@/components/Agent";
import { RelativeTime } from "@/components/Time";
import { Blankslate } from "@/components/Blankslate";

export const metadata: Metadata = { title: "Agents" };

export default async function AgentsPage() {
  const { agents } = await api<{ agents: Agent[] }>("/v1/agents");
  return (
    <div className="gh-container">
      <h1 className="f2 text-normal mb-1">Agents</h1>
      <p className="color-fg-muted mb-4">The identities that land changes on this gitai server, from its changeset log.</p>
      {agents.length === 0 ? (
        <div className="Box">
          <Blankslate icon={<CpuIcon size={24} />} title="No agent activity yet" />
        </div>
      ) : (
        <div className="d-flex flex-wrap" style={{ gap: 16 }}>
          {agents.map((a) => (
            <Link key={a.name} href={`/agents/${a.agentId}`} className="Box p-3 no-underline color-fg-default" style={{ width: 290 }} data-testid="agent-card">
              <div className="d-flex flex-items-center mb-3" style={{ gap: 10 }}>
                <AgentAvatar agent={a.agentId} size={40} />
                <div>
                  <div className="f4 text-bold">{a.agentId}</div>
                  <div className="f6 color-fg-muted">
                    Last active <RelativeTime iso={a.lastSeenTime} />
                  </div>
                </div>
              </div>
              <dl className="d-flex f6 m-0" style={{ gap: 16 }}>
                <Stat label="Lands" value={a.landCount} />
                <Stat label="Commits" value={a.commitCount} />
                <Stat label="Conflicts" value={a.conflictsResolvedCount} />
                <Stat label="Reverted" value={a.revertedCommitCount} />
              </dl>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <dt className="color-fg-muted">{label}</dt>
      <dd className="f3 text-bold m-0">{value}</dd>
    </div>
  );
}
