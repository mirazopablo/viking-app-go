package repositories

import (
	"errors"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// NotificationHistoryRepository defines data store interactions for notification history logs.
type NotificationHistoryRepository interface {
	Save(notification *models.NotificationHistory) (*models.NotificationHistory, error)
	FindByWorkOrderID(workOrderID string) ([]models.NotificationHistory, error)
	FindUnreadByWorkOrderID(workOrderID string) ([]models.NotificationHistory, error)
	MarkAsRead(id string) error
	MarkAllAsReadByWorkOrderID(workOrderID string) error
}

type notificationHistoryRepositoryImpl struct {
	db *gorm.DB
}

// NewNotificationHistoryRepository instantiates a new NotificationHistoryRepository with GORM.
func NewNotificationHistoryRepository() NotificationHistoryRepository {
	return &notificationHistoryRepositoryImpl{db: config.DB}
}

func (r *notificationHistoryRepositoryImpl) Save(notification *models.NotificationHistory) (*models.NotificationHistory, error) {
	err := r.db.Create(notification).Error
	if err != nil {
		return nil, err
	}
	return notification, nil
}

func (r *notificationHistoryRepositoryImpl) FindByWorkOrderID(workOrderID string) ([]models.NotificationHistory, error) {
	var history []models.NotificationHistory
	err := r.db.Where("work_order_id = ?", workOrderID).Order("created_at DESC").Find(&history).Error
	return history, err
}

func (r *notificationHistoryRepositoryImpl) FindUnreadByWorkOrderID(workOrderID string) ([]models.NotificationHistory, error) {
	var history []models.NotificationHistory
	err := r.db.Where("work_order_id = ? AND is_read = ?", workOrderID, false).Order("created_at DESC").Find(&history).Error
	return history, err
}

func (r *notificationHistoryRepositoryImpl) MarkAsRead(id string) error {
	result := r.db.Model(&models.NotificationHistory{}).Where("id = ?", id).Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("notification record not found")
	}
	return nil
}

func (r *notificationHistoryRepositoryImpl) MarkAllAsReadByWorkOrderID(workOrderID string) error {
	return r.db.Model(&models.NotificationHistory{}).Where("work_order_id = ? AND is_read = ?", workOrderID, false).Update("is_read", true).Error
}
