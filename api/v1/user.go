package v1

import (
	"errors"
	"fmt"
	"log"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UpdateUserInput struct {
	Name        string `json:"full_name,omitempty"`
	Email       string `json:"email,omitempty" validate:"email,school_email"`
	PhoneNumber string `json:"phone_number,omitempty" gorm:"column:phone_number;not null"`
}

func UpdateProfile(c *fiber.Ctx) error {
	userid := c.Locals("userID")
	updatedUser := new(UpdateUserInput)
	if err := c.BodyParser(&updatedUser); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Bad request", "errors": "Invalid request body"})
	}
	errs := pkg.LoginValidator().Validate(updatedUser)
	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	var user models.User
	result := models.DB.Where("uuid = ?", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{"status": "Failed", "messages": "User not found"})
	}
	if updatedUser.Email != "" {
		user.Email = updatedUser.Email
	}
	if updatedUser.Name != "" {
		user.FullName = updatedUser.Name
	}
	if updatedUser.PhoneNumber != "" {
		user.PhoneNumber = updatedUser.PhoneNumber
	}
	models.DB.Save(&user)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Success", "message": "User details updated successfully"})
}

func UpdateProfilePicture(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	var user models.User
	result := models.DB.Where("uuid = ?", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{"status": "Failed", "messages": "User not found"})
	}
	fileHeader, err := c.FormFile("image")

	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Image is required",
		})
	}
	// Open the file from the file header
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to open image file",
		})
	}

	defer file.Close()
	go func() {
		if err := pkg.UploadAsyncSave(file, fileHeader,user.ID , "profile"); err != nil {
			log.Printf("Async upload failed: %v", err) 
		}
	}()
	return c.SendStatus(fiber.StatusNoContent)
}

func GetProfile(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	var user models.User
	result := models.DB.Where("uuid = ?", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{"status": "Failed", "messages": "User not found"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"user":   &user,
	})
}

func DeleteProfile(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	var user models.User
	result := models.DB.Where("uuid = ?", userid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{"status": "Failed", "messages": "User not found"})
	}

	models.DB.Delete(&user)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "User successfully Deleted",
	})
}
