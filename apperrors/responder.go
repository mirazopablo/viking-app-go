package apperrors

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// OK writes a 200 OK JSON response.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// Created writes a 201 Created JSON response.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// RespondError writes an AppError using the unified ErrorResponse JSON contract.
// Internal errors for 5xx status codes are logged server-side and never exposed to the client.
func RespondError(c *gin.Context, appErr *AppError) {
	if appErr == nil {
		Internal(c, errors.New("nil AppError passed to RespondError"))
		return
	}
	if appErr.StatusCode >= http.StatusInternalServerError && appErr.Internal != nil {
		log.Printf("[apperrors] %s %s | code=%s | internal=%v",
			c.Request.Method, c.Request.URL.Path, appErr.Code, appErr.Internal)
	}
	c.JSON(appErr.StatusCode, appErr.ToResponse())
}

// AbortError aborts the Gin middleware chain and writes an AppError JSON response.
func AbortError(c *gin.Context, appErr *AppError) {
	if appErr == nil {
		appErr = NewInternal(errors.New("nil AppError passed to AbortError"))
	}
	if appErr.StatusCode >= http.StatusInternalServerError && appErr.Internal != nil {
		log.Printf("[apperrors] ABORT %s %s | code=%s | internal=%v",
			c.Request.Method, c.Request.URL.Path, appErr.Code, appErr.Internal)
	}
	c.AbortWithStatusJSON(appErr.StatusCode, appErr.ToResponse())
}

// HandleError translates any Go error into a standardized HTTP response.
// If err wraps or is an *AppError, its status, code, and message are used; otherwise 500 Internal is returned.
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		RespondError(c, appErr)
		return
	}
	Internal(c, err)
}

// BadRequest writes a 400 Bad Request JSON response.
func BadRequest(c *gin.Context, code, message string, details ...any) {
	appErr := NewBadRequest(code, message)
	if len(details) > 0 && details[0] != nil {
		appErr.WithDetails(details[0])
	}
	RespondError(c, appErr)
}

// ValidationError extracts field-level validation errors (when available) and writes a 400 VALIDATION_ERROR response.
func ValidationError(c *gin.Context, err error) {
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		fields := make([]FieldError, 0, len(valErrs))
		for _, fe := range valErrs {
			fields = append(fields, FieldError{
				Field:   fe.Field(),
				Reason:  fe.Tag(),
				Message: fmt.Sprintf("Field '%s' failed validation '%s'", fe.Field(), fe.Tag()),
			})
		}
		BadRequest(c, CodeValidationError, "Invalid request data", fields)
		return
	}
	BadRequest(c, CodeValidationError, "Invalid request payload")
}

// Unauthorized writes a 401 Unauthorized JSON response.
func Unauthorized(c *gin.Context, code, message string) {
	RespondError(c, NewUnauthorized(code, message))
}

// Forbidden writes a 403 Forbidden JSON response.
func Forbidden(c *gin.Context, code, message string) {
	RespondError(c, NewForbidden(code, message))
}

// NotFound writes a 404 Not Found JSON response.
func NotFound(c *gin.Context, code, message string) {
	RespondError(c, NewNotFound(code, message))
}

// Conflict writes a 409 Conflict JSON response.
func Conflict(c *gin.Context, code, message string) {
	RespondError(c, NewConflict(code, message))
}

// TooManyRequests writes a 429 Too Many Requests JSON response.
func TooManyRequests(c *gin.Context, code, message string) {
	RespondError(c, NewTooManyRequests(code, message))
}

// Internal logs the underlying error server-side and writes a safe 500 INTERNAL_ERROR JSON response.
func Internal(c *gin.Context, err error) {
	RespondError(c, NewInternal(err))
}

