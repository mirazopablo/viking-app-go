package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
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
		apperrors.ValidationError(c, err)
		return
	}

	_, err := uc.service.RegisterUser(&input)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, gin.H{"message": "User saved successfully"})
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
		apperrors.ValidationError(c, err)
		return
	}

	updated, err := uc.service.UpdateUser(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "User not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, updated)
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit < 1 {
		limit = 100
	}

	users, err := uc.service.SearchUsers(id, dni, name, email, phone, query, page, limit)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, users)
}

// AutocompleteUser godoc
// @Summary Autocompletado Rápido de Usuario / Cliente
// @Description Devuelve una proyección ligera de usuarios para selectores sin exponer PII sensible
// @Tags User Controller
// @ID autocompleteUser
// @Produce json
// @Param query query string false "Término de búsqueda (nombre, DNI o teléfono)"
// @Success 200 {array} models.UserAutocompleteDto "OK"
// @Security bearer-jwt
// @Router /api/user/autocomplete [get]
func (uc *UserController) AutocompleteUser(c *gin.Context) {
	query := c.Query("query")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}

	users, err := uc.service.AutocompleteUsers(query, limit)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, users)
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
	// The AuthMiddleware already extracted the claims and set the "userID" in context
	userID, exists := c.Get("userID")
	if !exists {
		apperrors.Unauthorized(c, apperrors.CodeAuthTokenInvalid, "Unauthorized")
		return
	}

	user, err := uc.service.GetUserByID(userID.(string))
	if err != nil {
		apperrors.NotFound(c, apperrors.CodeResourceNotFound, "User not found")
		return
	}

	apperrors.OK(c, user)
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
		apperrors.HandleError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
