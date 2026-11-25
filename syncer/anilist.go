package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"luyin/models"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AniListSyncer struct {
	client *http.Client
}

func NewAniListSyncer() *AniListSyncer {
	return &AniListSyncer{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

func (s *AniListSyncer) postGraphQL(ctx context.Context, accessToken string, payload gqlRequest, out any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anilist API status %d: %s", resp.StatusCode, string(raw))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *AniListSyncer) fetchViewerID(ctx context.Context, accessToken string) (int, error) {
	var res struct {
		Data struct {
			Viewer struct {
				ID int `json:"id"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	err := s.postGraphQL(ctx, accessToken, gqlRequest{
		Query: `query { Viewer { id } }`,
	}, &res)

	if err != nil {
		return 0, err
	}

	if res.Data.Viewer.ID == 0 {
		return 0, fmt.Errorf("anilist viewer ID returned 0")
	}

	return res.Data.Viewer.ID, nil
}

func (s *AniListSyncer) Sync(ctx context.Context, conn models.Connection) ([]NormalizedActivity, string, error) {
	viewerID, err := s.fetchViewerID(ctx, conn.AccessToken)
	if err != nil {
		return nil, "", err
	}

	var lastUpdated int64
	if conn.LastCursor != nil && *conn.LastCursor != "" {
		lastUpdated, _ = strconv.ParseInt(*conn.LastCursor, 10, 64)
	}

	query := `
		query ($userId: Int) {
			Page(page: 1, perPage: 50) {
				mediaList(userId: $userId, sort: UPDATED_TIME_DESC) {
					status
					progress
					updatedAt
					media {
						id
						title {
							english
							romaji
						}
					}
				}
			}
		}
	`

	var res struct {
		Data struct {
			Page struct {
				MediaList []struct {
					Status    string `json:"status"`
					Progress  int    `json:"progress"`
					UpdatedAt int64  `json:"updatedAt"`
					Media     struct {
						ID    int `json:"id"`
						Title struct {
							English string `json:"english"`
							Romaji  string `json:"romaji"`
						} `json:"title"`
					} `json:"media"`
				} `json:"mediaList"`
			} `json:"Page"`
		} `json:"data"`
	}

	err = s.postGraphQL(ctx, conn.AccessToken, gqlRequest{
		Query:     query,
		Variables: map[string]any{"userId": viewerID},
	}, &res)

	if err != nil {
		return nil, "", err
	}

	var activities []NormalizedActivity
	maxUpdated := lastUpdated

	for _, entry := range res.Data.Page.MediaList {
		if lastUpdated > 0 && entry.UpdatedAt <= lastUpdated {
			break
		}

		if entry.UpdatedAt > maxUpdated {
			maxUpdated = entry.UpdatedAt
		}

		title := entry.Media.Title.English
		if title == "" {
			title = entry.Media.Title.Romaji
		}

		action := strings.ToLower(entry.Status)
		switch entry.Status {
		case "CURRENT", "REPEATING":
			action = "watched"
			if entry.Progress > 0 {
				title = fmt.Sprintf("%s (Ep %d)", title, entry.Progress)
			}
		case "COMPLETED":
			action = "completed"
		case "PLANNING":
			action = "planned"
		}

		activities = append(activities, NormalizedActivity{
			ProviderItemID: strconv.Itoa(entry.Media.ID),
			Action:         action,
			Title:          title,
			OccurredAt:     time.Unix(entry.UpdatedAt, 0).UTC(),
		})
	}

	var nextCursor string
	if maxUpdated > 0 {
		nextCursor = strconv.FormatInt(maxUpdated, 10)
	}

	return activities, nextCursor, nil
}
