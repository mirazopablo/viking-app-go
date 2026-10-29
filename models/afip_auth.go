package models

import "time"

// AFIPAuthToken represents the cached credentials obtained from WSAA.
type AFIPAuthToken struct {
	Token      string    `json:"token"`
	Sign       string    `json:"sign"`
	Expiration time.Time `json:"expiration"`
}

// IsValid checks if the token is still valid (with a 10-minute safety buffer).
func (a *AFIPAuthToken) IsValid() bool {
	if a == nil || a.Token == "" || a.Sign == "" {
		return false
	}
	return time.Now().Add(10 * time.Minute).Before(a.Expiration)
}
