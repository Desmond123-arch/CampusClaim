package models

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Messages struct {
	ID string`bson:"_id, omitempty" json:"id"`
	Sender string `bson:"sender_id" json:"sender"`
	Receiver string `bson:"receiver_id" json:"receiver"`
	ChannelID  string `bson:"channel_id"`
	Content string `bson:"content" json:"content"`
	TimeStamp int64 `bson:"sent_at" json:"sent_at"`
}

type ChatChannel struct {
	ID           string   `bson:"_id,omitempty"`
	Participants []string `bson:"participants"`
	CreatedAt    int64    `bson:"created_at"`
}

func MongoSetup(uri string) (*mongo.Client, error) {
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
	return client, nil
}

func GetCollection(name string) *mongo.Collection {
	return  MDB.Database("Chats").Collection(name)
}