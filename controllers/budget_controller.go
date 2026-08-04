package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// BudgetController handles REST API endpoints for dynamic budgets.
type BudgetController struct {
	service services.BudgetService
}

// NewBudgetController instantiates a new BudgetController.
func NewBudgetController(service services.BudgetService) *BudgetController {
	return &BudgetController{service: service}
}

// SaveBudget godoc
// @Summary Guardar o actualizar presupuesto
// @Description Almacena o actualiza la estructura JSON de un presupuesto y registra auditoría en diagnostic_points
// @Tags Budget
// @ID saveBudget
// @Accept json
// @Produce json
// @Param budget body models.BudgetSaveDto true "Budget payload"
// @Success 200 {object} models.BudgetResponseDto "OK"
// @Security bearer-jwt
// @Router /api/budget/save [post]
func (bc *BudgetController) SaveBudget(c *gin.Context) {
	var dto models.BudgetSaveDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format: " + err.Error()})
		return
	}

	saved, err := bc.service.SaveBudget(&dto)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated work order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

// GetBudgetByWorkOrder godoc
// @Summary Obtener presupuesto por ID de Orden de Trabajo
// @Description Obtiene la estructura del presupuesto para su renderizado Web en tiempo real
// @Tags Budget
// @ID getBudgetByWorkOrder
// @Produce json
// @Param workOrderId path string true "Work Order UUID" format(uuid)
// @Param public query bool false "Is Public Request"
// @Success 200 {object} models.BudgetResponseDto "OK"
// @Security bearer-jwt
// @Router /api/budget/by-work-order/{workOrderId} [get]
func (bc *BudgetController) GetBudgetByWorkOrder(c *gin.Context) {
	workOrderID := c.Param("workOrderId")
	isPublic := c.Query("public") == "true"

	res, err := bc.service.GetBudgetByWorkOrder(workOrderID, isPublic)
	if err != nil {
		if errors.Is(err, services.ErrBudgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No budget found for this work order"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetPublicBudgetByWorkOrder godoc
// @Summary Obtener presupuesto público por ID de Orden de Trabajo
// @Description Obtiene la estructura desprotegida y sanitizada de un presupuesto para clientes en la vista pública /status
// @Tags Budget
// @ID getPublicBudgetByWorkOrder
// @Produce json
// @Param workOrderId path string true "Work Order UUID" format(uuid)
// @Param securityCode query string false "Security Code (WOVIK-XXXXX)"
// @Success 200 {object} models.BudgetResponseDto "OK"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not Found"
// @Router /public/work-order/budget/{workOrderId} [get]
func (bc *BudgetController) GetPublicBudgetByWorkOrder(c *gin.Context) {
	workOrderID := c.Param("workOrderId")
	securityCode := c.Query("securityCode")

	res, err := bc.service.GetBudgetByWorkOrder(workOrderID, true, securityCode)
	if err != nil {
		if errors.Is(err, services.ErrBudgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found for the specified work order"})
			return
		}
		if errors.Is(err, services.ErrInvalidSecurityCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid security code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateBudgetStatus godoc
// @Summary Actualizar estado del presupuesto
// @Description Permite actualizar el estado del presupuesto (DRAFT, SENT, APPROVED, REJECTED)
// @Tags Budget
// @ID updateBudgetStatus
// @Accept json
// @Produce json
// @Param id path string true "Budget UUID" format(uuid)
// @Param status body models.BudgetStatusUpdateDto true "Status payload"
// @Success 200 {object} models.BudgetResponseDto "OK"
// @Security bearer-jwt
// @Router /api/budget/update-status/{id} [patch]
func (bc *BudgetController) UpdateBudgetStatus(c *gin.Context) {
	id := c.Param("id")

	var dto models.BudgetStatusUpdateDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format: " + err.Error()})
		return
	}

	res, err := bc.service.UpdateBudgetStatus(id, dto.Status)
	if err != nil {
		if errors.Is(err, services.ErrBudgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// DeleteBudget godoc
// @Summary Eliminar presupuesto (Hard Delete)
// @Description Elimina físicamente un presupuesto de la base de datos por su UUID
// @Tags Budget
// @ID deleteBudget
// @Produce json
// @Param id path string true "Budget UUID" format(uuid)
// @Success 200 {object} map[string]string "OK"
// @Failure 404 {object} map[string]string "Not Found"
// @Security bearer-jwt
// @Router /api/budget/delete/{id} [delete]
func (bc *BudgetController) DeleteBudget(c *gin.Context) {
	id := c.Param("id")

	if err := bc.service.DeleteBudget(id); err != nil {
		if errors.Is(err, services.ErrBudgetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}
