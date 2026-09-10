package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/mirazopablo/viking-app-go/models"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarProvider defines the interface for calendar operations
type CalendarProvider interface {
	GetOccupiedSlots(date string, targetSlots []models.TimeSlotDto) (map[string]bool, error)
	InsertEvent(booking *models.Booking, timeSlotID string) (string, error)
	UpdateEventStatus(booking *models.Booking) error
	DeleteEvent(eventID string) error
}

type googleCalendarProvider struct {
	client     *calendar.Service
	calendarID string
	timeZone   string
}

// NewGoogleCalendarProvider initializes the calendar service once
func NewGoogleCalendarProvider() (CalendarProvider, error) {
	ctx := context.Background()
	b, err := loadGoogleCredentials()
	if err != nil {
		return nil, fmt.Errorf("could not load credentials: %w", err)
	}

	config, err := google.JWTConfigFromJSON(b, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("could not parse credentials: %v", err)
	}
	httpClient := config.Client(ctx)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve calendar client: %v", err)
	}

	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		calendarID = "primary"
	}

	timeZone := os.Getenv("TIMEZONE")
	if timeZone == "" {
		timeZone = "America/Argentina/Buenos_Aires"
	}

	return &googleCalendarProvider{
		client:     srv,
		calendarID: calendarID,
		timeZone:   timeZone,
	}, nil
}

func loadGoogleCredentials() ([]byte, error) {
	if encoded := os.Getenv("GOOGLE_CREDENTIALS_JSON"); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("GOOGLE_CREDENTIALS_JSON is not valid base64: %w", err)
		}
		return decoded, nil
	}
	return os.ReadFile("credentials.json")
}

func (p *googleCalendarProvider) GetOccupiedSlots(date string, targetSlots []models.TimeSlotDto) (map[string]bool, error) {
	// We use the fixed -03:00 offset here to match DB times in the original logic, 
	// or we can pass proper UTC depending on requirements.
	timeMin := fmt.Sprintf("%sT00:00:00-03:00", date)
	timeMax := fmt.Sprintf("%sT23:59:59-03:00", date)

	events, err := p.client.Events.List(p.calendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		Do()

	if err != nil {
		return nil, err
	}

	occupied := make(map[string]bool)

	for _, item := range events.Items {
		if item.Start == nil || item.End == nil || item.Start.DateTime == "" {
			continue // skip full day events or invalid ones
		}

		eventStart, err1 := time.Parse(time.RFC3339, item.Start.DateTime)
		eventEnd, err2 := time.Parse(time.RFC3339, item.End.DateTime)
		if err1 != nil || err2 != nil {
			continue
		}

		for _, slot := range targetSlots {
			startTimeStr, endTimeStr := generateSlotTimes(date, slot.ID)
			slotStart, err1 := time.Parse(time.RFC3339, startTimeStr)
			slotEnd, err2 := time.Parse(time.RFC3339, endTimeStr)
			if err1 != nil || err2 != nil {
				continue
			}

			// Overlap condition: max(StartA, StartB) < min(EndA, EndB)
			if eventStart.Before(slotEnd) && eventEnd.After(slotStart) {
				occupied[slot.ID] = true
			}
		}
	}

	return occupied, nil
}

func (p *googleCalendarProvider) InsertEvent(booking *models.Booking, timeSlotID string) (string, error) {
	startTime, endTime := generateSlotTimes(booking.Date, timeSlotID)

	event := &calendar.Event{
		Summary:     fmt.Sprintf("Turno: %s - %s", booking.DeviceType, booking.FullName),
		Description: fmt.Sprintf("Phone: %s\nNotes: %s", booking.Phone, booking.Notes),
		Start: &calendar.EventDateTime{
			DateTime: startTime,
			TimeZone: p.timeZone,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
			TimeZone: p.timeZone,
		},
		Reminders: &calendar.EventReminders{
			UseDefault:      false,
			ForceSendFields: []string{"UseDefault"},
			Overrides: []*calendar.EventReminder{
				{Method: "popup", Minutes: 60},
			},
		},
	}

	event, err := p.client.Events.Insert(p.calendarID, event).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create event: %v", err)
	}

	return event.Id, nil
}

func (p *googleCalendarProvider) UpdateEventStatus(booking *models.Booking) error {
	event, err := p.client.Events.Get(p.calendarID, booking.GoogleEventID).Do()
	if err != nil {
		return err
	}

	event.Summary = fmt.Sprintf("[%s] Turno: %s - %s", booking.Status, booking.DeviceType, booking.FullName)
	_, err = p.client.Events.Update(p.calendarID, booking.GoogleEventID, event).Do()
	return err
}

func (p *googleCalendarProvider) DeleteEvent(eventID string) error {
	return p.client.Events.Delete(p.calendarID, eventID).Do()
}
