package v1

import (
	"context"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	receiverID := c.Params("id")

	collection := models.GetCollection("channels")
	ctx := context.Background()
	filter := bson.M{"participants": bson.M{"$all": []string{userID, receiverID}}}
	var channel models.ChatChannel
	err := collection.FindOne(ctx, filter).Decode(&channel)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "Failed",
			"errors": "No channel found",
		})
	}
	msgCol := models.GetCollection("message")
	cursor, err := msgCol.Find(ctx, bson.M{"channel_id": channel.ID})

	if err != nil {
		return err
	}
	var messages []models.Messages
	if err = cursor.All(ctx, &messages); err != nil {
		return err
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":   "success",
		"messages": messages,
	})
}
