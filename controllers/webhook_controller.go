package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/services"
)

type WebhookController struct {
	debounceService services.WebhookDebounceService
}

func NewWebhookController(debounceService services.WebhookDebounceService) *WebhookController {
	return &WebhookController{
		debounceService: debounceService,
	}
}

// HandleWhatsAppWebhook intercepts the webhook from Evolution API
func (wc *WebhookController) HandleWhatsAppWebhook(c *gin.Context) {
	var payload map[string]interface{}
	
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// We process the webhook asynchronously so we can reply 200 OK immediately to Evolution API
	go wc.debounceService.HandleIncomingWebhook(payload)

	// Always reply 200 OK so the API doesn't retry
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}
