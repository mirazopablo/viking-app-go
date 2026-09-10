package services

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type WebhookDebounceService interface {
	HandleIncomingWebhook(payload map[string]interface{}) error
}

type webhookDebounceServiceImpl struct {
	sessions sync.Map // map[string]*debounceSession
}

type debounceSession struct {
	RemoteJid     string
	Messages      []string
	Timer         *time.Timer
	OriginalEvent map[string]interface{}
	mu            sync.Mutex
}

func NewWebhookDebounceService() WebhookDebounceService {
	return &webhookDebounceServiceImpl{}
}

// HandleIncomingWebhook process incoming webhooks from Evolution API
func (s *webhookDebounceServiceImpl) HandleIncomingWebhook(payload map[string]interface{}) error {
	// Extract data
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		// Not a standard message or missing data, forward immediately
		return s.forwardToN8N(payload)
	}

	key, ok := data["key"].(map[string]interface{})
	if !ok {
		return s.forwardToN8N(payload)
	}

	remoteJid, ok := key["remoteJid"].(string)
	if !ok || remoteJid == "" {
		return s.forwardToN8N(payload)
	}

	// We only debounce if it's from a user (fromMe = false)
	fromMe, ok := key["fromMe"].(bool)
	if ok && fromMe {
		return s.forwardToN8N(payload)
	}

	msg, ok := data["message"].(map[string]interface{})
	if !ok {
		return s.forwardToN8N(payload)
	}

	text := ""
	if conv, ok := msg["conversation"].(string); ok && conv != "" {
		text = conv
	} else if ext, ok := msg["extendedTextMessage"].(map[string]interface{}); ok {
		if extText, ok := ext["text"].(string); ok {
			text = extText
		}
	}

	if text == "" {
		// If it's an image or something else without text, forward immediately
		return s.forwardToN8N(payload)
	}

	var session *debounceSession
	sessionIntf, loaded := s.sessions.Load(remoteJid)
	if loaded {
		session = sessionIntf.(*debounceSession)
		session.mu.Lock()
		session.Messages = append(session.Messages, text)
		
		// Reset the timer
		if session.Timer != nil {
			session.Timer.Stop()
		}
		
		session.Timer = time.AfterFunc(10*time.Second, func() {
			s.flushSession(remoteJid)
		})
		session.mu.Unlock()
	} else {
		// Create new session
		session = &debounceSession{
			RemoteJid:     remoteJid,
			Messages:      []string{text},
			OriginalEvent: payload, // keep the first event as base
		}
		
		session.Timer = time.AfterFunc(10*time.Second, func() {
			s.flushSession(remoteJid)
		})
		
		s.sessions.Store(remoteJid, session)
	}

	return nil
}

func (s *webhookDebounceServiceImpl) flushSession(remoteJid string) {
	sessionIntf, loaded := s.sessions.LoadAndDelete(remoteJid)
	if !loaded {
		return
	}
	session := sessionIntf.(*debounceSession)
	
	session.mu.Lock()
	defer session.mu.Unlock()

	// Concatenate all messages
	finalText := ""
	for i, msg := range session.Messages {
		if i > 0 {
			finalText += "\\n"
		}
		finalText += msg
	}

	// Modify the original event's conversation text
	payload := session.OriginalEvent
	if data, ok := payload["data"].(map[string]interface{}); ok {
		if msg, ok := data["message"].(map[string]interface{}); ok {
			msg["conversation"] = finalText
			// Clear extendedTextMessage to force evolution/n8n to read conversation
			delete(msg, "extendedTextMessage")
			data["message"] = msg
		}
	}

	log.Printf("Debounce completed for %s, forwarding %d concatenated messages.", remoteJid, len(session.Messages))
	s.forwardToN8N(payload)
}

func (s *webhookDebounceServiceImpl) forwardToN8N(payload map[string]interface{}) error {
	// n8n Webhook URL. It uses the docker internal network by default, or an env variable.
	n8nURL := os.Getenv("N8N_WEBHOOK_URL")
	if n8nURL == "" {
		n8nURL = "http://n8n_automation:5678/webhook/vicki-bot"
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling debounced webhook: %v", err)
		return err
	}
	
	req, err := http.NewRequest("POST", n8nURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request to n8n: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding webhook to n8n: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("n8n responded with status %d", resp.StatusCode)
	}

	return nil
}
