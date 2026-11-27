// Package activity is the Agent activity bounded context: gitai's changeset
// log (every reference update, with the agent that made it), the commits each
// changeset landed, and per-agent statistics derived from them.
package activity

import (
	"context"
	"time"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Kind is GitaiHub's classification of a changeset, for the feed.
type Kind string

const (
	KindLand             Kind = "LAND"
	KindBatchLand        Kind = "BATCH_LAND"
	KindConflictResolved Kind = "CONFLICT_RESOLVED"
	KindAutoRevert       Kind = "AUTO_REVERT"
	KindBranchCreated    Kind = "BRANCH_CREATED"
	KindBranchDeleted    Kind = "BRANCH_DELETED"
	KindUpdate           Kind = "UPDATE"
)

// Changeset is one entry of a repository's changeset log.
type Changeset struct {
	Repository string
	Sequence   uint64
	Reference  string
	OldOID     string
	NewOID     string
	Operation  string
	Actor      string
	RecordTime time.Time
	Rewritten  bool
	Members    []Member
	Kind       Kind
	// Commits are the commits that this changeset landed (newest first).
	Commits []LandedCommit
}

// Member is one request of a batched land.
type Member struct {
	OriginalCommitID string
	CommitID         string
	Actor            string
	Rewritten        bool
	ResolvedPaths    []string
}

// LandedCommit is a commit that a changeset brought onto its reference.
type LandedCommit struct {
	CommitID     string
	Agent        string
	Subject      string
	AutoRevertOf string
	ResolvedBy   string
}

// Agent is an identity that lands changes, with its statistics.
type Agent struct {
	Name            string
	Changesets      int
	Lands           int
	CommitsLanded   int
	ConflictsSolved int
	AutoReverts     int
	// RevertedCommits counts this agent's commits that gitai auto-reverted.
	RevertedCommits int
	Repositories    []string
	FirstSeen       time.Time
	LastSeen        time.Time
	// Daily is the number of landed commits per day over the last 30 days.
	Daily []DailyCount
}

// DailyCount is a count for one day.
type DailyCount struct {
	Day   time.Time
	Count int
}

// Filter narrows changeset listings.
type Filter struct {
	Repository string
	Agent      string
	// BeforeSequence/BeforeRepo page through results newest first.
	BeforeTime *time.Time
	BeforeKey  string
	Limit      int
}

// Store persists the changeset log copy, landed commits and cursors.
type Store interface {
	Cursor(ctx context.Context, repo string) (uint64, error)
	SaveChangeset(ctx context.Context, cs *Changeset) error
	ListChangesets(ctx context.Context, f Filter) ([]Changeset, error)
	Landings(ctx context.Context, repo string, commitIDs []string) (map[string]source.Landing, error)
	ListAgents(ctx context.Context) ([]Agent, error)
	GetAgent(ctx context.Context, name string) (*Agent, error)
	RepositoryStats(ctx context.Context) (map[string]RepoStats, error)
}

// RepoStats is the activity summary of one repository.
type RepoStats struct {
	LastActivity time.Time
	Changesets   int
	TopAgents    []source.AgentCount
}

// Upstream is the port to gitai's ChangesetService and commit reads.
type Upstream interface {
	// WatchChangesets streams changesets after the cursor, then live ones.
	WatchChangesets(ctx context.Context, relativePath string, after uint64) (<-chan *Changeset, <-chan error)
	// LandedCommits lists the commits a reference update brought in: those
	// reachable from newOID and not from oldOID. For a new branch other than
	// the default one, commits already on the default branch are left out.
	LandedCommits(ctx context.Context, relativePath, reference, oldOID, newOID string, limit int) ([]*source.Commit, error)
}

// ZeroOID is the all-zeros object ID of a created or deleted reference.
func IsZeroOID(oid string) bool {
	if oid == "" {
		return true
	}
	for _, c := range oid {
		if c != '0' {
			return false
		}
	}
	return true
}
