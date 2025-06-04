package auth

import (
	"errors"
	"strings"
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
	verifier := new(models.EmailVerification)
	verifier.Code, _ = pkg.GenerateOTP()
	verifier.ExpiresAt = time.Now().Add(30 * time.Second)
	verifier.UserID = user.ID

	result = models.DB.Where("user_id = ?", user.ID).Assign(verifier).FirstOrCreate(&verifier)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": "User already exists"})
		} else {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": result.Error})
		}
	}

	//FIXME: SEND AN EMAIL HERE 
	

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

	if !dbUser.IsVerified {
		return c.Redirect("/auth/verify-account", fiber.StatusTemporaryRedirect)
	}

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
	type OTPRequest struct {
		Code string `json:"code" validate:"required,len=6"`
	}
	otprequest := new(OTPRequest)
	if err := c.BodyParser(&otprequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Invalid request body"})
	}

	errs := pkg.GeneralValidator().Validate(otprequest)
	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	token := c.GetReqHeaders()["Authorization"][0]
	token = strings.ReplaceAll(token, "Bearer ", "")
	verfiedtoken, err := VerifyToken(token)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}

	userid, _ := verfiedtoken.Claims.(jwt.MapClaims).GetSubject()
	var user models.User
	models.DB.Where("uuid = ? ", userid).First(&user)
	if user.IsVerified {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "User already verified"})
	}
	verifier := new(models.EmailVerification)
	err = models.DB.Where("user_id = ?", user.ID).First(&verifier).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "User not found"})

	}
	if time.Now().After(verifier.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Token has expired",
		})
	}

	if verifier.Code != otprequest.Code {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}

	models.DB.Model(&user).Update("is_verified", true)
	models.DB.Delete(&verifier)
	return c.SendStatus(fiber.StatusAccepted)
}

func ResetPassword(c *fiber.Ctx) error {
	type PasswordRequest struct {
		Password string `json:"password,omitempty" gorm:"column:password;not null" validate:"required"`
	}

	password := new(PasswordRequest)
	if err := c.BodyParser(&password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Invalid request body"})
	}
	errs := pkg.GeneralValidator().Validate(password)

	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	token := c.GetReqHeaders()["Authorization"][0]
	token = strings.ReplaceAll(token, "Bearer ", "")
	verfiedtoken, err := VerifyToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}
	userid, _ := verfiedtoken.Claims.(jwt.MapClaims).GetSubject()
	models.DB.Where("uuid = ? ", userid).Update("password", password.Password)

	return c.SendStatus(fiber.StatusAccepted)
}

func GetNewVerficationCode(c *fiber.Ctx) error {
	token := c.GetReqHeaders()["Authorization"][0]
	token = strings.ReplaceAll(token, "Bearer ", "")
	verfiedtoken, err := VerifyToken(token)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}

	userid, _ := verfiedtoken.Claims.(jwt.MapClaims).GetSubject()
	var user models.User
	models.DB.Where("uuid = ?", userid).First(&user)
	verifier := new(models.EmailVerification)
	verifier.Code, _ = pkg.GenerateOTP()
	verifier.ExpiresAt = time.Now().Add(30 * time.Second)
	verifier.UserID = user.ID

	result := models.DB.Where("user_id = ?", user.ID).Assign(verifier).FirstOrCreate(&verifier)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": "User already exists"})
		} else {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"status": "Failed", "errors": result.Error})
		}
	}
	//FIXME: SEND AN EMAIL HERE 
	return c.SendStatus(fiber.StatusAccepted)
}
