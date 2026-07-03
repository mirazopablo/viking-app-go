package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents the role entity in database.
type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"descripcion" binding:"required,min=2,max=100" example:"ADMIN"`
}

// RoleCreateRequest represents the request payload for creating or updating a role.
type RoleCreateRequest struct {
	Name string `json:"descripcion" binding:"required,min=2,max=100" example:"ADMIN"`
}

// TableName sets the table name for GORM.
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate is a GORM hook triggered before inserting a new record.
// It generates a new UUID in memory if one hasn't been provided, ensuring DB independence.
func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
