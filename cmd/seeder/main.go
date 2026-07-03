package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// promptString displays a label with an optional default value and reads a line from stdin.
func promptString(reader *bufio.Reader, label string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("➤ %s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("➤ %s: ", label)
	}
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// promptPassword repeatedly prompts the user until a valid matching password confirmation is provided.
func promptPassword(reader *bufio.Reader) string {
	for {
		fmt.Print("➤ Contraseña para el Admin (mínimo 6 caracteres): ")
		pass1, _ := reader.ReadString('\n')
		pass1 = strings.TrimSpace(pass1)

		if len(pass1) < 6 {
			fmt.Println("  [X] La contraseña debe tener al menos 6 caracteres. Intente nuevamente.")
			continue
		}

		fmt.Print("➤ Confirmar Contraseña: ")
		pass2, _ := reader.ReadString('\n')
		pass2 = strings.TrimSpace(pass2)

		if pass1 != pass2 {
			fmt.Println("  [X] Las contraseñas no coinciden. Intente nuevamente.")
			continue
		}

		return pass1
	}
}

// main executes the database seeding tool.
// It initializes core system roles (ADMIN, STAFF, CLIENTE) and seeds the primary Admin account interactively or via flags.
func main() {
	var (
		interactiveFlag    = flag.Bool("interactive", true, "Run in interactive console form mode")
		emailFlag          = flag.String("email", "", "Admin user email address (non-interactive mode)")
		passwordFlag       = flag.String("password", "", "Admin user initial password (non-interactive mode)")
		nameFlag           = flag.String("name", "Pablo Mirazo (Super Admin)", "Admin user full name")
		dniFlag            = flag.Int("dni", 30000000, "Admin user DNI document number")
		addressFlag        = flag.String("address", "Calle 123", "Admin user address")
		phoneFlag          = flag.String("phone", "5491100000000", "Admin user primary phone number")
		secondaryPhoneFlag = flag.String("secondary-phone", "", "Admin user secondary phone number (optional)")
	)
	flag.Parse()

	log.Println("--- Viking App Database Seeder ---")

	// 1. Load environment variables from .env
	config.LoadConfig()

	// 2. Connect to PostgreSQL database
	config.ConnectDatabase()

	// 3. Ensure database schema is migrated before seeding
	log.Println("Verifying database schema via AutoMigrate...")
	err := config.DB.AutoMigrate(&models.Role{}, &models.User{}, &models.UserRole{}, &models.Device{}, &models.WorkOrder{}, &models.DiagnosticPoint{})
	if err != nil {
		log.Fatalf("Database auto-migration failed during seeding: %v", err)
	}

	// 4. Seed core system roles
	log.Println("Seeding core system roles...")
	rolesMap := make(map[string]uuid.UUID)
	coreRoles := []string{"ADMIN", "STAFF", "CLIENT"}

	for _, desc := range coreRoles {
		var role models.Role
		err := config.DB.Where("name = ?", desc).First(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			role = models.Role{
				Name: desc,
			}
			if err := config.DB.Create(&role).Error; err != nil {
				log.Fatalf("Failed to create role %s: %v", desc, err)
			}
			log.Printf("[NEW] Created role: %-7s | ID: %s", desc, role.ID)
		} else if err != nil {
			log.Fatalf("Database error querying role %s: %v", desc, err)
		} else {
			log.Printf("[EXISTING] Role found: %-7s | ID: %s", desc, role.ID)
		}
		rolesMap[desc] = role.ID
	}

	// 5. Determine Admin user credentials via Interactive Console Form or Flags
	var (
		adminEmail          string
		adminPassword       string
		adminName           string
		adminDni            int
		adminAddress        string
		adminPhone          string
		adminSecondaryPhone string
	)

	if *interactiveFlag && *emailFlag == "" {
		fmt.Println("\n==================================================")
		fmt.Println("        VIKING-APP CONSOLE SEEDER FORM")
		fmt.Println("==================================================")
		fmt.Println("[!] Modo interactivo seguro. Ingrese los datos del Administrador:")

		reader := bufio.NewReader(os.Stdin)
		adminEmail = promptString(reader, "Email del Administrador", "admin@viking.com")
		adminName = promptString(reader, "Nombre Completo", *nameFlag)

		dniStr := promptString(reader, "DNI", strconv.Itoa(*dniFlag))
		if parsedDni, err := strconv.Atoi(dniStr); err == nil {
			adminDni = parsedDni
		} else {
			adminDni = *dniFlag
		}

		adminAddress = promptString(reader, "Dirección / Domicilio", *addressFlag)
		adminPhone = promptString(reader, "Teléfono Principal", *phoneFlag)
		adminSecondaryPhone = promptString(reader, "Teléfono Secundario (opcional)", *secondaryPhoneFlag)

		adminPassword = promptPassword(reader)
		fmt.Println("==================================================\n")
	} else {
		adminEmail = *emailFlag
		adminPassword = *passwordFlag
		adminName = *nameFlag
		adminDni = *dniFlag
		adminAddress = *addressFlag
		adminPhone = *phoneFlag
		adminSecondaryPhone = *secondaryPhoneFlag

		if adminEmail == "" || adminPassword == "" {
			log.Fatal("[SECURITY ERROR] En modo no-interactivo (-interactive=false), los flags -email y -password son obligatorios para no dejar contraseñas por defecto en el repositorio.")
		}
	}

	// 6. Seed primary Admin user account
	log.Printf("Checking primary Admin user account (%s)...", adminEmail)
	userRepo := repositories.NewUserRepository()
	existingAdmin, err := userRepo.FindByEmail(adminEmail)
	if err != nil {
		log.Fatalf("Database error checking existing admin user: %v", err)
	}

	if existingAdmin != nil {
		log.Printf("[EXISTING] Admin account already registered: %s | ID: %s", existingAdmin.Email, existingAdmin.ID)
		log.Println("--- Seeding completed without overriding existing user data ---")
		return
	}

	log.Printf("Generating secure bcrypt hash for admin password...")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}
	hashedStr := string(hashedPassword)

	var secPhonePtr *string
	if adminSecondaryPhone != "" {
		secPhonePtr = &adminSecondaryPhone
	}

	adminUser := &models.User{
		Name:                 adminName,
		Dni:                  int32(adminDni),
		Address:              adminAddress,
		PhoneNumber:          adminPhone,
		SecondaryPhoneNumber: secPhonePtr,
		Email:                adminEmail,
		Password:             &hashedStr,
	}

	adminRoleID, ok := rolesMap["ADMIN"]
	if !ok {
		log.Fatal("Critical Error: ADMIN role ID not found in roles map")
	}

	log.Printf("Creating Admin user %q within transaction...", adminEmail)
	if err := userRepo.CreateWithRole(adminUser, adminRoleID); err != nil {
		log.Fatalf("Failed to create Admin user account: %v", err)
	}

	log.Printf("[SUCCESS] Created Admin user: %s | ID: %s | Assigned Role ID: %s", adminUser.Email, adminUser.ID, adminRoleID)
	log.Println("--- Seeding completed successfully! ---")
}
