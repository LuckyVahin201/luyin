package syncer

import (
	"context"
	"errors"
	"time"

	"luyin/models"
)

var ErrUnauthorized = errors.New("provider unauthorized or token revoked")
var ErrRefreshRejected = errors.New("refresh token rejected by provider")

type NormalizedActivity struct {
	ProviderItemID string
	Action         string
	Title          string
	OccurredAt     time.Time
}

type ProviderSyncer interface {
	Sync(ctx context.Context, conn models.Connection) (activities []NormalizedActivity, nextCursor string, err error)
}
