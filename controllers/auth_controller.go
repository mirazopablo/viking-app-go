package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// AuthController handles authentication and user signup endpoints.
type AuthController struct {
	service services.UserService
}

// NewAuthController instantiates a new AuthController.
func NewAuthController(service services.UserService) *AuthController {
	return &AuthController{service: service}
}

// RegisterUser godoc
// @Summary Registrar Usuario
// @Description Crea y registra un nuevo usuario en la base de datos
// @Tags Auth
// @ID registerUser
// @Accept json
// @Produce json
// @Param user body models.RegisterDto true "Register Request"
// @Success 200 {string} string "OK"
// @Security bearer-jwt
// @Router /auth/signup [post]
func (ac *AuthController) RegisterUser(c *gin.Context) {
	var input models.RegisterDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := ac.service.RegisterUser(&input)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
		if errors.Is(err, services.ErrPasswordRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.String(http.StatusOK, "User registered successfully")
}

// AuthenticateUser godoc
// @Summary Iniciar Sesión
// @Description Autentica a un usuario y retorna su token JWT
// @Tags Auth
// @ID authenticateUser
// @Accept json
// @Produce json
// @Param creds body models.LoginUserDto true "Login Credentials"
// @Success 200 {object} map[string]string "OK"
// @Router /auth/login [post]
func (ac *AuthController) AuthenticateUser(c *gin.Context) {
	var input models.LoginUserDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := ac.service.LoginUser(&input)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCreds) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"type":  "Bearer",
	})
}

// ValidateToken godoc
// @Summary Validar Token JWT
// @Description Verifica si un token JWT enviado en la cabecera Authorization es válido
// @Tags Auth
// @ID validateToken
// @Produce json
// @Success 200 {boolean} boolean "OK"
// @Router /auth/validate [get]
func (ac *AuthController) ValidateToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, false)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	isValid := ac.service.ValidateTokenString(tokenString)
	c.JSON(http.StatusOK, isValid)
}
