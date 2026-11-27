// Command gitaihub-api serves GitaiHub's HTTP API in front of one gitai
// server. All configuration comes from environment variables (see
// internal/config and the README).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierreblais/gitaihub/api/internal/activity"
	"github.com/pierreblais/gitaihub/api/internal/config"
	"github.com/pierreblais/gitaihub/api/internal/gitai"
	"github.com/pierreblais/gitaihub/api/internal/httpapi"
	"github.com/pierreblais/gitaihub/api/internal/identity"
	"github.com/pierreblais/gitaihub/api/internal/observability"
	"github.com/pierreblais/gitaihub/api/internal/postgres"
	"github.com/pierreblais/gitaihub/api/internal/review"
	"github.com/pierreblais/gitaihub/api/internal/source"
)

func main() {
	if err := run(); err != nil {
		slog.Error("gitaihub-api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv(os.Getenv)
	if err != nil {
		return err
	}
	level := slog.LevelInfo
	_ = level.UnmarshalText([]byte(strings.ToUpper(cfg.LogLevel)))
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := connectWithRetry(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	gc, err := gitai.Dial(ctx, cfg.GitaiAddress, cfg.GitaiToken, cfg.GitaiStorage)
	if err != nil {
		return err
	}
	defer gc.Close()

	broker := activity.NewBroker()
	metrics := observability.New(func() float64 { return float64(broker.Subscribers()) })
	activityStore := &postgres.ActivityStore{DB: db}
	catalog := &postgres.Catalog{DB: db, Stats: activityStore}
	identityStore := &postgres.IdentityStore{DB: db}

	activitySvc := &activity.Service{Store: activityStore, Broker: broker}
	sourceSvc := &source.Service{Catalog: catalog, Gateway: gc, Landings: activitySvc}
	reviewSvc := &review.Service{Catalog: catalog, Gateway: gc, Tokens: cfg.UserGitaiTokens}
	identitySvc := &identity.Service{Store: identityStore, TTL: cfg.SessionTTL}

	if cfg.AdminUsername != "" {
		if _, created, err := identitySvc.EnsureUser(ctx, cfg.AdminUsername, cfg.AdminPassword, ""); err != nil {
			return err
		} else if created {
			slog.Info("created first user", "username", cfg.AdminUsername)
		}
	}

	ingestor := &activity.Ingestor{Store: activityStore, Upstream: gc, Broker: broker, Metrics: metrics, Log: slog.Default()}
	go discover(ctx, cfg, gc, catalog, ingestor)

	srv := &httpapi.Server{
		Source: sourceSvc, Reviews: reviewSvc, Activity: activitySvc, Identity: identitySvc,
		Metrics: metrics, CookieSecure: cfg.CookieSecure,
		Ready: func(ctx context.Context) error {
			if err := db.Ping(ctx); err != nil {
				return err
			}
			return gc.Ping(ctx)
		},
	}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr, "gitai", cfg.GitaiAddress)
		errc <- httpServer.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	ingestor.Wait()
	return nil
}

func connectWithRetry(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		db, err := postgres.Open(ctx, url)
		if err == nil {
			return db, nil
		}
		lastErr = err
		slog.Warn("database not ready, retrying", "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, lastErr
}

// discover registers the repositories of the gitai storage (or the
// configured list) and makes sure each one's changeset log is followed.
func discover(ctx context.Context, cfg config.Config, gc *gitai.Client, catalog *postgres.Catalog, ingestor *activity.Ingestor) {
	tick := time.NewTicker(cfg.DiscoveryInterval)
	defer tick.Stop()
	for {
		paths := cfg.Repositories
		if len(paths) == 0 {
			found, err := gc.Discover(ctx)
			if err != nil {
				slog.Warn("repository discovery failed; set GITAI_REPOSITORIES to list them", "error", err)
			}
			paths = found
		}
		if _, err := catalog.Upsert(ctx, paths); err != nil {
			slog.Warn("registering repositories failed", "error", err)
		}
		if all, err := catalog.All(ctx); err == nil {
			ingestor.Ensure(ctx, all)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
