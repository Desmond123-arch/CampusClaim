package v1

import (
	"context"
	"fmt"

	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
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

	result, err := pkg.SendAddImageURL(url, "", "search")
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

	if description == "" {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Description is required",
		})
	}
	fmt.Println("Search by Text")
	result, err := pkg.SendAddImageURL("", description, "search")
	
	if err != nil {
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

