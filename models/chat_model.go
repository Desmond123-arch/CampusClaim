package models

import (
	"context"
	"fmt"
	"time"

	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Messages struct {
	ID primitive.ObjectID `bson:"_id, omitempty" json:"id"`
	Sender int32 `bson:"sender_id" json:"sender"`
	Receiver int32 `bson:"receiver_id" json:"receiver"`
	Content string `bson:"content" json:"content"`
	SentAt time.Time `bson:"sent_at" json:"sent_at"`
}

type MongoDB struct {
	client *mongo.Client
	MessagesCollection *mongo.Collection
}
func MongoSetup(uri string) (*MongoDB, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("Failed to create mongo client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	err = client.Ping(ctx, nil)

	if err != nil {
        // Clean up if connection fails
        _ = client.Disconnect(ctx)
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

	database := client.Database("Chats")


	return &MongoDB{
        client:             client,
        MessagesCollection: database.Collection("messages"),
    }, nil
    
}

func (m *MongoDB) Close() error {
	if m.client != nil {
		return m.client.Disconnect(context.Background())
	}
	return nil
}