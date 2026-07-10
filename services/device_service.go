package services

import (
	"errors"
	"strings"

	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"gorm.io/gorm"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
)

// DeviceService defines business operations for devices.
type DeviceService interface {
	RegisterDevice(dto *models.DeviceCreateRequestDto) (*models.DeviceResponseDto, error)
	UpdateDevice(id string, dto *models.DeviceUpdateRequestDto) (*models.DeviceResponseDto, error)
	SearchDevices(id, serialNumber, brand, userDni, userName, query string) ([]models.DeviceResponseDto, error)
	DeleteDevice(id string) error
}

type deviceServiceImpl struct {
	repo    repositories.DeviceRepository
	userSvc UserService
}

// NewDeviceService instantiates a new DeviceService.
func NewDeviceService(repo repositories.DeviceRepository, userSvc UserService) DeviceService {
	return &deviceServiceImpl{repo: repo, userSvc: userSvc}
}

func (s *deviceServiceImpl) RegisterDevice(dto *models.DeviceCreateRequestDto) (*models.DeviceResponseDto, error) {
	// Verify user exists and fetch snapshot data via UserService
	user, err := s.userSvc.GetUserByID(dto.UserID)
	if err != nil {
		return nil, err
	}

	device := &models.Device{
		Type:         dto.Type,
		Brand:        dto.Brand,
		Model:        dto.Model,
		SerialNumber: dto.SerialNumber,
		UserID:       &dto.UserID,
		UserName:     user.Name,
		UserDni:      user.Dni,
		UserPhone:    user.PhoneNumber,
	}

	saved, err := s.repo.Save(device)
	if err != nil {
		return nil, err
	}

	return toDeviceResponseDto(saved), nil
}

func (s *deviceServiceImpl) UpdateDevice(id string, dto *models.DeviceUpdateRequestDto) (*models.DeviceResponseDto, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}

	if dto.UserID != "" && (existing.UserID == nil || dto.UserID != *existing.UserID) {
		user, err := s.userSvc.GetUserByID(dto.UserID)
		if err != nil {
			return nil, err
		}
		existing.UserID = &dto.UserID
		existing.UserName = user.Name
		existing.UserDni = user.Dni
		existing.UserPhone = user.PhoneNumber
	}

	if dto.Type != "" {
		existing.Type = dto.Type
	}
	if dto.Brand != "" {
		existing.Brand = dto.Brand
	}
	if dto.Model != "" {
		existing.Model = dto.Model
	}
	if dto.SerialNumber != "" {
		existing.SerialNumber = dto.SerialNumber
	}

	updated, err := s.repo.Update(existing)
	if err != nil {
		return nil, err
	}

	return toDeviceResponseDto(updated), nil
}

func (s *deviceServiceImpl) SearchDevices(id, serialNumber, brand, userDni, userName, query string) ([]models.DeviceResponseDto, error) {
	devices, err := s.repo.Search(id, serialNumber, brand, userDni, userName, query)
	if err != nil {
		return nil, err
	}

	res := make([]models.DeviceResponseDto, len(devices))
	for i, d := range devices {
		res[i] = *toDeviceResponseDto(&d)
	}
	return res, nil
}

func (s *deviceServiceImpl) DeleteDevice(id string) error {
	err := s.repo.Delete(id)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return ErrDeviceNotFound
	}
	return err
}

func toDeviceResponseDto(d *models.Device) *models.DeviceResponseDto {
	userIDStr := ""
	if d.UserID != nil {
		userIDStr = *d.UserID
	}

	userName := d.UserName
	if userName == "" && d.User.Name != "" {
		userName = d.User.Name
	}
	userDni := d.UserDni
	if userDni == 0 && d.User.Dni != 0 {
		userDni = d.User.Dni
	}
	userPhone := d.UserPhone
	if userPhone == "" && d.User.PhoneNumber != "" {
		userPhone = d.User.PhoneNumber
	}

	return &models.DeviceResponseDto{
		ID:           d.ID,
		Type:         d.Type,
		Brand:        d.Brand,
		Model:        d.Model,
		SerialNumber: d.SerialNumber,
		UserID:       userIDStr,
		UserName:     userName,
		UserDni:      userDni,
		UserPhone:    userPhone,
	}
}
