package repositories

import (
	"errors"
	"fmt"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// DiagnosticPointRepository defines data store interactions for diagnostic logs and multimedia evidence.
type DiagnosticPointRepository interface {
	Save(dp *models.DiagnosticPoint) (*models.DiagnosticPoint, error)
	FindByID(id string) (*models.DiagnosticPoint, error)
	FindByWorkOrderAndClient(workOrderID, clientID string) ([]models.DiagnosticPoint, error)
	Delete(id string) error
}

type diagnosticPointRepositoryImpl struct {
	db *gorm.DB
}

// NewDiagnosticPointRepository instantiates a new DiagnosticPointRepository with GORM.
func NewDiagnosticPointRepository() DiagnosticPointRepository {
	return &diagnosticPointRepositoryImpl{db: config.DB}
}

func (r *diagnosticPointRepositoryImpl) Save(dp *models.DiagnosticPoint) (*models.DiagnosticPoint, error) {
	err := r.db.Create(dp).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(dp.ID)
}

func (r *diagnosticPointRepositoryImpl) FindByID(id string) (*models.DiagnosticPoint, error) {
	var dp models.DiagnosticPoint
	err := r.db.First(&dp, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &dp, nil
}

func (r *diagnosticPointRepositoryImpl) FindByWorkOrderAndClient(workOrderID, clientID string) ([]models.DiagnosticPoint, error) {
	var points []models.DiagnosticPoint
	query := r.db.Model(&models.DiagnosticPoint{})
	if workOrderID != "" {
		query = query.Where("work_order_id = ?", workOrderID)
	}
	if clientID != "" {
		query = query.Where("client_id = ?", clientID)
	}
	err := query.Order("created_at DESC").Find(&points).Error
	return points, err
}

func (r *diagnosticPointRepositoryImpl) Delete(id string) error {
	result := r.db.Delete(&models.DiagnosticPoint{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("diagnostic point not found")
	}
	return nil
}
