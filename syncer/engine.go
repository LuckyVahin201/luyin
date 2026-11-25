package syncer

import (
	"context"
	"log"
	"log/slog"
	"luyin/config"
	"luyin/db"
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

func (e *Engine) EnqueueIfStale(ctx context.Context, userID ulid.ULID) {
	var needsSync bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM connections
			WHERE user_id = $1
			  AND (last_synced_at IS NULL OR last_synced_at < $2)
		)
	`, userID, time.Now().Add(-e.staleTTL)).Scan(&needsSync)

	if err == nil && needsSync {
		e.Enqueue(userID)
	}
}

func (e *Engine) scheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := db.Pool.Query(ctx, `
				SELECT DISTINCT user_id FROM connections
				WHERE last_synced_at IS NULL OR last_synced_at < $1
				LIMIT 100
			`, time.Now().Add(-e.staleTTL))

			if err != nil {
				log.Printf("scheduler query error: %v", err)
				continue
			}

			for rows.Next() {
				var uid ulid.ULID
				if err := rows.Scan(&uid); err == nil {
					e.Enqueue(uid)
				}
			}

			rows.Close()
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
