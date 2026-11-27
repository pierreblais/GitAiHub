// Package review is the Review bounded context: gitai's server-side code
// reviews, with files classified by change class, approval requirements,
// verdicts and line comments.
package review

import (
	"context"
	"errors"
	"time"
)

// ErrNoGitaiIdentity is returned when a user has no gitai token to write
// comments and verdicts with.
var ErrNoGitaiIdentity = errors.New("no gitai identity configured for this user")

// State is the lifecycle state of a review.
type State string

const (
	StateOpen   State = "OPEN"
	StateLanded State = "LANDED"
	StateClosed State = "CLOSED"
)

// ChangeClass is gitai's classification of a changed file.
type ChangeClass string

const (
	ClassCode        ChangeClass = "CODE"
	ClassCommentOnly ChangeClass = "COMMENT_ONLY"
	ClassDocs        ChangeClass = "DOCS"
	ClassTests       ChangeClass = "TESTS"
	ClassGenerated   ChangeClass = "GENERATED"
	ClassConfig      ChangeClass = "CONFIG"
)

// ClassOrder is the display order of change classes.
var ClassOrder = []ChangeClass{ClassCode, ClassTests, ClassConfig, ClassDocs, ClassCommentOnly, ClassGenerated}

// VerdictKind is an approval or a change request.
type VerdictKind string

const (
	VerdictApprove        VerdictKind = "APPROVE"
	VerdictRequestChanges VerdictKind = "REQUEST_CHANGES"
)

// Review is one gitai review.
type Review struct {
	ID           string
	TargetBranch string
	BaseCommitID string
	HeadCommitID string
	Author       string
	Title        string
	Description  string
	State        State
	Files        []File
	Verdicts     []Verdict
	Comments     []Comment
	Requirements []Requirement
	Approved     bool
	CreateTime   time.Time
	UpdateTime   time.Time
}

// File is a changed file of a review.
type File struct {
	Path   string
	Class  ChangeClass
	Reason string
}

// Verdict is an approval or change request by one identity on one head.
type Verdict struct {
	Actor      string
	Kind       VerdictKind
	CommitID   string
	Body       string
	CreateTime time.Time
	Current    bool
}

// Comment is a review comment, optionally anchored to a file line.
type Comment struct {
	ID         string
	Actor      string
	Body       string
	Path       string
	Line       int32
	CommitID   string
	CreateTime time.Time
}

// Requirement is one approval rule and whether it is met.
type Requirement struct {
	Rule              string
	Description       string
	RequiredApprovals int32
	Approvals         int32
	Satisfied         bool
	Paths             []string
}

// FileGroup is the files of one change class.
type FileGroup struct {
	Class ChangeClass
	Files []File
}

// GroupFiles groups the review's files by class in ClassOrder.
func (r *Review) GroupFiles() []FileGroup {
	by := map[ChangeClass][]File{}
	for _, f := range r.Files {
		by[f.Class] = append(by[f.Class], f)
	}
	var out []FileGroup
	for _, c := range ClassOrder {
		if fs := by[c]; len(fs) > 0 {
			out = append(out, FileGroup{Class: c, Files: fs})
		}
	}
	return out
}

// CurrentVerdicts returns the latest current verdict of each actor.
func (r *Review) CurrentVerdicts() []Verdict {
	latest := map[string]int{}
	var order []string
	for i, v := range r.Verdicts {
		if !v.Current {
			continue
		}
		if _, ok := latest[v.Actor]; !ok {
			order = append(order, v.Actor)
		}
		latest[v.Actor] = i
	}
	out := make([]Verdict, 0, len(order))
	for _, a := range order {
		out = append(out, r.Verdicts[latest[a]])
	}
	return out
}

// Gateway is the port to gitai's ReviewService. Writes carry the gitai token
// of the user, so gitai records the right actor.
type Gateway interface {
	ListReviews(ctx context.Context, repo string, includeClosed bool, limit int) ([]Review, error)
	GetReview(ctx context.Context, repo, id string) (*Review, error)
	AddComment(ctx context.Context, token, repo, id, body, path string, line int32) (*Comment, error)
	SubmitVerdict(ctx context.Context, token, repo, id string, kind VerdictKind, body, head string) (*Review, error)
}
