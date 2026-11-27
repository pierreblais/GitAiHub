-- Identity context
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    create_time   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash  TEXT PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    expire_time TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_expire_time ON sessions (expire_time);

-- Source context: the repositories GitaiHub knows on the gitai server.
CREATE TABLE repositories (
    name           TEXT PRIMARY KEY,
    relative_path  TEXT NOT NULL UNIQUE,
    default_branch TEXT NOT NULL DEFAULT '',
    discover_time  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Agent activity context: a copy of each repository's changeset log.
-- The cursor of a repository is its highest sequence, saved in the same
-- transaction as the changeset, so a restart resumes without gaps.
CREATE TABLE changesets (
    repository  TEXT NOT NULL REFERENCES repositories(name) ON DELETE CASCADE,
    sequence    BIGINT NOT NULL,
    reference   TEXT NOT NULL,
    old_oid     TEXT NOT NULL,
    new_oid     TEXT NOT NULL,
    operation   TEXT NOT NULL,
    actor       TEXT NOT NULL,
    record_time TIMESTAMPTZ NOT NULL,
    rewritten   BOOLEAN NOT NULL DEFAULT false,
    kind        TEXT NOT NULL,
    members     JSONB NOT NULL DEFAULT '[]',
    commits     JSONB NOT NULL DEFAULT '[]',
    PRIMARY KEY (repository, sequence)
);
CREATE INDEX changesets_record_time ON changesets (record_time DESC, repository, sequence DESC);
CREATE INDEX changesets_actor ON changesets (actor);

-- The first changeset that brought each commit into the repository: which
-- agent landed it. Blame and history read this to show the real agent.
CREATE TABLE landed_commits (
    repository     TEXT NOT NULL REFERENCES repositories(name) ON DELETE CASCADE,
    commit_id      TEXT NOT NULL,
    sequence       BIGINT NOT NULL,
    agent          TEXT NOT NULL,
    operation      TEXT NOT NULL,
    reference      TEXT NOT NULL,
    subject        TEXT NOT NULL DEFAULT '',
    auto_revert_of TEXT NOT NULL DEFAULT '',
    resolved_by    TEXT NOT NULL DEFAULT '',
    record_time    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (repository, commit_id)
);
CREATE INDEX landed_commits_agent ON landed_commits (agent, record_time DESC);
CREATE INDEX landed_commits_auto_revert_of ON landed_commits (auto_revert_of) WHERE auto_revert_of <> '';
