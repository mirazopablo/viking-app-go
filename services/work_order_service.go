package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrWorkOrderNotFound = errors.New("work order not found")
)

// WorkOrderService defines business logic for managing repair jobs.
type WorkOrderService interface {
	CreateWorkOrder(dto *models.WorkOrderCreateRequest, staffID string) (*models.WorkOrderResponseDto, error)
	UpdateWorkOrderStatus(id string, dto *models.WorkOrderStatusUpdateRequestDto) (*models.WorkOrderResponseDto, error)
	RegenerateSecurityCode(id string) (*models.WorkOrderResponseDto, error)
	SearchWorkOrders(staffId string, clientDni int32, deviceSerialNumber string, query string) ([]models.WorkOrderResponseDto, error)
	DeleteWorkOrder(id string) error
}

func generateSecurityCode() (string, string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	for i, b := range bytes {
		bytes[i] = charset[int(b)%len(charset)]
	}
	code := fmt.Sprintf("WOVIK-%s", string(bytes))
	hashed, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return code, string(hashed), nil
}

type workOrderServiceImpl struct {
	repo       repositories.WorkOrderRepository
	userRepo   repositories.UserRepository
	deviceRepo repositories.DeviceRepository
}

// NewWorkOrderService instantiates a new WorkOrderService.
func NewWorkOrderService(repo repositories.WorkOrderRepository, userRepo repositories.UserRepository, deviceRepo repositories.DeviceRepository) WorkOrderService {
	return &workOrderServiceImpl{repo: repo, userRepo: userRepo, deviceRepo: deviceRepo}
}

func (s *workOrderServiceImpl) CreateWorkOrder(dto *models.WorkOrderCreateRequest, staffID string) (*models.WorkOrderResponseDto, error) {
	if err := models.ValidateRepairStatus(dto.RepairStatus); err != nil {
		return nil, err
	}

	// Verify client exists
	client, err := s.userRepo.FindByID(dto.ClientID)
	if err != nil || client == nil {
		return nil, ErrUserNotFound
	}

	// Verify device exists
	device, err := s.deviceRepo.FindByID(dto.DeviceID)
	if err != nil || device == nil {
		return nil, ErrDeviceNotFound
	}

	plainCode, hashedCode, err := generateSecurityCode()
	if err != nil {
		return nil, err
	}

	var staffPtr *string
	if staffID != "" {
		staffPtr = &staffID
	}

	wo := &models.WorkOrder{
		ClientID:             &dto.ClientID,
		DeviceID:             &dto.DeviceID,
		StaffID:              staffPtr,
		SecurityCodeHash:     hashedCode,
		ClientNameSnapshot:   client.Name,
		ClientDniSnapshot:    client.Dni,
		ClientPhoneSnapshot:  client.PhoneNumber,
		DeviceBrandSnapshot:  device.Brand,
		DeviceModelSnapshot:  device.Model,
		DeviceSerialSnapshot: device.SerialNumber,
		IssueDescription:     dto.IssueDescription,
		RepairStatus:         strings.ToUpper(strings.TrimSpace(dto.RepairStatus)),
	}

	saved, err := s.repo.Save(wo)
	if err != nil {
		return nil, err
	}

	res := toWorkOrderResponseDto(saved)
	res.SecurityCode = plainCode
	return res, nil
}

func (s *workOrderServiceImpl) UpdateWorkOrderStatus(id string, dto *models.WorkOrderStatusUpdateRequestDto) (*models.WorkOrderResponseDto, error) {
	if err := models.ValidateRepairStatus(dto.RepairStatus); err != nil {
		return nil, err
	}

	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	existing.RepairStatus = strings.ToUpper(strings.TrimSpace(dto.RepairStatus))

	updated, err := s.repo.Update(existing)
	if err != nil {
		return nil, err
	}

	return toWorkOrderResponseDto(updated), nil
}

func (s *workOrderServiceImpl) SearchWorkOrders(staffId string, clientDni int32, deviceSerialNumber string, query string) ([]models.WorkOrderResponseDto, error) {
	orders, err := s.repo.Search(staffId, clientDni, deviceSerialNumber, query)
	if err != nil {
		return nil, err
	}

	res := make([]models.WorkOrderResponseDto, len(orders))
	for i, wo := range orders {
		res[i] = *toWorkOrderResponseDto(&wo)
	}
	return res, nil
}

func (s *workOrderServiceImpl) DeleteWorkOrder(id string) error {
	err := s.repo.Delete(id)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return ErrWorkOrderNotFound
	}

	workOrderDir := filepath.Join(config.AppConfig.UploadDir, id)
	if err := os.RemoveAll(workOrderDir); err != nil && !os.IsNotExist(err) {
		// Log or ignore non-existence errors to preserve DB operation consistency
	}

	return err
}

func (s *workOrderServiceImpl) RegenerateSecurityCode(id string) (*models.WorkOrderResponseDto, error) {
	wo, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	plainCode, hashedCode, err := generateSecurityCode()
	if err != nil {
		return nil, err
	}

	wo.SecurityCodeHash = hashedCode
	updated, err := s.repo.Update(wo)
	if err != nil {
		return nil, err
	}

	res := toWorkOrderResponseDto(updated)
	res.SecurityCode = plainCode
	return res, nil
}

func toWorkOrderResponseDto(wo *models.WorkOrder) *models.WorkOrderResponseDto {
	clientIDStr := ""
	if wo.ClientID != nil {
		clientIDStr = *wo.ClientID
	}
	deviceIDStr := ""
	if wo.DeviceID != nil {
		deviceIDStr = *wo.DeviceID
	}

	clientName := wo.ClientNameSnapshot
	if clientName == "" && wo.Client.ID != uuid.Nil {
		clientName = wo.Client.Name
	}
	clientDni := wo.ClientDniSnapshot
	if clientDni == 0 && wo.Client.ID != uuid.Nil {
		clientDni = wo.Client.Dni
	}
	clientPhone := wo.ClientPhoneSnapshot
	if clientPhone == "" && wo.Client.ID != uuid.Nil {
		clientPhone = wo.Client.PhoneNumber
	}

	deviceBrand := wo.DeviceBrandSnapshot
	if deviceBrand == "" && wo.Device.ID != "" {
		deviceBrand = wo.Device.Brand
	}
	deviceModel := wo.DeviceModelSnapshot
	if deviceModel == "" && wo.Device.ID != "" {
		deviceModel = wo.Device.Model
	}
	deviceSerial := wo.DeviceSerialSnapshot
	if deviceSerial == "" && wo.Device.ID != "" {
		deviceSerial = wo.Device.SerialNumber
	}

	dto := &models.WorkOrderResponseDto{
		ID:                 wo.ID,
		ClientID:           clientIDStr,
		ClientName:         clientName,
		ClientDni:          clientDni,
		ClientPhone:        clientPhone,
		DeviceID:           deviceIDStr,
		DeviceBrand:        deviceBrand,
		DeviceModel:        deviceModel,
		DeviceSerialNumber: deviceSerial,
		IssueDescription:   wo.IssueDescription,
		RepairStatus:       wo.RepairStatus,
		CreatedAt:          wo.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          wo.UpdatedAt.Format(time.RFC3339),
	}
	if wo.StaffID != nil && *wo.StaffID != "" {
		dto.StaffID = *wo.StaffID
		if wo.Staff != nil {
			dto.StaffName = wo.Staff.Name
		}
	}
	return dto
}
