package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Booking represents an appointment in the system and Google Calendar.
type Booking struct {
	ID            string    `gorm:"type:uuid;primary_key;" json:"id"`
	FullName      string    `gorm:"type:varchar(255);not null" json:"fullName"`
	Phone         string    `gorm:"type:varchar(50);not null" json:"phone"`
	DeviceType    string    `gorm:"type:varchar(100);not null" json:"deviceType"`
	Date          string    `gorm:"type:date;not null;uniqueIndex:idx_date_timeslot" json:"date"`
	TimeSlotID    string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_date_timeslot" json:"timeSlotId"`
	Notes         string    `gorm:"type:text" json:"notes"`
	GoogleEventID string    `gorm:"type:varchar(255)" json:"googleEventId"`
	Status        string    `gorm:"type:varchar(50);not null;default:'pending_confirmation'" json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// BeforeCreate hook to set UUID prior to database insertion.
func (b *Booking) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return
}
