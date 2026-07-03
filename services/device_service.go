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
	SearchDevices(id, serialNumber, brand string, userDni int32, query string) ([]models.DeviceResponseDto, error)
	DeleteDevice(id string) error
}

type deviceServiceImpl struct {
	repo     repositories.DeviceRepository
	userRepo repositories.UserRepository
}

// NewDeviceService instantiates a new DeviceService.
func NewDeviceService(repo repositories.DeviceRepository, userRepo repositories.UserRepository) DeviceService {
	return &deviceServiceImpl{repo: repo, userRepo: userRepo}
}

func (s *deviceServiceImpl) RegisterDevice(dto *models.DeviceCreateRequestDto) (*models.DeviceResponseDto, error) {
	// Verify user exists
	_, err := s.userRepo.FindByID(dto.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	device := &models.Device{
		Type:         dto.Type,
		Brand:        dto.Brand,
		Model:        dto.Model,
		SerialNumber: dto.SerialNumber,
		UserID:       dto.UserID,
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

	if dto.UserID != "" && dto.UserID != existing.UserID {
		_, err := s.userRepo.FindByID(dto.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
		existing.UserID = dto.UserID
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

func (s *deviceServiceImpl) SearchDevices(id, serialNumber, brand string, userDni int32, query string) ([]models.DeviceResponseDto, error) {
	devices, err := s.repo.Search(id, serialNumber, brand, userDni, query)
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
	return &models.DeviceResponseDto{
		ID:           d.ID,
		Type:         d.Type,
		Brand:        d.Brand,
		Model:        d.Model,
		SerialNumber: d.SerialNumber,
		UserID:       d.UserID,
		UserName:     d.User.Name,
		UserDni:      d.User.Dni,
	}
}
