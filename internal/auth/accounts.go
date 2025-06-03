package auth

import (
	"errors"

	"fmt"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterUser(c *fiber.Ctx) error {
	user := new(models.User)
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Bad request"})
	}

	errs := pkg.RegistrationValidatator().Validate(user)

	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	user.Password, _ = pkg.HashPassword(user.Password)
	result := models.DB.Create(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": "User already exists"})
		} else {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": result.Error})
		}
	}
	accessToken, err := CreateAccessToken(user.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	refreshToken, err := CreateRefreshToken(user.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}

	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	c.Cookie((*fiber.Cookie)(cookie))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        user,
		"accessToken": accessToken,
	})
}


func LoginUser(c *fiber.Ctx) error {
	user := new(models.LoginDetails)
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Bad request"})
	}
	errs := pkg.LoginValidator().Validate(user)

	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	var dbUser models.User
	models.DB.Where("email = ?", user.Email).First(&dbUser)
	fmt.Println(dbUser)
	return nil
}