package repositories

import (
	"errors"
	"strings"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

var (
	// ErrSlotTaken is thrown when a unique constraint violation occurs for a date + time_slot_id
	ErrSlotTaken = errors.New("time slot already booked for this date")
)

// BookingRepository defines data store interactions for bookings.
type BookingRepository interface {
	Save(booking *models.Booking) (*models.Booking, error)
	GetBookingsByDate(date string) ([]models.Booking, error)
	Update(booking *models.Booking) error
}

type bookingRepositoryImpl struct {
	db *gorm.DB
}

// NewBookingRepository instantiates a new BookingRepository with GORM.
func NewBookingRepository() BookingRepository {
	return &bookingRepositoryImpl{db: config.DB}
}

func (r *bookingRepositoryImpl) Save(booking *models.Booking) (*models.Booking, error) {
	err := r.db.Create(booking).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || containsDuplicateKeyError(err) {
			return nil, ErrSlotTaken
		}
		return nil, err
	}
	return booking, nil
}

func (r *bookingRepositoryImpl) GetBookingsByDate(date string) ([]models.Booking, error) {
	var bookings []models.Booking
	err := r.db.Where("date = ?", date).Find(&bookings).Error
	return bookings, err
}

func (r *bookingRepositoryImpl) Update(booking *models.Booking) error {
	return r.db.Save(booking).Error
}

func containsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}
