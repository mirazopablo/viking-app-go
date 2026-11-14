package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environmental variables for the application.
type Config struct {
	ServerPort           string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	DBSSLMode            string
	UploadDir            string
	JWTSecret            string
	JWTExpirationHours   int
	VAPIDPublicKey       string
	VAPIDPrivateKey      string
	VAPIDSubscriberEmail string

	// AFIP / ARCA Configuration
	AfipCUIT       string
	AfipPuntoVenta int
	AfipCertPath   string
	AfipKeyPath    string
	AfipEnv        string
	AfipWSAAURL    string
	AfipWSFEURL    string
}

// AppConfig is the global configuration instance.
var AppConfig *Config

// LoadConfig reads the .env file and populates AppConfig.
func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("[config/config.go] [LoadConfig] Warning: No .env file found or error loading it: %v", err)
	}

	AppConfig = &Config{
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", "postgres"),
		DBName:               getEnv("DB_NAME", "viking_db"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		UploadDir:            getEnv("UPLOAD_DIR", "./uploads"),
		JWTSecret:            getEnv("JWT_SECRET", "super-secret-viking-key"),
		JWTExpirationHours:   getEnvAsInt("JWT_EXPIRATION_HOURS", 4),
		VAPIDPublicKey:       getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:      getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubscriberEmail: getEnv("VAPID_SUBSCRIBER_EMAIL", "mailto:admin@viking-app.zondasolutions.com"),

		// AFIP / ARCA
		AfipCUIT:       getEnv("AFIP_CUIT", ""),
		AfipPuntoVenta: getEnvAsInt("AFIP_PUNTO_VENTA", 2),
		AfipCertPath:   getEnv("AFIP_CERT_PATH", "./config/certs/viking-app.crt"),
		AfipKeyPath:    getEnv("AFIP_KEY_PATH", "./config/certs/viking-app.key"),
		AfipEnv:        getEnv("AFIP_ENV", "production"),
		AfipWSAAURL:    getEnv("AFIP_WSAA_URL", "https://wsaa.afip.gov.ar/ws/services/LoginCms"),
		AfipWSFEURL:    getEnv("AFIP_WSFE_URL", "https://servicios1.afip.gov.ar/wsfev1/service.asmx"),
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(AppConfig.UploadDir, os.ModePerm); err != nil {
		log.Fatalf("[config/config.go] [LoadConfig] Failed to create upload directory: %v", err)
	}
}

// getEnv retrieves an environment variable or returns a default value if not set.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value if not set or invalid.
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil && value > 0 {
		return value
	}
	return defaultValue
}
