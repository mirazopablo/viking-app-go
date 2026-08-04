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
	err := config.DB.AutoMigrate(&models.Role{}, &models.User{}, &models.UserRole{}, &models.Device{}, &models.WorkOrder{}, &models.DiagnosticPoint{}, &models.PushSubscription{}, &models.NotificationHistory{}, &models.Budget{})
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
		clientRoleID, ok := rolesMap["CLIENT"]
		if !ok {
			log.Fatal("Critical Error: CLIENT role ID not found in roles map")
		}
		seedTestData(clientRoleID, existingAdmin.ID.String())
		log.Println("--- Seeding completed successfully! ---")
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

	// 7. Seed test data (clients, devices, work orders)
	clientRoleID, ok := rolesMap["CLIENT"]
	if !ok {
		log.Fatal("Critical Error: CLIENT role ID not found in roles map")
	}
	seedTestData(clientRoleID, adminUser.ID.String())

	log.Println("--- Seeding completed successfully! ---")
}

// seedTestData inserts sample clients, devices, and work orders for testing purposes.
func seedTestData(clientRoleID uuid.UUID, adminUserID string) {
	log.Println("\n--- Seeding Test Data (Clients, Devices & Work Orders) ---")

	userRepo := repositories.NewUserRepository()
	deviceRepo := repositories.NewDeviceRepository()
	workOrderRepo := repositories.NewWorkOrderRepository()

	// 1. Seed Sample Clients
	type clientSeed struct {
		Name    string
		Dni     int32
		Address string
		Phone   string
		Email   string
	}

	clientsSeedData := []clientSeed{
		{
			Name:    "Juan Pérez",
			Dni:     35123456,
			Address: "Av. Siempre Viva 742, Córdoba",
			Phone:   "5493514567890",
			Email:   "cliente1@ejemplo.com",
		},
		{
			Name:    "Maria Gomez",
			Dni:     38987654,
			Address: "Calle Falsa 123, Rosario",
			Phone:   "5493415678901",
			Email:   "cliente2@ejemplo.com",
		},
	}

	clientUsers := make(map[string]*models.User)

	for _, cs := range clientsSeedData {
		existing, err := userRepo.FindByEmail(cs.Email)
		if err != nil {
			log.Fatalf("Error checking client user %s: %v", cs.Email, err)
		}

		if existing != nil {
			log.Printf("[EXISTING] Client user found: %s | ID: %s", existing.Email, existing.ID)
			clientUsers[cs.Email] = existing
		} else {
			newUser := &models.User{
				Name:        cs.Name,
				Dni:         cs.Dni,
				Address:     cs.Address,
				PhoneNumber: cs.Phone,
				Email:       cs.Email,
			}
			if err := userRepo.CreateWithRole(newUser, clientRoleID); err != nil {
				log.Fatalf("Failed to create client user %s: %v", cs.Email, err)
			}
			log.Printf("[NEW] Created Client user: %s (%s) | ID: %s", newUser.Name, newUser.Email, newUser.ID)
			clientUsers[cs.Email] = newUser
		}
	}

	// 2. Seed Sample Devices (Equipos)
	// Client 1 has 2 devices (Dell XPS Notebook, Samsung Galaxy S21)
	// Client 2 has 1 device (Custom PC Gamer)
	type deviceSeed struct {
		ClientEmail  string
		Type         string
		Brand        string
		Model        string
		SerialNumber string
	}

	devicesSeedData := []deviceSeed{
		{
			ClientEmail:  "cliente1@ejemplo.com",
			Type:         "Notebook",
			Brand:        "Dell",
			Model:        "XPS 15 9520",
			SerialNumber: "DELL-XPS-9520-001",
		},
		{
			ClientEmail:  "cliente1@ejemplo.com",
			Type:         "Smartphone",
			Brand:        "Samsung",
			Model:        "Galaxy S21",
			SerialNumber: "SAMSUNG-S21-9988",
		},
		{
			ClientEmail:  "cliente2@ejemplo.com",
			Type:         "PC de Escritorio",
			Brand:        "Custom PC",
			Model:        "Ryzen 7 / RTX 3080",
			SerialNumber: "CUSTOM-PC-2024-X",
		},
	}

	seededDevices := make(map[string]*models.Device)

	for _, ds := range devicesSeedData {
		clientOwner, ok := clientUsers[ds.ClientEmail]
		if !ok {
			log.Fatalf("Owner client %s not found for device %s", ds.ClientEmail, ds.SerialNumber)
		}

		var existingDevice models.Device
		err := config.DB.Where("serial_number = ?", ds.SerialNumber).First(&existingDevice).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userIDStr := clientOwner.ID.String()
			newDevice := &models.Device{
				Type:         ds.Type,
				Brand:        ds.Brand,
				Model:        ds.Model,
				SerialNumber: ds.SerialNumber,
				UserID:       &userIDStr,
				UserName:     clientOwner.Name,
				UserDni:      clientOwner.Dni,
				UserPhone:    clientOwner.PhoneNumber,
			}
			saved, err := deviceRepo.Save(newDevice)
			if err != nil {
				log.Fatalf("Failed to create device %s: %v", ds.SerialNumber, err)
			}
			log.Printf("[NEW] Created Device: %s %s (SN: %s) | Owner: %s | ID: %s", saved.Brand, saved.Model, saved.SerialNumber, clientOwner.Name, saved.ID)
			seededDevices[ds.SerialNumber] = saved
		} else if err != nil {
			log.Fatalf("Error querying device %s: %v", ds.SerialNumber, err)
		} else {
			log.Printf("[EXISTING] Device found: %s %s (SN: %s) | ID: %s", existingDevice.Brand, existingDevice.Model, existingDevice.SerialNumber, existingDevice.ID)
			seededDevices[ds.SerialNumber] = &existingDevice
		}
	}

	// 3. Seed Sample Work Orders (Ordenes de Trabajo)
	type workOrderSeed struct {
		DeviceSerial     string
		IssueDescription string
		Notes            string
		RepairStatus     string
		PlainCode        string
	}

	workOrdersSeedData := []workOrderSeed{
		{
			DeviceSerial:     "DELL-XPS-9520-001",
			IssueDescription: "No enciende tras derrame ligero de líquido sobre el teclado.",
			Notes:            "Revisión de placa madre y limpieza ultrasónica recomendada.",
			RepairStatus:     models.StatusInProgress,
			PlainCode:        "WOVIK-NOTE1",
		},
		{
			DeviceSerial:     "SAMSUNG-S21-9988",
			IssueDescription: "Pantalla astillada y puerto de carga con falso contacto.",
			Notes:            "Presupuesto enviado al cliente por WhatsApp.",
			RepairStatus:     models.StatusReceived,
			PlainCode:        "WOVIK-S21XX",
		},
		{
			DeviceSerial:     "CUSTOM-PC-2024-X",
			IssueDescription: "Mantenimiento preventivo, cambio de pasta térmica y limpieza de polvo.",
			Notes:            "Finalizado con éxito. Pruebas de stress de CPU/GPU pasadas.",
			RepairStatus:     models.StatusDone,
			PlainCode:        "WOVIK-PCGAM",
		},
	}

	for _, wos := range workOrdersSeedData {
		targetDevice, ok := seededDevices[wos.DeviceSerial]
		if !ok {
			log.Fatalf("Target device %s not found for work order", wos.DeviceSerial)
		}

		var existingWO models.WorkOrder
		err := config.DB.Where("device_id = ? AND issue_description = ?", targetDevice.ID, wos.IssueDescription).First(&existingWO).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hashedCode, err := bcrypt.GenerateFromPassword([]byte(wos.PlainCode), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash security code for work order: %v", err)
			}

			clientOwner := targetDevice.User
			if clientOwner.ID == uuid.Nil && targetDevice.UserID != nil {
				fetchedUser, _ := userRepo.FindByID(*targetDevice.UserID)
				if fetchedUser != nil {
					clientOwner = *fetchedUser
				}
			}

			clientIDStr := clientOwner.ID.String()
			var staffPtr *string
			if adminUserID != "" {
				staffPtr = &adminUserID
			}

			newWO := &models.WorkOrder{
				ClientID:             &clientIDStr,
				DeviceID:             &targetDevice.ID,
				StaffID:              staffPtr,
				SecurityCodeHash:     string(hashedCode),
				ClientNameSnapshot:   clientOwner.Name,
				ClientDniSnapshot:    clientOwner.Dni,
				ClientPhoneSnapshot:  clientOwner.PhoneNumber,
				DeviceBrandSnapshot:  targetDevice.Brand,
				DeviceModelSnapshot:  targetDevice.Model,
				DeviceSerialSnapshot: targetDevice.SerialNumber,
				IssueDescription:     wos.IssueDescription,
				Notes:                wos.Notes,
				RepairStatus:         wos.RepairStatus,
			}

			savedWO, err := workOrderRepo.Save(newWO)
			if err != nil {
				log.Fatalf("Failed to create work order for device %s: %v", targetDevice.SerialNumber, err)
			}

			log.Printf("[NEW] Created Work Order: ID: %s | Status: %s | Client: %s | Device: %s %s | Security Code: %s",
				savedWO.ID, savedWO.RepairStatus, savedWO.ClientNameSnapshot, savedWO.DeviceBrandSnapshot, savedWO.DeviceModelSnapshot, wos.PlainCode)
		} else if err != nil {
			log.Fatalf("Error querying work order for device %s: %v", targetDevice.SerialNumber, err)
		} else {
			log.Printf("[EXISTING] Work Order found: ID: %s | Status: %s | Device: %s %s",
				existingWO.ID, existingWO.RepairStatus, existingWO.DeviceBrandSnapshot, existingWO.DeviceModelSnapshot)
		}
	}
}
