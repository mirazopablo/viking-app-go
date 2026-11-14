package apperrors

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ValidateUUID checks that rawID is a valid UUID. On failure it writes a 400 INVALID_ID response and returns false.
func ValidateUUID(c *gin.Context, rawID, fieldName string) bool {
	if _, err := uuid.Parse(rawID); err != nil {
		BadRequest(c, CodeInvalidID, fmt.Sprintf("Invalid %s format (expected UUID)", fieldName))
		return false
	}
	return true
}

// ValidateDate checks that rawDate strictly matches YYYY-MM-DD (10 chars, valid calendar date).
// On failure it writes a 400 VALIDATION_ERROR response and returns false.
func ValidateDate(c *gin.Context, rawDate, fieldName string) bool {
	if len(rawDate) != 10 {
		BadRequest(c, CodeValidationError, fmt.Sprintf("Invalid %s format (expected YYYY-MM-DD)", fieldName))
		return false
	}
	if _, err := time.Parse("2006-01-02", rawDate); err != nil {
		BadRequest(c, CodeValidationError, fmt.Sprintf("Invalid %s format (expected YYYY-MM-DD)", fieldName))
		return false
	}
	return true
}

