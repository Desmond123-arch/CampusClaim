package auth

import (
	"fmt"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
)

func RegisterUser(c *fiber.Ctx) error {
	input := new(models.User)
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Bad request"})
	}

	errs := pkg.SchoolEmailValidator().Validate(input)


	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status":"Failed", "errors": errs})
	}
	fmt.Printf("%+v\n", input)

	return nil
}