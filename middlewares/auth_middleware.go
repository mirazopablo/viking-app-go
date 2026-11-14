package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/apperrors"
	"github.com/mirazopablo/viking-app-go/services"
)

// AuthMiddleware creates a Gin middleware that validates JWT Bearer tokens.
func AuthMiddleware(jwtSvc services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apperrors.AbortError(c, apperrors.NewUnauthorized(
				apperrors.CodeAuthTokenMissing,
				"Authorization header is required",
			))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			apperrors.AbortError(c, apperrors.NewUnauthorized(
				apperrors.CodeAuthTokenMissing,
				"Bearer token is missing",
			))
			return
		}

		claims, err := jwtSvc.ValidateToken(tokenString)
		if err != nil {
			apperrors.AbortError(c, apperrors.NewUnauthorized(
				apperrors.CodeAuthTokenInvalid,
				"Invalid or expired token",
			))
			return
		}

		// Store user identity in context for downstream controllers
		c.Set("userID", claims.UserID)
		c.Set("roleID", claims.RoleID)
		c.Set("roleName", claims.RoleName)

		c.Next()
	}
}
