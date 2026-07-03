package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole represents the intermediate join entity between User and Role.
type UserRole struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_user_role;" json:"userId" example:"123e4567-e89b-12d3-a456-426614174000"`
	RoleID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_user_role;" json:"roleId" example:"123e4567-e89b-12d3-a456-426614174000"`
	User      User           `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Role      Role           `gorm:"foreignKey:RoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"role,omitempty"`
	CreatedAt time.Time      `json:"createdAt,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName sets the explicit table name for GORM.
func (UserRole) TableName() string {
	return "user_roles"
}

// BeforeCreate is a GORM hook triggered before inserting a new record.
func (ur *UserRole) BeforeCreate(tx *gorm.DB) (err error) {
	if ur.ID == uuid.Nil {
		ur.ID = uuid.New()
	}
	return
}
