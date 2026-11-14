package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
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
		apperrors.Unauthorized(c, apperrors.CodeAuthTokenInvalid, "Role ID not found in token context")
		return
	}

	roleID, ok := roleIDVal.(string)
	if !ok || roleID == "" {
		apperrors.Unauthorized(c, apperrors.CodeAuthTokenInvalid, "Invalid role ID in token context")
		return
	}

	role, err := urc.roleSvc.GetRoleByID(roleID)
	if err != nil {
		apperrors.HandleError(c, err)
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
		apperrors.OK(c, false)
		return
	}

	roleID, ok := roleIDVal.(string)
	if !ok || roleID == "" {
		apperrors.OK(c, false)
		return
	}

	role, err := urc.roleSvc.GetRoleByID(roleID)
	if err != nil {
		apperrors.OK(c, false)
		return
	}

	desc := strings.ToUpper(strings.TrimSpace(role.Name))
	isStaff := desc == "ADMIN" || desc == "STAFF" || desc == "TECNICO"
	apperrors.OK(c, isStaff)
}
