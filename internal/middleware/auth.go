package middleware

import (
	"fmt"
	"github.com/Desmond123-arch/CampusClaim/internal/auth"
	"github.com/gofiber/fiber/v2"
)

func AuthenticateMiddleware(c *fiber.Ctx) error {
	tokenString := c.Cookies("access-token")
	if tokenString != "" {
		c.Redirect("/login", fiber.StatusSeeOther)
		return fmt.Errorf("token is Required")
	}
	_, err := auth.VerifyToken(tokenString)
	if err != nil {
		c.Redirect("/login", fiber.StatusSeeOther)
		return err
	}
	c.Next()
	return nil
}


