package controllers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// BookingController handles REST API endpoints for bookings.
type BookingController struct {
	service services.BookingService
}

// NewBookingController instantiates a new BookingController.
func NewBookingController(service services.BookingService) *BookingController {
	return &BookingController{service: service}
}

// GetAvailability godoc
// @Summary Obtener slots disponibles
// @Description Retorna los bloques horarios (Time Slots) disponibles para una fecha específica. Para 'general' devuelve 14 slots; para el resto, 4 slots.
// @Tags Bookings
// @ID getAvailability
// @Produce json
// @Param date query string true "Fecha en formato YYYY-MM-DD"
// @Param deviceType query string false "Tipo de equipo/servicio (general | pc | laptop | mobile | gaming)"
// @Success 200 {object} models.AvailabilityResponseDto "OK"
// @Failure 400 {object} object "Bad Request"
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/bookings/availability [get]
func (bc *BookingController) GetAvailability(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		apperrors.BadRequest(c, apperrors.CodeValidationError, "Date parameter is required")
		return
	}
	if !apperrors.ValidateDate(c, date, "date") {
		return
	}

	deviceType := c.Query("deviceType")

	response, err := bc.service.GetAvailability(date, deviceType)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, response)
}

// CreateBooking godoc
// @Summary Crear reserva de turno
// @Description Crea un nuevo turno y reserva el slot en Google Calendar
// @Tags Bookings
// @ID createBooking
// @Accept json
// @Produce json
// @Param booking body models.BookingCreateDto true "Booking Payload"
// @Success 201 {object} models.BookingResponseDto "Created"
// @Failure 409 {object} apperrors.ErrorResponse "Conflict"
// @Router /api/v1/bookings [post]
func (bc *BookingController) CreateBooking(c *gin.Context) {
	var dto models.BookingCreateDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	response, err := bc.service.CreateBooking(&dto)
	if err != nil {
		if strings.Contains(err.Error(), "conflict") {
			apperrors.Conflict(c, apperrors.CodeConflict, "El horario ya se encuentra ocupado. Por favor, selecciona otro.")
			return
		}
		apperrors.HandleError(c, err)
		return
	}

	apperrors.Created(c, response)
}

// GetBookingsByDate godoc
// @Summary Obtener turnos por fecha
// @Description Obtiene la lista de turnos (bookings) para una fecha específica.
// @Tags Bookings
// @ID getBookingsByDate
// @Produce json
// @Param date path string true "Fecha en formato YYYY-MM-DD"
// @Success 200 {array} models.Booking
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/date/{date} [get]
func (bc *BookingController) GetBookingsByDate(c *gin.Context) {
	date := c.Param("date")
	if !apperrors.ValidateDate(c, date, "date") {
		return
	}

	bookings, err := bc.service.GetBookingsByDate(date)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, bookings)
}

// GetTodayBookings godoc
// @Summary Obtener turnos del día actual
// @Description Obtiene la lista de turnos (bookings) para el día de hoy automáticamente calculando la fecha.
// @Tags Bookings
// @ID getTodayBookings
// @Produce json
// @Success 200 {array} models.Booking
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/today [get]
func (bc *BookingController) GetTodayBookings(c *gin.Context) {
	loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	today := time.Now().In(loc).Format("2006-01-02")

	bookings, err := bc.service.GetBookingsByDate(today)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, bookings)
}

// GetTomorrowBookings godoc
// @Summary Obtener turnos del día de mañana
// @Description Obtiene la lista de turnos (bookings) para el día de mañana automáticamente calculando la fecha.
// @Tags Bookings
// @ID getTomorrowBookings
// @Produce json
// @Success 200 {array} models.Booking
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/tomorrow [get]
func (bc *BookingController) GetTomorrowBookings(c *gin.Context) {
	loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires")
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	bookings, err := bc.service.GetBookingsByDate(tomorrow)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}
	apperrors.OK(c, bookings)
}

// UpdateBookingStatus godoc
// @Summary Actualizar estado del turno
// @Description Actualiza el estado del turno tanto en DB como en Google Calendar
// @Tags Bookings
// @ID updateBookingStatus
// @Accept json
// @Produce json
// @Param id path string true "ID del turno (UUID)"
// @Param payload body models.UpdateBookingStatusDto true "Nuevo estado"
// @Success 200 {object} object "OK"
// @Failure 400 {object} object "Bad Request"
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/bookings/{id}/status [patch]
func (bc *BookingController) UpdateBookingStatus(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}

	var dto models.UpdateBookingStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	err := bc.service.UpdateBookingStatus(id, &dto)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, gin.H{"message": "Booking status updated successfully"})
}

// UpdateBotStatus godoc
// @Summary Actualizar estado del bot para Handoff
// @Description Habilita o deshabilita la atención automática por bot (n8n/Evolution API)
// @Tags Bookings
// @ID updateBotStatus
// @Accept json
// @Produce json
// @Param id path string true "ID del turno (UUID)"
// @Param payload body models.UpdateBotStatusDto true "Estado del bot (botActive)"
// @Success 200 {object} object "OK"
// @Failure 400 {object} apperrors.ErrorResponse "Bad Request"
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/{id}/bot-status [patch]
func (bc *BookingController) UpdateBotStatus(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}

	var dto models.UpdateBotStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	err := bc.service.UpdateBotStatus(id, &dto)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, gin.H{"message": "Bot status updated successfully"})
}

// ListBlocks godoc
// @Summary Listar bloqueos activos
// @Description Retorna los bloqueos configurados
// @Tags Bookings (Blocks)
// @ID listBlocks
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BlockResponseDto
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/blocks [get]
func (bc *BookingController) ListBlocks(c *gin.Context) {
	blocks, err := bc.service.ListBlocks()
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, blocks)
}

// CreateBlock godoc
// @Summary Crear nuevo bloqueo
// @Description Crea una excepción de disponibilidad
// @Tags Bookings (Blocks)
// @ID createBlock
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param block body models.BlockCreateDto true "Block Payload"
// @Success 201 {object} models.BlockResponseDto "Created"
// @Failure 400 {object} apperrors.ErrorResponse "Bad Request"
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/blocks [post]
func (bc *BookingController) CreateBlock(c *gin.Context) {
	var dto models.BlockCreateDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		apperrors.ValidationError(c, err)
		return
	}

	response, err := bc.service.CreateBlock(&dto)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.Created(c, response)
}

// DeleteBlock godoc
// @Summary Eliminar bloqueo
// @Description Elimina una excepción de disponibilidad
// @Tags Bookings (Blocks)
// @ID deleteBlock
// @Security BearerAuth
// @Param id path string true "ID del bloqueo (UUID)"
// @Success 200 {object} object "OK"
// @Failure 500 {object} apperrors.ErrorResponse "Internal Server Error"
// @Router /api/v1/bookings/blocks/{id} [delete]
func (bc *BookingController) DeleteBlock(c *gin.Context) {
	id := c.Param("id")
	if !apperrors.ValidateUUID(c, id, "id") {
		return
	}

	err := bc.service.DeleteBlock(id)
	if err != nil {
		apperrors.HandleError(c, err)
		return
	}

	apperrors.OK(c, gin.H{"message": "Block deleted successfully"})
}

// GetClientByPhone godoc
// @Summary Obtener cliente por teléfono
// @Description Obtiene el nombre del cliente basado en su número de teléfono
// @Tags Bookings
// @ID getClientByPhone
// @Produce json
// @Param phone path string true "Número de teléfono"
// @Success 200 {object} object "OK"
// @Failure 404 {object} apperrors.ErrorResponse "Not Found"
// @Router /api/v1/bookings/client/{phone} [get]
func (bc *BookingController) GetClientByPhone(c *gin.Context) {
	phone := c.Param("phone")

	user, err := bc.service.GetClientByPhone(phone)
	if err != nil {
		apperrors.NotFound(c, apperrors.CodeResourceNotFound, "Client not found")
		return
	}

	apperrors.OK(c, gin.H{
		"fullName": user.Name,
		"phone":    user.PhoneNumber,
	})
}
