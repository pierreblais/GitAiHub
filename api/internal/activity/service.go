package activity

import (
	"context"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Service answers Agent activity queries.
type Service struct {
	Store  Store
	Broker *Broker
}

// ListChangesets lists recorded changesets, newest first.
func (s *Service) ListChangesets(ctx context.Context, f Filter) ([]Changeset, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	return s.Store.ListChangesets(ctx, f)
}

// ListAgents lists every agent seen in the changeset log.
func (s *Service) ListAgents(ctx context.Context) ([]Agent, error) { return s.Store.ListAgents(ctx) }

// GetAgent returns one agent with its statistics.
func (s *Service) GetAgent(ctx context.Context, name string) (*Agent, error) {
	return s.Store.GetAgent(ctx, name)
}

// Landings implements source.LandingLookup.
func (s *Service) Landings(ctx context.Context, repo string, ids []string) (map[string]source.Landing, error) {
	return s.Store.Landings(ctx, repo, ids)
}
