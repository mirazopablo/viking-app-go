package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/repositories"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrBudgetNotFound = errors.New("budget not found")
)

// BudgetService defines business logic for managing dynamic budgets.
type BudgetService interface {
	SaveBudget(dto *models.BudgetSaveDto) (*models.BudgetResponseDto, error)
	GetBudgetByWorkOrder(workOrderID string, isPublic bool, securityCode ...string) (*models.BudgetResponseDto, error)
	UpdateBudgetStatus(id string, status string) (*models.BudgetResponseDto, error)
	DeleteBudget(id string) error
}

type budgetServiceImpl struct {
	repo                repositories.BudgetRepository
	workOrderRepo       repositories.WorkOrderRepository
	diagnosticPointRepo repositories.DiagnosticPointRepository
	notificationService NotificationService
}

// NewBudgetService instantiates a new BudgetService.
func NewBudgetService(
	repo repositories.BudgetRepository,
	workOrderRepo repositories.WorkOrderRepository,
	diagnosticPointRepo repositories.DiagnosticPointRepository,
	notificationService NotificationService,
) BudgetService {
	return &budgetServiceImpl{
		repo:                repo,
		workOrderRepo:       workOrderRepo,
		diagnosticPointRepo: diagnosticPointRepo,
		notificationService: notificationService,
	}
}

func (s *budgetServiceImpl) SaveBudget(dto *models.BudgetSaveDto) (*models.BudgetResponseDto, error) {
	if strings.TrimSpace(dto.WorkOrderID) == "" {
		return nil, errors.New("workOrderId is required")
	}

	wo, err := s.workOrderRepo.FindByID(dto.WorkOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	mode := strings.ToUpper(strings.TrimSpace(dto.Mode))
	if mode == "" {
		mode = models.BudgetModeMaintenance
	}
	if err := models.ValidateBudgetMode(mode); err != nil {
		return nil, err
	}

	status := strings.ToUpper(strings.TrimSpace(dto.Status))
	if status == "" {
		status = models.BudgetStatusDraft
	}
	if err := models.ValidateBudgetStatus(status); err != nil {
		return nil, err
	}

	currency := dto.Currency
	if currency == "" {
		currency = "$"
	}

	// Compute totals from items and labors
	var itemsSubtotal, totalSparePartsCost, totalSparePartsProfit float64
	for _, item := range dto.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		itemSubtotal := item.UnitPrice * qty
		if itemSubtotal == 0 && item.UnitPrice > 0 {
			itemSubtotal = item.UnitPrice
		}
		itemsSubtotal += itemSubtotal

		cost := item.CostPrice * qty
		totalSparePartsCost += cost

		profit := item.ProfitAmount
		if profit == 0 && (item.UnitPrice > 0 || item.CostPrice > 0) {
			profit = (item.UnitPrice - item.CostPrice) * qty
		}
		totalSparePartsProfit += profit
	}

	var laborTotal float64
	for _, labor := range dto.Labors {
		laborTotal += labor.Amount
	}

	grandTotal := itemsSubtotal + laborTotal
	totalEstimatedProfit := totalSparePartsProfit + laborTotal

	totalsDto := models.BudgetTotalsDto{
		ItemsSubtotal:         itemsSubtotal,
		LaborTotal:            laborTotal,
		GrandTotal:            grandTotal,
		TotalSparePartsCost:   totalSparePartsCost,
		TotalSparePartsProfit: totalSparePartsProfit,
		TotalEstimatedProfit:  totalEstimatedProfit,
	}

	// Prepare budget JSON object
	budgetJSONMap := map[string]interface{}{
		"workOrderId":        dto.WorkOrderID,
		"title":              dto.Title,
		"mode":               mode,
		"status":             status,
		"clientName":         dto.ClientName,
		"clientDni":          dto.ClientDni,
		"clientAddress":      dto.ClientAddress,
		"clientPhoneNumber":  dto.ClientPhoneNumber,
		"clientEmail":        dto.ClientEmail,
		"deviceModel":        dto.DeviceModel,
		"deviceSerialNumber": dto.DeviceSerialNumber,
		"currency":           currency,
		"taxPercentage":      dto.TaxPercentage,
		"items":              dto.Items,
		"labors":             dto.Labors,
		"notes":              dto.Notes,
		"termsAndConditions": dto.TermsAndConditions,
		"totals":             totalsDto,
	}
	if dto.Blocks != nil {
		budgetJSONMap["blocks"] = dto.Blocks
	}

	jsonBytes, err := json.Marshal(budgetJSONMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal budget data: %w", err)
	}

	// Check if budget already exists for this work order
	existing, err := s.repo.FindByWorkOrderID(dto.WorkOrderID)
	var savedBudget *models.Budget

	if err == nil && existing != nil {
		existing.Title = dto.Title
		existing.Mode = mode
		existing.Status = status
		existing.Currency = currency
		existing.TaxPercentage = dto.TaxPercentage
		existing.BudgetData = jsonBytes
		existing.ItemsSubtotal = itemsSubtotal
		existing.LaborTotal = laborTotal
		existing.GrandTotal = grandTotal
		existing.SparePartsCost = totalSparePartsCost
		existing.EstimatedProfit = totalEstimatedProfit

		savedBudget, err = s.repo.Update(existing)
		if err != nil {
			return nil, err
		}
	} else {
		newBudget := &models.Budget{
			WorkOrderID:     dto.WorkOrderID,
			Title:           dto.Title,
			Mode:            mode,
			Status:          status,
			Currency:        currency,
			TaxPercentage:   dto.TaxPercentage,
			BudgetData:      jsonBytes,
			ItemsSubtotal:   itemsSubtotal,
			LaborTotal:      laborTotal,
			GrandTotal:      grandTotal,
			SparePartsCost:  totalSparePartsCost,
			EstimatedProfit: totalEstimatedProfit,
		}
		savedBudget, err = s.repo.Save(newBudget)
		if err != nil {
			return nil, err
		}
	}

	// Automatically register diagnostic point for budget summary audit in timeline
	if s.diagnosticPointRepo != nil {
		summaryDesc := fmt.Sprintf("Presupuesto de taller publicado/actualizado: %s (Total: %s%.2f)", savedBudget.Title, currency, grandTotal)
		dp := &models.DiagnosticPoint{
			WorkOrderID: dto.WorkOrderID,
			ClientID:    wo.ClientID,
			EntryType:   models.EntryTypeBudgetSummary,
			Description: summaryDesc,
			ImageURL:    "",
		}
		savedDp, dpErr := s.diagnosticPointRepo.Save(dp)
		if dpErr == nil && s.notificationService != nil {
			s.notificationService.NotifyDiagnosticPointAdded(savedDp, wo)
		}
	}

	return toBudgetResponseDto(savedBudget, totalsDto, false), nil
}

func (s *budgetServiceImpl) GetBudgetByWorkOrder(workOrderID string, isPublic bool, securityCode ...string) (*models.BudgetResponseDto, error) {
	if strings.TrimSpace(workOrderID) == "" {
		return nil, errors.New("workOrderId is required")
	}

	if isPublic && len(securityCode) > 0 && strings.TrimSpace(securityCode[0]) != "" {
		secCode := strings.TrimSpace(securityCode[0])
		wo, err := s.workOrderRepo.FindByID(workOrderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrBudgetNotFound
			}
			return nil, err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(wo.SecurityCodeHash), []byte(secCode)); err != nil {
			return nil, ErrInvalidSecurityCode
		}
	}

	budget, err := s.repo.FindByWorkOrderID(workOrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, err
	}

	totals := extractTotalsFromBudget(budget)
	return toBudgetResponseDto(budget, totals, isPublic), nil
}

func (s *budgetServiceImpl) UpdateBudgetStatus(id string, status string) (*models.BudgetResponseDto, error) {
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	if err := models.ValidateBudgetStatus(upperStatus); err != nil {
		return nil, err
	}

	budget, err := s.repo.UpdateStatus(id, upperStatus)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBudgetNotFound
		}
		return nil, err
	}

	totals := extractTotalsFromBudget(budget)
	return toBudgetResponseDto(budget, totals, false), nil
}

func (s *budgetServiceImpl) DeleteBudget(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}

	err := s.repo.Delete(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBudgetNotFound
		}
		return err
	}

	return nil
}

func extractTotalsFromBudget(budget *models.Budget) models.BudgetTotalsDto {
	var totalSparePartsProfit float64
	if budget.ItemsSubtotal > 0 && budget.SparePartsCost > 0 {
		totalSparePartsProfit = budget.ItemsSubtotal - budget.SparePartsCost
	}
	return models.BudgetTotalsDto{
		ItemsSubtotal:         budget.ItemsSubtotal,
		LaborTotal:            budget.LaborTotal,
		GrandTotal:            budget.GrandTotal,
		TotalSparePartsCost:   budget.SparePartsCost,
		TotalSparePartsProfit: totalSparePartsProfit,
		TotalEstimatedProfit:  budget.EstimatedProfit,
	}
}

func toBudgetResponseDto(budget *models.Budget, totals models.BudgetTotalsDto, isPublic bool) *models.BudgetResponseDto {
	budgetData := budget.BudgetData

	if isPublic && len(budgetData) > 0 {
		// Sanitize public JSON data to remove internal financial costs and profit margins
		var parsed map[string]interface{}
		if err := json.Unmarshal(budgetData, &parsed); err == nil {
			if itemsRaw, ok := parsed["items"].([]interface{}); ok {
				sanitizedItems := make([]map[string]interface{}, len(itemsRaw))
				for i, itemRaw := range itemsRaw {
					if itemMap, ok := itemRaw.(map[string]interface{}); ok {
						cleanItem := make(map[string]interface{})
						for k, v := range itemMap {
							if k != "costPrice" && k != "profitMarginPercentage" && k != "profitAmount" {
								cleanItem[k] = v
							}
						}
						sanitizedItems[i] = cleanItem
					}
				}
				parsed["items"] = sanitizedItems
			}
			if totalsRaw, ok := parsed["totals"].(map[string]interface{}); ok {
				delete(totalsRaw, "totalSparePartsCost")
				delete(totalsRaw, "totalSparePartsProfit")
				delete(totalsRaw, "totalEstimatedProfit")
			}
			if cleanBytes, err := json.Marshal(parsed); err == nil {
				budgetData = cleanBytes
			}
		}

		// Also sanitize totals DTO for public views
		totals.TotalSparePartsCost = 0
		totals.TotalSparePartsProfit = 0
		totals.TotalEstimatedProfit = 0
	}

	return &models.BudgetResponseDto{
		ID:          budget.ID,
		WorkOrderID: budget.WorkOrderID,
		Title:       budget.Title,
		Mode:        budget.Mode,
		Status:      budget.Status,
		Currency:    budget.Currency,
		BudgetData:  budgetData,
		Totals:      totals,
		CreatedAt:   budget.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   budget.UpdatedAt.Format(time.RFC3339),
	}
}
