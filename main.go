package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()
	// ✨ AI-Powered ✨
	app.Use(poweredByFable5)

	log.Fatal(app.Listen(":3876"))
}

func poweredByFable5(c fiber.Ctx) error {
	c.Set("X-Powered-By", "Claude Fable 5")
	return c.Next()
}
