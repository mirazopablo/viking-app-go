package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// WorkOrderController handles repair job endpoints.
type WorkOrderController struct {
	service services.WorkOrderService
}

// NewWorkOrderController instantiates a new WorkOrderController.
func NewWorkOrderController(service services.WorkOrderService) *WorkOrderController {
	return &WorkOrderController{service: service}
}

// SaveWorkOrder godoc
// @Summary Crear Orden de Trabajo
// @Description Genera una nueva orden uniendo clientId, deviceId, descripción del problema y estado inicial
// @Tags Work Order Controller
// @ID saveWorkOrder
// @Accept json
// @Produce json
// @Param workOrder body models.WorkOrderCreateRequest true "Work Order Create Request"
// @Success 200 {object} models.WorkOrderResponseDto "OK"
// @Security bearer-jwt
// @Router /api/work-order/save [post]
func (woc *WorkOrderController) SaveWorkOrder(c *gin.Context) {
	var input models.WorkOrderCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var staffID string
	if userIDVal, exists := c.Get("userID"); exists {
		if uid, ok := userIDVal.(uuid.UUID); ok {
			staffID = uid.String()
		} else if uStr, ok := userIDVal.(string); ok {
			staffID = uStr
		}
	}

	saved, err := woc.service.CreateWorkOrder(&input, staffID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated client not found"})
			return
		}
		if errors.Is(err, services.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated device not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

// UpdateWorkOrderStatus godoc
// @Summary Actualizar Estado de Orden de Trabajo
// @Description Cambia el estado de reparación de una orden (RECEIVED -> IN_PROGRESS -> DONE)
// @Tags Work Order Controller
// @ID updateWorkOrderStatus
// @Accept json
// @Produce json
// @Param orderId path string true "Work Order UUID" format(uuid)
// @Param status body models.WorkOrderStatusUpdateRequestDto true "Status Update Request"
// @Success 200 {object} models.WorkOrderResponseDto "OK"
// @Security bearer-jwt
// @Router /api/work-order/update/{orderId} [patch]
func (woc *WorkOrderController) UpdateWorkOrderStatus(c *gin.Context) {
	id := c.Param("orderId")
	var input models.WorkOrderStatusUpdateRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := woc.service.UpdateWorkOrderStatus(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// SearchWorkOrder godoc
// @Summary Buscar Órdenes de Trabajo
// @Description Permite rastrear reparaciones por staffId, clientDni, deviceSerialNumber o término general
// @Tags Work Order Controller
// @ID searchWorkOrder
// @Produce json
// @Param staffId query string false "Staff UUID" format(uuid)
// @Param clientDni query integer false "Client DNI" format(int32)
// @Param deviceSerialNumber query string false "Device Serial Number"
// @Param query query string false "General Search Term"
// @Success 200 {array} models.WorkOrderResponseDto "OK"
// @Security bearer-jwt
// @Router /api/work-order/search [get]
func (woc *WorkOrderController) SearchWorkOrder(c *gin.Context) {
	staffId := c.Query("staffId")
	deviceSerialNumber := c.Query("deviceSerialNumber")
	queryStr := c.Query("query")

	var clientDni int32
	if dniStr := c.Query("clientDni"); dniStr != "" {
		if val, err := strconv.Atoi(dniStr); err == nil {
			clientDni = int32(val)
		}
	}

	orders, err := woc.service.SearchWorkOrders(staffId, clientDni, deviceSerialNumber, queryStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// DeleteWorkOrder godoc
// @Summary Eliminar Orden de Trabajo
// @Description Borra o cancela una orden de trabajo por UUID
// @Tags Work Order Controller
// @ID deleteWorkOrder
// @Param orderId path string true "Work Order UUID" format(uuid)
// @Success 200 "OK"
// @Security bearer-jwt
// @Router /api/work-order/delete/{orderId} [delete]
func (woc *WorkOrderController) DeleteWorkOrder(c *gin.Context) {
	id := c.Param("orderId")
	if err := woc.service.DeleteWorkOrder(id); err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Work order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
