package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// ErrAgentNotFound is returned for an agent never seen in the changeset log.
var ErrAgentNotFound = fmt.Errorf("%w: agent", source.ErrNotFound)

// ActivityStore implements activity.Store.
type ActivityStore struct{ DB *pgxpool.Pool }

func (s *ActivityStore) Cursor(ctx context.Context, repo string) (uint64, error) {
	var seq int64
	err := s.DB.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM changesets WHERE repository = $1`, repo).Scan(&seq)
	return uint64(seq), err
}

type memberJSON struct {
	OriginalCommitID string   `json:"original_commit_id"`
	CommitID         string   `json:"commit_id"`
	Actor            string   `json:"actor"`
	Rewritten        bool     `json:"rewritten"`
	ResolvedPaths    []string `json:"resolved_paths,omitempty"`
}

type commitJSON struct {
	CommitID     string `json:"commit_id"`
	Agent        string `json:"agent"`
	Subject      string `json:"subject"`
	AutoRevertOf string `json:"auto_revert_of,omitempty"`
	ResolvedBy   string `json:"resolved_by,omitempty"`
}

func (s *ActivityStore) SaveChangeset(ctx context.Context, cs *activity.Changeset) error {
	members := make([]memberJSON, 0, len(cs.Members))
	for _, m := range cs.Members {
		members = append(members, memberJSON(m))
	}
	commits := make([]commitJSON, 0, len(cs.Commits))
	for _, c := range cs.Commits {
		commits = append(commits, commitJSON(c))
	}
	mj, _ := json.Marshal(members)
	cj, _ := json.Marshal(commits)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `
		INSERT INTO changesets (repository, sequence, reference, old_oid, new_oid, operation, actor, record_time, rewritten, kind, members, commits)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (repository, sequence) DO NOTHING`,
		cs.Repository, int64(cs.Sequence), cs.Reference, cs.OldOID, cs.NewOID, cs.Operation, cs.Actor, cs.RecordTime,
		cs.Rewritten, string(cs.Kind), mj, cj); err != nil {
		return err
	}
	for _, c := range cs.Commits {
		if _, err := tx.Exec(ctx, `
			INSERT INTO landed_commits (repository, commit_id, sequence, agent, operation, reference, subject, auto_revert_of, resolved_by, record_time)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (repository, commit_id) DO NOTHING`,
			cs.Repository, c.CommitID, int64(cs.Sequence), c.Agent, cs.Operation, cs.Reference, c.Subject, c.AutoRevertOf, c.ResolvedBy, cs.RecordTime); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *ActivityStore) ListChangesets(ctx context.Context, f activity.Filter) ([]activity.Changeset, error) {
	args := []any{}
	where := "TRUE"
	add := func(cond string, v any) {
		args = append(args, v)
		where += fmt.Sprintf(" AND "+cond, len(args))
	}
	if f.Repository != "" {
		add("repository = $%d", f.Repository)
	}
	if f.Agent != "" {
		args = append(args, f.Agent)
		n := len(args)
		where += fmt.Sprintf(` AND (actor = $%d OR EXISTS (SELECT 1 FROM jsonb_array_elements(commits) c WHERE c->>'agent' = $%d))`, n, n)
	}
	if f.BeforeTime != nil {
		args = append(args, *f.BeforeTime, f.BeforeKey)
		where += fmt.Sprintf(` AND (record_time, repository || '/' || lpad(sequence::text, 20, '0')) < ($%d, $%d)`, len(args)-1, len(args))
	}
	args = append(args, f.Limit)
	rows, err := s.DB.Query(ctx, `
		SELECT repository, sequence, reference, old_oid, new_oid, operation, actor, record_time, rewritten, kind, members, commits
		FROM changesets WHERE `+where+`
		ORDER BY record_time DESC, repository || '/' || lpad(sequence::text, 20, '0') DESC
		LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []activity.Changeset
	for rows.Next() {
		var cs activity.Changeset
		var seq int64
		var kind string
		var mj, cj []byte
		if err := rows.Scan(&cs.Repository, &seq, &cs.Reference, &cs.OldOID, &cs.NewOID, &cs.Operation, &cs.Actor, &cs.RecordTime, &cs.Rewritten, &kind, &mj, &cj); err != nil {
			return nil, err
		}
		cs.Sequence = uint64(seq)
		cs.Kind = activity.Kind(kind)
		var members []memberJSON
		_ = json.Unmarshal(mj, &members)
		for _, m := range members {
			cs.Members = append(cs.Members, activity.Member(m))
		}
		var commits []commitJSON
		_ = json.Unmarshal(cj, &commits)
		for _, c := range commits {
			cs.Commits = append(cs.Commits, activity.LandedCommit(c))
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

// PageKey is the secondary sort key of a changeset, for page tokens.
func PageKey(repo string, seq uint64) string { return fmt.Sprintf("%s/%020d", repo, seq) }

func (s *ActivityStore) Landings(ctx context.Context, repo string, ids []string) (map[string]source.Landing, error) {
	out := map[string]source.Landing{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.DB.Query(ctx, `SELECT commit_id, agent, sequence, operation, reference, record_time FROM landed_commits WHERE repository = $1 AND commit_id = ANY($2)`, repo, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var l source.Landing
		var seq int64
		if err := rows.Scan(&id, &l.Agent, &seq, &l.Operation, &l.Reference, &l.Time); err != nil {
			return nil, err
		}
		l.Sequence = uint64(seq)
		out[id] = l
	}
	return out, rows.Err()
}

const agentStatsSQL = `
WITH names AS (
    SELECT agent AS name FROM landed_commits
    UNION SELECT actor FROM changesets WHERE operation <> 'land_batch'
),
lc AS (
    SELECT agent,
           COUNT(*) AS commits,
           COUNT(DISTINCT (repository, sequence)) AS lands,
           COUNT(*) FILTER (WHERE resolved_by <> '') AS conflicts,
           COUNT(*) FILTER (WHERE auto_revert_of <> '') AS reverts,
           MIN(record_time) AS first_seen, MAX(record_time) AS last_seen,
           ARRAY_AGG(DISTINCT repository) AS repos
    FROM landed_commits GROUP BY agent
),
cs AS (
    SELECT actor, COUNT(*) AS changesets, MIN(record_time) AS first_seen, MAX(record_time) AS last_seen,
           ARRAY_AGG(DISTINCT repository) AS repos
    FROM changesets GROUP BY actor
),
rv AS (
    SELECT culprit.agent, COUNT(*) AS reverted
    FROM landed_commits r
    JOIN landed_commits culprit ON culprit.repository = r.repository AND culprit.commit_id LIKE r.auto_revert_of || '%'
    WHERE r.auto_revert_of <> ''
    GROUP BY culprit.agent
)
SELECT n.name,
       COALESCE(cs.changesets, 0), COALESCE(lc.lands, 0), COALESCE(lc.commits, 0),
       COALESCE(lc.conflicts, 0), COALESCE(lc.reverts, 0), COALESCE(rv.reverted, 0),
       (SELECT ARRAY(SELECT DISTINCT unnest(COALESCE(lc.repos, '{}') || COALESCE(cs.repos, '{}')) ORDER BY 1)),
       LEAST(COALESCE(lc.first_seen, cs.first_seen), COALESCE(cs.first_seen, lc.first_seen)),
       GREATEST(COALESCE(lc.last_seen, cs.last_seen), COALESCE(cs.last_seen, lc.last_seen))
FROM names n
LEFT JOIN lc ON lc.agent = n.name
LEFT JOIN cs ON cs.actor = n.name
LEFT JOIN rv ON rv.agent = n.name
`

func scanAgent(row pgx.Row) (*activity.Agent, error) {
	var a activity.Agent
	err := row.Scan(&a.Name, &a.Changesets, &a.Lands, &a.CommitsLanded, &a.ConflictsSolved, &a.AutoReverts, &a.RevertedCommits, &a.Repositories, &a.FirstSeen, &a.LastSeen)
	return &a, err
}

func (s *ActivityStore) ListAgents(ctx context.Context) ([]activity.Agent, error) {
	rows, err := s.DB.Query(ctx, agentStatsSQL+` ORDER BY 10 DESC NULLS LAST, 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []activity.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *ActivityStore) GetAgent(ctx context.Context, name string) (*activity.Agent, error) {
	a, err := scanAgent(s.DB.QueryRow(ctx, agentStatsSQL+` WHERE n.name = $1`, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	since := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -29)
	rows, err := s.DB.Query(ctx, `
		SELECT d::date, COALESCE(c.n, 0)
		FROM generate_series($2::timestamptz, $2::timestamptz + interval '29 days', interval '1 day') d
		LEFT JOIN (SELECT date_trunc('day', record_time AT TIME ZONE 'UTC') AS day, COUNT(*) AS n
		           FROM landed_commits WHERE agent = $1 GROUP BY 1) c ON c.day = d AT TIME ZONE 'UTC'
		ORDER BY 1`, name, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var dc activity.DailyCount
		if err := rows.Scan(&dc.Day, &dc.Count); err != nil {
			return nil, err
		}
		a.Daily = append(a.Daily, dc)
	}
	return a, rows.Err()
}

func (s *ActivityStore) RepositoryStats(ctx context.Context) (map[string]activity.RepoStats, error) {
	out := map[string]activity.RepoStats{}
	rows, err := s.DB.Query(ctx, `SELECT repository, MAX(record_time), COUNT(*) FROM changesets GROUP BY repository`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name string
		var st activity.RepoStats
		if err := rows.Scan(&name, &st.LastActivity, &st.Changesets); err != nil {
			rows.Close()
			return nil, err
		}
		out[name] = st
	}
	rows.Close()
	rows, err = s.DB.Query(ctx, `
		SELECT repository, agent, n FROM (
		  SELECT repository, agent, COUNT(DISTINCT sequence) AS n,
		         ROW_NUMBER() OVER (PARTITION BY repository ORDER BY COUNT(DISTINCT sequence) DESC, agent) AS rk
		  FROM landed_commits GROUP BY repository, agent) t
		WHERE rk <= 3 ORDER BY repository, n DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var ac source.AgentCount
		if err := rows.Scan(&name, &ac.Agent, &ac.Count); err != nil {
			return nil, err
		}
		st := out[name]
		st.TopAgents = append(st.TopAgents, ac)
		out[name] = st
	}
	return out, rows.Err()
}
