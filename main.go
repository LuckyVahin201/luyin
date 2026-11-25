package main

import (
	"context"
	"log"
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
	auth := handlers.NewAuthHandler(conf.Providers, conf.RedirectURI)

	app.Get("/api/auth/:provider/url", auth.GetAuthURL)
	app.Get("/api/auth/callback", auth.AuthCallback)

	app.Post("/api/sync", func(c fiber.Ctx) error {
		queued := engine.Enqueue("11111111-1111-1111-1111-111111111111")

		return c.JSON(fiber.Map{"queued": queued})
	})

	log.Fatal(app.Listen(":3000"))
}
