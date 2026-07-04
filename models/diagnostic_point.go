package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DiagnosticPoint represents a multimedia evidence or technical diagnostic log attached to a work order.
type DiagnosticPoint struct {
	ID          string         `gorm:"type:uuid;primary_key;" json:"id"`
	WorkOrderID string    `gorm:"type:uuid;not null;index" json:"workOrderId"`
	WorkOrder   WorkOrder `gorm:"foreignKey:WorkOrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ClientID    *string   `gorm:"type:uuid;index" json:"clientId,omitempty"`
	Client      User      `gorm:"foreignKey:ClientID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Description string    `gorm:"type:text;not null" json:"description"`
	ImageURL    string    `gorm:"type:varchar(255);not null" json:"imageUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// BeforeCreate hooks into GORM to generate a UUID prior to database insertion.
func (dp *DiagnosticPoint) BeforeCreate(tx *gorm.DB) (err error) {
	if dp.ID == "" {
		dp.ID = uuid.New().String()
	}
	return
}

// DiagnosticPointResponseDto defines the external JSON representation of a diagnostic point.
type DiagnosticPointResponseDto struct {
	ID          string `json:"id"`
	WorkOrderID string `json:"workOrderId"`
	ClientID    string `json:"clientId"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	CreatedAt   string `json:"createdAt"`
}
