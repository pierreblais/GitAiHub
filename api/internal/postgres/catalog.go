package postgres

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Catalog implements source.RepositoryCatalog on the repositories table,
// joined with the activity summary.
type Catalog struct {
	DB    *pgxpool.Pool
	Stats interface {
		RepositoryStats(ctx context.Context) (map[string]activity.RepoStats, error)
	}
}

// NameFor returns the resource ID of a repository from its relative path:
// "agents/demo.git" becomes "agents-demo".
func NameFor(relativePath string) string {
	n := strings.TrimSuffix(strings.Trim(path.Clean(relativePath), "/"), ".git")
	n = strings.ReplaceAll(n, "/", "-")
	return strings.ToLower(n)
}

// Upsert registers repositories (relative paths) and returns name -> path.
func (c *Catalog) Upsert(ctx context.Context, relativePaths []string) (map[string]string, error) {
	out := map[string]string{}
	for _, rel := range relativePaths {
		name := NameFor(rel)
		if _, err := c.DB.Exec(ctx, `INSERT INTO repositories (name, relative_path) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING`, name, rel); err != nil {
			return nil, err
		}
		out[name] = rel
	}
	return out, nil
}

// All returns every registered repository as name -> relative path.
func (c *Catalog) All(ctx context.Context) (map[string]string, error) {
	rows, err := c.DB.Query(ctx, `SELECT name, relative_path FROM repositories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var n, p string
		if err := rows.Scan(&n, &p); err != nil {
			return nil, err
		}
		out[n] = p
	}
	return out, rows.Err()
}

func (c *Catalog) withStats(ctx context.Context, repos []source.Repository) []source.Repository {
	if c.Stats == nil {
		return repos
	}
	stats, err := c.Stats.RepositoryStats(ctx)
	if err != nil {
		return repos
	}
	for i := range repos {
		st := stats[repos[i].Name]
		repos[i].LastActivityTime = st.LastActivity
		repos[i].LandingCount = st.Changesets
		repos[i].TopAgents = st.TopAgents
	}
	return repos
}

func (c *Catalog) ListRepositories(ctx context.Context) ([]source.Repository, error) {
	rows, err := c.DB.Query(ctx, `SELECT name, relative_path, default_branch FROM repositories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []source.Repository
	for rows.Next() {
		var r source.Repository
		if err := rows.Scan(&r.Name, &r.RelativePath, &r.DefaultBranch); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return c.withStats(ctx, out), nil
}

func (c *Catalog) GetRepository(ctx context.Context, name string) (*source.Repository, error) {
	var r source.Repository
	err := c.DB.QueryRow(ctx, `SELECT name, relative_path, default_branch FROM repositories WHERE name = $1`, name).
		Scan(&r.Name, &r.RelativePath, &r.DefaultBranch)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: repository %q", source.ErrNotFound, name)
	}
	if err != nil {
		return nil, err
	}
	return &c.withStats(ctx, []source.Repository{r})[0], nil
}
