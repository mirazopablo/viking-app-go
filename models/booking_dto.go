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
