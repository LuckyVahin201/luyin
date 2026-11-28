package handlers

import (
	"errors"
	"log/slog"
	"luyin/models"
	"luyin/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgconn"
)

type LoginHandler struct {
	JWTSecret string
}

func NewLoginHandler(jwtSecret string) *LoginHandler {
	return &LoginHandler{JWTSecret: jwtSecret}
}

func (h LoginHandler) CreateUserWithPassword(c fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"Password"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, err := models.CreateUserWithPassword(c.Context(), req.Email, req.Username, utils.HashPassword(req.Password))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user already exists"})
		}

		return c.SendStatus(fiber.StatusInternalServerError)
	}

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

func (h LoginHandler) Login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"Password"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, err := models.GetUserByEmail(c, req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})

	}

	if !utils.CompareHashes(*user.PasswordHash, req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	token, err := utils.GenerateJWT(user.ID, h.JWTSecret)
	if err != nil {
		slog.Debug("")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":  user,
		"token": token,
	})
}

func (h LoginHandler) Me(c fiber.Ctx) error {
	user, err := UserFromContext(c)

	if err != nil {
		slog.Error("couldn't fetch user from context: ", "error", err, "userid", UserID(c))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "something broke internally"})
	}

	return c.Status(fiber.StatusOK).JSON(user)
}
