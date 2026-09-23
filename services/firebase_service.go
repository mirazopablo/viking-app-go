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
		log.Fatalf("[services/firebase_service.go] [InitFirebase] could not load firebase credentials: %v", err)
	}

	// Usamos WithCredentialsJSON para pasar los bytes directamente
	opt := option.WithCredentialsJSON(b)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("[services/firebase_service.go] [InitFirebase] Error initializing Firebase App: %v", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Fatalf("[services/firebase_service.go] [InitFirebase] Error getting FCM client: %v", err)
	}

	fcmClient = client
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
// Rationale: Utilizamos SendEachForMulticast ya que Google dio de baja la API de /batch usada por SendMulticast.
func SendMulticastPushNotification(tokens []string, title string, body string) error {
	if len(tokens) == 0 {
		return nil
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Tokens: tokens,
	}

	// NUEVO: Usamos SendEachForMulticast en lugar de SendMulticast
	br, err := fcmClient.SendEachForMulticast(context.Background(), message)
	if err != nil {
		log.Printf("[services/firebase_service.go] [SendMulticastPushNotification] Error sending multicast message: %v\n", err)
		return err
	}

	if br.FailureCount > 0 {
		log.Printf("[services/firebase_service.go] [SendMulticastPushNotification] Multicast push partially failed. Failure count: %d\n", br.FailureCount)
	}
	return nil
}
