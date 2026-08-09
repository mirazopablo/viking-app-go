package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date parameter is required"})
		return
	}
	
	deviceType := c.Query("deviceType")

	response, err := bc.service.GetAvailability(date, deviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
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
// @Failure 409 {object} object "Conflict"
// @Router /api/v1/bookings [post]
func (bc *BookingController) CreateBooking(c *gin.Context) {
	var dto models.BookingCreateDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format: " + err.Error()})
		return
	}

	response, err := bc.service.CreateBooking(&dto)
	if err != nil {
		if strings.Contains(err.Error(), "conflict") {
			c.JSON(http.StatusConflict, gin.H{"error": "El horario ya se encuentra ocupado. Por favor, selecciona otro."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}
