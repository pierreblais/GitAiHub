package gitai

import (
	"context"

	"github.com/feder-cr/git-ai/proto/go/gitalypb"
	"github.com/feder-cr/git-ai/sdk"

	"github.com/pierreblais/gitaihub/api/internal/review"
)

var _ review.Gateway = (*Client)(nil)

func (c *Client) ListReviews(ctx context.Context, rel string, includeClosed bool, limit int) ([]review.Review, error) {
	rs, err := c.sdk.Repo(rel).ListReviews(ctx, sdk.ListReviewsOptions{IncludeClosed: includeClosed, Limit: uint32(max(limit, 0))})
	if err != nil {
		return nil, translate(err)
	}
	out := make([]review.Review, 0, len(rs))
	for _, r := range rs {
		out = append(out, *toReview(r))
	}
	return out, nil
}

func (c *Client) GetReview(ctx context.Context, rel, id string) (*review.Review, error) {
	r, err := c.sdk.Repo(rel).GetReview(ctx, id)
	if err != nil {
		return nil, translate(err)
	}
	return toReview(r), nil
}

func (c *Client) AddComment(ctx context.Context, token, rel, id, body, path string, line int32) (*review.Comment, error) {
	u, err := c.userClient(ctx, token)
	if err != nil {
		return nil, err
	}
	cm, err := u.Repo(rel).CommentReview(ctx, sdk.CommentReviewOptions{ID: id, Body: body, Path: path, Line: line})
	if err != nil {
		return nil, translate(err)
	}
	out := toComment(cm)
	return &out, nil
}

func (c *Client) SubmitVerdict(ctx context.Context, token, rel, id string, kind review.VerdictKind, body, head string) (*review.Review, error) {
	u, err := c.userClient(ctx, token)
	if err != nil {
		return nil, err
	}
	opts := sdk.VerdictOptions{ID: id, Head: head, Body: body}
	var r *sdk.Review
	if kind == review.VerdictRequestChanges {
		r, err = u.Repo(rel).RequestChangesAt(ctx, opts)
	} else {
		r, err = u.Repo(rel).ApproveReviewAt(ctx, opts)
	}
	if err != nil {
		return nil, translate(err)
	}
	return toReview(r), nil
}

func toReview(r *gitalypb.Review) *review.Review {
	out := &review.Review{
		ID:           r.GetId(),
		TargetBranch: string(r.GetTargetBranch()),
		BaseCommitID: r.GetBaseCommitId(),
		HeadCommitID: r.GetHeadCommitId(),
		Author:       r.GetAuthor(),
		Title:        string(r.GetTitle()),
		Description:  string(r.GetDescription()),
		Approved:     r.GetApproved(),
	}
	switch r.GetState() {
	case gitalypb.Review_STATE_LANDED:
		out.State = review.StateLanded
	case gitalypb.Review_STATE_CLOSED:
		out.State = review.StateClosed
	default:
		out.State = review.StateOpen
	}
	if r.GetCreatedAt() != nil {
		out.CreateTime = r.GetCreatedAt().AsTime()
	}
	if r.GetUpdatedAt() != nil {
		out.UpdateTime = r.GetUpdatedAt().AsTime()
	}
	for _, f := range r.GetFiles() {
		out.Files = append(out.Files, review.File{Path: string(f.GetPath()), Class: toClass(f.GetClass()), Reason: f.GetReason()})
	}
	for _, v := range r.GetVerdicts() {
		vd := review.Verdict{Actor: v.GetActor(), CommitID: v.GetCommitId(), Body: string(v.GetBody()), Current: v.GetCurrent(), Kind: review.VerdictApprove}
		if v.GetKind() == gitalypb.ReviewVerdict_KIND_REQUEST_CHANGES {
			vd.Kind = review.VerdictRequestChanges
		}
		if v.GetCreatedAt() != nil {
			vd.CreateTime = v.GetCreatedAt().AsTime()
		}
		out.Verdicts = append(out.Verdicts, vd)
	}
	for _, cm := range r.GetComments() {
		out.Comments = append(out.Comments, toComment(cm))
	}
	for _, q := range r.GetRequirements() {
		req := review.Requirement{Rule: q.GetRule(), Description: q.GetDescription(), RequiredApprovals: q.GetRequiredApprovals(), Approvals: q.GetApprovals(), Satisfied: q.GetSatisfied()}
		for _, p := range q.GetPaths() {
			req.Paths = append(req.Paths, string(p))
		}
		out.Requirements = append(out.Requirements, req)
	}
	return out
}

func toComment(cm *gitalypb.ReviewComment) review.Comment {
	out := review.Comment{ID: cm.GetId(), Actor: cm.GetActor(), Body: string(cm.GetBody()), Path: string(cm.GetPath()), Line: cm.GetLine(), CommitID: cm.GetCommitId()}
	if cm.GetCreatedAt() != nil {
		out.CreateTime = cm.GetCreatedAt().AsTime()
	}
	return out
}

func toClass(c gitalypb.ReviewFile_ChangeClass) review.ChangeClass {
	switch c {
	case gitalypb.ReviewFile_CHANGE_CLASS_COMMENT_ONLY:
		return review.ClassCommentOnly
	case gitalypb.ReviewFile_CHANGE_CLASS_DOCS:
		return review.ClassDocs
	case gitalypb.ReviewFile_CHANGE_CLASS_TESTS:
		return review.ClassTests
	case gitalypb.ReviewFile_CHANGE_CLASS_GENERATED:
		return review.ClassGenerated
	case gitalypb.ReviewFile_CHANGE_CLASS_CONFIG:
		return review.ClassConfig
	}
	return review.ClassCode
}
