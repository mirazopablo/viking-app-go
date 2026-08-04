package repositories

import (
	"errors"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// BudgetRepository defines data store interactions for dynamic budgets.
type BudgetRepository interface {
	Save(budget *models.Budget) (*models.Budget, error)
	Update(budget *models.Budget) (*models.Budget, error)
	FindByID(id string) (*models.Budget, error)
	FindByWorkOrderID(workOrderID string) (*models.Budget, error)
	UpdateStatus(id string, status string) (*models.Budget, error)
	Delete(id string) error
}

type budgetRepositoryImpl struct {
	db *gorm.DB
}

// NewBudgetRepository instantiates a new BudgetRepository with GORM.
func NewBudgetRepository() BudgetRepository {
	return &budgetRepositoryImpl{db: config.DB}
}

func (r *budgetRepositoryImpl) Save(budget *models.Budget) (*models.Budget, error) {
	err := r.db.Create(budget).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(budget.ID)
}

func (r *budgetRepositoryImpl) Update(budget *models.Budget) (*models.Budget, error) {
	err := r.db.Save(budget).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(budget.ID)
}

func (r *budgetRepositoryImpl) FindByID(id string) (*models.Budget, error) {
	var budget models.Budget
	err := r.db.First(&budget, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &budget, nil
}

func (r *budgetRepositoryImpl) FindByWorkOrderID(workOrderID string) (*models.Budget, error) {
	var budget models.Budget
	err := r.db.Where("work_order_id = ?", workOrderID).Order("updated_at DESC").First(&budget).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &budget, nil
}

func (r *budgetRepositoryImpl) UpdateStatus(id string, status string) (*models.Budget, error) {
	budget, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}
	budget.Status = status
	err = r.db.Model(budget).Update("status", status).Error
	if err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

func (r *budgetRepositoryImpl) Delete(id string) error {
	result := r.db.Unscoped().Delete(&models.Budget{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
