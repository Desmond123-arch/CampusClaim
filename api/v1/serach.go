package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SearchByImage(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Image is required",
		})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to open file image",
		})
	}

	url, err := pkg.UploadFile(file, fileHeader, context.TODO(), "temp")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to upload image",
		})
	}

	result, err := pkg.SendAddImageURL(url, "", "search", "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"error":  "An error occurred while searching by image",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"result": result,
	})
}

func SearchByDescription(c *fiber.Ctx) error {
	description := c.FormValue("description")
	uuid := c.Locals("userID")
	var user models.User
	user_res := models.DB.Where("uuid = ?", uuid).First(&user)
	if errors.Is(user_res.Error, gorm.ErrRecordNotFound) {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Invalid user details",
			"status": "false",
		})
	}
	searchQuery := models.RecentSearches{
		SearchQuery: description, 
		UserID: user.ID,
		SearchTSV:   gorm.Expr("to_tsvector('english', ?)", description),
	}

	models.DB.Create(&searchQuery)
	if description == "" {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Description is required",
		})
	}
	fmt.Println("Search by Text")

	result, err := pkg.SendAddImageURL("", description, "search", "")

	if err != nil {
		fmt.Println(err)
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"error":  "An error occurred while searching by description",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"result": result["results"],
	})
}
