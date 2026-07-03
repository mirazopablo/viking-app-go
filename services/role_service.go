package services

import (
	"errors"

	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"gorm.io/gorm"
)

// ErrRoleNotFound indicates that the requested role does not exist in the database.
var ErrRoleNotFound = errors.New("role not found")

// RoleService defines the business logic interface for role management.
type RoleService interface {
	GetAllRoles() ([]models.Role, error)
	GetRoleByID(id string) (*models.Role, error)
	CreateRole(req *models.RoleCreateRequest) (*models.Role, error)
	UpdateRole(id string, req *models.RoleCreateRequest) (*models.Role, error)
	DeleteRole(id string) error
}

type roleServiceImpl struct {
	repo repositories.RoleRepository
}

// NewRoleService instantiates a new RoleService.
func NewRoleService(repo repositories.RoleRepository) RoleService {
	return &roleServiceImpl{repo: repo}
}

// GetAllRoles retrieves all roles.
func (s *roleServiceImpl) GetAllRoles() ([]models.Role, error) {
	return s.repo.FindAll()
}

// GetRoleByID fetches a role by ID or returns ErrRoleNotFound.
func (s *roleServiceImpl) GetRoleByID(id string) (*models.Role, error) {
	role, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

// CreateRole validates and creates a new role.
func (s *roleServiceImpl) CreateRole(req *models.RoleCreateRequest) (*models.Role, error) {
	role := &models.Role{
		Name: req.Name,
	}
	if err := s.repo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates an existing role's properties.
func (s *roleServiceImpl) UpdateRole(id string, req *models.RoleCreateRequest) (*models.Role, error) {
	existingRole, err := s.GetRoleByID(id)
	if err != nil {
		return nil, err
	}

	existingRole.Name = req.Name

	err = s.repo.Update(existingRole)
	if err != nil {
		return nil, err
	}
	return existingRole, nil
}

// DeleteRole removes a role by ID.
func (s *roleServiceImpl) DeleteRole(id string) error {
	err := s.repo.Delete(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrRoleNotFound
	}
	return err
}
