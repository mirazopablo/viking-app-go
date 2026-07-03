package repositories

import (
	"errors"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// RoleRepository defines the interface for role data operations.
type RoleRepository interface {
	FindAll() ([]models.Role, error)
	FindByID(id string) (*models.Role, error)
	Create(role *models.Role) error
	Update(role *models.Role) error
	Delete(id string) error
}

type roleRepositoryImpl struct{}

// NewRoleRepository instantiates a new RoleRepository.
func NewRoleRepository() RoleRepository {
	return &roleRepositoryImpl{}
}

// FindAll retrieves all roles from the database.
func (r *roleRepositoryImpl) FindAll() ([]models.Role, error) {
	var roles []models.Role
	err := config.DB.Find(&roles).Error
	return roles, err
}

// FindByID retrieves a single role by its UUID string.
func (r *roleRepositoryImpl) FindByID(id string) (*models.Role, error) {
	var role models.Role
	err := config.DB.Where("id = ?", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// Create inserts a new role into the database.
func (r *roleRepositoryImpl) Create(role *models.Role) error {
	return config.DB.Create(role).Error
}

// Update saves changes to an existing role.
func (r *roleRepositoryImpl) Update(role *models.Role) error {
	return config.DB.Save(role).Error
}

// Delete removes a role by its UUID string.
func (r *roleRepositoryImpl) Delete(id string) error {
	result := config.DB.Where("id = ?", id).Delete(&models.Role{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
