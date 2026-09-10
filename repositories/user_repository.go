package repositories

import (
	"errors"
	"strings"

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
	FindAll(page, limit int) ([]models.User, int64, error)
	Search(id, dni, name, email, phone, query string, page, limit int) ([]models.User, int64, error)
	Update(user *models.User) error
	UpdateWithRole(user *models.User, roleID uuid.UUID) error
	Delete(id string) error
	FindByPhone(phone string) (*models.User, error)
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

// FindByPhone retrieves a user by their phone number.
func (r *userRepositoryImpl) FindByPhone(phone string) (*models.User, error) {
	var user models.User
	err := config.DB.Where("phone_number = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindAll retrieves all user records from database, paginated.
func (r *userRepositoryImpl) FindAll(page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	db := config.DB.Model(&models.User{})
	db.Count(&total)

	offset := (page - 1) * limit
	err := db.Preload("UserRoles.Role").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// Search retrieves users filtered by partial matching or general query, paginated.
func (r *userRepositoryImpl) Search(id, dni, name, email, phone, query string, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	queryBuilder := config.DB.Model(&models.User{})

	if id != "" {
		queryBuilder = queryBuilder.Where("id = ?", id)
	}
	if dni != "" {
		queryBuilder = queryBuilder.Where("CAST(dni AS TEXT) ILIKE ?", "%"+dni+"%")
	}
	if name != "" {
		queryBuilder = queryBuilder.Where("name ILIKE ?", "%"+name+"%")
	}
	if email != "" {
		queryBuilder = queryBuilder.Where("email ILIKE ?", "%"+email+"%")
	}
	if phone != "" {
		queryBuilder = queryBuilder.Where("phone_number ILIKE ?", "%"+phone+"%")
	}
	if query != "" {
		lowerQ := strings.ToLower(strings.TrimSpace(query))
		q := "%" + lowerQ + "%"
		queryBuilder = queryBuilder.Where(
			"LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(phone_number) LIKE ? OR CAST(dni AS TEXT) LIKE ?",
			q, q, q, q,
		)
	}

	queryBuilder.Count(&total)

	offset := (page - 1) * limit
	err := queryBuilder.Preload("UserRoles.Role").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// Update modifies an existing user record without altering role associations.
func (r *userRepositoryImpl) Update(user *models.User) error {
	return config.DB.Save(user).Error
}

// UpdateWithRole modifies an existing user record and updates their primary role association using GORM.
func (r *userRepositoryImpl) UpdateWithRole(user *models.User, roleID uuid.UUID) error {
	if err := config.DB.Save(user).Error; err != nil {
		return err
	}

	userRole := &models.UserRole{
		UserID: user.ID,
		RoleID: roleID,
	}
	
	// Replace existing associations
	return config.DB.Model(user).Association("UserRoles").Replace(userRole)
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
