package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"gorm.io/gorm"
)

var (
	ErrDiagnosticPointNotFound = errors.New("diagnostic point not found")
)

// DiagnosticPointService defines business logic for managing multimedia evidences and diagnostics.
type DiagnosticPointService interface {
	AddDiagnosticPoint(id, workOrderID, clientID, description, imageURL string) (*models.DiagnosticPointResponseDto, error)
	GetByWorkOrderAndClient(workOrderID, clientID string) ([]models.DiagnosticPointResponseDto, error)
	DeleteDiagnosticPoint(id string) error
}

type diagnosticPointServiceImpl struct {
	repo          repositories.DiagnosticPointRepository
	workOrderRepo repositories.WorkOrderRepository
	userRepo      repositories.UserRepository
}

// NewDiagnosticPointService instantiates a new DiagnosticPointService.
func NewDiagnosticPointService(repo repositories.DiagnosticPointRepository, workOrderRepo repositories.WorkOrderRepository, userRepo repositories.UserRepository) DiagnosticPointService {
	return &diagnosticPointServiceImpl{repo: repo, workOrderRepo: workOrderRepo, userRepo: userRepo}
}

func (s *diagnosticPointServiceImpl) AddDiagnosticPoint(id, workOrderID, clientID, description, imageURL string) (*models.DiagnosticPointResponseDto, error) {
	if strings.TrimSpace(workOrderID) == "" {
		return nil, errors.New("workOrderId is required")
	}

	wo, err := s.workOrderRepo.FindByID(workOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	if clientID == "" {
		if wo.ClientID != nil {
			clientID = *wo.ClientID
		}
	} else {
		_, err = s.userRepo.FindByID(clientID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
	}

	dp := &models.DiagnosticPoint{
		ID:          id,
		WorkOrderID: workOrderID,
		ClientID:    clientID,
		Description: strings.TrimSpace(description),
		ImageURL:    imageURL,
	}

	saved, err := s.repo.Save(dp)
	if err != nil {
		return nil, err
	}

	return toDiagnosticPointResponseDto(saved), nil
}

func (s *diagnosticPointServiceImpl) GetByWorkOrderAndClient(workOrderID, clientID string) ([]models.DiagnosticPointResponseDto, error) {
	points, err := s.repo.FindByWorkOrderAndClient(workOrderID, clientID)
	if err != nil {
		return nil, err
	}

	res := make([]models.DiagnosticPointResponseDto, len(points))
	for i, dp := range points {
		res[i] = *toDiagnosticPointResponseDto(&dp)
	}
	return res, nil
}

func (s *diagnosticPointServiceImpl) DeleteDiagnosticPoint(id string) error {
	dp, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			return ErrDiagnosticPointNotFound
		}
		return err
	}

	err = s.repo.Delete(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrDiagnosticPointNotFound
		}
		return err
	}

	if dp.ImageURL != "" {
		relPath := strings.TrimPrefix(dp.ImageURL, "/auth/uploads/")
		relPath = strings.TrimPrefix(relPath, "/")
		if !strings.Contains(relPath, "..") {
			filePath := filepath.Join(config.AppConfig.UploadDir, relPath)
			_ = os.Remove(filePath)
		}
	}

	return nil
}

func toDiagnosticPointResponseDto(dp *models.DiagnosticPoint) *models.DiagnosticPointResponseDto {
	return &models.DiagnosticPointResponseDto{
		ID:          dp.ID,
		WorkOrderID: dp.WorkOrderID,
		ClientID:    dp.ClientID,
		Description: dp.Description,
		ImageURL:    dp.ImageURL,
		CreatedAt:   dp.CreatedAt.Format(time.RFC3339),
	}
}
