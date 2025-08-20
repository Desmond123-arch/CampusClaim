package v1

import (
	"errors"
	"fmt"
	"strconv"

	// "github.com/Desmond123-arch/CampusClaim/internal/firebase"
	"github.com/Desmond123-arch/CampusClaim/internal/firebase"
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
	type claimDetails struct {
		AgreedToTerms bool `json:"agreedToTerms"`
		LostDateTime string `json:"lostDateTime"`
		LostLocation string `json:"lostLocation"`
		UniqueFeature string `json:"uniqueFeature"`
		DeliveryPhone string `json:"deliveryPhone"`

	}
	userid := c.Locals("userID").(string)
	item_id := c.Params("id")

	var claimDetail claimDetails
	var item models.Item
	var user models.User
	var status models.Claim_Status //default is pending which is 1

	


	if err := c.BodyParser(&claimDetail); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"status":"Failed",
				"errors": "Incorrect request body",
			})
	}

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
			"error":  "Poster cannot claim posted item",
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
	var user_token models.UserTokens
	_ = models.DB.Where(`"user" = ?`, strconv.Itoa(int(item.UserID))).First(&user_token)
	agreed := "No"
	if claimDetail.AgreedToTerms {
		agreed = "Yes"
	}
	
	msg := fmt.Sprintf(
		"%s is trying to claim your item.\n\n"+
			"Here are the details they provided:\n\n"+
			"Agreed to terms: %s\n"+
			"Lost on: %s\n"+
			"Lost at: %s\n"+
			"Unique feature mentioned: %s",
		user.FullName,
		agreed,
		claimDetail.LostDateTime,
		claimDetail.LostLocation,
		claimDetail.UniqueFeature,
	)
	
	CreateConversation(userid, item.User.UUID.String(), string(msg))

	if user_token.Token != "" && user_token.IsSubscribed {
		firebase.SendNotifactionClaim([]string{user_token.Token}, strconv.Itoa(int(item.UserID)), item.UUID.String(), fmt.Sprintf("Claimed Submitted For your recent %s", item.Title), "Please verify claim")
	}
	// firebase.SendNotifactionClaim()
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
		"status":  "success",
		"message": "Claim deleted successfully",
	})
}
