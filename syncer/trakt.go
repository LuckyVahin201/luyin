package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"luyin/models"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	traktPageSize = 100
	traktMaxPages = 200
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

func (s *TraktSyncer) fetchHistoryPage(ctx context.Context, conn models.Connection, page int) ([]traktHistoryItem, int, error) {
	endpoint, _ := url.Parse("https://api.trakt.tv/sync/history")

	q := endpoint.Query()
	q.Set("limit", strconv.Itoa(traktPageSize))
	q.Set("page", strconv.Itoa(page))

	if conn.LastCursor != nil && *conn.LastCursor != "" {
		q.Set("start_at", *conn.LastCursor)
	}

	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Authorization", "Bearer "+conn.AccessToken)
	req.Header.Set("trakt-api-version", "2")
	req.Header.Set("trakt-api-key", s.clientID)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("trakt API status %d: %s", resp.StatusCode, string(body))
	}

	var items []traktHistoryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, 0, err
	}

	pageCount, _ := strconv.Atoi(resp.Header.Get("X-Pagination-Page-Count"))
	return items, pageCount, nil
}

func (s *TraktSyncer) Sync(ctx context.Context, conn models.Connection) ([]NormalizedActivity, string, error) {
	var activities []NormalizedActivity
	var latestWatched time.Time

	for page := 1; ; page++ {
		if page > traktMaxPages {
			return nil, "", fmt.Errorf("trakt: more than %d pages of history", traktMaxPages)
		}

		items, pageCount, err := s.fetchHistoryPage(ctx, conn, page)
		if err != nil {
			return nil, "", err
		}

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

		last := page >= pageCount
		if pageCount == 0 {
			last = len(items) < traktPageSize
		}

		if len(items) == 0 || last {
			break
		}
	}

	var nextCursor string
	if !latestWatched.IsZero() {
		nextCursor = latestWatched.UTC().Format(time.RFC3339Nano)
	}

	return activities, nextCursor, nil
}
