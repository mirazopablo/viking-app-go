package models

// PaginatedResponse wraps a slice of data with pagination metadata.
type PaginatedResponse[T any] struct {
	Data  []T   `json:"data"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}
