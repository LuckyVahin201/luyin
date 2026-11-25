package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"luyin/models"
	"net/http"
	"time"
)

func (e *Engine) refreshConnection(ctx context.Context, conn *models.Connection) error {
	providerCfg, exists := e.cfg.Providers[conn.Provider]
	if !exists {
		return fmt.Errorf("missing provider config for %s", conn.Provider)
	}

	if conn.RefreshToken == nil || *conn.RefreshToken == "" {
		_ = conn.MarkInvalidated(ctx)
		return fmt.Errorf("%w: no refresh token stored", ErrRefreshRejected)
	}

	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": *conn.RefreshToken,
		"client_id":     providerCfg.ClientID,
		"redirect_uri":  e.cfg.RedirectURI,
	}
	if !providerCfg.UsePKCE {
		payload["client_secret"] = providerCfg.ClientSecret
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, providerCfg.TokenURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		_ = conn.MarkInvalidated(ctx)

		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w (%d): %s", ErrRefreshRejected, resp.StatusCode, string(raw))
	}

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh transient error (%d): %s", resp.StatusCode, string(raw))
	}

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return err
	}

	var newRefresh *string
	if tokenData.RefreshToken != "" {
		newRefresh = &tokenData.RefreshToken
	}

	var newExpiresAt *int64
	if tokenData.ExpiresIn > 0 {
		exp := time.Now().Unix() + tokenData.ExpiresIn
		newExpiresAt = &exp
	}

	return conn.UpdateTokens(ctx, tokenData.AccessToken, newRefresh, newExpiresAt)
}
