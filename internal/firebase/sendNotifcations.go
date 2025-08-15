package firebase

import (
	"context"
	"fmt"
	"log"

	"firebase.google.com/go/v4/messaging"
)

func SendNotifactions(registrationTokens []string, item_id, title, body string) {
	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"item_id": item_id,
		},
		Notification: &messaging.Notification{
			Title: title,
			Body: body,
		},
		Tokens: registrationTokens,
	}
	br, err := FCMClient.SendEachForMulticast(context.Background(), message)
	if err != nil {
		log.Fatalln(err)
	}
	var failedTokens []string
	if br.FailureCount > 0 {

		for idx, resp := range br.Responses {
			if !resp.Success {
				failedTokens = append(failedTokens, registrationTokens[idx])
			}
		}
	}
	fmt.Printf("List of tokens that caused failures: %v\n", failedTokens)
}

func SendNotifactionClaim(registrationTokens[] string, user_id, item_id, title, body string) {
	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"item_id": item_id,
			"user_id": user_id,
		},
		Notification: &messaging.Notification{
			Title: title,
			Body: body,
		},
		Tokens: registrationTokens,
	}
	br, err := FCMClient.SendEachForMulticast(context.Background(), message)
	if err != nil {
		log.Fatalln(err)
	}
	var failedTokens []string
	if br.FailureCount > 0 {

		for idx, resp := range br.Responses {
			if !resp.Success {
				failedTokens = append(failedTokens, registrationTokens[idx])
			}
		}
	}
	fmt.Printf("List of tokens that caused failures: %v\n", failedTokens)
}