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
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyTaken = errors.New("email address is already registered")
	ErrInvalidCreds      = errors.New("invalid email or password")
	ErrPasswordRequired  = errors.New("password is required for non-client roles")
)

type UserService interface {
	RegisterUser(req *models.RegisterDto) (*models.UserResponseDto, error)
	LoginUser(req *models.LoginUserDto) (*models.LoginResponseDto, error)
	ValidateTokenString(tokenString string) bool
	GetAllUsers(page, limit int) (*models.PaginatedResponse[models.UserResponseDto], error)
	SearchUsers(id, dni, name, email, phone, query string, page, limit int) (*models.PaginatedResponse[models.UserResponseDto], error)
	AutocompleteUsers(query string, limit int) ([]models.UserAutocompleteDto, error)
	GetUserByID(id string) (*models.UserResponseDto, error)
	UpdateUser(id string, req *models.RegisterDto) (*models.UserResponseDto, error)
	DeleteUser(id string) error
}

type userServiceImpl struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
	jwtSvc   JWTService
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository, jwtSvc JWTService) UserService {
	return &userServiceImpl{
		userRepo: userRepo,
		roleRepo: roleRepo,
		jwtSvc:   jwtSvc,
	}
}

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
	isClientRole := desc == models.RoleClient || desc == "CLIENT"

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

func (s *userServiceImpl) LoginUser(req *models.LoginUserDto) (*models.LoginResponseDto, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Password == nil || strings.TrimSpace(*user.Password) == "" {
		return nil, ErrInvalidCreds
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.Password))
	if err != nil {
		return nil, ErrInvalidCreds
	}

	primaryRoleID := user.GetPrimaryRoleID()
	if primaryRoleID == uuid.Nil {
		return nil, ErrInvalidCreds
	}

	primaryRoleName := user.GetPrimaryRoleName()

	token, err := s.jwtSvc.GenerateToken(user.ID.String(), primaryRoleID.String(), primaryRoleName)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponseDto{
		Token: token,
		Type:  "Bearer",
		User:  user.ToResponseDto(),
	}, nil
}

func (s *userServiceImpl) ValidateTokenString(tokenString string) bool {
	_, err := s.jwtSvc.ValidateToken(tokenString)
	return err == nil
}

func (s *userServiceImpl) GetAllUsers(page, limit int) (*models.PaginatedResponse[models.UserResponseDto], error) {
	users, total, err := s.userRepo.FindAll(page, limit)
	if err != nil {
		return nil, err
	}

	var res = make([]models.UserResponseDto, 0, len(users))
	for _, u := range users {
		res = append(res, *u.ToResponseDto())
	}
	
	return &models.PaginatedResponse[models.UserResponseDto]{
		Data:  res,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *userServiceImpl) SearchUsers(id, dni, name, email, phone, query string, page, limit int) (*models.PaginatedResponse[models.UserResponseDto], error) {
	users, total, err := s.userRepo.Search(id, dni, name, email, phone, query, page, limit)
	if err != nil {
		return nil, err
	}

	var res = make([]models.UserResponseDto, 0, len(users))
	for _, u := range users {
		res = append(res, *u.ToResponseDto())
	}
	
	return &models.PaginatedResponse[models.UserResponseDto]{
		Data:  res,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *userServiceImpl) AutocompleteUsers(query string, limit int) ([]models.UserAutocompleteDto, error) {
	users, _, err := s.userRepo.Search("", "", "", "", "", query, 1, limit)
	if err != nil {
		return nil, err
	}

	var res = make([]models.UserAutocompleteDto, 0, len(users))
	for _, u := range users {
		res = append(res, *u.ToAutocompleteDto())
	}
	return res, nil
}

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

	if req.Password != "" {
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

func (s *userServiceImpl) DeleteUser(id string) error {
	return s.userRepo.Delete(id)
}
