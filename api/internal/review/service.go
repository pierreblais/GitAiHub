package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Service answers Review queries and the two commands the MVP allows:
// commenting and giving a verdict. Neither changes code.
type Service struct {
	Catalog source.RepositoryCatalog
	Gateway Gateway
	// Tokens maps a GitaiHub username to its gitai token.
	Tokens map[string]string
}

// ListReviews lists reviews of a repository, newest first.
func (s *Service) ListReviews(ctx context.Context, repo string, state string, limit int) ([]Review, error) {
	r, err := s.Catalog.GetRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	includeClosed := state != "" && state != string(StateOpen)
	all, err := s.Gateway.ListReviews(ctx, r.RelativePath, includeClosed, limit)
	if err != nil {
		return nil, err
	}
	if state == "" || state == "ALL" {
		return all, nil
	}
	out := all[:0]
	for _, rv := range all {
		if string(rv.State) == state {
			out = append(out, rv)
		}
	}
	return out, nil
}

// GetReview returns one review with comments and verdicts.
func (s *Service) GetReview(ctx context.Context, repo, id string) (*Review, error) {
	r, err := s.Catalog.GetRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	return s.Gateway.GetReview(ctx, r.RelativePath, id)
}

// CanWrite reports whether the user can comment and give verdicts.
func (s *Service) CanWrite(username string) bool { return s.Tokens[username] != "" }

// AddComment comments on a review as the user.
func (s *Service) AddComment(ctx context.Context, username, repo, id, body, path string, line int32) (*Comment, error) {
	tok := s.Tokens[username]
	if tok == "" {
		return nil, ErrNoGitaiIdentity
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%w: body is required", source.ErrInvalidArgument)
	}
	if line < 0 {
		return nil, fmt.Errorf("%w: line must not be negative", source.ErrInvalidArgument)
	}
	r, err := s.Catalog.GetRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	return s.Gateway.AddComment(ctx, tok, r.RelativePath, id, body, path, line)
}

// SubmitVerdict approves or requests changes as the user, on the head the
// user saw (gitai refuses a verdict on a head that moved).
func (s *Service) SubmitVerdict(ctx context.Context, username, repo, id string, kind VerdictKind, body, head string) (*Review, error) {
	tok := s.Tokens[username]
	if tok == "" {
		return nil, ErrNoGitaiIdentity
	}
	if kind != VerdictApprove && kind != VerdictRequestChanges {
		return nil, fmt.Errorf("%w: kind must be APPROVE or REQUEST_CHANGES", source.ErrInvalidArgument)
	}
	if head == "" {
		return nil, fmt.Errorf("%w: head_commit_id is required", source.ErrInvalidArgument)
	}
	r, err := s.Catalog.GetRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	return s.Gateway.SubmitVerdict(ctx, tok, r.RelativePath, id, kind, body, head)
}
