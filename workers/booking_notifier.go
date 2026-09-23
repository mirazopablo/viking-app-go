package workers

import (
	"fmt"
	"log"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"github.com/mirazopablo/viking-app-go/services"
	"github.com/robfig/cron/v3"
)

var argTimezone *time.Location

func init() {
	loc, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		log.Printf("[workers/booking_notifier.go] [init] Error al cargar la zona horaria: %v", err)
		argTimezone = time.Local
	} else {
		argTimezone = loc
	}
}

// StartCronJobs initializes and starts all background scheduled tasks.
func StartCronJobs(deviceTokenRepo repositories.DeviceTokenRepository) {
	c := cron.New(cron.WithLocation(argTimezone))

	// FASE 1: Resumen Diario a las 20:00 hs
	_, err := c.AddFunc("0 20 * * *", func() {
		notifyTomorrowBookings(deviceTokenRepo)
	})
	if err != nil {
		log.Printf("[workers/booking_notifier.go] [StartCronJobs] Error al programar la Fase 1: %v", err)
	}

	c.Start()
}

func notifyTomorrowBookings(tokenRepo repositories.DeviceTokenRepository) {
	nowArg := time.Now().In(argTimezone)
	tomorrowDate := nowArg.AddDate(0, 0, 1).Format("2006-01-02")

	var count int64
	err := config.DB.Model(&models.Booking{}).
		Where("date = ? AND type = ?", tomorrowDate, "booking").
		Count(&count).Error

	if err != nil {
		log.Printf("[workers/booking_notifier.go] [notifyTomorrowBookings] Error al contar los turnos de mañana: %v", err)
		return
	}

	if count == 0 {
		return
	}

	tokens, err := tokenRepo.GetAllTokens()
	if err != nil {
		log.Printf("[workers/booking_notifier.go] [notifyTomorrowBookings] Error al obtener los tokens de los dispositivos: %v", err)
		return
	}
	
	if len(tokens) == 0 {
		return
	}

	title := "Resumen de Agenda 📅"
	body := fmt.Sprintf("Mañana tienes %d turno(s) agendado(s). Revisa la app para más detalles.", count)
	
	services.SendMulticastPushNotification(tokens, title, body)
}
