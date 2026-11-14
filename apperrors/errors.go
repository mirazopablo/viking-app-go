package apperrors

import (
	"fmt"
	"net/http"
)

// ErrorResponse defines the unified JSON contract returned to clients on 4xx/5xx responses.
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Code    string `json:"code" example:"AUTH_INVALID_CREDENTIALS"`
	Message string `json:"message" example:"Invalid email or password"`
	Details any    `json:"details,omitempty"`
}

// FieldError describes a single validation failure on a request field.
type FieldError struct {
	Field   string `json:"field" example:"email"`
	Reason  string `json:"reason" example:"email"`
	Message string `json:"message" example:"Invalid value for field 'email'"`
}

// AppError represents a domain or application error decoupled from HTTP/Gin context.
type AppError struct {
	StatusCode int
	Code       string
	Message    string
	Details    any
	Internal   error
}

// Error implements the standard error interface.
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped internal error for errors.Is / errors.As compatibility.
func (e *AppError) Unwrap() error {
	return e.Internal
}

// WithDetails attaches structured details (e.g. validation field errors) to the AppError.
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// WithInternal attaches the underlying cause for server-side logging without exposing it to clients.
func (e *AppError) WithInternal(err error) *AppError {
	e.Internal = err
	return e
}

// ToResponse converts the AppError into the public ErrorResponse JSON contract.
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Success: false,
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	}
}

// New creates an AppError with an explicit HTTP status code, machine code, and user-facing message.
func New(statusCode int, code, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// NewBadRequest creates a 400 Bad Request AppError.
func NewBadRequest(code, message string) *AppError {
	return New(http.StatusBadRequest, code, message)
}

// NewUnauthorized creates a 401 Unauthorized AppError.
func NewUnauthorized(code, message string) *AppError {
	return New(http.StatusUnauthorized, code, message)
}

// NewForbidden creates a 403 Forbidden AppError.
func NewForbidden(code, message string) *AppError {
	return New(http.StatusForbidden, code, message)
}

// NewNotFound creates a 404 Not Found AppError.
func NewNotFound(code, message string) *AppError {
	return New(http.StatusNotFound, code, message)
}

// NewConflict creates a 409 Conflict AppError.
func NewConflict(code, message string) *AppError {
	return New(http.StatusConflict, code, message)
}

// NewTooManyRequests creates a 429 Too Many Requests AppError.
func NewTooManyRequests(code, message string) *AppError {
	return New(http.StatusTooManyRequests, code, message)
}

// NewInternal creates a 500 Internal Server Error AppError with a safe client message.
func NewInternal(err error) *AppError {
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       CodeInternalError,
		Message:    "An unexpected internal error occurred",
		Internal:   err,
	}
}

