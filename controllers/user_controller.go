package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// UserController handles CRUD operations for user profiles.
type UserController struct {
	service services.UserService
	jwtSvc  services.JWTService
}

// NewUserController instantiates a new UserController.
func NewUserController(service services.UserService, jwtSvc services.JWTService) *UserController {
	return &UserController{service: service, jwtSvc: jwtSvc}
}

// SaveUser godoc
// @Summary Guardar Usuario
// @Description Guarda un nuevo usuario en la base de datos
// @Tags User Controller
// @ID registerUser_1
// @Accept json
// @Produce json
// @Param user body models.RegisterDto true "User Create Request"
// @Success 200 {string} string "OK"
// @Security bearer-jwt
// @Router /api/user/save [post]
func (uc *UserController) SaveUser(c *gin.Context) {
	var input models.RegisterDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := uc.service.RegisterUser(&input)
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

	c.String(http.StatusOK, "User saved successfully")
}

// UpdateUser godoc
// @Summary Actualizar Usuario
// @Description Actualiza un usuario existente
// @Tags User Controller
// @ID updateUser
// @Accept json
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Param user body models.RegisterDto true "User Update Request"
// @Success 200 {object} models.UserResponseDto "OK"
// @Security bearer-jwt
// @Router /api/user/update/{id} [patch]
func (uc *UserController) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var input models.RegisterDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := uc.service.UpdateUser(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// SearchUser godoc
// @Summary Buscar Usuario
// @Description Busca un usuario en la base de datos por ID o lista todos
// @Tags User Controller
// @ID searchUser
// @Produce json
// @Param id query string false "User UUID" format(uuid)
// @Param dni query string false "User DNI (búsqueda parcial incremental)"
// @Param name query string false "User Name"
// @Param email query string false "User Email"
// @Param phone query string false "User Phone"
// @Param query query string false "Selector de campo o término general"
// @Success 200 {array} models.UserResponseDto "OK"
// @Security bearer-jwt
// @Router /api/user/search [get]
func (uc *UserController) SearchUser(c *gin.Context) {
	id := c.Query("id")
	dni := c.Query("dni")
	name := c.Query("name")
	email := c.Query("email")
	phone := c.Query("phone")
	query := c.Query("query")

	users, err := uc.service.SearchUsers(id, dni, name, email, phone, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetCurrentUser godoc
// @Summary Obtener Usuario Actual
// @Description Obtiene la información del usuario autenticado vía JWT en cabecera Authorization
// @Tags User Controller
// @ID getCurrentUser
// @Produce json
// @Success 200 {object} models.UserResponseDto "OK"
// @Security bearer-jwt
// @Router /api/user/current [get]
func (uc *UserController) GetCurrentUser(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	claims, err := uc.jwtSvc.ValidateToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	user, err := uc.service.GetUserByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Eliminar Usuario
// @Description Elimina un usuario existente
// @Tags User Controller
// @ID deleteUser
// @Param id path string true "User UUID" format(uuid)
// @Success 200 "OK"
// @Security bearer-jwt
// @Router /api/user/delete/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := uc.service.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
