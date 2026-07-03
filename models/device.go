package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Device represents the hardware equipment registered to a client in database.
type Device struct {
	ID           string         `gorm:"type:uuid;primary_key;" json:"id"`
	Type         string         `gorm:"type:varchar(100);not null" json:"type"`
	Brand        string         `gorm:"type:varchar(100);not null" json:"brand"`
	Model        string         `gorm:"type:varchar(150);not null" json:"model"`
	SerialNumber string         `gorm:"type:varchar(150);uniqueIndex;not null" json:"serialNumber"`
	UserID       string         `gorm:"type:uuid;not null;index" json:"userId"`
	User         User           `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (d *Device) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return
}

// DeviceCreateRequestDto defines the input structure for registering a device.
type DeviceCreateRequestDto struct {
	Type         string `json:"type" binding:"required"`
	Brand        string `json:"brand" binding:"required"`
	Model        string `json:"model" binding:"required"`
	SerialNumber string `json:"serialNumber" binding:"required"`
	UserID       string `json:"userId" binding:"required"`
}

// DeviceUpdateRequestDto defines the input structure for updating a device.
type DeviceUpdateRequestDto struct {
	ID           string `json:"id"`
	Type         string `json:"type" binding:"required"`
	Brand        string `json:"brand" binding:"required"`
	Model        string `json:"model" binding:"required"`
	SerialNumber string `json:"serialNumber" binding:"required"`
	UserID       string `json:"userId" binding:"required"`
}

// DeviceResponseDto defines the output data structure for device queries.
type DeviceResponseDto struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
	UserID       string `json:"userId"`
	UserName     string `json:"userName"`
	UserDni      int32  `json:"userDni"`
}
