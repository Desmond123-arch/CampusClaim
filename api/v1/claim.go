package v1

import (
	"errors"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetItemCliams(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	item_id := c.Params("id")
	var claims []models.Claims
	var item models.Item
	pagination := pkg.Pagination{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 20),
	}

	if err := models.DB.Preload("User").Where("item_uuid = ?", item_id).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "false",
			"error":  "Item not found",
		})
	}

	result := models.DB.
		Scopes(pkg.Pagainate(claims, &pagination, models.DB)).
		Preload("User").
		Preload("Item").
		Preload("ClaimStatus").
		Where("item_id = ?", item.ID).
		Find(&claims)

	if userid != item.User.UUID.String() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "false",
			"error":  "Unauthorized",
		})
	}
	pagination.Rows = claims
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"count":  result.RowsAffected,
		"data":   pagination,
	})
}

func SubmitClaim(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	item_id := c.Params("id")

	var item models.Item
	var user models.User
	var status models.Claim_Status //default is pending which is 1
	if err := models.DB.Preload("User").Where("item_uuid = ?", item_id).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "false",
			"error":  "Item not found",
		})
	}

	if err := models.DB.Where("uuid = ?", userid).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "false",
			"error":  "User not found",
		})
	}
	if err := models.DB.Where("status = ?", "Pending").First(&status).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "False", "error": "Invalid claim status"})
	}
	if item.User.UUID.String() == user.UUID.String() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "False",
			"error": "Submitted user cannot claim item",
		})
	}
	claim := models.Claims{
		ItemID:   item.ID,
		UserID:   user.ID,
		StatusID: status.ID,
	}

	result := models.DB.Create(&claim)

	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status": "false",
			"error":  "User has already claimed this item",
		})
	}

	if err := models.DB.
		Preload("User").
		Preload("Item").
		Preload("ClaimStatus").
		First(&claim, claim.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to load full claim details",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"claim":  claim,
	})
}


func DeleteClaim(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	claimID := c.Params("id")

	var claim models.Claims
	if err := models.DB.
		Preload("User").
		Where("claim_id = ?", claimID).
		First(&claim).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "false",
			"error":  "Claim not found",
		})
	}

	if claim.User.UUID.String() != userID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "false",
			"error":  "You are not authorized to delete this claim",
		})
	}

	// Proceed with deletion
	if err := models.DB.Delete(&claim).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to delete claim",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"message": "Claim deleted successfully",
	})
}
