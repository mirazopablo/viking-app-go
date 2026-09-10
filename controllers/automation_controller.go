package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// AutomationController handles internal REST API endpoints for n8n/Evolution automation.
type AutomationController struct {
	bookingService services.BookingService
}

// NewAutomationController instantiates a new AutomationController.
func NewAutomationController(bookingService services.BookingService) *AutomationController {
	return &AutomationController{bookingService: bookingService}
}

// GetTomorrowTasks godoc
// @Summary Obtener los turnos agendados para el día siguiente
// @Description Retorna los turnos de mañana en UTC-3 mapeados para n8n
// @Tags Automation
// @ID getTomorrowTasks
// @Produce json
// @Success 200 {array} models.TomorrowTaskDto
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/automation/tasks/tomorrow [get]
func (ac *AutomationController) GetTomorrowTasks(c *gin.Context) {
	// Calculate tomorrow in UTC-3
	loc, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load timezone"})
		return
	}

	tomorrow := time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	bookings, err := ac.bookingService.GetBookingsByDate(tomorrow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []models.TomorrowTaskDto
	for _, b := range bookings {
		response = append(response, models.TomorrowTaskDto{
			TaskID:      b.ID,
			EventID:     b.GoogleEventID,
			ClientName:  b.FullName,
			Phone:       b.Phone,
			ServiceType: b.DeviceType,
		})
	}

	// If nil, return empty array instead of null
	if response == nil {
		response = make([]models.TomorrowTaskDto, 0)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTaskStatus godoc
// @Summary Actualizar estado del turno desde la automatización
// @Description Actualiza el estado local y sincroniza con Google Calendar
// @Tags Automation
// @ID updateTaskStatus
// @Accept json
// @Produce json
// @Param id path string true "ID del turno (UUID)"
// @Param payload body models.UpdateBookingStatusDto true "Nuevo estado"
// @Success 200 {object} object "OK"
// @Failure 400 {object} object "Bad Request"
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/automation/tasks/{id}/status [patch]
func (ac *AutomationController) UpdateTaskStatus(c *gin.Context) {
	id := c.Param("id")
	
	var dto models.UpdateBookingStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format: " + err.Error()})
		return
	}

	err := ac.bookingService.UpdateBookingStatus(id, &dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task status updated successfully via automation"})
}

// UpdateTaskBotStatus godoc
// @Summary Actualizar estado del bot (Handoff)
// @Description Pausa o reactiva las respuestas automáticas para un turno
// @Tags Automation
// @ID updateTaskBotStatus
// @Accept json
// @Produce json
// @Param id path string true "ID del turno (UUID)"
// @Param payload body models.UpdateBotStatusDto true "Estado del bot (botActive)"
// @Success 200 {object} object "OK"
// @Failure 400 {object} object "Bad Request"
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/automation/tasks/{id}/bot-status [patch]
func (ac *AutomationController) UpdateTaskBotStatus(c *gin.Context) {
	id := c.Param("id")
	
	var dto models.UpdateBotStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format: " + err.Error()})
		return
	}

	err := ac.bookingService.UpdateBotStatus(id, &dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bot status updated successfully via automation"})
}

// DeleteTask godoc
// @Summary Eliminar turno desde automatización
// @Description Elimina el turno en DB y Google Calendar (ej. al cancelar por WhatsApp)
// @Tags Automation
// @ID deleteTask
// @Produce json
// @Param id path string true "ID del turno (UUID)"
// @Success 200 {object} object "OK"
// @Failure 500 {object} object "Internal Server Error"
// @Router /api/v1/automation/tasks/{id} [delete]
func (ac *AutomationController) DeleteTask(c *gin.Context) {
	id := c.Param("id")

	err := ac.bookingService.DeleteBooking(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully via automation"})
}
