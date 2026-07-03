package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a system user entity in database.
type User struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name                 string         `gorm:"type:varchar(150);not null;" json:"name" example:"Viking Admin"`
	Dni                  int32          `gorm:"not null;uniqueIndex;" json:"dni" example:"30123456"`
	Address              string         `gorm:"type:varchar(200);" json:"address" example:"Calle Valhalla 123"`
	PhoneNumber          string         `gorm:"type:varchar(50);" json:"phoneNumber" example:"5491112345678"`
	SecondaryPhoneNumber string         `gorm:"type:varchar(50);" json:"secondaryPhoneNumber" example:"5491187654321"`
	Email                string         `gorm:"type:varchar(150);not null;uniqueIndex;" json:"email" example:"admin@viking.com"`
	Password             *string        `gorm:"type:varchar(255);null;" json:"-"` // Nullable password pointer for optional client login
	UserRoles            []UserRole     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"userRoles,omitempty"`
	CreatedAt            time.Time      `json:"createdAt,omitempty"`
	UpdatedAt            time.Time      `json:"updatedAt,omitempty"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hooks into GORM to generate UUID before insertion if nil.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

// RegisterDto defines the request body for registering or saving a user.
type RegisterDto struct {
	Name                 string    `json:"name" binding:"required,min=2,max=150" example:"Viking Admin"`
	Dni                  int32     `json:"dni" binding:"required" example:"30123456"`
	Address              string    `json:"address" binding:"required" example:"Calle Valhalla 123"`
	PhoneNumber          string    `json:"phoneNumber" binding:"required" example:"5491112345678"`
	SecondaryPhoneNumber string    `json:"secondaryPhoneNumber" example:"5491187654321"`
	Email                string    `json:"email" binding:"required,email" example:"admin@viking.com"`
	Password             string    `json:"password" binding:"omitempty,min=6" example:"secret123"`
	RoleID               uuid.UUID `json:"roleId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// LoginUserDto defines the payload for login authentication.
type LoginUserDto struct {
	Email    string `json:"email" binding:"required,email" example:"admin@viking.com"`
	Password string `json:"password" binding:"required" example:"secret123"`
}

// UserResponseDto represents safe user data returned to clients without sensitive credentials.
type UserResponseDto struct {
	ID                   uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name                 string    `json:"name" example:"Viking Admin"`
	Dni                  int32     `json:"dni" example:"30123456"`
	Address              string    `json:"address" example:"Calle Valhalla 123"`
	PhoneNumber          string    `json:"phoneNumber" example:"5491112345678"`
	SecondaryPhoneNumber string    `json:"secondaryPhoneNumber" example:"5491187654321"`
	Email                string    `json:"email" example:"admin@viking.com"`
	RoleID               uuid.UUID `json:"roleId" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// GetPrimaryRoleID returns the first role ID assigned to the user from the intermediate relationship, or Nil if none exists.
func (u *User) GetPrimaryRoleID() uuid.UUID {
	if len(u.UserRoles) > 0 {
		return u.UserRoles[0].RoleID
	}
	return uuid.Nil
}

// ToResponseDto converts User model to safe UserResponseDto.
func (u *User) ToResponseDto() *UserResponseDto {
	return &UserResponseDto{
		ID:                   u.ID,
		Name:                 u.Name,
		Dni:                  u.Dni,
		Address:              u.Address,
		PhoneNumber:          u.PhoneNumber,
		SecondaryPhoneNumber: u.SecondaryPhoneNumber,
		Email:                u.Email,
		RoleID:               u.GetPrimaryRoleID(),
	}
}
