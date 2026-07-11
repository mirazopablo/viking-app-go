package services

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrUserNotFound indicates the requested user ID or email does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyTaken indicates the email is already in use by another account.
	ErrEmailAlreadyTaken = errors.New("email address is already registered")
	// ErrInvalidCreds indicates login credentials failed validation.
	ErrInvalidCreds = errors.New("invalid email or password")
	// ErrPasswordRequired indicates that a password is mandatory for non-client roles.
	ErrPasswordRequired = errors.New("password is required for non-client roles")
)

// UserService defines user management and authentication logic.
type UserService interface {
	RegisterUser(req *models.RegisterDto) (*models.UserResponseDto, error)
	LoginUser(req *models.LoginUserDto) (string, error)
	ValidateTokenString(tokenString string) bool
	GetAllUsers() ([]models.UserResponseDto, error)
	SearchUsers(id, dni, name, email, phone, query string) ([]models.UserResponseDto, error)
	AutocompleteUsers(query string) ([]models.UserAutocompleteDto, error)
	GetUserByID(id string) (*models.UserResponseDto, error)
	UpdateUser(id string, req *models.RegisterDto) (*models.UserResponseDto, error)
	DeleteUser(id string) error
}

type userServiceImpl struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	jwtSvc   JWTService
}

// NewUserService instantiates a new UserService.
func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, jwtSvc JWTService) UserService {
	return &userServiceImpl{
		userRepo: userRepo,
		roleRepo: roleRepo,
		jwtSvc:   jwtSvc,
	}
}

// RegisterUser hashes password and saves new user to database within a transaction.
func (s *userServiceImpl) RegisterUser(req *models.RegisterDto) (*models.UserResponseDto, error) {
	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyTaken
	}

	role, err := s.roleRepo.FindByID(req.RoleID.String())
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("invalid role specified")
	}

	desc := strings.ToUpper(strings.TrimSpace(role.Name))
	isClientRole := desc == "CLIENTE" || desc == "CLIENT"

	if !isClientRole && strings.TrimSpace(req.Password) == "" {
		return nil, ErrPasswordRequired
	}

	var passwordPtr *string
	if strings.TrimSpace(req.Password) != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedStr := string(hashedPassword)
		passwordPtr = &hashedStr
	}

	user := &models.User{
		Name:                 req.Name,
		Dni:                  req.Dni,
		Address:              req.Address,
		PhoneNumber:          req.PhoneNumber,
		SecondaryPhoneNumber: req.SecondaryPhoneNumber,
		Email:                req.Email,
		Password:             passwordPtr,
	}

	if err := s.userRepo.CreateWithRole(user, req.RoleID); err != nil {
		return nil, err
	}

	return user.ToResponseDto(), nil
}

// LoginUser verifies credentials and returns a signed JWT token.
func (s *userServiceImpl) LoginUser(req *models.LoginUserDto) (string, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return "", err
	}
	if user == nil || user.Password == nil || strings.TrimSpace(*user.Password) == "" {
		return "", ErrInvalidCreds
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.Password))
	if err != nil {
		return "", ErrInvalidCreds
	}

	primaryRoleID := user.GetPrimaryRoleID()
	if primaryRoleID == uuid.Nil {
		return "", ErrInvalidCreds
	}

	return s.jwtSvc.GenerateToken(user.ID.String(), primaryRoleID.String())
}

// ValidateTokenString checks if a token string is valid and unexpired.
func (s *userServiceImpl) ValidateTokenString(tokenString string) bool {
	_, err := s.jwtSvc.ValidateToken(tokenString)
	return err == nil
}

// GetAllUsers retrieves all users converted to safe response DTOs.
func (s *userServiceImpl) GetAllUsers() ([]models.UserResponseDto, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var res []models.UserResponseDto
	for _, u := range users {
		res = append(res, *u.ToResponseDto())
	}
	return res, nil
}

func (s *userServiceImpl) SearchUsers(id, dni, name, email, phone, query string) ([]models.UserResponseDto, error) {
	users, err := s.userRepo.Search(id, dni, name, email, phone, query)
	if err != nil {
		return nil, err
	}

	var res []models.UserResponseDto
	for _, u := range users {
		res = append(res, *u.ToResponseDto())
	}
	return res, nil
}

// AutocompleteUsers returns a lightweight projection of users for selectors without sensitive PII.
func (s *userServiceImpl) AutocompleteUsers(query string) ([]models.UserAutocompleteDto, error) {
	users, err := s.userRepo.Search("", "", "", "", "", query)
	if err != nil {
		return nil, err
	}

	var res []models.UserAutocompleteDto
	for _, u := range users {
		res = append(res, *u.ToAutocompleteDto())
	}
	return res, nil
}

// GetUserByID retrieves a specific user by ID.
func (s *userServiceImpl) GetUserByID(id string) (*models.UserResponseDto, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user.ToResponseDto(), nil
}

// UpdateUser updates user profile properties and optionally hashes a new password.
func (s *userServiceImpl) UpdateUser(id string, req *models.RegisterDto) (*models.UserResponseDto, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Name = req.Name
	user.Dni = req.Dni
	user.Address = req.Address
	user.PhoneNumber = req.PhoneNumber
	user.SecondaryPhoneNumber = req.SecondaryPhoneNumber
	user.Email = req.Email

	if req.Password != "" && req.Password != "------" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedStr := string(hashed)
		user.Password = &hashedStr
	}

	if err := s.userRepo.UpdateWithRole(user, req.RoleID); err != nil {
		return nil, err
	}
	return user.ToResponseDto(), nil
}

// DeleteUser removes a user record by ID.
func (s *userServiceImpl) DeleteUser(id string) error {
	return s.userRepo.Delete(id)
}
