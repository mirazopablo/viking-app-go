package main

import (
	"log"
	"net/http"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/routes"
)

// @title Viking-App ApiREST
// @version 1.0
// @description Documentación de mi API Rest
// @description
// @description Para autenticarte:
// @description 1. Usa el endpoint /auth/login para obtener el token
// @description 2. Copia el token devuelto
// @description 3. Click en el botón 'Authorize' (🔓) arriba
// @description 4. Pega el token en el campo 'Value' (incluye 'Bearer ')
// @contact.name mirazopablo
// @contact.email mirazopablo@gmail.com
// @host 0.0.0.0:8080
// @BasePath /
// @securityDefinitions.apikey bearer-jwt
// @in header
// @name Authorization
func main() {
	// 1. Load environment variables from .env
	config.LoadConfig()

	// 2. Connect to PostgreSQL database
	config.ConnectDatabase()

	// 3. Perform database auto-migration for models including PushSubscription & NotificationHistory
	err := config.DB.AutoMigrate(&models.Role{}, &models.User{}, &models.UserRole{}, &models.Device{}, &models.WorkOrder{}, &models.DiagnosticPoint{}, &models.PushSubscription{}, &models.NotificationHistory{})
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}
	log.Println("Database auto-migration completed successfully.")

	// 4. Initialize Gin router and endpoints
	r := routes.SetupRouter()

	// 5. Start HTTP server
	port := ":" + config.AppConfig.ServerPort
	log.Printf("App is running on port %s", port)
	
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
