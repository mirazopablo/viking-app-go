package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var fcmClient *messaging.Client

// InitFirebase initializes the Firebase app and the FCM client.
func InitFirebase() {
	b, err := loadFirebaseCredentials()
	if err != nil {
		log.Fatalf("could not load firebase credentials: %v", err)
	}

	// Usamos WithCredentialsJSON para pasar los bytes directamente
	opt := option.WithCredentialsJSON(b)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase App: %v", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("Error getting FCM client: %v", err)
	}

	fcmClient = client
	log.Println("Firebase Cloud Messaging (FCM) initialized successfully!")
}

// loadFirebaseCredentials lee el JSON desde Base64 en el .env, o cae en el archivo local.
func loadFirebaseCredentials() ([]byte, error) {
	if encoded := os.Getenv("FIREBASE_CREDENTIALS_JSON"); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("FIREBASE_CREDENTIALS_JSON is not valid base64: %w", err)
		}
		return decoded, nil
	}
	// Fallback para desarrollo local
	return os.ReadFile("firebase_credentials.json")
}

// SendMulticastPushNotification sends a single message to multiple device tokens (Broadcast).
// Rationale: SendMulticast is highly optimized by Google to process up to 500 tokens in a single network request.
func SendMulticastPushNotification(tokens []string, title string, body string) error {
	if len(tokens) == 0 {
		log.Println("No tokens provided for multicast push notification. Skipping.")
		return nil
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Tokens: tokens,
	}

	br, err := fcmClient.SendMulticast(context.Background(), message)
	if err != nil {
		log.Printf("Error sending multicast message: %v\n", err)
		return err
	}

	log.Printf("Multicast push sent. Success count: %d, Failure count: %d\n", br.SuccessCount, br.FailureCount)
	return nil
}
