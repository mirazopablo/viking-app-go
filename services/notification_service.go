package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrSubscriptionNotFound = errors.New("push subscription not found")
)

// NotificationService defines business logic for managing WebPush and deferred notifications.
type NotificationService interface {
	Subscribe(dto *models.PushSubscriptionCreateRequestDto) (*models.PushSubscription, error)
	Unsubscribe(endpoint string) error
	GetHistoryByWorkOrder(workOrderID, securityCode string) ([]models.NotificationHistoryResponseDto, error)
	MarkHistoryAsRead(workOrderID, securityCode string) error
	NotifyOrderStatusChanged(wo *models.WorkOrder, oldStatus, newStatus string)
	NotifyDiagnosticPointAdded(dp *models.DiagnosticPoint, wo *models.WorkOrder)
}

type notificationServiceImpl struct {
	pushRepo      repositories.PushSubscriptionRepository
	historyRepo   repositories.NotificationHistoryRepository
	workOrderRepo repositories.WorkOrderRepository

	// Debounce state for diagnostic points to prevent alert spamming when multiple files are uploaded quickly
	mu            sync.Mutex
	pendingTimers map[string]*time.Timer
	pendingCounts map[string]int
}

// NewNotificationService instantiates a new NotificationService with repositories and debounce state.
func NewNotificationService(
	pushRepo repositories.PushSubscriptionRepository,
	historyRepo repositories.NotificationHistoryRepository,
	workOrderRepo repositories.WorkOrderRepository,
) NotificationService {
	return &notificationServiceImpl{
		pushRepo:      pushRepo,
		historyRepo:   historyRepo,
		workOrderRepo: workOrderRepo,
		pendingTimers: make(map[string]*time.Timer),
		pendingCounts: make(map[string]int),
	}
}

func (s *notificationServiceImpl) Subscribe(dto *models.PushSubscriptionCreateRequestDto) (*models.PushSubscription, error) {
	if strings.TrimSpace(dto.WorkOrderID) == "" || strings.TrimSpace(dto.SecurityCode) == "" {
		return nil, ErrInvalidSecurityCode
	}

	wo, err := s.workOrderRepo.FindByID(dto.WorkOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(wo.SecurityCodeHash), []byte(strings.TrimSpace(dto.SecurityCode))); err != nil {
		return nil, ErrInvalidSecurityCode
	}

	sub := &models.PushSubscription{
		WorkOrderID: dto.WorkOrderID,
		Endpoint:    strings.TrimSpace(dto.Endpoint),
		P256dh:      strings.TrimSpace(dto.Keys.P256dh),
		Auth:        strings.TrimSpace(dto.Keys.Auth),
		UserAgent:   strings.TrimSpace(dto.UserAgent),
	}

	return s.pushRepo.Save(sub)
}

func (s *notificationServiceImpl) Unsubscribe(endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("endpoint is required")
	}
	return s.pushRepo.DeleteByEndpoint(endpoint)
}

func (s *notificationServiceImpl) GetHistoryByWorkOrder(workOrderID, securityCode string) ([]models.NotificationHistoryResponseDto, error) {
	if strings.TrimSpace(workOrderID) == "" || strings.TrimSpace(securityCode) == "" {
		return nil, ErrInvalidSecurityCode
	}

	wo, err := s.workOrderRepo.FindByID(workOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(wo.SecurityCodeHash), []byte(strings.TrimSpace(securityCode))); err != nil {
		return nil, ErrInvalidSecurityCode
	}

	history, err := s.historyRepo.FindByWorkOrderID(workOrderID)
	if err != nil {
		return nil, err
	}

	res := make([]models.NotificationHistoryResponseDto, len(history))
	for i, nh := range history {
		res[i] = *toNotificationHistoryResponseDto(&nh)
	}
	return res, nil
}

func (s *notificationServiceImpl) MarkHistoryAsRead(workOrderID, securityCode string) error {
	if strings.TrimSpace(workOrderID) == "" || strings.TrimSpace(securityCode) == "" {
		return ErrInvalidSecurityCode
	}

	wo, err := s.workOrderRepo.FindByID(workOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrWorkOrderNotFound
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(wo.SecurityCodeHash), []byte(strings.TrimSpace(securityCode))); err != nil {
		return ErrInvalidSecurityCode
	}

	return s.historyRepo.MarkAllAsReadByWorkOrderID(workOrderID)
}

func (s *notificationServiceImpl) NotifyOrderStatusChanged(wo *models.WorkOrder, oldStatus, newStatus string) {
	if wo == nil || strings.EqualFold(oldStatus, newStatus) {
		return
	}

	// Format a human-readable status description for client notifications (omit raw UUIDs and technical jargon)
	readableStatus := formatReadableStatus(newStatus)
	title := "Viking-APP: Actualización de Orden"
	body := fmt.Sprintf("Tu equipo ha cambiado de estado a: %s.", readableStatus)
	targetURL := "/public/work-order/status"

	// 1. Persist notification in database history right away to ensure morning sync / offline bell readiness
	nh := &models.NotificationHistory{
		WorkOrderID: wo.ID,
		Title:       title,
		Body:        body,
		Type:        "STATUS_CHANGE",
		TargetURL:   targetURL,
		IsRead:      false,
	}
	if _, err := s.historyRepo.Save(nh); err != nil {
		log.Printf("[NotificationService] Error saving status change notification history for order %s: %v", wo.ID, err)
	}

	// 2. Dispatch live WebPush notification asynchronously
	go s.sendWebPushToWorkOrder(wo.ID, title, body, targetURL)
}

func (s *notificationServiceImpl) NotifyDiagnosticPointAdded(dp *models.DiagnosticPoint, wo *models.WorkOrder) {
	if dp == nil || wo == nil {
		return
	}

	// 1. Immediately persist individual diagnostic point addition in database history for morning sync
	title := "Viking-APP: Observación Técnica"
	body := "Se ha registrado un nuevo punto de diagnóstico en tu orden de trabajo."
	if dp.Description != "" {
		body = fmt.Sprintf("Observación registrada: %s", truncateString(dp.Description, 80))
	}
	targetURL := "/public/work-order/status"

	nh := &models.NotificationHistory{
		WorkOrderID: wo.ID,
		Title:       title,
		Body:        body,
		Type:        "DIAGNOSTIC_POINT",
		TargetURL:   targetURL,
		IsRead:      false,
	}
	if _, err := s.historyRepo.Save(nh); err != nil {
		log.Printf("[NotificationService] Error saving diagnostic point history for order %s: %v", wo.ID, err)
	}

	// 2. Apply memory debounce window (2 minutes) to coalesce multiple quick diagnostic point uploads
	s.mu.Lock()
	defer s.mu.Unlock()

	orderID := wo.ID
	s.pendingCounts[orderID]++

	if timer, exists := s.pendingTimers[orderID]; exists && timer != nil {
		timer.Reset(2 * time.Minute)
		return
	}

	s.pendingTimers[orderID] = time.AfterFunc(2*time.Minute, func() {
		s.flushDiagnosticDebounce(orderID)
	})
}

func (s *notificationServiceImpl) flushDiagnosticDebounce(workOrderID string) {
	s.mu.Lock()
	count := s.pendingCounts[workOrderID]
	delete(s.pendingCounts, workOrderID)
	delete(s.pendingTimers, workOrderID)
	s.mu.Unlock()

	if count <= 0 {
		return
	}

	title := "Viking-APP: Actualización Técnica"
	body := "Se ha añadido una nueva observación técnica o evidencia multimedia a tu orden."
	if count > 1 {
		body = fmt.Sprintf("Se han añadido %d nuevas observaciones técnicas a tu orden de reparación.", count)
	}
	targetURL := "/public/work-order/status"

	s.sendWebPushToWorkOrder(workOrderID, title, body, targetURL)
}

func (s *notificationServiceImpl) sendWebPushToWorkOrder(workOrderID, title, body, targetURL string) {
	if config.AppConfig.VAPIDPublicKey == "" || config.AppConfig.VAPIDPrivateKey == "" {
		log.Println("[NotificationService] VAPID keys not configured. Skipping WebPush dispatch.")
		return
	}

	subs, err := s.pushRepo.FindByWorkOrderID(workOrderID)
	if err != nil || len(subs) == 0 {
		return
	}

	payloadMap := map[string]interface{}{
		"title":     title,
		"body":      body,
		"targetUrl": targetURL,
		"timestamp": time.Now().Unix(),
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("[NotificationService] Error marshaling push payload: %v", err)
		return
	}

	var wg sync.WaitGroup
	// Create an HTTP client with a strict timeout to prevent goroutine leaks
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	for _, sub := range subs {
		wg.Add(1)
		go func(subscription models.PushSubscription) {
			defer wg.Done()
			
			resp, err := webpush.SendNotification(
				payloadBytes,
				&webpush.Subscription{
					Endpoint: subscription.Endpoint,
					Keys: webpush.Keys{
						P256dh: subscription.P256dh,
						Auth:   subscription.Auth,
					},
				},
				&webpush.Options{
					Subscriber:      config.AppConfig.VAPIDSubscriberEmail,
					VAPIDPublicKey:  config.AppConfig.VAPIDPublicKey,
					VAPIDPrivateKey: config.AppConfig.VAPIDPrivateKey,
					TTL:             43200, // 12 hours TTL so push gateways retry if mobile client is offline
					HTTPClient:      httpClient,
				},
			)
			if err != nil {
				log.Printf("[NotificationService] Push error to endpoint %s: %v", subscription.Endpoint, err)
				return
			}
			defer resp.Body.Close()

			// Check HTTP status code: 410 Gone or 404 Not Found indicates subscription expired or revoked by browser
			if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
				log.Printf("[NotificationService] Subscription expired (status %d), cleaning up endpoint: %s", resp.StatusCode, subscription.Endpoint)
				_ = s.pushRepo.DeleteByEndpoint(subscription.Endpoint)
			}
		}(sub)
	}
	
	// Wait for all concurrent push notifications to finish before exiting the parent goroutine
	wg.Wait()
}

func formatReadableStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case models.StatusReceived:
		return "Recibido en Taller"
	case models.StatusInQueue:
		return "En Cola de Espera"
	case models.StatusInProgress:
		return "En Proceso / Diagnosticando"
	case models.StatusDone:
		return "Reparación Finalizada (Listo para Retirar)"
	case models.StatusWithdrawn:
		return "Entregado y Retirado"
	default:
		return status
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func toNotificationHistoryResponseDto(nh *models.NotificationHistory) *models.NotificationHistoryResponseDto {
	return &models.NotificationHistoryResponseDto{
		ID:        nh.ID,
		Title:     nh.Title,
		Body:      nh.Body,
		Type:      nh.Type,
		TargetURL: nh.TargetURL,
		IsRead:    nh.IsRead,
		CreatedAt: nh.CreatedAt.Format(time.RFC3339),
	}
}
