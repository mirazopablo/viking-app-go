package repositories

import (
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm/clause"
)

type DeviceTokenRepository interface {
	SaveToken(token *models.DeviceToken) error
	DeleteToken(fcmToken string) error
	GetAllTokens() ([]string, error)
}

type deviceTokenRepository struct{}

func NewDeviceTokenRepository() DeviceTokenRepository {
	return &deviceTokenRepository{}
}

func (r *deviceTokenRepository) SaveToken(token *models.DeviceToken) error {
	// Usamos OnConflict para que si el token ya existe, simplemente actualice el userID y la fecha.
	return config.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fcm_token"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "device", "updated_at"}),
	}).Create(token).Error
}

func (r *deviceTokenRepository) DeleteToken(fcmToken string) error {
	return config.DB.Where("fcm_token = ?", fcmToken).Delete(&models.DeviceToken{}).Error
}

func (r *deviceTokenRepository) GetAllTokens() ([]string, error) {
	var tokens []string
	err := config.DB.Model(&models.DeviceToken{}).Pluck("fcm_token", &tokens).Error
	return tokens, err
}
