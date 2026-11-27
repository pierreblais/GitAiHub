package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/identity"
)

// These tests need a disposable Postgres database. They run when
// GITAIHUB_TEST_DATABASE_URL is set (CI sets it) and are skipped otherwise.
func testDB(t *testing.T) (*ActivityStore, *Catalog, *IdentityStore) {
	t.Helper()
	url := os.Getenv("GITAIHUB_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("GITAIHUB_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// Migrating twice must be a no-op.
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"landed_commits", "changesets", "sessions", "users", "repositories"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatal(err)
		}
	}
	store := &ActivityStore{DB: pool}
	return store, &Catalog{DB: pool, Stats: store}, &IdentityStore{DB: pool}
}

func TestChangesetLog(t *testing.T) {
	store, catalog, _ := testDB(t)
	ctx := context.Background()
	names, err := catalog.Upsert(ctx, []string{"orbit.git", "Team/Nebula-Web.git"})
	if err != nil {
		t.Fatal(err)
	}
	if names["orbit"] != "orbit.git" || names["team-nebula-web"] != "Team/Nebula-Web.git" {
		t.Fatalf("names = %v", names)
	}

	base := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	for seq := uint64(1); seq <= 5; seq++ {
		cs := &activity.Changeset{
			Repository: "orbit", Sequence: seq, Reference: "refs/heads/main",
			OldOID: fmt.Sprintf("%040d", seq-1), NewOID: fmt.Sprintf("%040d", seq),
			Operation: "land", Actor: "builder", RecordTime: base.Add(time.Duration(seq) * time.Minute),
			Kind:    activity.KindLand,
			Commits: []activity.LandedCommit{{CommitID: fmt.Sprintf("c%d", seq), Agent: "builder", Subject: "change"}},
		}
		if err := store.SaveChangeset(ctx, cs); err != nil {
			t.Fatal(err)
		}
	}
	// Saving the same sequence again is idempotent (watchers may replay).
	if err := store.SaveChangeset(ctx, &activity.Changeset{Repository: "orbit", Sequence: 5, RecordTime: base, Actor: "x", Kind: activity.KindLand}); err != nil {
		t.Fatal(err)
	}
	if cur, err := store.Cursor(ctx, "orbit"); err != nil || cur != 5 {
		t.Fatalf("cursor = %d, %v; want 5", cur, err)
	}

	page, err := store.ListChangesets(ctx, activity.Filter{Repository: "orbit", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].Sequence != 5 || page[1].Sequence != 4 {
		t.Fatalf("first page = %+v", page)
	}
	last := page[1]
	page, err = store.ListChangesets(ctx, activity.Filter{
		Repository: "orbit", Limit: 10,
		BeforeTime: &last.RecordTime, BeforeKey: PageKey(last.Repository, last.Sequence),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 || page[0].Sequence != 3 {
		t.Fatalf("second page = %+v", page)
	}

	landings, err := store.Landings(ctx, "orbit", []string{"c2", "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := landings["c2"]; !ok || len(landings) != 1 {
		t.Fatalf("landings = %+v", landings)
	}

	agent, err := store.GetAgent(ctx, "builder")
	if err != nil {
		t.Fatal(err)
	}
	if agent.Name != "builder" {
		t.Fatalf("agent = %+v", agent)
	}
	if _, err := store.GetAgent(ctx, "ghost"); err == nil {
		t.Fatal("unknown agent: want an error")
	}

	repos, err := catalog.ListRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("repositories = %+v", repos)
	}
}

func TestIdentityStore(t *testing.T) {
	_, _, ids := testDB(t)
	ctx := context.Background()
	svc := &identity.Service{Store: ids, TTL: time.Hour}
	if _, created, err := svc.EnsureUser(ctx, "pierre", "long enough", "Pierre"); err != nil || !created {
		t.Fatalf("EnsureUser: created=%v err=%v", created, err)
	}
	if _, created, err := svc.EnsureUser(ctx, "pierre", "long enough", "Pierre"); err != nil || created {
		t.Fatalf("EnsureUser again: created=%v err=%v", created, err)
	}
	token, _, _, err := svc.Login(ctx, "pierre", "long enough")
	if err != nil {
		t.Fatal(err)
	}
	u, err := svc.Authenticate(ctx, token)
	if err != nil || u.Username != "pierre" {
		t.Fatalf("Authenticate = %v, %v", u, err)
	}
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, token); err == nil {
		t.Fatal("token still valid after logout")
	}
}
