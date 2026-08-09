package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// BookingService defines business logic for managing bookings and Calendar sync.
type BookingService interface {
	GetAvailability(date string, deviceType string) (*models.AvailabilityResponseDto, error)
	CreateBooking(dto *models.BookingCreateDto) (*models.BookingResponseDto, error)
}

type bookingServiceImpl struct {
	repo repositories.BookingRepository
}

// NewBookingService instantiates a new BookingService.
func NewBookingService(repo repositories.BookingRepository) BookingService {
	return &bookingServiceImpl{repo: repo}
}

var generalSlots = []models.TimeSlotDto{
	{ID: "slot_1000", Time: "10:00 AM"},
	{ID: "slot_1030", Time: "10:30 AM"},
	{ID: "slot_1100", Time: "11:00 AM"},
	{ID: "slot_1130", Time: "11:30 AM"},
	{ID: "slot_1200", Time: "12:00 PM"},
	{ID: "slot_1230", Time: "12:30 PM"},
	{ID: "slot_1700", Time: "05:00 PM"},
	{ID: "slot_1730", Time: "05:30 PM"},
	{ID: "slot_1800", Time: "06:00 PM"},
	{ID: "slot_1830", Time: "06:30 PM"},
	{ID: "slot_1900", Time: "07:00 PM"},
	{ID: "slot_1930", Time: "07:30 PM"},
	{ID: "slot_2000", Time: "08:00 PM"},
	{ID: "slot_2030", Time: "08:30 PM"},
}

var reducedSlots = []models.TimeSlotDto{
	{ID: "slot_0930", Time: "09:30 AM"},
	{ID: "slot_1100", Time: "11:00 AM"},
	{ID: "slot_1730", Time: "05:30 PM"},
	{ID: "slot_1930", Time: "07:30 PM"},
}

func (s *bookingServiceImpl) GetAvailability(date string, deviceType string) (*models.AvailabilityResponseDto, error) {
	cleanDate := extractDate(date)

	var targetSlots []models.TimeSlotDto
	if deviceType == "general" {
		targetSlots = generalSlots
	} else {
		targetSlots = reducedSlots
	}

	// Google Calendar is the single source of truth for availability.
	// The DB is only used to persist client metadata, not to determine slot availability.
	calendarOccupied, err := getOccupiedSlotsFromCalendar(cleanDate, targetSlots)
	if err != nil {
		// If Calendar is unreachable, fail loudly so the frontend knows
		return nil, fmt.Errorf("could not verify availability from Google Calendar: %w", err)
	}

	var availableSlots []models.TimeSlotDto
	for _, slot := range targetSlots {
		s := slot // copy
		s.IsAvailable = !calendarOccupied[s.ID]
		availableSlots = append(availableSlots, s)
	}

	return &models.AvailabilityResponseDto{
		Date:           cleanDate,
		AvailableSlots: availableSlots,
	}, nil
}

func (s *bookingServiceImpl) CreateBooking(dto *models.BookingCreateDto) (*models.BookingResponseDto, error) {
	cleanDate := extractDate(dto.Date)
	
	booking := &models.Booking{
		FullName:   dto.FullName,
		Phone:      dto.Phone,
		DeviceType: dto.DeviceType,
		Date:       cleanDate,
		TimeSlotID: dto.TimeSlotID,
		Notes:      dto.Notes,
		Status:     "pending_confirmation",
	}

	// 1. Save in DB (Locks by UniqueIndex constraint)
	saved, err := s.repo.Save(booking)
	if err != nil {
		if errors.Is(err, repositories.ErrSlotTaken) {
			return nil, fmt.Errorf("conflict: %w", err)
		}
		return nil, err
	}

	// 2. Call Google Calendar
	eventID, err := insertToGoogleCalendar(saved, dto.TimeSlotID)
	if err == nil && eventID != "" {
		saved.GoogleEventID = eventID
		_ = s.repo.Update(saved) // Update silently
	}

	return &models.BookingResponseDto{
		Message:   "Booking created successfully",
		BookingID: saved.ID,
		Status:    saved.Status,
	}, nil
}

func extractDate(input string) string {
	if len(input) > 10 {
		return input[:10] // Extract YYYY-MM-DD from YYYY-MM-DDTHH:MM:SSZ
	}
	return input
}

// loadGoogleCredentials loads the service account JSON from an environment
// variable (base64-encoded) in production, falling back to a local file for
// development. The two variables are:
//   - GOOGLE_CREDENTIALS_JSON  → base64 of the full credentials.json content
//   - GOOGLE_CALENDAR_ID       → target calendar (e.g. your personal gmail)
func loadGoogleCredentials() ([]byte, error) {
	if encoded := os.Getenv("GOOGLE_CREDENTIALS_JSON"); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("GOOGLE_CREDENTIALS_JSON is not valid base64: %w", err)
		}
		return decoded, nil
	}
	// Local development fallback
	return os.ReadFile("credentials.json")
}

func getOccupiedSlotsFromCalendar(date string, targetSlots []models.TimeSlotDto) (map[string]bool, error) {
	ctx := context.Background()
	b, err := loadGoogleCredentials()
	if err != nil {
		return nil, err
	}

	config, err := google.JWTConfigFromJSON(b, calendar.CalendarEventsReadonlyScope)
	if err != nil {
		return nil, err
	}
	client := config.Client(ctx)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		calendarID = "primary"
	}

	timeMin := fmt.Sprintf("%sT00:00:00-03:00", date)
	timeMax := fmt.Sprintf("%sT23:59:59-03:00", date)

	events, err := srv.Events.List(calendarID).
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
			startTimeStr, endTimeStr := getSlotTimes(date, slot.ID)
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

func insertToGoogleCalendar(booking *models.Booking, timeSlotID string) (string, error) {
	ctx := context.Background()
	b, err := loadGoogleCredentials()
	if err != nil {
		return "", fmt.Errorf("could not load credentials: %w", err)
	}

	// Parse JSON config to get JWT
	config, err := google.JWTConfigFromJSON(b, calendar.CalendarEventsScope)
	if err != nil {
		return "", fmt.Errorf("could not parse credentials: %v", err)
	}
	client := config.Client(ctx)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve calendar client: %v", err)
	}

	startTime, endTime := getSlotTimes(booking.Date, timeSlotID)

	event := &calendar.Event{
		Summary:     fmt.Sprintf("Turno: %s - %s", booking.DeviceType, booking.FullName),
		Description: fmt.Sprintf("Phone: %s\nNotes: %s", booking.Phone, booking.Notes),
		Start: &calendar.EventDateTime{
			DateTime: startTime,
			TimeZone: "America/Argentina/Buenos_Aires",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
			TimeZone: "America/Argentina/Buenos_Aires",
		},
	}

	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		calendarID = "primary"
	}

	event, err = srv.Events.Insert(calendarID, event).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create event: %v", err)
	}

	return event.Id, nil
}

func getSlotTimes(date string, timeSlotID string) (string, string) {
	hourMap := map[string]string{
		"slot_0930": "09:30:00",
		"slot_1000": "10:00:00",
		"slot_1030": "10:30:00",
		"slot_1100": "11:00:00",
		"slot_1130": "11:30:00",
		"slot_1200": "12:00:00",
		"slot_1230": "12:30:00",
		"slot_1700": "17:00:00",
		"slot_1730": "17:30:00",
		"slot_1800": "18:00:00",
		"slot_1830": "18:30:00",
		"slot_1900": "19:00:00",
		"slot_1930": "19:30:00",
		"slot_2000": "20:00:00",
		"slot_2030": "20:30:00",
	}

	startHour := hourMap[timeSlotID]
	if startHour == "" {
		startHour = "10:00:00" // Default fallback
	}

	// Assuming 30 minutes slots
	t, _ := time.Parse("15:04:05", startHour)
	endT := t.Add(30 * time.Minute)
	endHour := endT.Format("15:04:05")

	// Format: 2026-08-10T09:00:00-03:00
	startTimeStr := fmt.Sprintf("%sT%s-03:00", date, startHour)
	endTimeStr := fmt.Sprintf("%sT%s-03:00", date, endHour)

	return startTimeStr, endTimeStr
}
