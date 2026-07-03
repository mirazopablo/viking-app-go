package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
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
	role, err := rc.service.GetRoleByID(id)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdRole, err := rc.service.CreateRole(&input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, createdRole)
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
	var input models.RoleCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedRole, err := rc.service.UpdateRole(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedRole)
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
	if err := rc.service.DeleteRole(id); err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
