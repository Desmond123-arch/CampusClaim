package firebase

import (
	"context"
	"fmt"
	"os"

	"firebase.google.com/go/v4/messaging"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var FCMClient *messaging.Client

func Init() error {
	credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	opt := option.WithCredentialsFile(credFile)

	app, err := firebase.NewApp(context.Background(), nil, opt)

	if err != nil {
		return fmt.Errorf("error initializing firebase app: %v", err)
	}
	fcmClient, err := app.Messaging(context.Background())
	if err != nil {
		return fmt.Errorf("error initializing FCM client: %v", err) 
	}
	FCMClient = fcmClient
	return nil
}
