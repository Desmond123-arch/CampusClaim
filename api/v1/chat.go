package v1

import (
	"context"
	"errors"
	// "fmt"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetMessages(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	receiverID := c.Params("userId")

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
	msgCol := models.GetCollection("messages")
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

func GetConversations(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	collection := models.GetCollection("channels")
	ctx := context.Background()

	filter := bson.M{"participants": userID}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Failed to fetch conversations",
		})
	}
	defer cursor.Close(ctx)

	var channels []models.ChatChannel
	if err = cursor.All(ctx, &channels); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "Failed",
			"errors": "Failed to decode conversations",
		})
	}
	if len(channels) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":        "success",
			"conversations": []interface{}{},
		})
	}

	msgCollection := models.GetCollection("messages")
	var conversations []fiber.Map

	for _, channel := range channels {
		var otherParticipantID string
		for _, participant := range channel.Participants {
			if participant != userID {
				otherParticipantID = participant
				break
			}
		}

		var latestMessage models.Messages
		err := msgCollection.FindOne(ctx,
			bson.M{"channel_id": channel.ID},
			options.FindOne().SetSort(bson.M{"sent_at": -1}),
		).Decode(&latestMessage)
		var otherUser models.User
		err = models.DB.Where("uuid = ?", otherParticipantID).First(&otherUser).Error

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":"Failed",
				"result":"Internal server error",
			})
		}
		conversation := fiber.Map{
			"channel_id":      channel.ID,
			"other_user_id":   otherParticipantID,
			"created_at":      channel.CreatedAt,
			"last_message":    nil,
			"last_message_at": nil,
			"other_user_name": otherUser.FullName,
		}

		if err == nil {
			conversation["last_message"] = latestMessage.Content
			conversation["last_message_at"] = latestMessage.TimeStamp
			conversation["last_message_sender"] = latestMessage.Sender
		}

		conversations = append(conversations, conversation)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":        "success",
		"conversations": conversations,
	})
}

func CreateConversation(senderID, receiverID, message string) error {
	if senderID == receiverID {
		return errors.New("Cannont create conversation with yourself")
	}

	ctx := context.Background()
	collection := models.GetCollection("channels")

	filter := bson.M{"participants": bson.M{"$all": []string{senderID, receiverID}}}
	var existingChannel models.ChatChannel
	err := collection.FindOne(ctx, filter).Decode(&existingChannel)

	var channel models.ChatChannel
	if err != nil {
		newChannel := models.ChatChannel{
			Participants: []string{senderID, receiverID},
		}
		result, err := collection.InsertOne(ctx, newChannel)

		if err != nil {
			return err
		}

		if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
			newChannel.ID = oid.Hex()
		} else {
			return errors.New("failed to convert InsertedID to ObjectID")
		}
		channel = newChannel
	} else {
		channel = existingChannel
	}
	msgCollection := models.GetCollection("messages")
	newMeesage := models.Messages{
		ChannelID: channel.ID,
		Sender:    senderID,
		Receiver:  receiverID,
		Content:   message,
		TimeStamp: time.Now().Unix(),
	}
	_, err = msgCollection.InsertOne(ctx, newMeesage)
	if err != nil {
		return err
	}
	return nil
}
