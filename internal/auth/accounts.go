package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"

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
	verifier.ExpiresAt = time.Now().Add(3 * time.Minute)
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
	cookie.SameSite = "None"

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        user,
		"accessToken": accessToken,
	})
}

func LoginWithSchoolCred(c *fiber.Ctx) error {
	type SchoolCred struct {
		ReferenceNumber string `json:"username" validate:"required"`
		Password        string `json:"password" validate:"required"`
	}
	var user SchoolCred

	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid request body"})
	}

	if user.ReferenceNumber == "" && user.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid details"})
	}

	details, err := pkg.GetSchoolDetails(user.ReferenceNumber, user.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid Login Details"})
	}

	// fmt.Println(details["name"], details["email"])
	schoolUser := models.User{
		FullName:   details["name"],
		IsVerified: true,
		Email:      details["email"],
	}
	result := models.DB.Where(models.User{Email: details["email"]}).FirstOrCreate(&schoolUser)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": result.Error})
	}

	accessToken, err := CreateAccessToken(schoolUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	refreshToken, err := CreateRefreshToken(schoolUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "None"

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        &schoolUser,
		"accessToken": accessToken,
	})
}

func LoginWithGoogle(c *fiber.Ctx) error {
	type Token struct {
		IdToken string `json:"token"`
	}
	var token Token
	client_token := os.Getenv("GOOGLE_CLIENT_ID")
	if err := c.BodyParser(&token); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid data"})
	}
	if token.IdToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid request details"})
	}

	fmt.Println(token)
	payload, err := idtoken.Validate(c.Context(), token.IdToken, client_token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Invalid Google Tokens"})
	}
	email := payload.Claims["email"].(string)
	full_name := payload.Claims["name"].(string)
	picture := payload.Claims["picture"].(string)

	fmt.Println(email, full_name, picture)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@st\.umat\.edu\.gh$`, email)
	if  !matched {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": "Email must be a school Email"})
	}

	schoolUser := models.User{
		FullName:   full_name,
		IsVerified: true,
		Email:      email,
		ImageURL:   picture,
	}
	result := models.DB.Where(models.User{Email: email}).FirstOrCreate(&schoolUser)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": result.Error})
	}

	accessToken, err := CreateAccessToken(schoolUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	refreshToken, err := CreateRefreshToken(schoolUser.UUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "Failed", "errors": "An unexpected error occured"})
	}
	cookie := new(fiber.Cookie)
	cookie.Name = "RefreshToken"
	cookie.Value = refreshToken
	cookie.Expires = time.Now().Add(24 * time.Hour * 72)
	cookie.HTTPOnly = true
	cookie.Secure = true
	cookie.SameSite = "None"

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "success",
		"user":        &schoolUser,
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
	cookie.SameSite = "None"
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
	cookie.SameSite = "None"
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Success", "accessToken": newAccessToken})
}

func VerifyAccount(c *fiber.Ctx) error {
	type OTPRequest struct {
		Code string `json:"code" validate:"required,len=4"`
	}
	otprequest := new(OTPRequest)
	if err := c.BodyParser(&otprequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"errors": "Invalid request body", "success": "false"})
	}

	errs := pkg.GeneralValidator().Validate(otprequest)
	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	// token := c.GetReqHeaders()["Authorization"][0]
	// token = strings.ReplaceAll(token, "Bearer ", "")
	// verfiedtoken, err := VerifyToken(token)

	// if err != nil {
	// 	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	// }

	userid := c.Locals("userID").(string)
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
	fmt.Println(verifier.Code, otprequest.Code, user.ID)
	if time.Now().After(verifier.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Token has expired",
		})
	}

	if verifier.Code != otprequest.Code {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid token"})
	}

	models.DB.Model(&user).Update("is_verified", true)
	models.DB.Delete(&verifier)
	return c.SendStatus(fiber.StatusAccepted)
}

func ChangePassword(c *fiber.Ctx) error {
	type PasswordRequest struct {
		OldPassword string `json:"old_password" gorm:"column:old_password;not null" validate:"required"`
		Password    string `json:"password" gorm:"column:password;not null" validate:"required"`
	}
	var user models.User
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

	models.DB.Where("uuid = ? ", userid).First(&user)

	if !pkg.VerifyHash(password.OldPassword, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "errors": "Invalid credentials"})
	}
	hashedPassword, err := pkg.HashPassword(password.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Could not hash password",
		})
	}

	if err := models.DB.Model(&user).Update("password", hashedPassword).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Could not update password",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "success"})
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
	// fmt.Println(resul)
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "message": "Invalid or expired token"})
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered in email goroutine:", r)
			}
		}()
		pkg.SendResetEmail(user.Email, newToken)
	}()
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "success"})
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "Failed", "message": "Invalid or expired token"})
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
