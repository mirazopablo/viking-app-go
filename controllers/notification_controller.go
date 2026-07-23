package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// NotificationController handles public WebPush subscription and notification history endpoints.
type NotificationController struct {
	service services.NotificationService
}

// NewNotificationController instantiates a new NotificationController.
func NewNotificationController(service services.NotificationService) *NotificationController {
	return &NotificationController{service: service}
}

// Subscribe godoc
// @Summary Suscribir Navegador a Notificaciones WebPush
// @Description Registra el endpoint y claves de cifrado del navegador para alertas en tiempo real de una orden
// @Tags Notification Controller
// @ID subscribePushNotification
// @Accept json
// @Produce json
// @Param subscription body models.PushSubscriptionCreateRequestDto true "Push Subscription Request"
// @Success 200 {object} models.PushSubscription "OK"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not Found"
// @Router /public/notifications/subscribe [post]
func (nc *NotificationController) Subscribe(c *gin.Context) {
	var input models.PushSubscriptionCreateRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	saved, err := nc.service.Subscribe(&input)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work order not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidSecurityCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid security code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

// Unsubscribe godoc
// @Summary Desuscribir Navegador de Notificaciones WebPush
// @Description Revoca o elimina la suscripción WebPush utilizando la URL del endpoint
// @Tags Notification Controller
// @ID unsubscribePushNotification
// @Accept json
// @Produce json
// @Param request body models.PushSubscriptionUnsubscribeRequestDto true "Unsubscribe Request"
// @Success 200 {object} map[string]string "OK"
// @Router /public/notifications/unsubscribe [post]
func (nc *NotificationController) Unsubscribe(c *gin.Context) {
	var input models.PushSubscriptionUnsubscribeRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := nc.service.Unsubscribe(input.Endpoint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unsubscribed"})
}

// GetHistory godoc
// @Summary Obtener Historial de Notificaciones por Orden de Trabajo
// @Description Permite la sincronización matutina (pull sync) al encender el servidor y actualiza la campanita interna
// @Tags Notification Controller
// @ID getNotificationHistory
// @Produce json
// @Param workOrderId query string true "Work Order UUID" format(uuid)
// @Param securityCode query string true "Security Code"
// @Success 200 {array} models.NotificationHistoryResponseDto "OK"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not Found"
// @Router /public/notifications/history [get]
func (nc *NotificationController) GetHistory(c *gin.Context) {
	workOrderID := c.Query("workOrderId")
	securityCode := c.Query("securityCode")

	history, err := nc.service.GetHistoryByWorkOrder(workOrderID, securityCode)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work order not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidSecurityCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid security code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// MarkAsRead godoc
// @Summary Marcar Todas las Notificaciones de una Orden como Leídas
// @Description Actualiza el estado de las alertas en la base de datos a leídas para el contador de la campanita
// @Tags Notification Controller
// @ID markNotificationHistoryAsRead
// @Accept json
// @Produce json
// @Param workOrderId query string true "Work Order UUID" format(uuid)
// @Param securityCode query string true "Security Code"
// @Success 200 {object} map[string]string "OK"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not Found"
// @Router /public/notifications/mark-read [post]
func (nc *NotificationController) MarkAsRead(c *gin.Context) {
	workOrderID := c.Query("workOrderId")
	securityCode := c.Query("securityCode")

	err := nc.service.MarkHistoryAsRead(workOrderID, securityCode)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work order not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidSecurityCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid security code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "marked_as_read"})
}
