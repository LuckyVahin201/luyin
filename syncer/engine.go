package syncer

import (
	"context"
	"log/slog"
	"luyin/config"
	"luyin/models"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type Engine struct {
	cfg       config.Config
	providers map[string]ProviderSyncer
	queue     chan ulid.ULID
	inFlight  sync.Map
	staleTTL  time.Duration
}

func NewEngine(cfg config.Config, providers map[string]ProviderSyncer) *Engine {
	return &Engine{
		cfg:       cfg,
		providers: providers,
		queue:     make(chan ulid.ULID, 500),
		staleTTL:  10 * time.Minute,
	}
}

func (e *Engine) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go e.worker(ctx)
	}

	go e.scheduler(ctx, 15*time.Minute)
}

func (e *Engine) Enqueue(userID ulid.ULID) bool {
	if _, alreadyQueued := e.inFlight.LoadOrStore(userID, true); alreadyQueued {
		return false
	}

	select {
	case e.queue <- userID:
		return true
	default:
		e.inFlight.Delete(userID)
		return false
	}
}

func (e *Engine) enqueueStale(ctx context.Context) {
	ids, err := models.GetStaleConnectionUserIDs(ctx, time.Now().Add(-e.staleTTL), 100)
	if err != nil {
		slog.Error("scheduler: stale user query failed", "err", err)
		return
	}

	queued := 0
	for _, id := range ids {
		if e.Enqueue(id) {
			queued++
		}
	}

	slog.Debug("scheduler tick", "stale", len(ids), "queued", queued)
}

func (e *Engine) EnqueueIfStale(ctx context.Context, userID ulid.ULID) {
	user := &models.User{ID: userID}

	stale, err := user.HasStaleConnections(ctx, time.Now().Add(-e.staleTTL))
	if err != nil {
		slog.Error("stale check failed", "user_id", userID, "err", err)
		return
	}

	if stale {
		e.Enqueue(userID)
	}
}

func (e *Engine) scheduler(ctx context.Context, interval time.Duration) {
	e.enqueueStale(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.enqueueStale(ctx)
		}
	}
}

func (e *Engine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case userID := <-e.queue:
			e.syncUser(ctx, userID)
			e.inFlight.Delete(userID)
		}
	}
}

func (e *Engine) syncUser(ctx context.Context, userID ulid.ULID) {
	user := &models.User{ID: userID}
	logger := slog.With("user_id", userID)

	conns, err := user.GetConnections(ctx)
	if err != nil {
		logger.Error("fetching connections failed", "err", err)
		return
	}

	for _, conn := range conns {
		if conn.Invalidated {
			continue
		}

		providerSyncer, ok := e.providers[conn.Provider]
		if !ok {
			continue
		}

		clog := logger.With("provider", conn.Provider)

		if conn.NeedsRefresh(5 * time.Minute) {
			if err := e.refreshConnection(ctx, &conn); err != nil {
				clog.Warn("token refresh failed", "err", err)
				continue
			}
		}

		activities, nextCursor, err := providerSyncer.Sync(ctx, conn)
		if err != nil {
			clog.Error("sync failed", "err", err)
			continue
		}

		inserted, err := user.CreateActivities(ctx, toActivities(conn.Provider, activities))
		if err != nil {
			clog.Error("saving activities failed, cursor not advanced", "err", err)
			continue
		}

		if err := conn.MarkSynced(ctx, nextCursor); err != nil {
			clog.Error("mark synced failed", "err", err)
			continue
		}

		clog.Info("sync complete", "fetched", len(activities), "inserted", inserted)
	}
}

func toActivities(provider string, in []NormalizedActivity) []models.Activity {
	out := make([]models.Activity, len(in))
	for i, a := range in {
		out[i] = models.Activity{
			Provider:       provider,
			ProviderItemID: a.ProviderItemID,
			Action:         a.Action,
			Title:          a.Title,
			OccurredAt:     a.OccurredAt,
		}
	}

	return out
}
