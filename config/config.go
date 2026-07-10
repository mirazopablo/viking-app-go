package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environmental variables for the application.
type Config struct {
	ServerPort         string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	UploadDir          string
	JWTSecret          string
	JWTExpirationHours int
}

// AppConfig is the global configuration instance.
var AppConfig *Config

// LoadConfig reads the .env file and populates AppConfig.
func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading it. Using environment variables directly.")
	}

	AppConfig = &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "viking_db"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		UploadDir:          getEnv("UPLOAD_DIR", "./uploads"),
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-viking-key"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 4),
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(AppConfig.UploadDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
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
