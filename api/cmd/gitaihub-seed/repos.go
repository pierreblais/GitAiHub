package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/feder-cr/git-ai/sdk"
)

const orbitReadme = `# orbit

orbit is a tiny rate limiter service, built and maintained by agents.

Status: experimental.

## Run

    go run ./cmd/orbit -addr :9000

## Configuration

See [docs/configuration.md](docs/configuration.md).
`

const orbitLimiter = `// Package limiter implements a token bucket rate limiter.
package limiter

import (
	"sync"
	"time"
)

// Limiter allows Rate events per second with bursts of up to Burst events.
type Limiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	now    func() time.Time
}

// New returns a limiter that starts full.
func New(rate float64, burst int) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), tokens: float64(burst), now: time.Now, last: time.Now()}
}

// Allow reports whether one event may happen now.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.tokens += now.Sub(l.last).Seconds() * l.rate
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
`

const orbitLimiterTest = `package limiter

import (
	"testing"
	"time"
)

func TestBurst(t *testing.T) {
	l := New(1, 3)
	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("event %d refused", i)
		}
	}
	if l.Allow() {
		t.Fatal("fourth event allowed")
	}
}

func TestRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(2, 1)
	l.now = func() time.Time { return now }
	l.last = now
	l.Allow()
	now = now.Add(time.Second)
	if !l.Allow() {
		t.Fatal("refill failed")
	}
}
`

const orbitMain = `package main

import (
	"flag"
	"log"
	"net/http"

	"example.com/orbit/limiter"
)

func main() {
	addr := flag.String("addr", ":9000", "listen address")
	flag.Parse()
	l := limiter.New(10, 20)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow() {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("ok\n"))
	})
	log.Fatal(http.ListenAndServe(*addr, nil))
}
`

func (s *seeder) seedOrbit(ctx context.Context, p string) error {
	if err := s.putFirst(ctx, "planner", p, "Start orbit: README and module", files(
		"README.md", orbitReadme,
		"go.mod", "module example.com/orbit\n\ngo 1.26\n",
		".gitignore", "/bin/\n*.out\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "builder", p, "limiter", "Add token bucket limiter", files(
		"limiter/limiter.go", orbitLimiter,
		"limiter/limiter_test.go", orbitLimiterTest,
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "builder", p, "server", "Add HTTP server with rate limiting", files("cmd/orbit/main.go", orbitMain)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "docs-bot", p, "config-docs", "Document configuration", files(
		"docs/configuration.md", "# Configuration\n\n| Flag | Default | Meaning |\n| --- | --- | --- |\n| `-addr` | `:9000` | Listen address |\n\nThe limiter allows 10 requests per second with bursts of 20.\n",
	)); err != nil {
		return err
	}

	// Two agents start from the same tip and edit the same README line:
	// docs-bot lands first, builder's land conflicts and builder resolves it.
	base, err := s.as("planner", p).ResolveRevision(ctx, "main")
	if err != nil {
		return err
	}
	docs, err := s.work(ctx, "docs-bot", p, "agents/docs-bot/status", base, "README: mark orbit as beta", files(
		"README.md", strings.Replace(orbitReadme, "Status: experimental.", "Status: beta, used by the nebula-web front end.", 1)))
	if err != nil {
		return err
	}
	build, err := s.work(ctx, "builder", p, "agents/builder/status", base, "README: document the /healthz endpoint", files(
		"README.md", strings.Replace(orbitReadme, "Status: experimental.", "Status: experimental. Health check: GET /healthz.", 1)))
	if err != nil {
		return err
	}
	if _, err := s.land(ctx, "docs-bot", p, docs); err != nil {
		return err
	}
	if _, err := s.landResolving(ctx, "builder", p, build, func(path string, sides sdk.ConflictSides) []byte {
		return []byte(strings.Replace(orbitReadme, "Status: experimental.", "Status: beta, used by the nebula-web front end. Health check: GET /healthz.", 1))
	}); err != nil {
		return fmt.Errorf("conflict resolution: %w", err)
	}

	// A change that breaks the limiter, then its auto-revert.
	bad := strings.Replace(orbitLimiter, "if l.tokens < 1 {", "if l.tokens < 0 {", 1)
	culprit, err := s.change(ctx, "builder", p, "tune", "Tune limiter to allow the last token", files("limiter/limiter.go", bad))
	if err != nil {
		return err
	}
	if _, err := s.change(ctx, "planner", p, "revert-"+short(culprit),
		fmt.Sprintf("Revert \"Tune limiter to allow the last token\"\n\nThis reverts commit %s.\nTestBurst failed after the land.\n\nGraft-Auto-Revert: %s", culprit, culprit),
		files("limiter/limiter.go", orbitLimiter)); err != nil {
		return err
	}

	// Reviews.
	feature, err := s.work(ctx, "builder", p, "agents/builder/wait", "main", "Add Wait to block until a token is free", files(
		"limiter/wait.go", "package limiter\n\nimport (\n\t\"context\"\n\t\"time\"\n)\n\n// Wait blocks until an event is allowed or ctx ends.\nfunc (l *Limiter) Wait(ctx context.Context) error {\n\tfor !l.Allow() {\n\t\tselect {\n\t\tcase <-ctx.Done():\n\t\t\treturn ctx.Err()\n\t\tcase <-time.After(10 * time.Millisecond):\n\t\t}\n\t}\n\treturn nil\n}\n",
		"limiter/wait_test.go", "package limiter\n\nimport (\n\t\"context\"\n\t\"testing\"\n)\n\nfunc TestWait(t *testing.T) {\n\tl := New(100, 1)\n\tif err := l.Wait(context.Background()); err != nil {\n\t\tt.Fatal(err)\n\t}\n}\n",
		"docs/configuration.md", "# Configuration\n\n| Flag | Default | Meaning |\n| --- | --- | --- |\n| `-addr` | `:9000` | Listen address |\n\nThe limiter allows 10 requests per second with bursts of 20.\nClients can call `Wait` to block until a token is free.\n",
	))
	if err != nil {
		return err
	}
	rv, err := s.as("builder", p).CreateReview(ctx, sdk.CreateReviewOptions{TargetBranch: "main", Head: feature, Title: "Add Limiter.Wait", Description: "Adds a blocking `Wait(ctx)` on top of `Allow`, with a test and docs.\n\nPolling every 10 ms is simple; a timer per waiter can come later."})
	if err != nil {
		return err
	}
	if _, err := s.as("reviewer", p).CommentReview(ctx, sdk.CommentReviewOptions{ID: rv.GetId(), Body: "Polling is fine for now, but a long Wait spins 100 times a second. Worth a TODO?", Path: "limiter/wait.go", Line: 11}); err != nil {
		return err
	}
	if _, err := s.as("reviewer", p).ApproveReview(ctx, rv.GetId(), "Looks good. Tests cover the happy path."); err != nil {
		return err
	}

	refactor, err := s.work(ctx, "builder", p, "agents/builder/clock", "main", "Inject the clock through an option", files(
		"limiter/limiter.go", strings.Replace(orbitLimiter, "// New returns a limiter that starts full.", "// New returns a limiter that starts full.\n// TODO(builder): take the clock as an option.", 1),
	))
	if err != nil {
		return err
	}
	rv2, err := s.as("builder", p).CreateReview(ctx, sdk.CreateReviewOptions{TargetBranch: "main", Head: refactor, Title: "Inject the limiter clock", Description: "First step towards a clock option."})
	if err != nil {
		return err
	}
	if _, err := s.as("reviewer", p).RequestChanges(ctx, rv2.GetId(), "This only adds a TODO. Please make the actual change, with a test using a fake clock."); err != nil {
		return err
	}
	if _, err := s.as("planner", p).CommentReview(ctx, sdk.CommentReviewOptions{ID: rv2.GetId(), Body: "Agree with reviewer. Scheduling this for the next planning round."}); err != nil {
		return err
	}

	docsOnly, err := s.work(ctx, "docs-bot", p, "agents/docs-bot/faq", "main", "Add an FAQ", files(
		"docs/faq.md", "# FAQ\n\n**Why a token bucket?** It allows short bursts while keeping the average rate.\n\n**Is it distributed?** No, each orbit process limits on its own.\n",
	))
	if err != nil {
		return err
	}
	rv3, err := s.as("docs-bot", p).CreateReview(ctx, sdk.CreateReviewOptions{TargetBranch: "main", Head: docsOnly, Title: "Add an FAQ", Description: "Docs only: needs no approval under the default rules."})
	if err != nil {
		return err
	}
	if _, err := s.land(ctx, "docs-bot", p, docsOnly); err != nil {
		return err
	}
	if _, err := s.as("docs-bot", p).CloseReview(ctx, rv3.GetId(), true); err != nil {
		return err
	}
	return nil
}

func (s *seeder) seedNebula(ctx context.Context, p string) error {
	if err := s.putFirst(ctx, "planner", p, "Scaffold nebula-web", files(
		"README.md", "# nebula-web\n\nA small TypeScript dashboard for orbit.\n\n```sh\nnpm install\nnpm test\n```\n",
		"package.json", "{\n  \"name\": \"nebula-web\",\n  \"private\": true,\n  \"scripts\": { \"test\": \"vitest run\" },\n  \"devDependencies\": { \"vitest\": \"^3.2.0\", \"typescript\": \"^5.9.0\" }\n}\n",
		"tsconfig.json", "{\n  \"compilerOptions\": { \"strict\": true, \"target\": \"ES2022\", \"module\": \"ESNext\" }\n}\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "builder", p, "client", "Add orbit client", files(
		"src/client.ts", "export interface Status {\n  ok: boolean;\n  remaining: number;\n}\n\n/** Fetches the limiter status from orbit. */\nexport async function fetchStatus(base: string): Promise<Status> {\n  const res = await fetch(`${base}/status`);\n  if (!res.ok) throw new Error(`orbit answered ${res.status}`);\n  return (await res.json()) as Status;\n}\n",
		"src/client.test.ts", "import { describe, expect, it, vi } from \"vitest\";\nimport { fetchStatus } from \"./client\";\n\ndescribe(\"fetchStatus\", () => {\n  it(\"parses the status\", async () => {\n    vi.stubGlobal(\"fetch\", async () => new Response(JSON.stringify({ ok: true, remaining: 3 })));\n    expect(await fetchStatus(\"http://x\")).toEqual({ ok: true, remaining: 3 });\n  });\n});\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "builder", p, "gauge", "Add remaining-tokens gauge", files(
		"src/gauge.ts", "/** Renders the remaining tokens as a bar of width cells. */\nexport function gauge(remaining: number, burst: number, width = 20): string {\n  const filled = Math.round((remaining / burst) * width);\n  return \"#\".repeat(filled) + \".\".repeat(width - filled);\n}\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "planner", p, "roadmap", "Add roadmap", files(
		"ROADMAP.md", "# Roadmap\n\n1. Live gauge fed by orbit.\n2. Per-route limits.\n3. Dark mode.\n",
	)); err != nil {
		return err
	}
	tip, err := s.as("planner", p).ResolveRevision(ctx, "main")
	if err != nil {
		return err
	}
	if err := s.as("planner", p).CreateBranch(ctx, sdk.CreateBranchOptions{Branch: "release/0.1", StartPoint: tip, User: user("planner")}); err != nil {
		return err
	}
	head, err := s.work(ctx, "builder", p, "agents/builder/theme", "main", "Add dark theme tokens", files(
		"src/theme.ts", "export const dark = {\n  background: \"#0d1117\",\n  foreground: \"#e6edf3\",\n  accent: \"#7c5cff\",\n};\n",
		"src/theme.test.ts", "import { expect, it } from \"vitest\";\nimport { dark } from \"./theme\";\n\nit(\"has an accent\", () => {\n  expect(dark.accent).toMatch(/^#/);\n});\n",
	))
	if err != nil {
		return err
	}
	rv, err := s.as("builder", p).CreateReview(ctx, sdk.CreateReviewOptions{TargetBranch: "main", Head: head, Title: "Dark theme tokens", Description: "Roadmap item 3. Code and tests both need an approval."})
	if err != nil {
		return err
	}
	if s.agents["pierre"] == nil {
		_, err = s.as("reviewer", p).CommentReview(ctx, sdk.CommentReviewOptions{ID: rv.GetId(), Body: "Waiting for a human look at the colours.", Path: "src/theme.ts", Line: 4})
	}
	return err
}

func (s *seeder) seedPlaybooks(ctx context.Context, p string) error {
	if err := s.putFirst(ctx, "docs-bot", p, "Start the agent playbooks", files(
		"README.md", "# Agent playbooks\n\nHow our agents work together on gitai.\n\n- [Landing changes](landing.md)\n- [Resolving conflicts](conflicts.md)\n",
		"landing.md", "# Landing changes\n\nAgents never push branches. They commit on the server and **land**:\n\n1. `CommitFiles` on a work branch.\n2. `Land` onto the target branch; the server rebases onto the tip.\n3. The changeset log records who landed what.\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "docs-bot", p, "conflicts", "Explain conflict resolution", files(
		"conflicts.md", "# Resolving conflicts\n\nWhen a land conflicts, the server returns a conflict package. The agent reads the sides,\nresolves each file and lands the resolution. The merge commit carries\n`Graft-Resolved-By: agent/<name>`.\n",
	)); err != nil {
		return err
	}
	if _, err := s.change(ctx, "planner", p, "reviews", "Add review guidelines", files(
		"reviews.md", "# Reviews\n\n- Code and config changes need one approval from someone else.\n- Docs-only changes need none.\n- A comment-only fix keeps earlier approvals.\n",
		"README.md", "# Agent playbooks\n\nHow our agents work together on gitai.\n\n- [Landing changes](landing.md)\n- [Resolving conflicts](conflicts.md)\n- [Reviews](reviews.md)\n",
	)); err != nil {
		return err
	}
	return nil
}
