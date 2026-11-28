package handlers

import (
	"context"
	"errors"
	"log/slog"
	"luyin/config"
	"luyin/models"
	"luyin/utils"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/oauth2"
)

type OIDCHandler struct {
	Providers map[string]*OIDCProvider
	JWTSecret string
}

type OIDCProvider struct {
	Provider      *oidc.Provider
	Oauth2        oauth2.Config
	TokenVerifier *oidc.IDTokenVerifier
}

func NewOIDCHandler(c context.Context, jwtSecret string, oidcProviders map[string]config.OIDCProviderConfig) (*OIDCHandler, error) {
	handler := &OIDCHandler{
		JWTSecret: jwtSecret,
		Providers: make(map[string]*OIDCProvider),
	}

	for name, providerConfig := range oidcProviders {
		provider, err := NewOIDCProvider(
			c, providerConfig.Issuer, providerConfig.ClientID,
			providerConfig.ClientSecret, providerConfig.RedirectURL,
			providerConfig.Scopes,
		)
		if err != nil {
			return nil, err
		}
		slog.Debug("oidc: provider ready", "name", name, "issuer", providerConfig.Issuer)
		handler.Providers[name] = provider
	}

	return handler, nil
}

func NewOIDCProvider(c context.Context, issuer string, clientID string, clientSecret string, redirectURL string, scopes []string) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(c, issuer)
	if err != nil {
		return nil, err
	}

	oauth := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return &OIDCProvider{
		Provider:      provider,
		Oauth2:        oauth,
		TokenVerifier: verifier,
	}, nil
}

func (h *OIDCHandler) OIDCLogin(c fiber.Ctx) error {
	p, ok := h.Providers[c.Params("provider")]
	if !ok {
		return c.SendStatus(fiber.StatusNotFound)
	}

	state := utils.GenerateCodeVerifier()
	verifier := utils.GenerateCodeVerifier()

	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	c.Cookie(&fiber.Cookie{
		Name:     "oidc_verifier",
		Value:    verifier,
		Path:     "/",
		MaxAge:   300,
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	authCodeUrl := p.Oauth2.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	return c.Status(fiber.StatusFound).Redirect().To(authCodeUrl)
}

func (h *OIDCHandler) OIDCCallback(c fiber.Ctx) error {
	p, ok := h.Providers[c.Params("provider")]
	if !ok {
		return c.SendStatus(fiber.StatusNotFound)
	}

	stateCookie := c.Cookies("oidc_state")
	verifierCookie := c.Cookies("oidc_verifier")

	if stateCookie == "" || verifierCookie == "" {
		slog.Debug("missing cookies", "cookies", c.GetCookies())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing oauth cookies"})
	}

	clearOIDCCookies(c)

	if c.Query("state") != stateCookie {
		slog.Debug("oauth state mismatch",
			"cookie", stateCookie,
			"url", c.Query("state"),
		)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "bad oauth state"})
	}

	authCode := c.Query("code")
	if authCode == "" {
		slog.Debug("oauth code missing from url")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "bad oauth state"})
	}

	token, err := p.Oauth2.Exchange(c, authCode, oauth2.VerifierOption(verifierCookie))

	if err != nil {
		slog.Debug("oauth token exchange failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication failed"})
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		slog.Debug("token response did not contain an id_token", "token_type", token.TokenType)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication failed"})
	}

	idToken, err := p.TokenVerifier.Verify(c, rawIDToken)
	if err != nil {
		slog.Debug("idToken verification failed", "error", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication failed"})
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"given_name"`
		PreferredUsername string `json:"preferred_username"`
		AvatarURL         string `json:"picture"`
	}

	if err := idToken.Claims(&claims); err != nil {
		slog.Error("couldn't extract claims from token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "authentication failed"})
	}

	if identity, err := models.GetIdentityByIssuerAndSubject(c, idToken.Issuer, idToken.Subject); err == nil {
		updated, err := models.UpdateIdentityEmail(c, identity.ID, claims.Email)
		if err != nil {
			slog.Error("failed to update identity email", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "something broke while updating your email"})
		}

		return c.Status(fiber.StatusOK).JSON(updated)
	}


	id, err := models.UpsertPendingIdentity(c, &claims.PreferredUsername, c.Params("provider"), idToken.Issuer, idToken.Subject, claims.Email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "signup already in progress"})
		} // this should no longer run but who knows?

		slog.Error("failed to create a pending identity", "claims", claims, "provider", c.Params("provider"), "subject", idToken.Subject, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "welp something broke"})
	}

	return c.Status(fiber.StatusCreated).JSON(id)
}

func (h *OIDCHandler) OIDCSignup(c fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Token    string `json:"token"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	exists, err := models.DoesUserExistEmail(c, req.Email)

	if err != nil {
		slog.Error("something broke while checking if a user exists", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	if exists {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "this user already exists, go to the user page to approve this as a sign in method"})
	}

	pending, err := models.GetPendingIdentityByToken(c, req.Token)

	if errors.Is(err, pgx.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pending identity not found or expired"})
	} else if err != nil {
		slog.Error("failed to get pending identity", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	user, err := models.CreateUserOauth(c, req.Email, req.Username, pending)

	token, err := utils.GenerateJWT(user.ID, h.JWTSecret)
	if err != nil {
		slog.Error("failed to generate jwt token", "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})

	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user":  user,
		"token": token,
	})
}

func clearOIDCCookies(c fiber.Ctx) {
	for _, name := range []string{"oidc_state", "oidc_verifier"} {
		c.ClearCookie(name)
	}
}
