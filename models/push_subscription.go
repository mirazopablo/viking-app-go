package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PushSubscription represents a web browser WebPush subscription linked to a specific work order.
// Rationale: Storing Endpoint along with P256dh and Auth keys allows sending encrypted payloads using VAPID (RFC 8292).
type PushSubscription struct {
	ID          string    `gorm:"type:uuid;primary_key;" json:"id"`
	WorkOrderID string    `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"workOrderId"`
	Endpoint    string    `gorm:"type:text;not null;uniqueIndex" json:"endpoint"`
	P256dh      string    `gorm:"type:varchar(255);not null;column:p256dh" json:"p256dh"`
	Auth        string    `gorm:"type:varchar(255);not null;column:auth" json:"auth"`
	UserAgent   string    `gorm:"type:varchar(255);" json:"userAgent,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// BeforeCreate hooks into GORM to generate a unique UUID prior to database insertion.
func (ps *PushSubscription) BeforeCreate(tx *gorm.DB) (err error) {
	if ps.ID == "" {
		ps.ID = uuid.New().String()
	}
	return
}

// PushSubscriptionCreateRequestDto defines the JSON payload required from the PWA to subscribe.
type PushSubscriptionCreateRequestDto struct {
	WorkOrderID  string `json:"workOrderId" binding:"required,uuid"`
	SecurityCode string `json:"securityCode" binding:"required"`
	Endpoint     string `json:"endpoint" binding:"required,url"`
	Keys         struct {
		P256dh string `json:"p256dh" binding:"required"`
		Auth   string `json:"auth" binding:"required"`
	} `json:"keys" binding:"required"`
	UserAgent string `json:"userAgent"`
}

// PushSubscriptionUnsubscribeRequestDto defines the JSON payload to unsubscribe or revoke a push subscription.
type PushSubscriptionUnsubscribeRequestDto struct {
	Endpoint string `json:"endpoint" binding:"required,url"`
}
