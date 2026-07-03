package repositories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// WorkOrderRepository defines data store interactions for repair jobs.
type WorkOrderRepository interface {
	Save(wo *models.WorkOrder) (*models.WorkOrder, error)
	FindByID(id string) (*models.WorkOrder, error)
	FindAll() ([]models.WorkOrder, error)
	Search(staffId string, clientDni int32, deviceSerialNumber string, query string) ([]models.WorkOrder, error)
	Update(wo *models.WorkOrder) (*models.WorkOrder, error)
	Delete(id string) error
}

type workOrderRepositoryImpl struct {
	db *gorm.DB
}

// NewWorkOrderRepository instantiates a new WorkOrderRepository with GORM.
func NewWorkOrderRepository() WorkOrderRepository {
	return &workOrderRepositoryImpl{db: config.DB}
}

func (r *workOrderRepositoryImpl) Save(wo *models.WorkOrder) (*models.WorkOrder, error) {
	err := r.db.Create(wo).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(wo.ID)
}

func (r *workOrderRepositoryImpl) FindByID(id string) (*models.WorkOrder, error) {
	var wo models.WorkOrder
	err := r.db.Preload("Client").Preload("Device").Preload("Staff").First(&wo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &wo, nil
}

func (r *workOrderRepositoryImpl) FindAll() ([]models.WorkOrder, error) {
	var orders []models.WorkOrder
	err := r.db.Preload("Client").Preload("Device").Preload("Staff").Find(&orders).Error
	return orders, err
}

func (r *workOrderRepositoryImpl) Search(staffId string, clientDni int32, deviceSerialNumber string, query string) ([]models.WorkOrder, error) {
	var orders []models.WorkOrder
	queryBuilder := r.db.Preload("Client").Preload("Device").Preload("Staff").
		Joins("JOIN users AS client_user ON client_user.id = work_orders.client_id").
		Joins("JOIN devices ON devices.id = work_orders.device_id")

	if staffId != "" {
		queryBuilder = queryBuilder.Where("work_orders.staff_id = ?", staffId)
	}
	if clientDni != 0 {
		queryBuilder = queryBuilder.Where("client_user.dni = ?", clientDni)
	}
	if deviceSerialNumber != "" {
		queryBuilder = queryBuilder.Where("devices.serial_number ILIKE ?", "%"+deviceSerialNumber+"%")
	}
	if query != "" {
		lowerQ := strings.ToLower(strings.TrimSpace(query))
		// Ignore control mode strings from frontend (e.g. "all", "by-client", "by-device", "by-status", "by-id")
		if lowerQ != "all" && !strings.HasPrefix(lowerQ, "by-") {
			q := "%" + lowerQ + "%"
			queryBuilder = queryBuilder.Where(
				"LOWER(work_orders.issue_description) LIKE ? OR LOWER(devices.brand) LIKE ? OR LOWER(devices.model) LIKE ? OR LOWER(devices.serial_number) LIKE ? OR LOWER(client_user.name) LIKE ?",
				q, q, q, q, q,
			)
		}
	}

	err := queryBuilder.Order("work_orders.created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *workOrderRepositoryImpl) Update(wo *models.WorkOrder) (*models.WorkOrder, error) {
	err := r.db.Save(wo).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(wo.ID)
}

func (r *workOrderRepositoryImpl) Delete(id string) error {
	result := r.db.Delete(&models.WorkOrder{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("work order not found")
	}
	return nil
}
