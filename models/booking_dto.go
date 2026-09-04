package models

// TimeSlotDto represents a single available or unavailable time slot.
type TimeSlotDto struct {
	ID          string `json:"id"`
	Time        string `json:"time"`
	IsAvailable bool   `json:"isAvailable"`
}

// AvailabilityResponseDto represents the response for the availability endpoint.
type AvailabilityResponseDto struct {
	Date           string        `json:"date"`
	AvailableSlots []TimeSlotDto `json:"availableSlots"`
}

// BookingCreateDto represents the incoming request to create a booking.
type BookingCreateDto struct {
	FullName   string `json:"fullName" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	DeviceType string `json:"deviceType" binding:"required"`
	Date       string `json:"date" binding:"required"` 
	TimeSlotID string `json:"timeSlotId" binding:"required"`
	Notes      string `json:"notes"`
}

// BookingResponseDto represents the response after successfully creating a booking.
type BookingResponseDto struct {
	Message   string `json:"message"`
	BookingID string `json:"bookingId"`
	Status    string `json:"status"`
}

// UpdateBookingStatusDto represents the request payload to update a booking's status.
type UpdateBookingStatusDto struct {
	Status string `json:"status" binding:"required"`
}

// UpdateBotStatusDto represents the request payload to update the bot active flag for a booking.
type UpdateBotStatusDto struct {
	BotActive *bool `json:"botActive" binding:"required"`
}

// BlockCreateDto represents the incoming request to create a block (exception).
type BlockCreateDto struct {
	Date      string `json:"date" binding:"required"`
	IsFullDay bool   `json:"isFullDay"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Reason    string `json:"reason"`
}

// BlockResponseDto represents the response for a block.
type BlockResponseDto struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	IsFullDay bool   `json:"isFullDay"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"createdAt"`
}
