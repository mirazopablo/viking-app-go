package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
)

type PushController interface {
	RegisterToken(c *gin.Context)
	UnregisterToken(c *gin.Context)
}

type pushController struct {
	repo repositories.DeviceTokenRepository
}

func NewPushController(repo repositories.DeviceTokenRepository) PushController {
	return &pushController{repo: repo}
}

type RegisterTokenDto struct {
	FCMToken string `json:"fcmToken" binding:"required"`
	Device   string `json:"device"` // Ej: "android", "ios"
}

func (pc *pushController) RegisterToken(c *gin.Context) {
	// El middleware de Auth (JWT) inyecta el userID en el contexto
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No user ID found in token"})
		return
	}

	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	var dto RegisterTokenDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := &models.DeviceToken{
		UserID:   userID,
		FCMToken: dto.FCMToken,
		Device:   dto.Device,
	}

	if err := pc.repo.SaveToken(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token registered successfully"})
}

func (pc *pushController) UnregisterToken(c *gin.Context) {
	var dto RegisterTokenDto // Reutilizamos el DTO porque solo necesitamos el FCMToken
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := pc.repo.DeleteToken(dto.FCMToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unregister token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token unregistered successfully"})
}
