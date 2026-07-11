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
	ErrWorkOrderNotFound   = errors.New("work order not found")
	ErrInvalidSecurityCode = errors.New("invalid security code")
)

// WorkOrderService defines business logic for managing repair jobs.
type WorkOrderService interface {
	CreateWorkOrder(dto *models.WorkOrderCreateRequest, staffID string) (*models.WorkOrderResponseDto, error)
	UpdateWorkOrderStatus(id string, dto *models.WorkOrderStatusUpdateRequestDto) (*models.WorkOrderResponseDto, error)
	GetWorkOrderByID(id string) (*models.WorkOrderResponseDto, error)
	RegenerateSecurityCode(id string) (*models.SecurityCodeResponseDto, error)
	SearchWorkOrders(staffId string, clientDni string, deviceSerialNumber string, query string) ([]models.WorkOrderSummaryDto, error)
	DeleteWorkOrder(id string) error
	GetPublicWorkOrderStatus(id string, securityCode string) (*models.WorkOrderPublicStatusResponseDto, error)
	GetPublicWorkOrderStatusByDNI(clientDni int32, securityCode string) (*models.WorkOrderPublicStatusResponseDto, error)
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
	repo                repositories.WorkOrderRepository
	userRepo            repositories.UserRepository
	deviceRepo          repositories.DeviceRepository
	diagnosticPointRepo repositories.DiagnosticPointRepository
}

// NewWorkOrderService instantiates a new WorkOrderService.
func NewWorkOrderService(repo repositories.WorkOrderRepository, userRepo repositories.UserRepository, deviceRepo repositories.DeviceRepository, diagnosticPointRepo repositories.DiagnosticPointRepository) WorkOrderService {
	return &workOrderServiceImpl{repo: repo, userRepo: userRepo, deviceRepo: deviceRepo, diagnosticPointRepo: diagnosticPointRepo}
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
		Notes:                dto.Notes,
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

func (s *workOrderServiceImpl) SearchWorkOrders(staffId string, clientDni string, deviceSerialNumber string, query string) ([]models.WorkOrderSummaryDto, error) {
	orders, err := s.repo.Search(staffId, clientDni, deviceSerialNumber, query)
	if err != nil {
		return nil, err
	}

	res := make([]models.WorkOrderSummaryDto, len(orders))
	for i, wo := range orders {
		res[i] = *toWorkOrderSummaryDto(&wo)
	}
	return res, nil
}

func (s *workOrderServiceImpl) GetWorkOrderByID(id string) (*models.WorkOrderResponseDto, error) {
	wo, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}
	return toWorkOrderResponseDto(wo), nil
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

func (s *workOrderServiceImpl) RegenerateSecurityCode(id string) (*models.SecurityCodeResponseDto, error) {
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

	return toSecurityCodeResponseDto(updated, plainCode), nil
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
		Notes:              wo.Notes,
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

func toSecurityCodeResponseDto(wo *models.WorkOrder, plainCode string) *models.SecurityCodeResponseDto {
	clientName := wo.ClientNameSnapshot
	if clientName == "" && wo.Client.ID != uuid.Nil {
		clientName = wo.Client.Name
	}
	return &models.SecurityCodeResponseDto{
		ID:           wo.ID,
		SecurityCode: plainCode,
		ClientName:   clientName,
	}
}

func toWorkOrderSummaryDto(wo *models.WorkOrder) *models.WorkOrderSummaryDto {
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

	deviceBrand := wo.DeviceBrandSnapshot
	if deviceBrand == "" && wo.Device.ID != "" {
		deviceBrand = wo.Device.Brand
	}
	deviceModel := wo.DeviceModelSnapshot
	if deviceModel == "" && wo.Device.ID != "" {
		deviceModel = wo.Device.Model
	}

	return &models.WorkOrderSummaryDto{
		ID:               wo.ID,
		ClientID:         clientIDStr,
		ClientName:       clientName,
		DeviceID:         deviceIDStr,
		DeviceBrand:      deviceBrand,
		DeviceModel:      deviceModel,
		IssueDescription: wo.IssueDescription,
		RepairStatus:     wo.RepairStatus,
		CreatedAt:        wo.CreatedAt.Format(time.RFC3339),
	}
}

func (s *workOrderServiceImpl) GetPublicWorkOrderStatus(id string, securityCode string) (*models.WorkOrderPublicStatusResponseDto, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(securityCode) == "" {
		return nil, ErrInvalidSecurityCode
	}

	wo, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(wo.SecurityCodeHash), []byte(strings.TrimSpace(securityCode))); err != nil {
		return nil, ErrInvalidSecurityCode
	}

	points, err := s.diagnosticPointRepo.FindByWorkOrderAndClient(wo.ID, "")
	if err != nil {
		return nil, err
	}

	dpDtos := make([]models.DiagnosticPointResponseDto, len(points))
	for i, dp := range points {
		dpDtos[i] = *toDiagnosticPointResponseDto(&dp)
	}

	return &models.WorkOrderPublicStatusResponseDto{
		WorkOrder:        *toWorkOrderResponseDto(wo),
		DiagnosticPoints: dpDtos,
	}, nil
}

func (s *workOrderServiceImpl) GetPublicWorkOrderStatusByDNI(clientDni int32, securityCode string) (*models.WorkOrderPublicStatusResponseDto, error) {
	if clientDni <= 0 || strings.TrimSpace(securityCode) == "" {
		return nil, ErrInvalidSecurityCode
	}

	orders, err := s.repo.FindByClientDNI(clientDni)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrWorkOrderNotFound
	}

	var matchedOrder *models.WorkOrder
	trimmedCode := strings.TrimSpace(securityCode)
	for i := range orders {
		if err := bcrypt.CompareHashAndPassword([]byte(orders[i].SecurityCodeHash), []byte(trimmedCode)); err == nil {
			matchedOrder = &orders[i]
			break
		}
	}

	if matchedOrder == nil {
		return nil, ErrInvalidSecurityCode
	}

	points, err := s.diagnosticPointRepo.FindByWorkOrderAndClient(matchedOrder.ID, "")
	if err != nil {
		return nil, err
	}

	dpDtos := make([]models.DiagnosticPointResponseDto, len(points))
	for i, dp := range points {
		dpDtos[i] = *toDiagnosticPointResponseDto(&dp)
	}

	return &models.WorkOrderPublicStatusResponseDto{
		WorkOrder:        *toWorkOrderResponseDto(matchedOrder),
		DiagnosticPoints: dpDtos,
	}, nil
}
