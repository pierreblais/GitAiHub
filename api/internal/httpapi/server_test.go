package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/identity"
	"github.com/pierreblais/gitaihub/api/internal/review"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// fakeCatalog is an in-memory source.RepositoryCatalog.
type fakeCatalog struct{ repos []source.Repository }

func (c *fakeCatalog) ListRepositories(context.Context) ([]source.Repository, error) {
	return append([]source.Repository(nil), c.repos...), nil
}

func (c *fakeCatalog) GetRepository(_ context.Context, name string) (*source.Repository, error) {
	for _, r := range c.repos {
		if r.Name == name {
			r := r
			return &r, nil
		}
	}
	return nil, source.ErrNotFound
}

// fakeSource implements the calls the tests make; the embedded interface
// panics on anything else, which flags an unexpected call.
type fakeSource struct{ source.Gateway }

func (fakeSource) DefaultBranch(context.Context, string) (string, error) { return "main", nil }

// fakeReviews is an in-memory review.Gateway.
type fakeReviews struct{ verdicts []string }

func (f *fakeReviews) ListReviews(context.Context, string, bool, int) ([]review.Review, error) {
	return nil, nil
}

func (f *fakeReviews) GetReview(context.Context, string, string) (*review.Review, error) {
	return &review.Review{ID: "1"}, nil
}

func (f *fakeReviews) AddComment(_ context.Context, token, _, _, body, _ string, _ int32) (*review.Comment, error) {
	return &review.Comment{Body: body}, nil
}

func (f *fakeReviews) SubmitVerdict(_ context.Context, token, _, id string, kind review.VerdictKind, _, _ string) (*review.Review, error) {
	f.verdicts = append(f.verdicts, token+":"+string(kind))
	return &review.Review{ID: id}, nil
}

// fakeStore is an activity.Store with no recorded activity.
type fakeStore struct{ activity.Store }

func (fakeStore) RepositoryStats(context.Context) (map[string]activity.RepoStats, error) {
	return map[string]activity.RepoStats{}, nil
}

func (fakeStore) Landings(context.Context, string, []string) (map[string]source.Landing, error) {
	return map[string]source.Landing{}, nil
}

type harness struct {
	srv     *httptest.Server
	broker  *activity.Broker
	reviews *fakeReviews
	token   string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	ids := &identity.Service{Store: &identity.MemStore{}, TTL: time.Hour}
	if _, _, err := ids.EnsureUser(ctx, "pierre", "correct horse", "Pierre"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ids.EnsureUser(ctx, "guest", "guest password", "Guest"); err != nil {
		t.Fatal(err)
	}
	catalog := &fakeCatalog{repos: []source.Repository{
		{Name: "alpha", RelativePath: "alpha.git"},
		{Name: "beta", RelativePath: "beta.git"},
		{Name: "gamma", RelativePath: "gamma.git"},
	}}
	broker := activity.NewBroker()
	revs := &fakeReviews{}
	act := &activity.Service{Store: fakeStore{}, Broker: broker}
	s := &Server{
		Source:    &source.Service{Catalog: catalog, Gateway: fakeSource{}, Landings: act},
		Reviews:   &review.Service{Catalog: catalog, Gateway: revs, Tokens: map[string]string{"pierre": "tok-pierre"}},
		Activity:  act,
		Identity:  ids,
		Ready:     func(context.Context) error { return nil },
		Heartbeat: time.Hour,
	}
	h := &harness{srv: httptest.NewServer(s.Handler()), broker: broker, reviews: revs}
	t.Cleanup(h.srv.Close)
	h.token = h.login(t, "pierre", "correct horse")
	return h
}

func (h *harness) login(t *testing.T, user, password string) string {
	t.Helper()
	resp, err := http.Post(h.srv.URL+"/v1/sessions", "application/json",
		strings.NewReader(`{"username":"`+user+`","password":"`+password+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Token == "" {
		t.Fatalf("login %s: status %d, err %v", user, resp.StatusCode, err)
	}
	return body.Token
}

func (h *harness) do(t *testing.T, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, h.srv.URL+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func errStatus(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	s, _ := e["status"].(string)
	return s
}

func TestRequiresAuthentication(t *testing.T) {
	h := newHarness(t)
	code, body := h.do(t, "GET", "/v1/repositories", "", "")
	if code != 401 || errStatus(body) != "UNAUTHENTICATED" {
		t.Fatalf("got %d %v, want 401 UNAUTHENTICATED", code, body)
	}
}

func TestWrongPassword(t *testing.T) {
	h := newHarness(t)
	code, body := h.do(t, "POST", "/v1/sessions", "", `{"username":"pierre","password":"nope"}`)
	if code != 401 || errStatus(body) != "UNAUTHENTICATED" {
		t.Fatalf("got %d %v", code, body)
	}
}

func TestMeReportsWriteAccess(t *testing.T) {
	h := newHarness(t)
	_, me := h.do(t, "GET", "/v1/users/me", h.token, "")
	if me["username"] != "pierre" || me["canReview"] != true {
		t.Fatalf("me = %v", me)
	}
}

func TestListRepositoriesPages(t *testing.T) {
	h := newHarness(t)
	code, page1 := h.do(t, "GET", "/v1/repositories?pageSize=2", h.token, "")
	if code != 200 {
		t.Fatalf("status %d: %v", code, page1)
	}
	if n := len(page1["repositories"].([]any)); n != 2 {
		t.Fatalf("page 1 has %d repositories, want 2", n)
	}
	next, _ := page1["nextPageToken"].(string)
	if next == "" {
		t.Fatal("no nextPageToken on a partial page")
	}
	_, page2 := h.do(t, "GET", "/v1/repositories?page_size=2&page_token="+next, h.token, "")
	if n := len(page2["repositories"].([]any)); n != 1 {
		t.Fatalf("page 2 has %d repositories, want 1", n)
	}
	if _, ok := page2["nextPageToken"]; ok {
		t.Fatal("last page has a nextPageToken")
	}
	code, body := h.do(t, "GET", "/v1/repositories?pageToken=garbage", h.token, "")
	if code != 400 || errStatus(body) != "INVALID_ARGUMENT" {
		t.Fatalf("bad token: got %d %v", code, body)
	}
}

func TestGetRepository(t *testing.T) {
	h := newHarness(t)
	code, repo := h.do(t, "GET", "/v1/repositories/beta", h.token, "")
	if code != 200 || repo["name"] != "repositories/beta" || repo["defaultBranch"] != "main" {
		t.Fatalf("got %d %v", code, repo)
	}
	code, body := h.do(t, "GET", "/v1/repositories/missing", h.token, "")
	if code != 404 || errStatus(body) != "NOT_FOUND" {
		t.Fatalf("missing: got %d %v", code, body)
	}
}

func TestSubmitVerdictUsesUsersGitaiToken(t *testing.T) {
	h := newHarness(t)
	code, body := h.do(t, "POST", "/v1/repositories/alpha/reviews/1:submitVerdict", h.token, `{"kind":"APPROVE","body":"ok","headCommitId":"abc123"}`)
	if code != 200 {
		t.Fatalf("got %d %v", code, body)
	}
	if len(h.reviews.verdicts) != 1 || !strings.HasPrefix(h.reviews.verdicts[0], "tok-pierre:") {
		t.Fatalf("verdicts = %v", h.reviews.verdicts)
	}

	guest := h.login(t, "guest", "guest password")
	code, body = h.do(t, "POST", "/v1/repositories/alpha/reviews/1:submitVerdict", guest, `{"kind":"APPROVE","headCommitId":"abc123"}`)
	if code != 403 || errStatus(body) != "PERMISSION_DENIED" {
		t.Fatalf("guest: got %d %v", code, body)
	}
}

func TestWatchChangesetsStreamsPublished(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", h.srv.URL+"/v1/changesets:watch?repository=repositories/alpha", nil)
	req.Header.Set("Authorization", "Bearer "+h.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	r := bufio.NewReader(resp.Body)
	// Wait for the "connected" comment so the subscription exists.
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, ": connected") {
			break
		}
	}
	// A changeset of another repository is filtered out; alpha's is sent.
	h.broker.Publish(activity.Changeset{Repository: "beta", Sequence: 1, Actor: "planner", Kind: activity.KindLand})
	h.broker.Publish(activity.Changeset{Repository: "alpha", Sequence: 7, Actor: "builder", Kind: activity.KindLand})
	var event, data string
	for data == "" {
		line, err := r.ReadString('\n')
		if err == io.EOF || err != nil {
			t.Fatalf("stream ended: %v", err)
		}
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event: "))
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		}
	}
	var cs map[string]any
	if err := json.Unmarshal([]byte(data), &cs); err != nil {
		t.Fatal(err)
	}
	if event != "changeset" || cs["name"] != "repositories/alpha/changesets/7" || cs["actor"] != "builder" {
		t.Fatalf("event %q data %v", event, cs)
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	h := newHarness(t)
	code, body := h.do(t, "GET", "/v2/nothing", h.token, "")
	if code != 404 || errStatus(body) != "NOT_FOUND" {
		t.Fatalf("got %d %v", code, body)
	}
}
