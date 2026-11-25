package syncer

import (
	"context"
	"log"
	"luyin/config"
	"luyin/db"
	"luyin/models"
	"sync"
	"time"
)

type Engine struct {
	cfg       config.Config
	providers map[string]ProviderSyncer
	queue     chan string
	inFlight  sync.Map
	staleTTL  time.Duration
}

func NewEngine(cfg config.Config, providers map[string]ProviderSyncer) *Engine {
	return &Engine{
		cfg:       cfg,
		providers: providers,
		queue:     make(chan string, 500),
		staleTTL:  10 * time.Minute,
	}
}

func (e *Engine) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go e.worker(ctx)
	}

	go e.scheduler(ctx, 15*time.Minute)
}

func (e *Engine) Enqueue(userID string) bool {
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

func (e *Engine) EnqueueIfStale(ctx context.Context, userID string) {
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
				var uid string
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

func (e *Engine) syncUser(ctx context.Context, userID string) {
	user := &models.User{ID: userID}

	conns, err := user.GetConnections(ctx)
	if err != nil {
		log.Printf("syncUser fetch connections error: %v", err)
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

		if conn.NeedsRefresh(5 * time.Minute) {
			if err := e.refreshConnection(ctx, &conn); err != nil {
				log.Printf("token refresh failed [%s/%s]: %v", userID, conn.Provider, err)
				continue
			}
		}

		activities, nextCursor, err := providerSyncer.Sync(ctx, conn)
		if err != nil {
			log.Printf("sync failed [%s/%s]: %v", userID, conn.Provider, err)
			continue
		}

		for _, act := range activities {
			_ = user.CreateActivity(ctx, &models.Activity{
				Provider:       conn.Provider,
				ProviderItemID: act.ProviderItemID,
				Action:         act.Action,
				Title:          act.Title,
				OccurredAt:     act.OccurredAt,
			})
		}

		if err := conn.MarkSynced(ctx, nextCursor); err != nil {
			log.Printf("markSynced failed [%s/%s]: %v", userID, conn.Provider, err)
		}
	}

}
