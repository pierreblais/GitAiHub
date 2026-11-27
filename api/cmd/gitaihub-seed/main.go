// Command gitaihub-seed fills a gitai server with demo repositories and
// realistic agent activity: lands, rebases, an AI-resolved conflict, an
// auto-revert and reviews with verdicts and line comments. It talks to gitai
// only through gitai's Go SDK, the way real agents do.
//
// Environment:
//
//	GITAI_ADDRESS        e.g. tcp://gitai:8075
//	GITAI_TOKEN          operator (shared) token, to create repositories
//	GITAI_STORAGE        storage name (default "default")
//	SEED_AGENT_TOKENS    agent=token pairs, comma separated; needs planner,
//	                     builder, reviewer and docs-bot, optionally pierre
//	SEED_LIVE_INTERVAL   if set (e.g. 40s), keep landing small changes forever
//	                     so the live feed has something to show
//
// Seeding is idempotent: a repository that exists already is left alone.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/feder-cr/git-ai/sdk"
)

type seeder struct {
	address string
	storage string
	admin   *sdk.Client
	agents  map[string]*sdk.Client
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	address, token := os.Getenv("GITAI_ADDRESS"), os.Getenv("GITAI_TOKEN")
	storage := os.Getenv("GITAI_STORAGE")
	if storage == "" {
		storage = sdk.DefaultStorage
	}
	if address == "" || token == "" {
		return errors.New("GITAI_ADDRESS and GITAI_TOKEN are required")
	}
	s := &seeder{address: address, storage: storage, agents: map[string]*sdk.Client{}}
	var err error
	if s.admin, err = sdk.Dial(ctx, address, sdk.WithToken(token), sdk.WithStorage(storage)); err != nil {
		return err
	}
	for _, pair := range strings.Split(os.Getenv("SEED_AGENT_TOKENS"), ",") {
		name, tok, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok {
			continue
		}
		c, err := sdk.Dial(ctx, address, sdk.WithAgentToken(tok), sdk.WithStorage(storage))
		if err != nil {
			return err
		}
		s.agents[name] = c
	}
	for _, a := range []string{"planner", "builder", "reviewer", "docs-bot"} {
		if s.agents[a] == nil {
			return fmt.Errorf("SEED_AGENT_TOKENS must include %s", a)
		}
	}
	if err := s.waitReady(ctx); err != nil {
		return err
	}
	for _, r := range []struct {
		path string
		fn   func(context.Context, string) error
	}{
		{"orbit.git", s.seedOrbit},
		{"nebula-web.git", s.seedNebula},
		{"agent-playbooks.git", s.seedPlaybooks},
	} {
		created, err := s.create(ctx, r.path)
		if err != nil {
			return err
		}
		if !created {
			slog.Info("repository exists, skipping", "repository", r.path)
			continue
		}
		slog.Info("seeding", "repository", r.path)
		if err := r.fn(ctx, r.path); err != nil {
			return fmt.Errorf("seeding %s: %w", r.path, err)
		}
	}
	slog.Info("seed complete")
	if iv := os.Getenv("SEED_LIVE_INTERVAL"); iv != "" {
		d, err := time.ParseDuration(iv)
		if err != nil {
			return fmt.Errorf("SEED_LIVE_INTERVAL: %w", err)
		}
		return s.live(ctx, d)
	}
	return nil
}

func (s *seeder) waitReady(ctx context.Context) error {
	repo := s.admin.Repo("readiness-probe.git")
	for i := 0; i < 60; i++ {
		_, err := repo.HasReferences(ctx)
		if err == nil || sdk.IsNotFound(err) {
			return nil
		}
		slog.Info("waiting for gitai", "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return errors.New("gitai did not become ready")
}

func (s *seeder) create(ctx context.Context, path string) (bool, error) {
	err := s.admin.Repo(path).Create(ctx, sdk.CreateOptions{DefaultBranch: "main"})
	if sdk.IsAlreadyExists(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *seeder) as(agent, path string) *sdk.Repo { return s.agents[agent].Repo(path) }

func user(agent string) sdk.User {
	return sdk.User{Name: agent, Email: agent + "@agents.gitaihub.local", Username: agent}
}

func author(agent string) sdk.Signature {
	return sdk.Signature{Name: agent, Email: agent + "@agents.gitaihub.local"}
}

// files turns path/content pairs into Put actions that create or overwrite.
func files(kv ...string) []sdk.Action {
	var out []sdk.Action
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, sdk.Put{Path: kv[i], Content: []byte(kv[i+1]), Overwrite: true})
	}
	return out
}

// work commits actions on a fresh work branch started at start (a revision)
// and returns the commit. Agents work on their own branches, then land.
func (s *seeder) work(ctx context.Context, agent, path, branch, start, msg string, actions []sdk.Action) (string, error) {
	// Put with Overwrite fails on a new file, and without it on an existing one.
	for i, a := range actions {
		if p, ok := a.(sdk.Put); ok {
			_, err := s.as(agent, path).ReadFile(ctx, start, p.Path)
			p.Overwrite = err == nil
			actions[i] = p
		}
	}
	res, err := s.as(agent, path).CommitFiles(ctx, sdk.CommitOptions{
		Branch: branch, StartRev: start, Message: msg, Author: author(agent), User: user(agent), Actions: actions,
	})
	if err != nil {
		return "", fmt.Errorf("%s commit %q: %w", agent, msg, err)
	}
	return res.CommitID, nil
}

// putFirst writes the first commit of an empty repository on main.
func (s *seeder) putFirst(ctx context.Context, agent, path, msg string, actions []sdk.Action) error {
	put := make([]sdk.Action, 0, len(actions))
	for _, a := range actions {
		p := a.(sdk.Put)
		p.Overwrite = false
		put = append(put, p)
	}
	_, err := s.as(agent, path).CommitFiles(ctx, sdk.CommitOptions{Branch: "main", Message: msg, Author: author(agent), User: user(agent), Actions: put})
	return err
}

func (s *seeder) land(ctx context.Context, agent, path, commit string) (*sdk.LandResult, error) {
	res, err := s.as(agent, path).Land(ctx, sdk.LandOptions{Branch: "main", Commit: commit, User: user(agent)})
	if err != nil {
		return nil, fmt.Errorf("%s land %s: %w", agent, short(commit), err)
	}
	return res, nil
}

// change is work + land in one step.
func (s *seeder) change(ctx context.Context, agent, path, slug, msg string, actions []sdk.Action) (string, error) {
	c, err := s.work(ctx, agent, path, "agents/"+agent+"/"+slug, "main", msg, actions)
	if err != nil {
		return "", err
	}
	res, err := s.land(ctx, agent, path, c)
	if err != nil {
		return "", err
	}
	return res.LandedCommitID, nil
}

// landResolving lands commit and, on a conflict, resolves every conflicting
// file with resolve(path, sides) and lands the resolution as the agent.
func (s *seeder) landResolving(ctx context.Context, agent, path, commit string, resolve func(string, sdk.ConflictSides) []byte) (*sdk.LandResult, error) {
	repo := s.as(agent, path)
	res, err := repo.Land(ctx, sdk.LandOptions{Branch: "main", Commit: commit, User: user(agent), ResolveConflicts: true})
	if err == nil {
		return res, nil
	}
	conflict, ok := sdk.LandConflict(err)
	if !ok {
		return nil, err
	}
	sides, err := repo.ConflictSides(ctx, conflict)
	if err != nil {
		return nil, err
	}
	resolution := sdk.ResolutionFor(conflict)
	for _, f := range conflict.GetFiles() {
		p := string(f.GetPath())
		resolution.Files = append(resolution.Files, sdk.ResolvedFile{Path: p, Mode: "100644", Content: resolve(p, sides[p])})
	}
	return repo.Land(ctx, sdk.LandOptions{Branch: "main", Commit: commit, User: user(agent), Resolution: resolution})
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func (s *seeder) live(ctx context.Context, every time.Duration) error {
	slog.Info("live mode: landing a small change periodically", "interval", every)
	agents := []string{"planner", "builder", "docs-bot"}
	repos := []string{"orbit.git", "nebula-web.git", "agent-playbooks.git"}
	n := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(every):
		}
		n++
		agent := agents[rand.IntN(len(agents))]
		repo := repos[rand.IntN(len(repos))]
		stamp := time.Now().UTC().Format(time.RFC3339)
		file := fmt.Sprintf("notes/%s.md", agent)
		old, _ := s.as(agent, repo).ReadFile(ctx, "main", file)
		content := string(old) + fmt.Sprintf("- %s: heartbeat %d from %s\n", stamp, n, agent)
		if _, err := s.change(ctx, agent, repo, fmt.Sprintf("heartbeat-%d", time.Now().Unix()), fmt.Sprintf("Log heartbeat %d", n), files(file, content)); err != nil {
			slog.Warn("live change failed", "error", err)
		}
	}
}
