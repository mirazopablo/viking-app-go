package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationHistory persists every alert triggered for a work order.
// Rationale: Acts as the single source of truth for offline-to-online pull synchronization and UI bell icon state.
type NotificationHistory struct {
	ID          string    `gorm:"type:uuid;primary_key;" json:"id"`
	WorkOrderID string    `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"workOrderId"`
	Title       string    `gorm:"type:varchar(150);not null" json:"title"`
	Body        string    `gorm:"type:text;not null" json:"body"`
	Type        string    `gorm:"type:varchar(50);not null;index" json:"type"` // e.g., STATUS_CHANGE, DIAGNOSTIC_SUMMARY
	TargetURL   string    `gorm:"type:varchar(255);" json:"targetUrl"`
	IsRead      bool      `gorm:"default:false;index" json:"isRead"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BeforeCreate hooks into GORM to generate a unique UUID prior to database insertion.
func (nh *NotificationHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if nh.ID == "" {
		nh.ID = uuid.New().String()
	}
	return
}

// NotificationHistoryResponseDto projects the stored notification for frontend UI bell synchronization.
type NotificationHistoryResponseDto struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Type      string `json:"type"`
	TargetURL string `json:"targetUrl"`
	IsRead    bool   `json:"isRead"`
	CreatedAt string `json:"createdAt"`
}
