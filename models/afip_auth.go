package models

import (
	"time"
)

// AFIPAuthToken represents credentials obtained from WSAA, persisted in DB to survive server restarts.
type AFIPAuthToken struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Service     string    `gorm:"size:50;not null;uniqueIndex:idx_service_env" json:"service"` // e.g. "wsfe"
	Environment string    `gorm:"size:20;not null;uniqueIndex:idx_service_env" json:"environment"` // "production" or "homologation"
	Token       string    `gorm:"type:text;not null" json:"token"`
	Sign        string    `gorm:"type:text;not null" json:"sign"`
	Expiration  time.Time `gorm:"not null" json:"expiration"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsValid checks if the token is still valid (with a 5-minute safety buffer).
func (a *AFIPAuthToken) IsValid() bool {
	if a == nil || a.Token == "" || a.Sign == "" {
		return false
	}
	return time.Now().Add(5 * time.Minute).Before(a.Expiration)
}

