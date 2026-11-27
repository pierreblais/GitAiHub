package httpapi

import (
	"strings"
	"time"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/identity"
	"github.com/pierreblais/gitaihub/api/internal/review"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

// The JSON resources below mirror proto/gitaihub/v1/*.proto in the proto3
// JSON mapping (lowerCamelCase fields, RFC 3339 timestamps, enums as names).

func ts(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

type agentCountJSON struct {
	Agent string `json:"agent"`
	Count int    `json:"count"`
}

type repositoryJSON struct {
	Name             string           `json:"name"`
	RepositoryID     string           `json:"repositoryId"`
	RelativePath     string           `json:"relativePath"`
	DefaultBranch    string           `json:"defaultBranch,omitempty"`
	LastActivityTime string           `json:"lastActivityTime,omitempty"`
	ChangesetCount   int              `json:"changesetCount"`
	TopAgents        []agentCountJSON `json:"topAgents"`
}

func toRepository(r source.Repository) repositoryJSON {
	out := repositoryJSON{
		Name: "repositories/" + r.Name, RepositoryID: r.Name, RelativePath: r.RelativePath,
		DefaultBranch: r.DefaultBranch, LastActivityTime: ts(r.LastActivityTime), ChangesetCount: r.LandingCount,
		TopAgents: []agentCountJSON{},
	}
	for _, a := range r.TopAgents {
		out.TopAgents = append(out.TopAgents, agentCountJSON(a))
	}
	return out
}

type signatureJSON struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Time  string `json:"time,omitempty"`
}

type landingJSON struct {
	Agent      string `json:"agent"`
	Sequence   uint64 `json:"sequence,string"`
	Operation  string `json:"operation"`
	Reference  string `json:"reference"`
	RecordTime string `json:"recordTime,omitempty"`
}

type trailerJSON struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type commitJSON struct {
	Name          string        `json:"name"`
	ID            string        `json:"id"`
	ParentIDs     []string      `json:"parentIds"`
	Subject       string        `json:"subject"`
	Body          string        `json:"body"`
	Author        signatureJSON `json:"author"`
	Committer     signatureJSON `json:"committer"`
	Agent         string        `json:"agent"`
	AgentVerified bool          `json:"agentVerified"`
	Landing       *landingJSON  `json:"landing,omitempty"`
	Trailers      []trailerJSON `json:"trailers"`
	AutoRevertOf  string        `json:"autoRevertOf,omitempty"`
	ResolvedBy    string        `json:"resolvedBy,omitempty"`
}

func toSig(s source.Signature) signatureJSON {
	return signatureJSON{Name: s.Name, Email: s.Email, Time: ts(s.Time)}
}

func toCommit(repo string, c *source.Commit) *commitJSON {
	if c == nil {
		return nil
	}
	out := &commitJSON{
		Name: "repositories/" + repo + "/commits/" + c.ID, ID: c.ID, ParentIDs: c.ParentIDs,
		Subject: c.Subject, Body: c.Body, Author: toSig(c.Author), Committer: toSig(c.Committer),
		Agent: c.Agent(), AgentVerified: c.AgentVerified(), Trailers: []trailerJSON{},
		AutoRevertOf: c.AutoRevertOf, ResolvedBy: c.ResolvedBy,
	}
	if out.ParentIDs == nil {
		out.ParentIDs = []string{}
	}
	if c.Landing != nil {
		out.Landing = &landingJSON{Agent: c.Landing.Agent, Sequence: c.Landing.Sequence, Operation: c.Landing.Operation, Reference: c.Landing.Reference, RecordTime: ts(c.Landing.Time)}
	}
	for _, t := range c.Trailers {
		out.Trailers = append(out.Trailers, trailerJSON(t))
	}
	return out
}

type refJSON struct {
	Name   string      `json:"name"`
	RefID  string      `json:"refId"`
	Kind   string      `json:"kind"`
	Target *commitJSON `json:"target,omitempty"`
}

type treeEntryJSON struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Type       string      `json:"type"`
	Mode       int32       `json:"mode"`
	OID        string      `json:"oid"`
	Size       int64       `json:"size,string"`
	LastCommit *commitJSON `json:"lastCommit,omitempty"`
}

type blobJSON struct {
	Path      string `json:"path"`
	OID       string `json:"oid"`
	Size      int64  `json:"size,string"`
	Content   string `json:"content"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
}

func toBlob(b *source.Blob) *blobJSON {
	if b == nil {
		return nil
	}
	return &blobJSON{Path: b.Path, OID: b.OID, Size: b.Size, Content: string(b.Content), Binary: b.Binary, Truncated: b.Truncated}
}

type fileDiffJSON struct {
	OldPath   string `json:"oldPath"`
	NewPath   string `json:"newPath"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
	TooLarge  bool   `json:"tooLarge"`
	Patch     string `json:"patch"`
}

type blameLineJSON struct {
	LineNumber int32  `json:"lineNumber"`
	CommitID   string `json:"commitId"`
	Content    string `json:"content"`
}

type userJSON struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	CanReview   bool   `json:"canReview"`
	CreateTime  string `json:"createTime"`
}

func toUser(u *identity.User, canReview bool) userJSON {
	return userJSON{Name: "users/" + u.Username, Username: u.Username, DisplayName: u.DisplayName, CanReview: canReview, CreateTime: ts(u.CreateTime)}
}

type reviewFileJSON struct {
	Path   string `json:"path"`
	Class  string `json:"class"`
	Reason string `json:"reason"`
}

type verdictJSON struct {
	Actor      string `json:"actor"`
	Kind       string `json:"kind"`
	CommitID   string `json:"commitId"`
	Body       string `json:"body"`
	CreateTime string `json:"createTime"`
	Current    bool   `json:"current"`
}

type commentJSON struct {
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	Body       string `json:"body"`
	Path       string `json:"path,omitempty"`
	Line       int32  `json:"line,omitempty"`
	CommitID   string `json:"commitId"`
	CreateTime string `json:"createTime"`
}

type requirementJSON struct {
	Rule              string   `json:"rule"`
	Description       string   `json:"description"`
	RequiredApprovals int32    `json:"requiredApprovals"`
	Approvals         int32    `json:"approvals"`
	Satisfied         bool     `json:"satisfied"`
	Paths             []string `json:"paths"`
}

type fileGroupJSON struct {
	Class string           `json:"class"`
	Files []reviewFileJSON `json:"files"`
}

type reviewJSON struct {
	Name            string            `json:"name"`
	ID              string            `json:"id"`
	TargetBranch    string            `json:"targetBranch"`
	BaseCommitID    string            `json:"baseCommitId"`
	HeadCommitID    string            `json:"headCommitId"`
	Author          string            `json:"author"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	State           string            `json:"state"`
	Files           []reviewFileJSON  `json:"files"`
	FileGroups      []fileGroupJSON   `json:"fileGroups"`
	Verdicts        []verdictJSON     `json:"verdicts"`
	CurrentVerdicts []verdictJSON     `json:"currentVerdicts"`
	Comments        []commentJSON     `json:"comments"`
	Requirements    []requirementJSON `json:"requirements"`
	Approved        bool              `json:"approved"`
	CreateTime      string            `json:"createTime"`
	UpdateTime      string            `json:"updateTime"`
}

func toVerdict(v review.Verdict) verdictJSON {
	return verdictJSON{Actor: v.Actor, Kind: string(v.Kind), CommitID: v.CommitID, Body: v.Body, CreateTime: ts(v.CreateTime), Current: v.Current}
}

func toComment(c review.Comment) commentJSON {
	return commentJSON{ID: c.ID, Actor: c.Actor, Body: c.Body, Path: c.Path, Line: c.Line, CommitID: c.CommitID, CreateTime: ts(c.CreateTime)}
}

func toReview(repo string, r *review.Review) reviewJSON {
	out := reviewJSON{
		Name: "repositories/" + repo + "/reviews/" + r.ID, ID: r.ID, TargetBranch: r.TargetBranch,
		BaseCommitID: r.BaseCommitID, HeadCommitID: r.HeadCommitID, Author: r.Author, Title: r.Title,
		Description: r.Description, State: string(r.State), Approved: r.Approved,
		CreateTime: ts(r.CreateTime), UpdateTime: ts(r.UpdateTime),
		Files: []reviewFileJSON{}, FileGroups: []fileGroupJSON{}, Verdicts: []verdictJSON{}, CurrentVerdicts: []verdictJSON{},
		Comments: []commentJSON{}, Requirements: []requirementJSON{},
	}
	for _, f := range r.Files {
		out.Files = append(out.Files, reviewFileJSON{Path: f.Path, Class: string(f.Class), Reason: f.Reason})
	}
	for _, g := range r.GroupFiles() {
		fg := fileGroupJSON{Class: string(g.Class)}
		for _, f := range g.Files {
			fg.Files = append(fg.Files, reviewFileJSON{Path: f.Path, Class: string(f.Class), Reason: f.Reason})
		}
		out.FileGroups = append(out.FileGroups, fg)
	}
	for _, v := range r.Verdicts {
		out.Verdicts = append(out.Verdicts, toVerdict(v))
	}
	for _, v := range r.CurrentVerdicts() {
		out.CurrentVerdicts = append(out.CurrentVerdicts, toVerdict(v))
	}
	for _, c := range r.Comments {
		out.Comments = append(out.Comments, toComment(c))
	}
	for _, q := range r.Requirements {
		rq := requirementJSON{Rule: q.Rule, Description: q.Description, RequiredApprovals: q.RequiredApprovals, Approvals: q.Approvals, Satisfied: q.Satisfied, Paths: q.Paths}
		if rq.Paths == nil {
			rq.Paths = []string{}
		}
		out.Requirements = append(out.Requirements, rq)
	}
	return out
}

type memberJSON struct {
	OriginalCommitID string   `json:"originalCommitId"`
	CommitID         string   `json:"commitId"`
	Actor            string   `json:"actor"`
	Rewritten        bool     `json:"rewritten"`
	ResolvedPaths    []string `json:"resolvedPaths"`
}

type landedCommitJSON struct {
	CommitID     string `json:"commitId"`
	Agent        string `json:"agent"`
	Subject      string `json:"subject"`
	AutoRevertOf string `json:"autoRevertOf,omitempty"`
	ResolvedBy   string `json:"resolvedBy,omitempty"`
}

type changesetJSON struct {
	Name       string             `json:"name"`
	Repository string             `json:"repository"`
	Sequence   uint64             `json:"sequence,string"`
	Reference  string             `json:"reference"`
	Branch     string             `json:"branch,omitempty"`
	OldOID     string             `json:"oldOid"`
	NewOID     string             `json:"newOid"`
	Operation  string             `json:"operation"`
	Actor      string             `json:"actor"`
	Agents     []string           `json:"agents"`
	RecordTime string             `json:"recordTime"`
	Rewritten  bool               `json:"rewritten"`
	Kind       string             `json:"kind"`
	Members    []memberJSON       `json:"members"`
	Commits    []landedCommitJSON `json:"commits"`
}

func toChangeset(cs activity.Changeset) changesetJSON {
	out := changesetJSON{
		Name:       "repositories/" + cs.Repository + "/changesets/" + itoa(cs.Sequence),
		Repository: "repositories/" + cs.Repository, Sequence: cs.Sequence, Reference: cs.Reference,
		OldOID: cs.OldOID, NewOID: cs.NewOID, Operation: cs.Operation, Actor: cs.Actor,
		RecordTime: ts(cs.RecordTime), Rewritten: cs.Rewritten, Kind: string(cs.Kind),
		Members: []memberJSON{}, Commits: []landedCommitJSON{}, Agents: []string{},
	}
	if b, ok := strings.CutPrefix(cs.Reference, "refs/heads/"); ok {
		out.Branch = b
	}
	seen := map[string]bool{}
	addAgent := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out.Agents = append(out.Agents, a)
		}
	}
	for _, m := range cs.Members {
		mm := memberJSON(m)
		if mm.ResolvedPaths == nil {
			mm.ResolvedPaths = []string{}
		}
		out.Members = append(out.Members, mm)
		addAgent(m.Actor)
	}
	for _, c := range cs.Commits {
		out.Commits = append(out.Commits, landedCommitJSON(c))
		addAgent(c.Agent)
	}
	if len(out.Agents) == 0 {
		addAgent(cs.Actor)
	}
	return out
}

type dailyJSON struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type agentJSON struct {
	Name            string      `json:"name"`
	AgentID         string      `json:"agentId"`
	Changesets      int         `json:"changesetCount"`
	Lands           int         `json:"landCount"`
	CommitsLanded   int         `json:"commitCount"`
	ConflictsSolved int         `json:"conflictsResolvedCount"`
	AutoReverts     int         `json:"autoRevertCount"`
	RevertedCommits int         `json:"revertedCommitCount"`
	Repositories    []string    `json:"repositories"`
	FirstSeenTime   string      `json:"firstSeenTime,omitempty"`
	LastSeenTime    string      `json:"lastSeenTime,omitempty"`
	Daily           []dailyJSON `json:"dailyCommits,omitempty"`
}

func toAgent(a activity.Agent) agentJSON {
	out := agentJSON{
		Name: "agents/" + a.Name, AgentID: a.Name, Changesets: a.Changesets, Lands: a.Lands,
		CommitsLanded: a.CommitsLanded, ConflictsSolved: a.ConflictsSolved, AutoReverts: a.AutoReverts,
		RevertedCommits: a.RevertedCommits, FirstSeenTime: ts(a.FirstSeen), LastSeenTime: ts(a.LastSeen),
		Repositories: []string{},
	}
	for _, r := range a.Repositories {
		out.Repositories = append(out.Repositories, "repositories/"+r)
	}
	for _, d := range a.Daily {
		out.Daily = append(out.Daily, dailyJSON{Day: d.Day.Format("2006-01-02"), Count: d.Count})
	}
	return out
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
