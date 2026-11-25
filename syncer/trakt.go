package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"luyin/models"
	"net/http"
	"net/url"
	"time"
)

type TraktSyncer struct {
	clientID string
	client   *http.Client
}

func NewTraktSyncer(clientID string) *TraktSyncer {
	return &TraktSyncer{
		clientID: clientID,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

type traktHistoryItem struct {
	ID        int64  `json:"id"`
	WatchedAt string `json:"watched_at"`
	Action    string `json:"action"`
	Type      string `json:"type"` // movie/episode

	Movie *struct {
		Title string `json:"title"`
		IDs   struct {
			Trakt int `json:"trakt"`
		} `json:"ids"`
	} `json:"movie"`

	Episode *struct {
		Season int    `json:"season"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		IDs    struct {
			Trakt int `json:"trakt"`
		} `json:"ids"`
	} `json:"episode"`

	Show *struct {
		Title string `json:"title"`
		IDs   struct {
			Trakt int `json:"trakt"`
		} `json:"ids"`
	} `json:"show"`
}

func (s *TraktSyncer) Sync(ctx context.Context, conn models.Connection) ([]NormalizedActivity, string, error) {
	endpoint, _ := url.Parse("https://api.trakt.tv/sync/history")

	q := endpoint.Query()
	q.Set("limit", "100")

	if conn.LastCursor != nil && *conn.LastCursor != "" {
		q.Set("start_at", *conn.LastCursor)
	}

	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+conn.AccessToken)
	req.Header.Set("trakt-api-version", "2")
	req.Header.Set("trakt-api-key", s.clientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("trakt API status %d: %s", resp.StatusCode, string(body))
	}

	var items []traktHistoryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, "", err
	}

	var activities []NormalizedActivity
	var latestWatched time.Time

	for _, item := range items {
		watchedAt, err := time.Parse(time.RFC3339Nano, item.WatchedAt)
		if err != nil {
			continue
		}

		if watchedAt.After(latestWatched) {
			latestWatched = watchedAt
		}

		var itemID, title string
		switch item.Type {
		case "movie":
			if item.Movie == nil {
				continue
			}

			itemID = fmt.Sprintf("movie:%d", item.Movie.IDs.Trakt)
			title = item.Movie.Title
		case "episode":
			if item.Episode == nil || item.Show == nil {
				continue
			}

			itemID = fmt.Sprintf("episode:%d", item.Episode.IDs.Trakt)
			title = fmt.Sprintf("%s S%02dE%02d", item.Show.Title, item.Episode.Season, item.Episode.Number)
		default:
			continue
		}

		activities = append(activities, NormalizedActivity{
			ProviderItemID: itemID,
			Action:         "watched",
			Title:          title,
			OccurredAt:     watchedAt.UTC(),
		})
	}

	var nextCursor string
	if !latestWatched.IsZero() {
		nextCursor = latestWatched.UTC().Format(time.RFC3339Nano)
	}

	return activities, nextCursor, nil
}
