package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceToken stores the FCM push tokens for staff/admin devices.
// Rationale: Separamos esto de User para permitir que un admin tenga sesión en múltiples dispositivos.
type DeviceToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"userId"`
	FCMToken  string    `gorm:"type:text;not null;uniqueIndex;" json:"fcmToken"`
	Device    string    `gorm:"type:varchar(50);" json:"device,omitempty"` // Ej: "android", "ios"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate hooks into GORM to generate UUID before insertion if nil.
func (dt *DeviceToken) BeforeCreate(tx *gorm.DB) (err error) {
	if dt.ID == uuid.Nil {
		dt.ID = uuid.New()
	}
	return
}
