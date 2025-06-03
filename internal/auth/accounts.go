package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
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
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "Lax"

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        user,
		"accessToken": accessToken,
	})
}

func LoginUser(c *fiber.Ctx) error {
	user := new(models.LoginDetails)
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid data"})
	}
	errs := pkg.LoginValidator().Validate(user)

	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	var dbUser models.User
	models.DB.Where("email = ?", user.Email).First(&dbUser)
	isValid := pkg.VerifyHash(user.Password, dbUser.Password)
	if !isValid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
	}

	accessToken, err := CreateAccessToken(dbUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	refreshToken, err := CreateRefreshToken(dbUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}

	// if !dbUser.IsVerified {
	// 	fmt.Println("Redirecting")
	// 	return c.Redirect("/auth/verify-account", fiber.StatusTemporaryRedirect)
	// }

	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "Lax"
	c.Cookie((*fiber.Cookie)(cookie))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        &dbUser,
		"accessToken": accessToken,
	})
}

func GetNewRefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("RefreshToken")
	if refreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid request body"})
	}
	token, err := VerifyToken(refreshToken)
	fmt.Println(err)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}
	userid, _ := token.Claims.(jwt.MapClaims).GetSubject()
	var user models.User
	models.DB.Where("uuid = ? ", userid).First(&user)

	if user.RefreshToken != refreshToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}
	newRefreshToken, err := CreateRefreshToken(userid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})

	}
	newAccessToken, err := CreateAccessToken(userid)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})

	}

	models.DB.Where("uuid = ? ", userid).Update("refresh_token", newRefreshToken)
	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = newRefreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "Lax"
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Success", "accessToken": newAccessToken})
}

func VerifyAccount(c *fiber.Ctx) error {

	return c.SendStatus(fiber.StatusAccepted)
}
