package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// RoleController handles role management HTTP requests.
type RoleController struct {
	service services.RoleService
}

// NewRoleController instantiates a new RoleController.
func NewRoleController(service services.RoleService) *RoleController {
	return &RoleController{service: service}
}

// GetAllRoles godoc
// @Summary Listar Roles
// @Description Retorna la lista completa de roles disponibles en el sistema
// @Tags Role Controller
// @ID getAllRoles
// @Produce json
// @Success 200 {array} models.Role "OK"
// @Security bearer-jwt
// @Router /auth/roles [get]
func (rc *RoleController) GetAllRoles(c *gin.Context) {
	roles, err := rc.service.GetAllRoles()
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, roles)
}

// GetRoleByID godoc
// @Summary Buscar Rol
// @Description Busca un rol en la base de datos
// @Tags Role Controller
// @ID getRoleById
// @Produce json
// @Param id path string true "Role UUID" format(uuid)
// @Success 200 {object} models.Role "OK"
// @Security bearer-jwt
// @Router /auth/roles/{id} [get]
func (rc *RoleController) GetRoleByID(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}
	role, err := rc.service.GetRoleByID(id)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Role not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, role)
}

// CreateRole godoc
// @Summary Crear Rol
// @Description Crea un nuevo rol
// @Tags Role Controller
// @ID createRole
// @Accept json
// @Produce json
// @Param role body models.RoleCreateRequest true "Role Create Request"
// @Success 200 {object} models.Role "OK"
// @Security bearer-jwt
// @Router /auth/roles [post]
func (rc *RoleController) CreateRole(c *gin.Context) {
	var input models.RoleCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	createdRole, err := rc.service.CreateRole(&input)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, createdRole)
}

// UpdateRole godoc
// @Summary Actualizar Rol
// @Description Actualiza un rol existente
// @Tags Role Controller
// @ID updateRole
// @Accept json
// @Produce json
// @Param id path string true "Role UUID" format(uuid)
// @Param role body models.RoleCreateRequest true "Role Update Request"
// @Success 200 {object} models.Role "OK"
// @Security bearer-jwt
// @Router /auth/roles/{id} [put]
func (rc *RoleController) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}
	var input models.RoleCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	updatedRole, err := rc.service.UpdateRole(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Role not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, updatedRole)
}

// DeleteRole godoc
// @Summary Eliminar Rol
// @Description Elimina un rol existente
// @Tags Role Controller
// @ID deleteRole
// @Param id path string true "Role UUID" format(uuid)
// @Success 200 "OK"
// @Security bearer-jwt
// @Router /auth/roles/{id} [delete]
func (rc *RoleController) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}
	if err := rc.service.DeleteRole(id); err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Role not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
