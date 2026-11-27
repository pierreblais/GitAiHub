// Package source is the Source bounded context: repositories, refs, trees,
// files, commits, diffs and blame, as read from the gitai server.
//
// Ubiquitous language: a commit is "landed" on a branch by an "agent"
// (the authenticated gitai identity recorded in the changeset log).
// We never say "pushed" or "pull request".
package source

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a repository, revision or path does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidArgument is returned for malformed requests.
var ErrInvalidArgument = errors.New("invalid argument")

// Repository is a gitai repository known to GitaiHub.
type Repository struct {
	// Name is the resource ID used in "repositories/{name}".
	Name string
	// RelativePath is the path of the repository on the gitai storage.
	RelativePath  string
	DefaultBranch string
	// LastActivityTime is when the last changeset was recorded (zero if none).
	LastActivityTime time.Time
	// TopAgents are the agents with the most landings, most active first.
	TopAgents []AgentCount
	// LandingCount is the number of changesets recorded for the repository.
	LandingCount int
}

// AgentCount is an agent with a number of landings.
type AgentCount struct {
	Agent string
	Count int
}

// RefKind distinguishes branches and tags.
type RefKind string

const (
	RefBranch RefKind = "BRANCH"
	RefTag    RefKind = "TAG"
)

// Ref is a branch or a tag.
type Ref struct {
	Name   string
	Kind   RefKind
	Target *Commit
}

// EntryType is the kind of a tree entry.
type EntryType string

const (
	EntryTree      EntryType = "TREE"
	EntryBlob      EntryType = "BLOB"
	EntrySubmodule EntryType = "SUBMODULE"
)

// TreeEntry is one entry of a directory.
type TreeEntry struct {
	Name string
	Path string
	Type EntryType
	Mode int32
	OID  string
	Size int64
	// LastCommit is the last commit that changed the entry, if known.
	LastCommit *Commit
}

// Blob is the content of one file.
type Blob struct {
	Path      string
	OID       string
	Size      int64
	Content   []byte
	Binary    bool
	Truncated bool
}

// Signature is the author or committer of a commit.
type Signature struct {
	Name  string
	Email string
	Time  time.Time
}

// Landing says how a commit reached its branch, from the changeset log.
type Landing struct {
	Agent     string
	Sequence  uint64
	Operation string
	Reference string
	Time      time.Time
}

// Commit is a commit with the gitai-specific facts GitaiHub shows.
type Commit struct {
	ID        string
	ParentIDs []string
	Subject   string
	Body      string
	Author    Signature
	Committer Signature
	Trailers  []Trailer
	// Landing is set when the changeset log says which agent landed it.
	Landing *Landing
	// AutoRevertOf is the culprit commit of a Graft-Auto-Revert trailer.
	AutoRevertOf string
	// ResolvedBy is the resolver named by a Graft-Resolved-By trailer.
	ResolvedBy string
}

// Agent returns the agent shown for the commit: the landing actor when the
// changeset log knows it, otherwise the author name (which clients can set
// freely, hence only a fallback).
func (c *Commit) Agent() string {
	if c.Landing != nil && c.Landing.Agent != "" {
		return c.Landing.Agent
	}
	return c.Author.Name
}

// AgentVerified reports whether Agent comes from the authenticated landing.
func (c *Commit) AgentVerified() bool { return c.Landing != nil && c.Landing.Agent != "" }

// Trailer is a "Key: value" line at the end of a commit message.
type Trailer struct {
	Key   string
	Value string
}

// FileStatus is the kind of change of one file in a diff.
type FileStatus string

const (
	FileAdded    FileStatus = "ADDED"
	FileModified FileStatus = "MODIFIED"
	FileDeleted  FileStatus = "DELETED"
	FileRenamed  FileStatus = "RENAMED"
)

// FileDiff is the change of one file in a commit.
type FileDiff struct {
	OldPath   string
	NewPath   string
	Status    FileStatus
	Additions int
	Deletions int
	Binary    bool
	TooLarge  bool
	Patch     string
}

// BlameLine is one line of a file with the commit that last changed it.
type BlameLine struct {
	LineNumber int32
	CommitID   string
	Content    string
}

// Blame is the blame of a file, with the commits it refers to.
type Blame struct {
	CommitID string
	Lines    []BlameLine
	Commits  map[string]*Commit
}

// SearchMatch is one line matching a code search.
type SearchMatch struct {
	Path       string
	LineNumber int
	Line       string
	Before     []string
	After      []string
}

// SearchResult is the result of a code search.
type SearchResult struct {
	CommitID  string
	Matches   []SearchMatch
	Truncated bool
}

// Gateway is the port to the upstream gitai server (implemented by the
// anticorruption layer in package gitai).
type Gateway interface {
	DefaultBranch(ctx context.Context, repo string) (string, error)
	ListBranches(ctx context.Context, repo string) ([]Ref, error)
	ListTags(ctx context.Context, repo string) ([]Ref, error)
	ResolveRevision(ctx context.Context, repo, rev string) (string, error)
	ListTree(ctx context.Context, repo, rev, path string) ([]TreeEntry, error)
	ReadBlob(ctx context.Context, repo, rev, path string, maxBytes int64) (*Blob, error)
	ListCommits(ctx context.Context, repo, rev, path string, offset, limit int) ([]*Commit, error)
	CountCommits(ctx context.Context, repo, rev, path string) (int, error)
	GetCommits(ctx context.Context, repo string, ids []string) (map[string]*Commit, error)
	CommitDiff(ctx context.Context, repo string, commit *Commit) ([]FileDiff, error)
	Blame(ctx context.Context, repo, rev, path string) (*Blame, error)
	SearchCode(ctx context.Context, repo, rev, query string, max int) (*SearchResult, error)
}

// RepositoryCatalog lists the repositories GitaiHub knows about.
type RepositoryCatalog interface {
	ListRepositories(ctx context.Context) ([]Repository, error)
	GetRepository(ctx context.Context, name string) (*Repository, error)
}

// LandingLookup finds the landing of commits in the changeset log (supplied
// by the Agent activity context).
type LandingLookup interface {
	Landings(ctx context.Context, repo string, commitIDs []string) (map[string]Landing, error)
}
