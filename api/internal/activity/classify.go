package activity

import (
	"strings"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Attribute fills cs.Commits from the commits the changeset landed and sets
// cs.Kind. For a batched land, each member's commit (and the commits below
// it, down to the previous member) belong to the member's actor.
func Attribute(cs *Changeset, commits []*source.Commit) {
	memberActor := map[string]string{}
	for _, m := range cs.Members {
		memberActor[m.CommitID] = m.Actor
	}
	// commits are newest first; walk them and switch owner at member tips.
	owner := cs.Actor
	if len(cs.Members) > 0 {
		owner = cs.Members[len(cs.Members)-1].Actor
	}
	cs.Commits = cs.Commits[:0]
	for _, c := range commits {
		source.ApplyTrailers(c)
		if a, ok := memberActor[c.ID]; ok {
			owner = a
		}
		cs.Commits = append(cs.Commits, LandedCommit{
			CommitID:     c.ID,
			Agent:        owner,
			Subject:      c.Subject,
			AutoRevertOf: c.AutoRevertOf,
			ResolvedBy:   c.ResolvedBy,
		})
	}
	cs.Kind = classify(cs)
}

func classify(cs *Changeset) Kind {
	if strings.HasPrefix(cs.Reference, "refs/heads/") {
		if IsZeroOID(cs.NewOID) {
			return KindBranchDeleted
		}
	}
	for _, c := range cs.Commits {
		if c.AutoRevertOf != "" {
			return KindAutoRevert
		}
	}
	if cs.Operation == "revert" {
		return KindAutoRevert
	}
	for _, c := range cs.Commits {
		if c.ResolvedBy != "" {
			return KindConflictResolved
		}
	}
	for _, m := range cs.Members {
		if len(m.ResolvedPaths) > 0 {
			return KindConflictResolved
		}
	}
	if cs.Operation == "land_batch" {
		return KindBatchLand
	}
	if IsZeroOID(cs.OldOID) && strings.HasPrefix(cs.Reference, "refs/heads/") {
		return KindBranchCreated
	}
	if strings.HasPrefix(cs.Operation, "land") {
		return KindLand
	}
	return KindUpdate
}
