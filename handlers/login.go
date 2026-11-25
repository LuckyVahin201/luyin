package handlers

import "github.com/gofiber/fiber/v3"

type LoginHandler struct {

}

func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

func (h LoginHandler) CreateUserWithPassword(c fiber.Ctx, email string, password string) {
	
}