package v1

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"

	// "os"
	"strconv"
	"strings"

	// "time"

	"github.com/Desmond123-arch/CampusClaim/internal/firebase"
	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func GetItems(c *fiber.Ctx) error {
	var items []models.Item
	var total int64

	status := c.Query("status", "")
	location := c.Query("location", "")
	pagination := pkg.Pagination{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 20),
	}
	result := models.DB.
		Scopes(pkg.Pagainate(items, &pagination, models.DB)).
		Preload("User").
		Preload("Item_Status").
		Preload("Categories").
		Preload("Images").
		Joins("JOIN item_statuses ON item_statuses.id = items.status_id")
	countQuery := models.DB.
		Model(&models.Item{}).
		Joins("JOIN item_statuses ON item_statuses.id = items.status_id")
	if status != "" {
		result = result.Where("LOWER(item_statuses.status) LIKE ?", "%"+strings.ToLower(status)+"%")
		countQuery = countQuery.Where("LOWER(item_statuses.status) LIKE ?", "%"+strings.ToLower(status)+"%")
	}
	if location != "" {
		result = result.Where("items.found_at ILIKE ?", "%"+strings.ToLower(location)+"%").Find(&items)
		countQuery = countQuery.Where("items.found_at ILIKE ?", "%"+strings.ToLower(location)+"%")
	} else {
		result.Find(&items)
	}
	countQuery.Count(&total)

	pagination.Rows = items
	pagination.TotalRows = total

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"count":  result.RowsAffected,
		"data":   pagination,
	})
}
func GetMyItems(c *fiber.Ctx) error {
	var items []models.Item
	var total int64
	uid := c.Locals("userID").(string)
	status := c.Query("status", "")
	location := c.Query("location", "")
	pagination := pkg.Pagination{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 20),
	}
	countQuery := models.DB.
		Model(&models.Item{}).
		Joins("JOIN item_statuses ON item_statuses.id = items.status_id")
	result := models.DB.
		Scopes(pkg.Pagainate(items, &pagination, models.DB)).
		Preload("User").
		Preload("Item_Status").
		Preload("Categories").
		Preload("Images").
		Joins("JOIN item_statuses ON item_statuses.id = items.status_id").
		Joins("JOIN users ON users.id = items.posted_by").
		Where("users.uuid = ?", uid)
	if status != "" {
		result.Where("LOWER(item_statuses.status) LIKE ?", "%"+strings.ToLower(status)+"%")
		countQuery = countQuery.Where("LOWER(item_statuses.status) LIKE ?", "%"+strings.ToLower(status)+"%")
	}
	if location != "" {
		result.Where("items.found_at ILIKE ?", "%"+strings.ToLower(location)+"%").Find(&items)
		countQuery = countQuery.Where("items.found_at ILIKE ?", "%"+strings.ToLower(location)+"%")
	} else {
		result.Find(&items)
	}
	countQuery.Count(&total)
	pagination.Rows = items
	pagination.TotalRows = total
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"count":  result.RowsAffected,
		"data":   pagination,
	})
}
func GetItem(c *fiber.Ctx) error {
	uuid := c.Params("id")

	var item models.Item
	result := models.DB.Preload("User").Preload("Item_Status").Preload("Categories").Preload("Images").
		Joins("JOIN categories ON categories.id = items.category_id").Where("item_uuid = ?", uuid).First(&item)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{
			"status": "false",
			"count":  0,
			"errors": "Item not found",
		})
	} else if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"errors": "Internal Server Error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"count":  1,
		"item":   &item,
	})
}

func AddItem(c *fiber.Ctx) error {
	type CreateItemRequest struct {
		Title       string `json:"title" validate:"required"`
		Description string `json:"description" validate:"required"`
		Bounty      uint   `json:"bounty" validate:"numeric"`
		Found_At    string `json:"found_at" validate:"required"`
		Category    string `json:"category" validate:"required"`
		Status      string `json:"status" validate:"required"`
	}
	var user models.User
	uuid := c.Locals("userID").(string)

	result := models.DB.Where("uuid = ?", uuid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Invalid user details",
			"status": "false",
		})
	}
	var categories models.Categories
	var item_status models.Item_Status
	bounty, err := strconv.ParseUint(c.FormValue("bounty"), 10, 32)
	if err != nil {
		fmt.Println(err)
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Invalid Bounty Value",
		})
	}
	requestBody := &CreateItemRequest{
		Title:       c.FormValue("title"),
		Description: c.FormValue("description"),
		Bounty:      uint(bounty),
		Category:    c.FormValue("category"),
		Status:      strings.ToUpper(c.FormValue("status")),
		Found_At:    c.FormValue("found_at"),
	}
	fmt.Println(requestBody.Bounty)
	errs := pkg.GeneralValidator().Validate(requestBody)
	if len(errs) != 0 {
		fmt.Println(errs)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "Failed", "errors": errs})
	}
	if err := models.DB.Where("categories.category = ?", requestBody.Category).First(&categories).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status": "Failed",
			"error":  "Invalid Category",
		})
	}
	//TODO: Handle item status conversion
	if err := models.DB.Where("item_statuses.status = ?", requestBody.Status).First(&item_status).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status": "Failed",
			"error":  "Invalid Item Status",
		})
	}
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Invalid multipart form",
		})
	}
	files := form.File["images"]
	if len(files) > 3 {
		return c.Status(400).JSON(fiber.Map{
			"status": "false",
			"error":  "Maximum 3 images allowed",
		})
	}

	item := models.Item{
		Title:       requestBody.Title,
		Description: requestBody.Description,
		Bounty:      requestBody.Bounty,
		UserID:      user.ID,
		CategoryID:  categories.ID,
		StatusID:    item_status.ID,
		User:        user,
		Item_Status: item_status,
		Categories:  categories,
		Found_At:    requestBody.Found_At,
	}
	if err := models.DB.Create(&item).Error; err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to create item",
		})
	}
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"status": "false",
				"error":  "Failed to open image file",
			})
		}
		go func(file multipart.File) {
			url, err := pkg.UploadAsyncSave(file, fileHeader, item.ID, "item")
			if err != nil {
				log.Printf("Async upload failed: %v", err)
			}
			_, err = pkg.SendAddImageURL(url, item.Description, "add", item.UUID.String())
			if err != nil {
				log.Printf("Async upload failed: %v", err)
			}
		}(file)
	}

	if err := models.DB.Preload("User").Preload("Item_Status").Preload("Categories").Preload("Images").
		First(&item, item.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "false",
			"error":  "Failed to fetch item details",
		})
	}

	recentSearchesTokens, err := pkg.GetRecentSearched(item.Title)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "false",
			"errors": "Internal Server Error",
		})
	}
	fmt.Println("Recent Search Tokens");
	fmt.Println(recentSearchesTokens);
	if len(recentSearchesTokens) != 0 {
		go firebase.SendNotifactions(recentSearchesTokens, item.UUID.String(), "New Item Posted", fmt.Sprintf("New item: %s", item.Title))
	}
	// fmt.Print(item)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"item":   &item,
	})
}

func UpdateItem(c *fiber.Ctx) error {
	type UpdateItemRequest struct {
		Title       string `json:"title"`
		Description string `json:"description" `
		Bounty      uint   `json:"bounty"`
		Category    string `json:"category"`
		Status      string `json:"status" `
	}
	userid := c.Locals("userID").(string)
	itemid := c.Params("id")
	var itemrequest UpdateItemRequest
	var item models.Item

	if err := c.BodyParser(&itemrequest); err != nil {

	}

	errs := pkg.GeneralValidator().Validate(itemrequest)
	if len(errs) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "Failed",
			"error":  errs,
		})
	}
	if itemrequest.Status != "" {
		itemrequest.Status = strings.ToUpper(string(itemrequest.Status[0])) + strings.ToLower(itemrequest.Status[1:])
	}
	result := models.DB.Preload("User").Where("item_uuid = ?", itemid).First(&item)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{
			"status": "false",
			"error":  "Item not found",
		})
	}
	if item.User.UUID.String() != userid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "false",
			"error":  "Unauthorized",
		})
	}
	if itemrequest.Category != "" {
		var catergory models.Categories
		result := models.DB.Where("category = ?", itemrequest.Category).First(&catergory)
		if result.Error != nil {
			return c.Status(400).JSON(fiber.Map{
				"status": "false",
				"error":  "Invalid Category",
			})
		}
		item.Categories = catergory
		item.CategoryID = catergory.ID
	}
	if itemrequest.Status != "" {
		var status models.Item_Status
		result := models.DB.Where("status = ?", itemrequest.Status).First(&status)
		if result.Error != nil {
			return c.Status(400).JSON(fiber.Map{
				"status": "false",
				"error":  "Invalid Item Status",
			})
		}
		item.Item_Status = status
		item.StatusID = status.ID
	}
	if itemrequest.Title != "" {
		item.Title = itemrequest.Title
	}
	if itemrequest.Description != "" {
		item.Description = itemrequest.Description
	}
	if itemrequest.Bounty != 0 && itemrequest.Bounty > 0 {
		item.Bounty = itemrequest.Bounty
	}
	models.DB.Save(&item)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "Success", "message": "Item details updated successfully"})
}

func DeleteItem(c *fiber.Ctx) error {
	userid := c.Locals("userID").(string)
	itemid := c.Params("id")
	var item models.Item
	result := models.DB.Where("item_uuid = ?", itemid).First(&item)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Status(404).JSON(fiber.Map{
			"status": "false",
			"error":  "Item not found",
		})
	}
	if item.User.UUID.String() != userid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "false",
			"error":  "Unauthorized",
		})
	}
	models.DB.Where("uuid = ?", itemid).Delete(&item)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Item details deleted successfully",
	})
}
