package syncer

import (
	"context"
	"time"

	"luyin/models"
)

type NormalizedActivity struct {
	ProviderItemID string
	Action         string
	Title          string
	OccurredAt     time.Time
}

type ProviderSyncer interface {
	Sync(ctx context.Context, conn models.Connection) (activities []NormalizedActivity, nextCursor string, err error)
}
