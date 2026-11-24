package main

import (
	"log"
	"luyin/config"
	"luyin/db"
	"luyin/handlers"

	"github.com/gofiber/fiber/v3"
)

func main() {
	conf := config.New()
	if err := db.Connect(conf.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	defer db.Pool.Close()

	if err := db.Migrate(conf.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()
	auth := handlers.NewAuthHandler(conf.Providers, conf.RedirectURI)

	app.Get("/api/auth/:provider/url", auth.GetAuthURL)
	app.Get("/api/auth/callback", auth.AuthCallback)

	log.Fatal(app.Listen(":3000"))
}
