package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	saved, err := dc.service.RegisterDevice(&input)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

// UpdateDevice godoc
// @Summary Actualizar Dispositivo
// @Description Actualiza un dispositivo existente
// @Tags Device Controller
// @ID updateDevice
// @Accept json
// @Produce json
// @Param id path string true "Device UUID" format(uuid)
// @Param device body models.DeviceUpdateRequestDto true "Device Update Request"
// @Success 200 {object} models.DeviceResponseDto "OK"
// @Security bearer-jwt
// @Router /api/device/update/{id} [put]
func (dc *DeviceController) UpdateDevice(c *gin.Context) {
	id := c.Param("id")
	var input models.DeviceUpdateRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := dc.service.UpdateDevice(id, &input)
	if err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
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
// @Param userDni query integer false "User DNI" format(int32)
// @Param query query string false "General Search Term"
// @Success 200 {array} models.DeviceResponseDto "OK"
// @Security bearer-jwt
// @Router /api/device/search [get]
func (dc *DeviceController) SearchDevice(c *gin.Context) {
	id := c.Query("id")
	serialNumber := c.Query("serialNumber")
	brand := c.Query("brand")
	queryStr := c.Query("query")

	var userDni int32
	if dniStr := c.Query("userDni"); dniStr != "" {
		if val, err := strconv.Atoi(dniStr); err == nil {
			userDni = int32(val)
		}
	}

	devices, err := dc.service.SearchDevices(id, serialNumber, brand, userDni, queryStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
