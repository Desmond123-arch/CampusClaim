package chat

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatChannel struct {
	ID           string   `bson:"_id,omitempty"`
	Participants []string `bson:"participants"`
	CreatedAt    int64    `bson:"created_at"`
}

var Clients = map[string]*websocket.Conn{}

func WebSocketUpgradeMiddleware() fiber.Handler {
	return func(c * fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return  c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	}
}


type IncomingMessage struct {
	ReceiverID string `json:"receiver_id"`
	Message string `json:"message"`
}

func HandleWebSocket(c *websocket.Conn) {
	userID := c.Locals("userID").(string)
	fmt.Println(userID)
	Clients[userID] = c
	defer delete(Clients, userID)

	for {
		var msg IncomingMessage
		if err := c.ReadJSON(&msg); err != nil {
			log.Println("Websocket read error", err)
			break
		}
		ctx := context.Background()
		channelCol := models.GetCollection("channels")
		filter := bson.M{"participants": bson.M{"$all": []string{userID, msg.ReceiverID}}}
		
		var channel models.ChatChannel
		err := channelCol.FindOne(ctx, filter).Decode(&channel)
		if err != nil {
			channel = models.ChatChannel{
				ID: primitive.NewObjectID().Hex(),
				Participants: []string{userID, msg.ReceiverID},
				CreatedAt: time.Now().Unix(),
			}
			_, err = channelCol.InsertOne(ctx, channel)
			if err != nil {
				log.Println("error creating channel:", err)
				continue
			}
		}

		message := models.Messages{
			ID: primitive.NewObjectID().Hex(),
			ChannelID: channel.ID,
			Sender: userID,
			Receiver: msg.ReceiverID,
			Content: msg.Message,
			TimeStamp: time.Now().Unix(),
		}
		msgCol := models.GetCollection("messages")
		_, err = msgCol.InsertOne(ctx, message)

		if err != nil {
			log.Println("Error saving message", err)
			continue
		}
		if conn, ok := Clients[msg.ReceiverID]; ok {
			conn.WriteJSON(message)
		}
	}

}