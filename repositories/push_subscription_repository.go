package repositories

import (
	"errors"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// PushSubscriptionRepository defines data store interactions for web push subscriptions.
type PushSubscriptionRepository interface {
	Save(subscription *models.PushSubscription) (*models.PushSubscription, error)
	FindByWorkOrderID(workOrderID string) ([]models.PushSubscription, error)
	FindByEndpoint(endpoint string) (*models.PushSubscription, error)
	DeleteByEndpoint(endpoint string) error
	DeleteByID(id string) error
}

type pushSubscriptionRepositoryImpl struct {
	db *gorm.DB
}

// NewPushSubscriptionRepository instantiates a new PushSubscriptionRepository with GORM.
func NewPushSubscriptionRepository() PushSubscriptionRepository {
	return &pushSubscriptionRepositoryImpl{db: config.DB}
}

func (r *pushSubscriptionRepositoryImpl) Save(subscription *models.PushSubscription) (*models.PushSubscription, error) {
	// If subscription with same endpoint exists, update keys/workOrderID rather than failing unique constraint
	var existing models.PushSubscription
	err := r.db.Where("endpoint = ?", subscription.Endpoint).First(&existing).Error
	if err == nil {
		existing.WorkOrderID = subscription.WorkOrderID
		existing.P256dh = subscription.P256dh
		existing.Auth = subscription.Auth
		existing.UserAgent = subscription.UserAgent
		if err := r.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	err = r.db.Create(subscription).Error
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (r *pushSubscriptionRepositoryImpl) FindByWorkOrderID(workOrderID string) ([]models.PushSubscription, error) {
	var subscriptions []models.PushSubscription
	err := r.db.Where("work_order_id = ?", workOrderID).Find(&subscriptions).Error
	return subscriptions, err
}

func (r *pushSubscriptionRepositoryImpl) FindByEndpoint(endpoint string) (*models.PushSubscription, error) {
	var subscription models.PushSubscription
	err := r.db.Where("endpoint = ?", endpoint).First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &subscription, nil
}

func (r *pushSubscriptionRepositoryImpl) DeleteByEndpoint(endpoint string) error {
	result := r.db.Where("endpoint = ?", endpoint).Delete(&models.PushSubscription{})
	return result.Error
}

func (r *pushSubscriptionRepositoryImpl) DeleteByID(id string) error {
	result := r.db.Delete(&models.PushSubscription{}, "id = ?", id)
	return result.Error
}
