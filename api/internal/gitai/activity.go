package gitai

import (
	"context"

	"github.com/feder-cr/git-ai/proto/go/gitalypb"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

var _ activity.Upstream = (*Client)(nil)

// WatchChangesets follows a repository's changeset log through the SDK,
// which retries transient failures itself.
func (c *Client) WatchChangesets(ctx context.Context, rel string, after uint64) (<-chan *activity.Changeset, <-chan error) {
	in, errs := c.sdk.Repo(rel).WatchChangesets(ctx, after)
	out := make(chan *activity.Changeset)
	go func() {
		defer close(out)
		for cs := range in {
			select {
			case out <- toChangeset(cs):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errs
}

// LandedCommits lists the commits a reference update brought in.
func (c *Client) LandedCommits(ctx context.Context, rel, ref, oldOID, newOID string, limit int) ([]*source.Commit, error) {
	revs := []string{newOID}
	if !activity.IsZeroOID(oldOID) {
		revs = append(revs, "^"+oldOID)
	} else if def, err := c.DefaultBranch(ctx, rel); err == nil && def != "" && ref != "refs/heads/"+def {
		if _, err := c.ResolveRevision(ctx, rel, "refs/heads/"+def); err == nil {
			revs = append(revs, "^refs/heads/"+def)
		}
	}
	stream, err := c.commits.ListCommits(ctx, &gitalypb.ListCommitsRequest{
		Repository: c.repo(rel), Revisions: revs, PaginationParams: &gitalypb.PaginationParameter{Limit: int32(limit)},
	})
	if err != nil {
		return nil, translate(err)
	}
	var out []*source.Commit
	err = recvAll(stream.Recv, func(m *gitalypb.ListCommitsResponse) {
		for _, gc := range m.GetCommits() {
			out = append(out, toCommit(gc))
		}
	})
	return out, err
}

func toChangeset(cs *gitalypb.Changeset) *activity.Changeset {
	out := &activity.Changeset{
		Sequence:  cs.GetSequence(),
		Reference: string(cs.GetReference()),
		OldOID:    cs.GetOldOid(),
		NewOID:    cs.GetNewOid(),
		Operation: cs.GetOperation(),
		Actor:     cs.GetActor(),
		Rewritten: cs.GetRewritten(),
	}
	if cs.GetRecordedAt() != nil {
		out.RecordTime = cs.GetRecordedAt().AsTime()
	}
	for _, m := range cs.GetMembers() {
		mm := activity.Member{OriginalCommitID: m.GetOriginalCommitId(), CommitID: m.GetCommitId(), Actor: m.GetActor(), Rewritten: m.GetRewritten()}
		for _, p := range m.GetResolvedPaths() {
			mm.ResolvedPaths = append(mm.ResolvedPaths, string(p))
		}
		out.Members = append(out.Members, mm)
	}
	return out
}
