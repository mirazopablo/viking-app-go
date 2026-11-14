package controllers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
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
// @Success 200 {object} map[string]string "OK"
// @Failure 400 {object} apperrors.ErrorResponse "Validation or business rule error"
// @Failure 409 {object} apperrors.ErrorResponse "Email already registered"
// @Failure 500 {object} apperrors.ErrorResponse "Internal server error"
// @Security bearer-jwt
// @Router /auth/signup [post]
func (ac *AuthController) RegisterUser(c *gin.Context) {
	var input models.RegisterDto
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	_, err := ac.service.RegisterUser(&input)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, gin.H{"message": "User registered successfully"})
}

// AuthenticateUser godoc
// @Summary Iniciar Sesión
// @Description Autentica a un usuario y retorna su token JWT junto con la información de perfil y rol
// @Tags Auth
// @ID authenticateUser
// @Accept json
// @Produce json
// @Param creds body models.LoginUserDto true "Login Credentials"
// @Success 200 {object} models.LoginResponseDto "OK"
// @Failure 400 {object} apperrors.ErrorResponse "Validation error"
// @Failure 401 {object} apperrors.ErrorResponse "Invalid credentials"
// @Failure 500 {object} apperrors.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (ac *AuthController) AuthenticateUser(c *gin.Context) {
	var input models.LoginUserDto
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	resp, err := ac.service.LoginUser(&input)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, resp)
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
		apperrors.OK(c, false)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	isValid := ac.service.ValidateTokenString(tokenString)
	apperrors.OK(c, isValid)
}
