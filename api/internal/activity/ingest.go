package activity

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// Metrics receives ingestion observations (Prometheus in production).
type Metrics interface {
	ChangesetIngested(repo string, lag time.Duration)
	WatchError(repo string)
}

type noMetrics struct{}

func (noMetrics) ChangesetIngested(string, time.Duration) {}
func (noMetrics) WatchError(string)                       {}

// Ingestor follows the changeset log of every repository and copies it into
// the store. Each repository has one watcher that resumes from its saved
// cursor, so a restart neither loses nor repeats changesets (freshness SLO).
type Ingestor struct {
	Store    Store
	Upstream Upstream
	Broker   *Broker
	Metrics  Metrics
	Log      *slog.Logger
	// MaxCommitsPerChangeset bounds the commits listed for one changeset.
	MaxCommitsPerChangeset int

	mu       sync.Mutex
	watching map[string]context.CancelFunc
	wg       sync.WaitGroup
}

// Ensure starts a watcher for each repository (name -> relative path) not
// yet watched. It is safe to call repeatedly as repositories are discovered.
func (in *Ingestor) Ensure(ctx context.Context, repos map[string]string) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.watching == nil {
		in.watching = map[string]context.CancelFunc{}
	}
	for name, rel := range repos {
		if _, ok := in.watching[name]; ok {
			continue
		}
		wctx, cancel := context.WithCancel(ctx)
		in.watching[name] = cancel
		in.wg.Add(1)
		go func(name, rel string) {
			defer in.wg.Done()
			in.watch(wctx, name, rel)
		}(name, rel)
	}
}

// Wait blocks until every watcher has stopped (after ctx is cancelled).
func (in *Ingestor) Wait() { in.wg.Wait() }

func (in *Ingestor) metrics() Metrics {
	if in.Metrics == nil {
		return noMetrics{}
	}
	return in.Metrics
}

func (in *Ingestor) logger() *slog.Logger {
	if in.Log == nil {
		return slog.Default()
	}
	return in.Log
}

func (in *Ingestor) watch(ctx context.Context, name, rel string) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := in.watchOnce(ctx, name, rel)
		if ctx.Err() != nil {
			return
		}
		in.metrics().WatchError(name)
		in.logger().Warn("changeset watch ended, retrying", "repository", name, "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (in *Ingestor) watchOnce(ctx context.Context, name, rel string) error {
	after, err := in.Store.Cursor(ctx, name)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	changes, errs := in.Upstream.WatchChangesets(ctx, rel, after)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cs, ok := <-changes:
			if !ok {
				if errs != nil {
					if err, ok := <-errs; ok && err != nil {
						return err
					}
				}
				return errors.New("changeset stream ended")
			}
			if err := in.Ingest(ctx, name, rel, cs); err != nil {
				return err
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil // closed: keep reading changes until they close too
				continue
			}
			if err != nil {
				return err
			}
		}
	}
}

// Ingest records one changeset: lists the commits it landed, attributes them
// to agents, saves it and publishes it to the live feed.
func (in *Ingestor) Ingest(ctx context.Context, name, rel string, cs *Changeset) error {
	cs.Repository = name
	if !IsZeroOID(cs.NewOID) {
		limit := in.MaxCommitsPerChangeset
		if limit <= 0 {
			limit = 200
		}
		commits, err := in.Upstream.LandedCommits(ctx, rel, cs.Reference, cs.OldOID, cs.NewOID, limit)
		if err != nil {
			in.logger().Warn("listing landed commits failed", "repository", name, "sequence", cs.Sequence, "error", err)
		}
		Attribute(cs, commits)
	} else {
		Attribute(cs, nil)
	}
	if err := in.Store.SaveChangeset(ctx, cs); err != nil {
		return err
	}
	in.metrics().ChangesetIngested(name, time.Since(cs.RecordTime))
	if in.Broker != nil {
		in.Broker.Publish(*cs)
	}
	return nil
}
