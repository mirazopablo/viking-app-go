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
	GetBlocksByDate(date string) ([]models.Booking, error)
	GetBlocksByDateRange(startDate string) ([]models.Booking, error)
	FindByID(id string) (*models.Booking, error)
	Update(booking *models.Booking) error
	UpdateStatus(id string, status string) error
	UpdateBotStatus(id string, botActive bool) error
	Delete(id string) error
	GetLatestBookingByPhone(phone string) (*models.Booking, error)
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
	err := r.db.Where("date = ? AND type != ?", date, "block").Find(&bookings).Error
	return bookings, err
}

func (r *bookingRepositoryImpl) GetBlocksByDate(date string) ([]models.Booking, error) {
	var bookings []models.Booking
	err := r.db.Where("date = ? AND type = ?", date, "block").Find(&bookings).Error
	return bookings, err
}

func (r *bookingRepositoryImpl) GetBlocksByDateRange(startDate string) ([]models.Booking, error) {
	var bookings []models.Booking
	err := r.db.Where("date >= ? AND type = ?", startDate, "block").Order("date ASC").Find(&bookings).Error
	return bookings, err
}

func (r *bookingRepositoryImpl) Update(booking *models.Booking) error {
	return r.db.Save(booking).Error
}

func (r *bookingRepositoryImpl) FindByID(id string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.Where("id = ?", id).First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *bookingRepositoryImpl) UpdateStatus(id string, status string) error {
	return r.db.Model(&models.Booking{}).Where("id = ?", id).Update("status", status).Error
}

func (r *bookingRepositoryImpl) UpdateBotStatus(id string, botActive bool) error {
	return r.db.Model(&models.Booking{}).Where("id = ?", id).Update("bot_active", botActive).Error
}

func (r *bookingRepositoryImpl) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Booking{}).Error
}

func (r *bookingRepositoryImpl) GetLatestBookingByPhone(phone string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.Where("phone = ?", phone).Order("created_at desc").First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
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
