// Package httpapi is GitaiHub's resource-oriented HTTP/JSON API, following
// the Google API design guide: resources named like "repositories/{repo}",
// standard List and Get methods with page_size/page_token, a few custom
// methods ("reviews/{review}:submitVerdict", "changesets:watch") and the
// Google error model. The contract is written down in proto/gitaihub/v1.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/identity"
	"github.com/pierreblais/gitaihub/api/internal/observability"
	"github.com/pierreblais/gitaihub/api/internal/review"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// SessionCookie is the name of the session cookie.
const SessionCookie = "gitaihub_session"

// Server wires the bounded contexts' application services to HTTP.
type Server struct {
	Source       *source.Service
	Reviews      *review.Service
	Activity     *activity.Service
	Identity     *identity.Service
	Metrics      *observability.Metrics
	Ready        func(ctx context.Context) error
	CookieSecure bool
	// Heartbeat is the interval of SSE keep-alive comments.
	Heartbeat time.Duration
}

type ctxKey int

const userKey ctxKey = 0

func currentUser(ctx context.Context) *identity.User {
	u, _ := ctx.Value(userKey).(*identity.User)
	return u
}

// Handler returns the HTTP handler of the API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", s.readyz)
	if s.Metrics != nil {
		mux.Handle("GET /metrics", promhttp.HandlerFor(s.Metrics.Registry, promhttp.HandlerOpts{}))
	}

	mux.HandleFunc("POST /v1/sessions", s.login)
	auth := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, s.requireUser(h)) }
	auth("DELETE /v1/sessions/current", s.logout)
	auth("GET /v1/users/me", s.me)

	auth("GET /v1/repositories", s.listRepositories)
	auth("GET /v1/repositories/{repo}", s.getRepository)
	auth("GET /v1/repositories/{repo}/branches", s.listRefs(source.RefBranch))
	auth("GET /v1/repositories/{repo}/tags", s.listRefs(source.RefTag))
	auth("GET /v1/repositories/{repo}/tree", s.getTree)
	auth("GET /v1/repositories/{repo}/blob", s.getBlob)
	auth("GET /v1/repositories/{repo}/blame", s.getBlame)
	auth("GET /v1/repositories/{repo}/commits", s.listCommits)
	auth("GET /v1/repositories/{repo}/commits/{commit}", s.getCommit)
	auth("GET /v1/repositories/{repo}/search", s.searchCode)
	auth("GET /v1/repositories/{repo}/reviews", s.listReviews)
	auth("GET /v1/repositories/{repo}/reviews/{review}", s.getReview)
	auth("POST /v1/repositories/{repo}/reviews/{review}", s.reviewCustomMethod)
	auth("POST /v1/repositories/{repo}/reviews/{review}/comments", s.addComment)
	auth("GET /v1/repositories/{repo}/changesets", s.listChangesets)
	auth("GET /v1/changesets", s.listChangesets)
	auth("GET /v1/changesets:watch", s.watchChangesets)
	auth("GET /v1/agents", s.listAgents)
	auth("GET /v1/agents/{agent}", s.getAgent)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, fmt.Errorf("%w: no route for %s %s", source.ErrNotFound, r.Method, r.URL.Path))
	})
	return s.observe(mux)
}

// statusRecorder captures the status code for metrics and logs.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) { r.code = code; r.ResponseWriter.WriteHeader(code) }
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: 200}
		if s.Metrics != nil {
			s.Metrics.InFlight.Inc()
			defer s.Metrics.InFlight.Dec()
		}
		next.ServeHTTP(rec, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		if i := strings.IndexByte(route, ' '); i >= 0 {
			route = route[i+1:]
		}
		d := time.Since(start)
		if s.Metrics != nil {
			s.Metrics.Requests.WithLabelValues(route, r.Method, strconv.Itoa(rec.code)).Inc()
			if !strings.HasSuffix(route, ":watch") {
				s.Metrics.Latency.WithLabelValues(route, r.Method).Observe(d.Seconds())
			}
		}
		if route != "/healthz" && route != "/readyz" && route != "/metrics" {
			slog.Info("http", "method", r.Method, "path", r.URL.Path, "route", route, "status", rec.code, "duration_ms", d.Milliseconds())
		}
	})
}

func sessionToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if c, err := r.Cookie(SessionCookie); err == nil {
		return c.Value
	}
	return ""
}

func (s *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.Identity.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			writeError(w, r, identity.ErrUnauthenticated)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if s.Ready != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := s.Ready(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "error": err.Error()})
			return
		}
	}
	writeJSON(w, 200, map[string]string{"status": "ready"})
}

func decodeBody(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: invalid JSON body: %v", source.ErrInvalidArgument, err)
	}
	return nil
}

// --- Identity ---

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	token, u, sess, err := s.Identity.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: s.CookieSecure,
		SameSite: http.SameSiteLaxMode, Expires: sess.ExpireTime,
	})
	writeJSON(w, http.StatusOK, map[string]any{"user": toUser(u, s.Reviews.CanWrite(u.Username)), "token": token, "expireTime": ts(sess.ExpireTime)})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	_ = s.Identity.Logout(r.Context(), sessionToken(r))
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.CookieSecure})
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r.Context())
	writeJSON(w, http.StatusOK, toUser(u, s.Reviews.CanWrite(u.Username)))
}

// --- Source ---

func (s *Server) listRepositories(w http.ResponseWriter, r *http.Request) {
	size, err := pageSize(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	tok, err := decodeToken(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	repos, err := s.Source.ListRepositories(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	// Most recently active first, like a GitHub dashboard.
	sortRepositories(repos)
	resp := map[string]any{"repositories": []repositoryJSON{}, "totalSize": len(repos)}
	list := []repositoryJSON{}
	end := min(tok.Offset+size, len(repos))
	for i := tok.Offset; i < end; i++ {
		list = append(list, toRepository(repos[i]))
	}
	resp["repositories"] = list
	if end < len(repos) {
		resp["nextPageToken"] = encodeToken(pageToken{Offset: end})
	}
	writeJSON(w, 200, resp)
}

func sortRepositories(repos []source.Repository) {
	for i := 1; i < len(repos); i++ {
		for j := i; j > 0 && repoLess(repos[j], repos[j-1]); j-- {
			repos[j], repos[j-1] = repos[j-1], repos[j]
		}
	}
}

func repoLess(a, b source.Repository) bool {
	if !a.LastActivityTime.Equal(b.LastActivityTime) {
		return a.LastActivityTime.After(b.LastActivityTime)
	}
	return a.Name < b.Name
}

func (s *Server) getRepository(w http.ResponseWriter, r *http.Request) {
	repo, err := s.Source.GetRepository(r.Context(), r.PathValue("repo"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, toRepository(*repo))
}

func (s *Server) listRefs(kind source.RefKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("repo")
		refs, err := s.Source.ListRefs(r.Context(), name, kind)
		if err != nil {
			writeError(w, r, err)
			return
		}
		coll := "branches"
		if kind == source.RefTag {
			coll = "tags"
		}
		list := []refJSON{}
		for _, ref := range refs {
			list = append(list, refJSON{Name: "repositories/" + name + "/" + coll + "/" + ref.Name, RefID: ref.Name, Kind: string(ref.Kind), Target: toCommit(name, ref.Target)})
		}
		writeJSON(w, 200, map[string]any{coll: list})
	}
}

func (s *Server) getTree(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	t, err := s.Source.GetTree(r.Context(), name, query(r, "revision", "revision"), query(r, "path", "path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	entries := []treeEntryJSON{}
	for _, e := range t.Entries {
		entries = append(entries, treeEntryJSON{Name: e.Name, Path: e.Path, Type: string(e.Type), Mode: e.Mode, OID: e.OID, Size: e.Size, LastCommit: toCommit(name, e.LastCommit)})
	}
	resp := map[string]any{"revision": t.Revision, "commitId": t.CommitID, "path": t.Path, "entries": entries}
	if t.Readme != nil {
		resp["readme"] = toBlob(t.Readme)
	}
	writeJSON(w, 200, resp)
}

func (s *Server) getBlob(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	b, last, commitID, err := s.Source.GetBlob(r.Context(), name, query(r, "revision", "revision"), query(r, "path", "path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := map[string]any{"blob": toBlob(b), "commitId": commitID}
	if last != nil {
		resp["lastCommit"] = toCommit(name, last)
	}
	writeJSON(w, 200, resp)
}

func (s *Server) getBlame(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	b, err := s.Source.GetBlame(r.Context(), name, query(r, "revision", "revision"), query(r, "path", "path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	lines := make([]blameLineJSON, 0, len(b.Lines))
	for _, l := range b.Lines {
		lines = append(lines, blameLineJSON(l))
	}
	commits := map[string]*commitJSON{}
	for id, c := range b.Commits {
		commits[id] = toCommit(name, c)
	}
	writeJSON(w, 200, map[string]any{"commitId": b.CommitID, "lines": lines, "commits": commits})
}

func (s *Server) listCommits(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	size, err := pageSize(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	tok, err := decodeToken(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	// Ask for one more than the page to know whether another page exists.
	cs, err := s.Source.ListCommits(r.Context(), name, query(r, "revision", "revision"), query(r, "path", "path"), tok.Offset, size+1)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := map[string]any{}
	if len(cs) > size {
		cs = cs[:size]
		resp["nextPageToken"] = encodeToken(pageToken{Offset: tok.Offset + size})
	}
	list := make([]*commitJSON, 0, len(cs))
	for _, c := range cs {
		list = append(list, toCommit(name, c))
	}
	resp["commits"] = list
	// The total is only computed for the first page, like a repository's
	// "N commits" link; later pages leave it out.
	if tok.Offset == 0 {
		if n, err := s.Source.CountCommits(r.Context(), name, query(r, "revision", "revision"), query(r, "path", "path")); err == nil {
			resp["totalSize"] = n
		}
	}
	writeJSON(w, 200, resp)
}

func (s *Server) getCommit(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	c, diff, err := s.Source.GetCommit(r.Context(), name, r.PathValue("commit"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	files := make([]fileDiffJSON, 0, len(diff))
	add, del := 0, 0
	for _, d := range diff {
		files = append(files, fileDiffJSON{OldPath: d.OldPath, NewPath: d.NewPath, Status: string(d.Status), Additions: d.Additions, Deletions: d.Deletions, Binary: d.Binary, TooLarge: d.TooLarge, Patch: d.Patch})
		add += d.Additions
		del += d.Deletions
	}
	writeJSON(w, 200, map[string]any{"commit": toCommit(name, c), "files": files, "stats": map[string]int{"additions": add, "deletions": del, "files": len(files)}})
}

func (s *Server) searchCode(w http.ResponseWriter, r *http.Request) {
	res, err := s.Source.SearchCode(r.Context(), r.PathValue("repo"), query(r, "revision", "revision"), query(r, "query", "query"), 200)
	if err != nil {
		writeError(w, r, err)
		return
	}
	type match struct {
		Path       string   `json:"path"`
		LineNumber int      `json:"lineNumber"`
		Line       string   `json:"line"`
		Before     []string `json:"before"`
		After      []string `json:"after"`
	}
	list := []match{}
	for _, m := range res.Matches {
		list = append(list, match{Path: m.Path, LineNumber: m.LineNumber, Line: m.Line, Before: nonNil(m.Before), After: nonNil(m.After)})
	}
	writeJSON(w, 200, map[string]any{"commitId": res.CommitID, "matches": list, "truncated": res.Truncated})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// --- Review ---

func (s *Server) listReviews(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	state := strings.ToUpper(query(r, "state", "state"))
	if state == "" {
		state = string(review.StateOpen)
	}
	rs, err := s.Reviews.ListReviews(r.Context(), name, state, 100)
	if err != nil {
		writeError(w, r, err)
		return
	}
	list := make([]reviewJSON, 0, len(rs))
	for i := range rs {
		list = append(list, toReview(name, &rs[i]))
	}
	writeJSON(w, 200, map[string]any{"reviews": list})
}

func (s *Server) getReview(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("repo")
	rv, err := s.Reviews.GetReview(r.Context(), name, r.PathValue("review"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, toReview(name, rv))
}

// reviewCustomMethod serves POST .../reviews/{review}:submitVerdict.
func (s *Server) reviewCustomMethod(w http.ResponseWriter, r *http.Request) {
	id, method, ok := strings.Cut(r.PathValue("review"), ":")
	if !ok || method != "submitVerdict" {
		writeError(w, r, fmt.Errorf("%w: unknown method %q", source.ErrNotFound, method))
		return
	}
	var req struct {
		Kind         string `json:"kind"`
		Body         string `json:"body"`
		HeadCommitID string `json:"headCommitId"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	name := r.PathValue("repo")
	rv, err := s.Reviews.SubmitVerdict(r.Context(), currentUser(r.Context()).Username, name, id, review.VerdictKind(strings.ToUpper(req.Kind)), req.Body, req.HeadCommitID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, toReview(name, rv))
}

func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body string `json:"body"`
		Path string `json:"path"`
		Line int32  `json:"line"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.Reviews.AddComment(r.Context(), currentUser(r.Context()).Username, r.PathValue("repo"), r.PathValue("review"), req.Body, req.Path, req.Line)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, toComment(*c))
}

// --- Agent activity ---

func (s *Server) changesetFilter(r *http.Request) (activity.Filter, error) {
	f := activity.Filter{Repository: r.PathValue("repo"), Agent: query(r, "agent", "agent")}
	if f.Repository == "" {
		f.Repository = strings.TrimPrefix(query(r, "repository", "repository"), "repositories/")
	}
	f.Agent = strings.TrimPrefix(f.Agent, "agents/")
	size, err := pageSize(r)
	if err != nil {
		return f, err
	}
	f.Limit = size
	tok, err := decodeToken(r)
	if err != nil {
		return f, err
	}
	if tok.Time != 0 {
		t := time.Unix(0, tok.Time).UTC()
		f.BeforeTime, f.BeforeKey = &t, tok.Key
	}
	return f, nil
}

func (s *Server) listChangesets(w http.ResponseWriter, r *http.Request) {
	f, err := s.changesetFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if f.Repository != "" {
		if _, err := s.Source.Catalog.GetRepository(r.Context(), f.Repository); err != nil {
			writeError(w, r, err)
			return
		}
	}
	cs, err := s.Activity.ListChangesets(r.Context(), f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	list := make([]changesetJSON, 0, len(cs))
	for _, c := range cs {
		list = append(list, toChangeset(c))
	}
	resp := map[string]any{"changesets": list}
	if len(cs) == f.Limit {
		last := cs[len(cs)-1]
		resp["nextPageToken"] = encodeToken(pageToken{Time: last.RecordTime.UnixNano(), Key: fmt.Sprintf("%s/%020d", last.Repository, last.Sequence)})
	}
	writeJSON(w, 200, resp)
}

// watchChangesets streams new changesets as server-sent events.
func (s *Server) watchChangesets(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, errors.New("streaming unsupported"))
		return
	}
	repo := strings.TrimPrefix(query(r, "repository", "repository"), "repositories/")
	agent := strings.TrimPrefix(query(r, "agent", "agent"), "agents/")
	sub, cancel := s.Activity.Broker.Subscribe()
	defer cancel()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	fmt.Fprint(w, "retry: 3000\n: connected\n\n")
	flusher.Flush()
	hb := s.Heartbeat
	if hb <= 0 {
		hb = 15 * time.Second
	}
	ticker := time.NewTicker(hb)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		case cs, ok := <-sub:
			if !ok {
				return
			}
			j := toChangeset(cs)
			if repo != "" && cs.Repository != repo {
				continue
			}
			if agent != "" && !contains(j.Agents, agent) && cs.Actor != agent {
				continue
			}
			b, _ := json.Marshal(j)
			fmt.Fprintf(w, "id: %s\nevent: changeset\ndata: %s\n\n", j.Name, b)
			flusher.Flush()
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	as, err := s.Activity.ListAgents(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	list := make([]agentJSON, 0, len(as))
	for _, a := range as {
		list = append(list, toAgent(a))
	}
	writeJSON(w, 200, map[string]any{"agents": list})
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	a, err := s.Activity.GetAgent(r.Context(), r.PathValue("agent"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, 200, toAgent(*a))
}
