package activity

import (
	"testing"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

func TestAttributeBatch(t *testing.T) {
	cs := &Changeset{
		Reference: "refs/heads/main", OldOID: "a", NewOID: "c3", Operation: "land_batch", Actor: "graft-batch",
		Members: []Member{{CommitID: "c1", Actor: "alpha"}, {CommitID: "c3", Actor: "beta"}},
	}
	Attribute(cs, []*source.Commit{{ID: "c3"}, {ID: "c2"}, {ID: "c1"}})
	got := []string{cs.Commits[0].Agent, cs.Commits[1].Agent, cs.Commits[2].Agent}
	if got[0] != "beta" || got[1] != "beta" || got[2] != "alpha" {
		t.Fatalf("attribution: %v", got)
	}
	if cs.Kind != KindBatchLand {
		t.Fatalf("kind %s", cs.Kind)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		cs   Changeset
		cms  []*source.Commit
		want Kind
	}{
		{Changeset{Reference: "refs/heads/main", OldOID: "a", NewOID: "b", Operation: "land"}, []*source.Commit{{ID: "b", Subject: "x"}}, KindLand},
		{Changeset{Reference: "refs/heads/main", OldOID: "a", NewOID: "b", Operation: "land"}, []*source.Commit{{ID: "b", Subject: "Merge", Body: "Graft-Resolved-By: agent/beta"}}, KindConflictResolved},
		{Changeset{Reference: "refs/heads/main", OldOID: "a", NewOID: "b", Operation: "commit_files"}, []*source.Commit{{ID: "b", Subject: "Revert", Body: "Graft-Auto-Revert: deadbeef"}}, KindAutoRevert},
		{Changeset{Reference: "refs/heads/x", OldOID: "0000", NewOID: "b", Operation: "commit_files"}, nil, KindBranchCreated},
		{Changeset{Reference: "refs/heads/x", OldOID: "b", NewOID: "0000000", Operation: "delete"}, nil, KindBranchDeleted},
		{Changeset{Reference: "refs/tags/v1", OldOID: "0", NewOID: "b", Operation: "receive_pack"}, nil, KindUpdate},
	}
	for i, c := range cases {
		cs := c.cs
		Attribute(&cs, c.cms)
		if cs.Kind != c.want {
			t.Errorf("case %d: got %s want %s", i, cs.Kind, c.want)
		}
	}
}
