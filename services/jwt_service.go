package services

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mirazopablo/viking-app-go/config"
)

var (
	// ErrInvalidToken indicates the token is malformed or expired.
	ErrInvalidToken = errors.New("invalid or expired JWT token")
)

// JWTClaims defines custom claims embedded in our JWT token.
type JWTClaims struct {
	UserID string `json:"userId"`
	RoleID string `json:"roleId"`
	jwt.RegisteredClaims
}

// JWTService defines operations for creating and verifying JWT tokens.
type JWTService interface {
	GenerateToken(userID string, roleID string) (string, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}

type jwtServiceImpl struct {
	secretKey      []byte
	expirationTime time.Duration
}

// NewJWTService instantiates a new JWTService.
func NewJWTService() JWTService {
	secret := ""
	if config.AppConfig != nil {
		secret = config.AppConfig.JWTSecret
	}
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		secret = "super-secret-viking-key-change-in-prod"
	}

	expirationHours := 4
	if config.AppConfig != nil && config.AppConfig.JWTExpirationHours > 0 {
		expirationHours = config.AppConfig.JWTExpirationHours
	} else if envHours := os.Getenv("JWT_EXPIRATION_HOURS"); envHours != "" {
		if parsed, err := strconv.Atoi(envHours); err == nil && parsed > 0 {
			expirationHours = parsed
		}
	}

	return &jwtServiceImpl{
		secretKey:      []byte(secret),
		expirationTime: time.Duration(expirationHours) * time.Hour,
	}
}

// GenerateToken creates a signed JWT string valid for the configured expiration duration.
func (s *jwtServiceImpl) GenerateToken(userID string, roleID string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		RoleID: roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "viking-app-go",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken parses and verifies a token string against our secret key.
func (s *jwtServiceImpl) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
