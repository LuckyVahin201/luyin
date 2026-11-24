package handlers

import (
	"bytes"
	ctx "context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"luyin/config"
	"luyin/db"
	"luyin/utils"

	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	RedirectURI string
	Providers   map[string]config.ProviderConfig
	Store       sync.Map
}

type AuthSession struct {
	ProviderName string
	Verifier     string
}

func NewAuthHandler(providers map[string]config.ProviderConfig, redirectURI string) *AuthHandler {
	return &AuthHandler{
		RedirectURI: redirectURI,
		Providers:   providers,
	}
}

func (h *AuthHandler) GetAuthURL(c fiber.Ctx) error {
	providerName := c.Params("provider")
	provider, exists := h.Providers[providerName]
	if !exists {
		return c.Status(400).SendString("Unknown provider")
	}

	state, _ := utils.GenerateCodeVerifier()

	sessionData := AuthSession{ProviderName: providerName}

	url := fmt.Sprintf(
		"%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		provider.AuthURL, provider.ClientID, h.RedirectURI, state,
	)

	if provider.UsePKCE {
		verifier, _ := utils.GenerateCodeVerifier()
		challenge := utils.GenerateCodeChallenge(verifier)

		sessionData.Verifier = verifier
		url += fmt.Sprintf("&code_challenge=%s&code_challenge_method=S256", challenge)
	}

	h.Store.Store(state, sessionData)

	return c.JSON(fiber.Map{"url": url})
}

func (h *AuthHandler) AuthCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")

	sessionDataIfc, exists := h.Store.LoadAndDelete(state)
	if !exists {
		return c.Status(400).SendString("Invalid or expired state")
	}

	sessionData := sessionDataIfc.(AuthSession)
	providerName := sessionData.ProviderName

	provider, exists := h.Providers[providerName]
	if !exists {
		return c.Status(500).SendString("Provider config missing")
	}

	payload := map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": h.RedirectURI,
		"client_id":    provider.ClientID,
	}

	if provider.UsePKCE {
		payload["code_verifier"] = sessionData.Verifier
	} else {
		payload["client_secret"] = provider.ClientSecret
	}

	return h.exchangeAndSave(c, provider.TokenURL, payload, providerName)
}

func (h *AuthHandler) exchangeAndSave(c fiber.Ctx, tokenURL string, payload map[string]string, providerName string) error {
	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(tokenURL, "application/json", bytes.NewBuffer(jsonPayload))

	if err != nil || resp.StatusCode != 200 {
		return c.Status(500).SendString("OAuth failure with provider")
	}

	defer resp.Body.Close()

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	json.NewDecoder(resp.Body).Decode(&tokenData)

	expiresAt := time.Now().Unix() + tokenData.ExpiresIn
	if tokenData.ExpiresIn == 0 {
		expiresAt = time.Now().Unix() + 31536000
	}

	// TODO: remove when users are impl'd
	testUserID := "11111111-1111-1111-1111-111111111111"
	_, err = db.Pool.Exec(
		ctx.Background(),
		`
		INSERT INTO linked_accounts (user_id, provider, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, provider) DO UPDATE SET 
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at;
		`,
		testUserID, providerName, tokenData.AccessToken, tokenData.RefreshToken, expiresAt,
	)

	if err != nil {
		return c.Status(500).SendString("Failed to save tokens")
	}

	return c.SendString(fmt.Sprintf("%s successfully linked!", providerName))
}
