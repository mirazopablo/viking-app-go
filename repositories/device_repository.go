package repositories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// DeviceRepository defines data store interactions for devices.
type DeviceRepository interface {
	Save(device *models.Device) (*models.Device, error)
	FindByID(id string) (*models.Device, error)
	FindAll() ([]models.Device, error)
	Search(id, serialNumber, brand, userDni, userName, query string) ([]models.Device, error)
	Update(device *models.Device) (*models.Device, error)
	Delete(id string) error
}

type deviceRepositoryImpl struct {
	db *gorm.DB
}

// NewDeviceRepository instantiates a new DeviceRepository with GORM.
func NewDeviceRepository() DeviceRepository {
	return &deviceRepositoryImpl{db: config.DB}
}

func (r *deviceRepositoryImpl) Save(device *models.Device) (*models.Device, error) {
	err := r.db.Create(device).Error
	if err != nil {
		return nil, err
	}
	// Reload with user association
	return r.FindByID(device.ID)
}

func (r *deviceRepositoryImpl) FindByID(id string) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("User").First(&device, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &device, nil
}

func (r *deviceRepositoryImpl) FindAll() ([]models.Device, error) {
	var devices []models.Device
	err := r.db.Preload("User").Find(&devices).Error
	return devices, err
}

func (r *deviceRepositoryImpl) Search(id, serialNumber, brand, userDni, userName, query string) ([]models.Device, error) {
	var devices []models.Device
	queryBuilder := r.db.Preload("User").Joins("LEFT JOIN users ON users.id = devices.user_id")

	if id != "" {
		queryBuilder = queryBuilder.Where("devices.id = ?", id)
	}
	if serialNumber != "" {
		queryBuilder = queryBuilder.Where("devices.serial_number ILIKE ?", "%"+serialNumber+"%")
	}
	if brand != "" {
		queryBuilder = queryBuilder.Where("devices.brand ILIKE ?", "%"+brand+"%")
	}
	if userDni != "" {
		queryBuilder = queryBuilder.Where("CAST(users.dni AS TEXT) ILIKE ? OR CAST(devices.user_dni AS TEXT) ILIKE ?", "%"+userDni+"%", "%"+userDni+"%")
	}
	if userName != "" {
		queryBuilder = queryBuilder.Where("devices.user_name ILIKE ? OR users.name ILIKE ?", "%"+userName+"%", "%"+userName+"%")
	}
	if query != "" {
		lowerQ := strings.ToLower(strings.TrimSpace(query))
		// Ignore control mode strings / field selectors from frontend (e.g. "all", "by-brand", "by-serial", "by-dni", "by-id", "userdni", "username", "brand", "serialnumber")
		if lowerQ != "all" && !strings.HasPrefix(lowerQ, "by-") && lowerQ != "userdni" && lowerQ != "username" && lowerQ != "brand" && lowerQ != "serialnumber" {
			q := "%" + lowerQ + "%"
			queryBuilder = queryBuilder.Where(
				"LOWER(devices.brand) LIKE ? OR LOWER(devices.model) LIKE ? OR LOWER(devices.serial_number) LIKE ? OR LOWER(devices.user_name) LIKE ? OR LOWER(users.name) LIKE ? OR CAST(users.dni AS TEXT) LIKE ? OR CAST(devices.user_dni AS TEXT) LIKE ?",
				q, q, q, q, q, q, q,
			)
		}
	}

	err := queryBuilder.Find(&devices).Error
	return devices, err
}

func (r *deviceRepositoryImpl) Update(device *models.Device) (*models.Device, error) {
	err := r.db.Save(device).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(device.ID)
}

func (r *deviceRepositoryImpl) Delete(id string) error {
	result := r.db.Delete(&models.Device{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}
