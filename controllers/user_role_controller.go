package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/services"
)

// UserRoleController handles role and permission checks for the authenticated user.
type UserRoleController struct {
	roleSvc services.RoleService
}

// NewUserRoleController instantiates a new UserRoleController.
func NewUserRoleController(roleSvc services.RoleService) *UserRoleController {
	return &UserRoleController{roleSvc: roleSvc}
}

// GetUserPermission godoc
// @Summary Verificar si el usuario es admin
// @Description Verifica si el usuario es admin
// @Tags User Role Controller
// @ID getUserPermission
// @Produce json
// @Success 200 {string} string "OK"
// @Security bearer-jwt
// @Router /api/user-roles/user-permission [get]
func (urc *UserRoleController) GetUserPermission(c *gin.Context) {
	roleIDVal, exists := c.Get("roleID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Role ID not found in token context"})
		return
	}

	roleID, ok := roleIDVal.(string)
	if !ok || roleID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid role ID in token context"})
		return
	}

	role, err := urc.roleSvc.GetRoleByID(roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.String(http.StatusOK, role.Name)
}

// IsUserStaff godoc
// @Summary Verificar si el usuario es staff
// @Description Verifica si el usuario es staff
// @Tags User Role Controller
// @ID isUserStaff
// @Produce json
// @Success 200 {boolean} boolean "OK"
// @Security bearer-jwt
// @Router /api/user-roles/is-staff [get]
func (urc *UserRoleController) IsUserStaff(c *gin.Context) {
	roleIDVal, exists := c.Get("roleID")
	if !exists {
		c.JSON(http.StatusOK, false)
		return
	}

	roleID, ok := roleIDVal.(string)
	if !ok || roleID == "" {
		c.JSON(http.StatusOK, false)
		return
	}

	role, err := urc.roleSvc.GetRoleByID(roleID)
	if err != nil {
		c.JSON(http.StatusOK, false)
		return
	}

	desc := strings.ToUpper(strings.TrimSpace(role.Name))
	isStaff := desc == "ADMIN" || desc == "STAFF" || desc == "TECNICO"
	c.JSON(http.StatusOK, isStaff)
}
