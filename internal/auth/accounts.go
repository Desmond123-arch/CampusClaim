package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered in email goroutine:", r)
			}
		}()
		pkg.SendVerficationEmail(user.Email, user.FullName, verifier)
	}()

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
	result := models.DB.Where("email = ?", user.Email).First(&dbUser)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
	}
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
	// 	//make user verify account by routing and requesting for a new email
	// 	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Account not verified"})
	// }

	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "Lax"
	c.Cookie((*fiber.Cookie)(cookie))

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
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
	result := models.DB.Where("uuid = ? ", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
	}
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
	result := models.DB.Where("uuid = ? ", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
	}
	if user.IsVerified {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "User already verified"})
	}
	verifier := new(models.EmailVerification)
	models.DB.Where("user_id = ?", user.ID).First(&verifier)
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

func ChangePassword(c *fiber.Ctx) error {
	type PasswordRequest struct {
		Password string `json:"password,omitempty" gorm:"column:password;not null" validate:"required"`
	}

	password := new(PasswordRequest)
	if err := c.BodyParser(&password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Invalid request body"})
	}
	password.Password, _ = pkg.HashPassword(password.Password)
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

func RequestPasswordreset(c *fiber.Ctx) error {
	type EmailRequest struct {
		Email string `json:"email" validate:"required,email"`
	}
	req := new(EmailRequest)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}
	var user models.User
	result := models.DB.Where("email = ?", req.Email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
	}

	b := make([]byte, 8)
	rand.Read(b)
	newToken := fmt.Sprintf("%x", b)
	result = models.DB.Model(&user).Update("password_token", newToken)
	fmt.Println(result)
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "message": "Invalid or expired token"})
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered in email goroutine:", r)
			}
		}()
		pkg.SendResetEmail(user.Email, newToken)
	}()
	return c.SendStatus(fiber.StatusAccepted)
}

func ResetPassword(c *fiber.Ctx) error {
	token := c.Query("token")
	type PasswordResetRequest struct {
		Password string `json:"password" validate:"required"`
	}

	req := new(PasswordResetRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid request"})
	}
	req.Password, _ = pkg.HashPassword(req.Password)
	result := models.DB.Model(&models.User{}).
		Where("password_token = ?", token).
		Updates(map[string]interface{}{
			"password":       string(req.Password),
			"password_token": "",
		})

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "message": "Invalid or expired token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password successfully reset"})
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
	result := models.DB.Where("uuid = ?", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Incorrect User Details"})
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered in email goroutine:", r)
			}
		}()
		pkg.SendVerficationEmail(user.Email, user.FullName, verifier)
	}()
	return c.SendStatus(fiber.StatusAccepted)
}
