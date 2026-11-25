package handlers

import (
	"luyin/models"
	"luyin/utils"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/oklog/ulid/v2"
)

var userKey = "userID"

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid authorization header"})
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		userID, err := utils.ParseJWT(tokenStr, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		ulidID, err := ulid.Parse(userID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or malformed user id"})
		}

		c.Locals(userKey, ulidID)

		return c.Next()

	}
}

func UserID(c fiber.Ctx) ulid.ULID {
	id, _ := c.Locals(userKey).(ulid.ULID)
	return id
}

func UserFromContext(c fiber.Ctx) (*models.User, error) {
	id, _ := c.Locals(userKey).(ulid.ULID)
	user, err := models.GetUser(c, id)

	return user, err
}
