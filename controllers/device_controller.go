package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// DeviceController handles CRUD operations for client hardware devices.
type DeviceController struct {
	service services.DeviceService
}

// NewDeviceController instantiates a new DeviceController.
func NewDeviceController(service services.DeviceService) *DeviceController {
	return &DeviceController{service: service}
}

// RegisterDevice godoc
// @Summary Guardar Dispositivos
// @Description Guarda un dispositivo en la base de datos
// @Tags Device Controller
// @ID registerDevice
// @Accept json
// @Produce json
// @Param device body models.DeviceCreateRequestDto true "Device Create Request"
// @Success 200 {object} models.DeviceResponseDto "OK"
// @Security bearer-jwt
// @Router /api/device/save [post]
func (dc *DeviceController) RegisterDevice(c *gin.Context) {
	var input models.DeviceCreateRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	saved, err := dc.service.RegisterDevice(&input)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Associated user not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, saved)
}

// UpdateDevice godoc
// @Summary Actualizar Dispositivo (Soporta PATCH parcial para reasignación de dueño)
// @Description Actualiza un dispositivo existente de forma parcial o total
// @Tags Device Controller
// @ID updateDevice
// @Accept json
// @Produce json
// @Param id path string true "Device UUID" format(uuid)
// @Param device body models.DeviceUpdateRequestDto true "Device Update Request"
// @Success 200 {object} models.DeviceResponseDto "OK"
// @Security bearer-jwt
// @Router /api/device/update/{id} [patch]
func (dc *DeviceController) UpdateDevice(c *gin.Context) {
	id := c.Param("id")
	var input models.DeviceUpdateRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	updated, err := dc.service.UpdateDevice(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Device not found")
			return
		}
		if errors.Is(err, services.ErrUserNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Associated user not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, updated)
}

// SearchDevice godoc
// @Summary Buscar Dispositivos
// @Description Busca dispositivos en la base de datos
// @Tags Device Controller
// @ID searchDevice
// @Produce json
// @Param id query string false "Device UUID" format(uuid)
// @Param serialNumber query string false "Serial Number"
// @Param brand query string false "Brand"
// @Param userDni query string false "User DNI (búsqueda parcial incremental)"
// @Param userName query string false "User Name (búsqueda parcial incremental)"
// @Param query query string false "Selector de campo o término general"
// @Success 200 {array} models.DeviceResponseDto "OK"
// @Security bearer-jwt
// @Router /api/device/search [get]
func (dc *DeviceController) SearchDevice(c *gin.Context) {
	id := c.Query("id")
	serialNumber := c.Query("serialNumber")
	brand := c.Query("brand")
	userDni := c.Query("userDni")
	userName := c.Query("userName")
	queryStr := c.Query("query")

	devices, err := dc.service.SearchDevices(id, serialNumber, brand, userDni, userName, queryStr)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, devices)
}

// DeleteDevice godoc
// @Summary Eliminar Dispositivo
// @Description Elimina un dispositivo de la base de datos
// @Tags Device Controller
// @ID deleteDevice
// @Param id path string true "Device UUID" format(uuid)
// @Success 200 "OK"
// @Security bearer-jwt
// @Router /api/device/delete/{id} [delete]
func (dc *DeviceController) DeleteDevice(c *gin.Context) {
	id := c.Param("id")
	if err := dc.service.DeleteDevice(id); err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Device not found")
			return
		}
		apperrors.HandleError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
