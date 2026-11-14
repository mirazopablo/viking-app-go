package apperrors

// Standardized machine-readable error and success codes for the frontend contract.
const (
	// Success codes
	CodeOK      = "OK"
	CodeCreated = "CREATED"

	// 400 Bad Request
	CodeValidationError = "VALIDATION_ERROR"
	CodeInvalidID       = "INVALID_ID"
	CodeBusinessRule    = "BUSINESS_RULE_VIOLATION"

	// 401 Unauthorized
	CodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	CodeAuthTokenMissing       = "AUTH_TOKEN_MISSING"
	CodeAuthTokenInvalid       = "AUTH_TOKEN_INVALID"
	CodeInvalidSecurityCode    = "INVALID_SECURITY_CODE"

	// 403 Forbidden
	CodeAuthForbidden = "AUTH_FORBIDDEN"

	// 404 Not Found
	CodeResourceNotFound = "RESOURCE_NOT_FOUND"

	// 409 Conflict
	CodeAuthEmailExists = "AUTH_EMAIL_EXISTS"
	CodeConflict        = "RESOURCE_CONFLICT"

	// 429 Too Many Requests
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// 500 Internal Server Error
	CodeInternalError = "INTERNAL_ERROR"
)

