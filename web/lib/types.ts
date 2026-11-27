// Resources of the GitaiHub API (see proto/gitaihub/v1). Field names follow
// the proto3 JSON mapping.

export interface AgentCount {
  agent: string;
  count: number;
}

export interface Repository {
  name: string;
  repositoryId: string;
  relativePath: string;
  defaultBranch?: string;
  lastActivityTime?: string;
  changesetCount: number;
  topAgents: AgentCount[];
}

export interface Signature {
  name: string;
  email: string;
  time?: string;
}

export interface Landing {
  agent: string;
  sequence: string;
  operation: string;
  reference: string;
  recordTime?: string;
}

export interface Trailer {
  key: string;
  value: string;
}

export interface Commit {
  name: string;
  id: string;
  parentIds: string[];
  subject: string;
  body: string;
  author: Signature;
  committer: Signature;
  agent: string;
  agentVerified: boolean;
  landing?: Landing;
  trailers: Trailer[];
  autoRevertOf?: string;
  resolvedBy?: string;
}

export interface Ref {
  name: string;
  refId: string;
  kind: "BRANCH" | "TAG";
  target?: Commit;
}

export interface TreeEntry {
  name: string;
  path: string;
  type: "TREE" | "BLOB" | "SUBMODULE";
  mode: number;
  oid: string;
  size: string;
  lastCommit?: Commit;
}

export interface Blob {
  path: string;
  oid: string;
  size: string;
  content: string;
  binary: boolean;
  truncated: boolean;
}

export interface Tree {
  revision: string;
  commitId: string;
  path: string;
  entries: TreeEntry[];
  readme?: Blob;
}

export interface FileDiff {
  oldPath: string;
  newPath: string;
  status: "ADDED" | "MODIFIED" | "DELETED" | "RENAMED";
  additions: number;
  deletions: number;
  binary: boolean;
  tooLarge: boolean;
  patch: string;
}

export interface CommitDetail {
  commit: Commit;
  files: FileDiff[];
  stats: { additions: number; deletions: number; files: number };
}

export interface BlameLine {
  lineNumber: number;
  commitId: string;
  content: string;
}

export interface Blame {
  commitId: string;
  lines: BlameLine[];
  commits: Record<string, Commit>;
}

export type ChangeClass = "CODE" | "COMMENT_ONLY" | "DOCS" | "TESTS" | "GENERATED" | "CONFIG";

export interface ReviewFile {
  path: string;
  class: ChangeClass;
  reason: string;
}

export interface Verdict {
  actor: string;
  kind: "APPROVE" | "REQUEST_CHANGES";
  commitId: string;
  body: string;
  createTime: string;
  current: boolean;
}

export interface ReviewComment {
  id: string;
  actor: string;
  body: string;
  path?: string;
  line?: number;
  commitId: string;
  createTime: string;
}

export interface Requirement {
  rule: string;
  description: string;
  requiredApprovals: number;
  approvals: number;
  satisfied: boolean;
  paths: string[];
}

export interface Review {
  name: string;
  id: string;
  targetBranch: string;
  baseCommitId: string;
  headCommitId: string;
  author: string;
  title: string;
  description: string;
  state: "OPEN" | "LANDED" | "CLOSED";
  files: ReviewFile[];
  fileGroups: { class: ChangeClass; files: ReviewFile[] }[];
  verdicts: Verdict[];
  currentVerdicts: Verdict[];
  comments: ReviewComment[];
  requirements: Requirement[];
  approved: boolean;
  createTime: string;
  updateTime: string;
}

export type ChangesetKind =
  | "LAND"
  | "BATCH_LAND"
  | "CONFLICT_RESOLVED"
  | "AUTO_REVERT"
  | "BRANCH_CREATED"
  | "BRANCH_DELETED"
  | "UPDATE";

export interface Changeset {
  name: string;
  repository: string;
  sequence: string;
  reference: string;
  branch?: string;
  oldOid: string;
  newOid: string;
  operation: string;
  actor: string;
  agents: string[];
  recordTime: string;
  rewritten: boolean;
  kind: ChangesetKind;
  members: { originalCommitId: string; commitId: string; actor: string; rewritten: boolean; resolvedPaths: string[] }[];
  commits: { commitId: string; agent: string; subject: string; autoRevertOf?: string; resolvedBy?: string }[];
}

export interface Agent {
  name: string;
  agentId: string;
  changesetCount: number;
  landCount: number;
  commitCount: number;
  conflictsResolvedCount: number;
  autoRevertCount: number;
  revertedCommitCount: number;
  repositories: string[];
  firstSeenTime?: string;
  lastSeenTime?: string;
  dailyCommits?: { day: string; count: number }[];
}

export interface User {
  name: string;
  username: string;
  displayName: string;
  canReview: boolean;
  createTime: string;
}

export interface SearchResult {
  commitId: string;
  matches: { path: string; lineNumber: number; line: string; before: string[]; after: string[] }[];
  truncated: boolean;
}

export interface ApiErrorBody {
  error: { code: number; message: string; status: string };
}
