package repositories

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"gorm.io/gorm"
)

// UserRepository defines operations for interacting with User records in database.
type UserRepository interface {
	Create(user *models.User) error
	CreateWithRole(user *models.User, roleID uuid.UUID) error
	FindByID(id string) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindAll() ([]models.User, error)
	Update(user *models.User) error
	UpdateWithRole(user *models.User, roleID uuid.UUID) error
	Delete(id string) error
}

type userRepositoryImpl struct{}

// NewUserRepository instantiates a new UserRepository.
func NewUserRepository() UserRepository {
	return &userRepositoryImpl{}
}

// Create inserts a new user record into database without role association.
func (r *userRepositoryImpl) Create(user *models.User) error {
	return config.DB.Create(user).Error
}

// CreateWithRole inserts a new user record into database and creates the user-role association within a transaction.
func (r *userRepositoryImpl) CreateWithRole(user *models.User, roleID uuid.UUID) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		userRole := &models.UserRole{
			UserID: user.ID,
			RoleID: roleID,
		}
		if err := tx.Create(userRole).Error; err != nil {
			return err
		}

		user.UserRoles = []models.UserRole{*userRole}
		return nil
	})
}

// FindByID retrieves a user by ID, preloading their associated UserRoles and Roles.
func (r *userRepositoryImpl) FindByID(id string) (*models.User, error) {
	var user models.User
	err := config.DB.Preload("UserRoles.Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email address, preloading roles.
func (r *userRepositoryImpl) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := config.DB.Preload("UserRoles.Role").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindAll retrieves all user records from database, preloading roles.
func (r *userRepositoryImpl) FindAll() ([]models.User, error) {
	var users []models.User
	err := config.DB.Preload("UserRoles.Role").Find(&users).Error
	return users, err
}

// Update modifies an existing user record without altering role associations.
func (r *userRepositoryImpl) Update(user *models.User) error {
	return config.DB.Save(user).Error
}

// UpdateWithRole modifies an existing user record and updates their primary role association within a transaction.
func (r *userRepositoryImpl) UpdateWithRole(user *models.User, roleID uuid.UUID) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}

		// Remove existing role associations and replace with the updated role ID
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}

		userRole := &models.UserRole{
			UserID: user.ID,
			RoleID: roleID,
		}
		if err := tx.Create(userRole).Error; err != nil {
			return err
		}

		user.UserRoles = []models.UserRole{*userRole}
		return nil
	})
}

// Delete physically deletes a user and all their associated UserRoles within a transaction.
func (r *userRepositoryImpl) Delete(id string) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.User{}, "id = ?", id).Error
	})
}
