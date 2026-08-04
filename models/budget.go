package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Budget mode constants
const (
	BudgetModeMaintenance  = "MAINTENANCE"
	BudgetModeNewEquipment = "NEW_EQUIPMENT"
)

// Budget status constants
const (
	BudgetStatusDraft    = "DRAFT"
	BudgetStatusSent     = "SENT"
	BudgetStatusApproved = "APPROVED"
	BudgetStatusRejected = "REJECTED"
)

// Budget represents a structured dynamic budget attached to a work order.
type Budget struct {
	ID              string          `gorm:"type:uuid;primary_key;" json:"id"`
	WorkOrderID     string          `gorm:"type:uuid;not null;index" json:"workOrderId"`
	WorkOrder       WorkOrder       `gorm:"foreignKey:WorkOrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Title           string          `gorm:"type:varchar(255);not null" json:"title"`
	Mode            string          `gorm:"type:varchar(50);not null" json:"mode"`
	Status          string          `gorm:"type:varchar(50);not null;default:'DRAFT'" json:"status"`
	Currency        string          `gorm:"type:varchar(10);not null;default:'$'" json:"currency"`
	TaxPercentage   float64         `gorm:"type:numeric(5,2);not null;default:0.00" json:"taxPercentage"`
	BudgetData      json.RawMessage `gorm:"type:jsonb;not null" json:"budgetData"`
	ItemsSubtotal   float64         `gorm:"type:numeric(12,2);not null;default:0.00" json:"itemsSubtotal"`
	LaborTotal      float64         `gorm:"type:numeric(12,2);not null;default:0.00" json:"laborTotal"`
	GrandTotal      float64         `gorm:"type:numeric(12,2);not null;default:0.00" json:"grandTotal"`
	SparePartsCost  float64         `gorm:"type:numeric(12,2);not null;default:0.00" json:"sparePartsCost"`
	EstimatedProfit float64         `gorm:"type:numeric(12,2);not null;default:0.00" json:"estimatedProfit"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// BeforeCreate hooks into GORM to generate a UUID prior to database insertion.
func (b *Budget) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return
}

// ValidateBudgetStatus checks if the status is valid.
func ValidateBudgetStatus(status string) error {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch s {
	case BudgetStatusDraft, BudgetStatusSent, BudgetStatusApproved, BudgetStatusRejected:
		return nil
	default:
		return errors.New("invalid budget status: must be DRAFT, SENT, APPROVED, or REJECTED")
	}
}

// ValidateBudgetMode checks if the mode is valid.
func ValidateBudgetMode(mode string) error {
	m := strings.ToUpper(strings.TrimSpace(mode))
	switch m {
	case BudgetModeMaintenance, BudgetModeNewEquipment:
		return nil
	default:
		return errors.New("invalid budget mode: must be MAINTENANCE or NEW_EQUIPMENT")
	}
}

// BudgetItemSaveDto represents an individual item row in budget requests.
type BudgetItemSaveDto struct {
	ID                     string  `json:"id,omitempty"`
	RowType                string  `json:"rowType,omitempty"`
	Description            string  `json:"description"`
	Quantity               float64 `json:"quantity"`
	CostPrice              float64 `json:"costPrice"`
	ProfitMarginPercentage float64 `json:"profitMarginPercentage"`
	UnitPrice              float64 `json:"unitPrice"`
	ProfitAmount           float64 `json:"profitAmount"`
}

// BudgetLaborSaveDto represents a labor cost row in budget requests.
type BudgetLaborSaveDto struct {
	ID          string  `json:"id,omitempty"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// BudgetSaveDto defines the payload for saving/updating a budget.
type BudgetSaveDto struct {
	WorkOrderID        string               `json:"workOrderId" binding:"required"`
	Title              string               `json:"title" binding:"required"`
	Mode               string               `json:"mode" binding:"required"`
	Status             string               `json:"status,omitempty"`
	ClientName         string               `json:"clientName,omitempty"`
	ClientDni          interface{}          `json:"clientDni,omitempty"`
	ClientAddress      string               `json:"clientAddress,omitempty"`
	ClientPhoneNumber  string               `json:"clientPhoneNumber,omitempty"`
	ClientEmail        string               `json:"clientEmail,omitempty"`
	DeviceModel        string               `json:"deviceModel,omitempty"`
	DeviceSerialNumber string               `json:"deviceSerialNumber,omitempty"`
	Currency           string               `json:"currency,omitempty"`
	TaxPercentage      float64              `json:"taxPercentage,omitempty"`
	Blocks             json.RawMessage      `json:"blocks,omitempty"`
	Items              []BudgetItemSaveDto  `json:"items,omitempty"`
	Labors             []BudgetLaborSaveDto `json:"labors,omitempty"`
	Notes              string               `json:"notes,omitempty"`
	TermsAndConditions string               `json:"termsAndConditions,omitempty"`
}

// BudgetTotalsDto contains aggregated metrics and calculated financial data.
type BudgetTotalsDto struct {
	ItemsSubtotal         float64 `json:"itemsSubtotal"`
	LaborTotal            float64 `json:"laborTotal"`
	GrandTotal            float64 `json:"grandTotal"`
	TotalSparePartsCost   float64 `json:"totalSparePartsCost,omitempty"`
	TotalSparePartsProfit float64 `json:"totalSparePartsProfit,omitempty"`
	TotalEstimatedProfit  float64 `json:"totalEstimatedProfit,omitempty"`
}

// BudgetStatusUpdateDto defines payload for updating budget status.
type BudgetStatusUpdateDto struct {
	Status string `json:"status" binding:"required"`
}

// BudgetResponseDto represents the complete API response for a budget.
type BudgetResponseDto struct {
	ID          string          `json:"id"`
	WorkOrderID string          `json:"workOrderId"`
	Status      string          `json:"status"`
	Title       string          `json:"title,omitempty"`
	Mode        string          `json:"mode,omitempty"`
	Currency    string          `json:"currency,omitempty"`
	BudgetData  json.RawMessage `json:"budgetData,omitempty"`
	Totals      BudgetTotalsDto `json:"totals"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}
