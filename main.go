package main

import (
	"context"
	"log"
	"log/slog"
	"luyin/config"
	"luyin/db"
	"luyin/handlers"
	"luyin/syncer"

	"github.com/gofiber/fiber/v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.New()

	slog.Info("Hello, World!")

	if err := db.Connect(conf.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	defer db.Pool.Close()

	if err := db.Migrate(conf.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	engine := syncer.NewEngine(conf, map[string]syncer.ProviderSyncer{
		"trakt":   syncer.NewTraktSyncer(conf.Providers["trakt"].ClientID),
		"anilist": syncer.NewAniListSyncer(),
	})

	engine.Start(ctx, 3)

	app := fiber.New()

	middleware := handlers.RequireAuth(conf.JWTSecret)

	login := handlers.NewLoginHandler(conf.JWTSecret)
	auth := handlers.NewAuthHandler(conf.Providers, conf.RedirectURI)
	oidc, _ := handlers.NewOIDCHandler(ctx, conf.JWTSecret, conf.OIDCProviders)

	_ = oidc

	app.Post("/api/users/register", login.CreateUserWithPassword)
	app.Post("/api/users/login", login.Login)

	app.Get("/api/oidc/:provider/login", oidc.OIDCLogin)
	app.Get("/api/oidc/:provider/callback", oidc.OIDCCallback)
	app.Post("/api/oidc/signup", oidc.OIDCSignup)

	// authenticated routes
	app.Get("/api/users/me", middleware, login.Me)
	app.Get("/api/auth/:provider/url", middleware, auth.GetAuthURL)

	app.Post("/api/sync", middleware, func(c fiber.Ctx) error {
		queued := engine.Enqueue(handlers.UserID(c))
		return c.JSON(fiber.Map{"queued": queued})
	})

	app.Get("/api/auth/callback", auth.AuthCallback)

	log.Fatal(app.Listen(":3000"))
}
