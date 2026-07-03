package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HomeController handles home and health check endpoints.
type HomeController struct{}

// NewHomeController instantiates a new HomeController.
func NewHomeController() *HomeController {
	return &HomeController{}
}

// Greeting godoc
// @Summary Saludo
// @Description Mensaje de bienvenida
// @Tags Home Controller
// @ID greeting
// @Produce text/plain
// @Success 200 {string} string "OK"
// @Security bearer-jwt
// @Router /api/ [get]
func (h *HomeController) Greeting(c *gin.Context) {
	c.String(http.StatusOK, "Welcome to Viking-App API REST")
}
