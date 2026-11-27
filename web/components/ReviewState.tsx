import { CodeReviewIcon, GitMergeIcon, XCircleFillIcon } from "@primer/octicons-react";
import type { Review } from "@/lib/types";

export function ReviewStateLabel({ state, large = false }: { state: Review["state"]; large?: boolean }) {
  const cls = large ? "State" : "Label";
  if (state === "LANDED")
    return (
      <span className={`${cls} ${large ? "State--merged" : "Label--done"}`}>
        <GitMergeIcon /> Landed
      </span>
    );
  if (state === "CLOSED")
    return (
      <span className={`${cls} ${large ? "State--closed" : "Label--danger"}`}>
        <XCircleFillIcon /> Closed
      </span>
    );
  return (
    <span className={`${cls} ${large ? "State--open" : "Label--success"}`}>
      <CodeReviewIcon /> Open
    </span>
  );
}

export const CLASS_LABEL: Record<string, { label: string; className: string }> = {
  CODE: { label: "Code", className: "Label--accent" },
  TESTS: { label: "Tests", className: "Label--success" },
  CONFIG: { label: "Config", className: "Label--attention" },
  DOCS: { label: "Docs", className: "Label--secondary" },
  COMMENT_ONLY: { label: "Comment only", className: "Label--secondary" },
  GENERATED: { label: "Generated", className: "Label--secondary" },
};
