package main

import (
	"log"

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

	// 3. Perform database auto-migration for Stage 1, 2, 4, 5 & 7 models
	err := config.DB.AutoMigrate(&models.Role{}, &models.User{}, &models.UserRole{}, &models.Device{}, &models.WorkOrder{}, &models.DiagnosticPoint{})
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}
	log.Println("Database auto-migration completed successfully.")

	// 4. Initialize Gin router and endpoints
	r := routes.SetupRouter()

	// 5. Start HTTP server
	port := ":" + config.AppConfig.ServerPort
	log.Printf("App is running on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
