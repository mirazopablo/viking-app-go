package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
)

// BookingService defines business logic for managing bookings and Calendar sync.
type BookingService interface {
	GetAvailability(date string, deviceType string) (*models.AvailabilityResponseDto, error)
	CreateBooking(dto *models.BookingCreateDto) (*models.BookingResponseDto, error)
	GetBookingsByDate(date string) ([]models.Booking, error)
	UpdateBookingStatus(id string, dto *models.UpdateBookingStatusDto) error
	UpdateBotStatus(id string, dto *models.UpdateBotStatusDto) error
	DeleteBooking(id string) error
	ListBlocks() ([]models.BlockResponseDto, error)
	CreateBlock(dto *models.BlockCreateDto) (*models.BlockResponseDto, error)
	DeleteBlock(id string) error
}

type bookingServiceImpl struct {
	repo         repositories.BookingRepository
	calendarProv CalendarProvider
}

// NewBookingService instantiates a new BookingService.
func NewBookingService(repo repositories.BookingRepository, calProv CalendarProvider) BookingService {
	return &bookingServiceImpl{repo: repo, calendarProv: calProv}
}

func getDynamicSlots(isGeneral bool) []models.TimeSlotDto {
	// Generating slots dynamically instead of hardcoding
	var times []string
	if isGeneral {
		times = []string{
			"10:00", "10:30", "11:00", "11:30", "12:00", "12:30",
			"17:00", "17:30", "18:00", "18:30", "19:00", "19:30", "20:00", "20:30",
		}
	} else {
		times = []string{"09:30", "11:00", "17:30", "19:30"}
	}

	slots := make([]models.TimeSlotDto, 0, len(times))
	for _, t := range times {
		parsed, _ := time.Parse("15:04", t)
		id := fmt.Sprintf("slot_%s", parsed.Format("1504"))
		slots = append(slots, models.TimeSlotDto{ID: id, Time: parsed.Format("03:04 PM")})
	}
	return slots
}

func (s *bookingServiceImpl) GetAvailability(date string, deviceType string) (*models.AvailabilityResponseDto, error) {
	cleanDate := extractDate(date)

	blocks, err := s.repo.GetBlocksByDate(cleanDate)
	if err != nil {
		return nil, fmt.Errorf("could not fetch blocks: %w", err)
	}

	for _, block := range blocks {
		if block.IsFullDay {
			return &models.AvailabilityResponseDto{
				Date:           cleanDate,
				AvailableSlots: []models.TimeSlotDto{},
			}, nil
		}
	}

	targetSlots := getDynamicSlots(deviceType == "general")

	// Google Calendar is the single source of truth for availability.
	// The DB is only used to persist client metadata, not to determine slot availability.
	calendarOccupied, err := s.calendarProv.GetOccupiedSlots(cleanDate, targetSlots)
	if err != nil {
		// If Calendar is unreachable, fail loudly so the frontend knows
		return nil, fmt.Errorf("could not verify availability from Calendar Provider: %w", err)
	}

	var availableSlots []models.TimeSlotDto
	for _, slot := range targetSlots {
		sCopy := slot // copy
		sCopy.IsAvailable = !calendarOccupied[sCopy.ID]

		isBlocked := false
		for _, block := range blocks {
			if !block.IsFullDay && block.StartTime != "" && block.EndTime != "" {
				startTimeStr, endTimeStr := generateSlotTimes(cleanDate, slot.ID)
				slotStart, err1 := time.Parse(time.RFC3339, startTimeStr)
				slotEnd, err2 := time.Parse(time.RFC3339, endTimeStr)

				blockStart, err3 := time.Parse(time.RFC3339, fmt.Sprintf("%sT%s:00-03:00", cleanDate, block.StartTime))
				blockEnd, err4 := time.Parse(time.RFC3339, fmt.Sprintf("%sT%s:00-03:00", cleanDate, block.EndTime))

				if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
					if blockStart.Before(slotEnd) && !blockEnd.Before(slotStart) {
						isBlocked = true
						break
					}
				}
			}
		}

		if !isBlocked {
			availableSlots = append(availableSlots, sCopy)
		}
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

	// 2. Call Calendar Provider
	eventID, err := s.calendarProv.InsertEvent(saved, dto.TimeSlotID)
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

func (s *bookingServiceImpl) GetBookingsByDate(date string) ([]models.Booking, error) {
	cleanDate := extractDate(date)
	return s.repo.GetBookingsByDate(cleanDate)
}

func (s *bookingServiceImpl) UpdateBookingStatus(id string, dto *models.UpdateBookingStatusDto) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	err = s.repo.UpdateStatus(id, dto.Status)
	if err != nil {
		return err
	}

	booking.Status = dto.Status
	if booking.GoogleEventID != "" {
		_ = s.calendarProv.UpdateEventStatus(booking)
	}

	return nil
}

func (s *bookingServiceImpl) UpdateBotStatus(id string, dto *models.UpdateBotStatusDto) error {
	if dto.BotActive == nil {
		return errors.New("botActive field is required")
	}
	return s.repo.UpdateBotStatus(id, *dto.BotActive)
}

func (s *bookingServiceImpl) DeleteBooking(id string) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	if booking.GoogleEventID != "" {
		_ = s.calendarProv.DeleteEvent(booking.GoogleEventID)
	}

	return nil
}

func (s *bookingServiceImpl) ListBlocks() ([]models.BlockResponseDto, error) {
	today := time.Now().Format("2006-01-02")
	blocks, err := s.repo.GetBlocksByDateRange(today)
	if err != nil {
		return nil, err
	}

	var response = make([]models.BlockResponseDto, 0)
	for _, b := range blocks {
		response = append(response, models.BlockResponseDto{
			ID:        b.ID,
			Date:      extractDate(b.Date),
			IsFullDay: b.IsFullDay,
			StartTime: b.StartTime,
			EndTime:   b.EndTime,
			Reason:    b.Notes,
			CreatedAt: b.CreatedAt.Format(time.RFC3339),
		})
	}

	return response, nil
}

func (s *bookingServiceImpl) CreateBlock(dto *models.BlockCreateDto) (*models.BlockResponseDto, error) {
	cleanDate := extractDate(dto.Date)

	blockID := uuid.New().String()

	block := &models.Booking{
		ID:         blockID,
		FullName:   "BLOCK",
		Phone:      "-",
		DeviceType: "-",
		Date:       cleanDate,
		TimeSlotID: "BLOCK-" + blockID,
		Notes:      dto.Reason,
		Status:     "confirmed",
		BotActive:  false,
		Type:       "block",
		IsFullDay:  dto.IsFullDay,
		StartTime:  dto.StartTime,
		EndTime:    dto.EndTime,
	}

	saved, err := s.repo.Save(block)
	if err != nil {
		return nil, err
	}

	return &models.BlockResponseDto{
		ID:        saved.ID,
		Date:      saved.Date,
		IsFullDay: saved.IsFullDay,
		StartTime: saved.StartTime,
		EndTime:   saved.EndTime,
		Reason:    saved.Notes,
		CreatedAt: saved.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *bookingServiceImpl) DeleteBlock(id string) error {
	booking, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if booking.Type != "block" {
		return errors.New("cannot delete a regular booking from block endpoint")
	}

	return s.repo.Delete(id)
}

func extractDate(input string) string {
	if len(input) > 10 {
		return input[:10] // Extract YYYY-MM-DD from YYYY-MM-DDTHH:MM:SSZ
	}
	return input
}

// generateSlotTimes parses slot_HHMM and returns RFC3339 start and end strings (30 min duration)
func generateSlotTimes(date string, timeSlotID string) (string, string) {
	if !strings.HasPrefix(timeSlotID, "slot_") || len(timeSlotID) < 9 {
		// Fallback for custom blocks
		if strings.HasPrefix(timeSlotID, "BLOCK-") {
			return fmt.Sprintf("%sT00:00:00-03:00", date), fmt.Sprintf("%sT00:00:00-03:00", date)
		}
		return fmt.Sprintf("%sT10:00:00-03:00", date), fmt.Sprintf("%sT10:30:00-03:00", date)
	}

	hourStr := timeSlotID[5:7]
	minStr := timeSlotID[7:9]
	
	startHour := fmt.Sprintf("%s:%s:00", hourStr, minStr)
	t, _ := time.Parse("15:04:05", startHour)
	endT := t.Add(30 * time.Minute)
	endHour := endT.Format("15:04:05")

	startTimeStr := fmt.Sprintf("%sT%s-03:00", date, startHour)
	endTimeStr := fmt.Sprintf("%sT%s-03:00", date, endHour)

	return startTimeStr, endTimeStr
}
