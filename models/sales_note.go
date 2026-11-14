package models

import (
	"time"
)

// PaymentMethod constants
const (
	PaymentMethodCash       = "CASH"
	PaymentMethodTransfer   = "TRANSFER"
	PaymentMethodMercadoPago = "MERCADOPAGO"
	PaymentMethodDebit      = "DEBIT"
	PaymentMethodCredit     = "CREDIT"
)

// PaymentStatus constants
const (
	PaymentStatusPaid    = "PAID"
	PaymentStatusPending = "PENDING"
	PaymentStatusPartial = "PARTIAL"
)

// SalesNote represents an internal non-fiscal sales document (Letra [ X ]).
type SalesNote struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	Number            string           `gorm:"size:50;not null;uniqueIndex" json:"number"` // e.g. NV-0001-00000015
	WorkOrderID       *string          `gorm:"type:uuid;index" json:"work_order_id,omitempty"`
	WorkOrder         *WorkOrder       `gorm:"foreignKey:WorkOrderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	BudgetID          *string          `gorm:"type:uuid;index" json:"budget_id,omitempty"`
	Budget            *Budget          `gorm:"foreignKey:BudgetID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	ClientID          *string          `gorm:"type:uuid;index" json:"client_id,omitempty"`
	Client            *User            `gorm:"foreignKey:ClientID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	CreatedByStaffID  *string          `gorm:"type:uuid;index" json:"created_by_staff_id,omitempty"`
	Staff             *User            `gorm:"foreignKey:CreatedByStaffID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	ClientName        string           `gorm:"size:150;not null" json:"client_name"`
	ClientDni         int32            `gorm:"not null" json:"client_dni"`
	ClientPhone       string           `gorm:"size:50" json:"client_phone"`
	Subtotal          float64          `gorm:"type:numeric(12,2);not null;default:0.00" json:"subtotal"`
	Discount          float64          `gorm:"type:numeric(12,2);not null;default:0.00" json:"discount"`
	TotalAmount       float64          `gorm:"type:numeric(12,2);not null" json:"total_amount"`
	CostAmount        float64          `gorm:"type:numeric(12,2);not null;default:0.00" json:"cost_amount"`
	NetProfit         float64          `gorm:"type:numeric(12,2);not null;default:0.00" json:"net_profit"`
	PaymentMethod     string           `gorm:"size:30;not null" json:"payment_method"`
	PaymentStatus     string           `gorm:"size:20;not null;default:'PAID'" json:"payment_status"`
	IsFiscal          bool             `gorm:"not null;default:false;index" json:"is_fiscal"`
	InvoiceID         *uint            `gorm:"index" json:"invoice_id,omitempty"`
	Invoice           *Invoice         `gorm:"foreignKey:InvoiceID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"invoice,omitempty"`
	Notes             string           `gorm:"type:text" json:"notes,omitempty"`
	Items             []SalesNoteItem  `gorm:"foreignKey:SalesNoteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// SalesNoteItem represents individual line items within a sales note.
type SalesNoteItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SalesNoteID uint      `gorm:"not null;index" json:"sales_note_id"`
	Description string    `gorm:"size:255;not null" json:"description"`
	ItemType    string    `gorm:"size:20;not null" json:"item_type"` // LABOR, SPARE_PART, ACCESSORY
	Quantity    float64   `gorm:"type:numeric(10,2);not null;default:1.00" json:"quantity"`
	UnitPrice   float64   `gorm:"type:numeric(12,2);not null" json:"unit_price"`
	UnitCost    float64   `gorm:"type:numeric(12,2);not null;default:0.00" json:"unit_cost"`
	TotalPrice  float64   `gorm:"type:numeric(12,2);not null" json:"total_price"`
}

// CreateSalesNoteRequest defines the payload to issue a new Sales Note.
type CreateSalesNoteRequest struct {
	WorkOrderID   *string             `json:"work_order_id,omitempty"`
	BudgetID      *string             `json:"budget_id,omitempty"`
	ClientID      *string             `json:"client_id,omitempty"`
	ClientName    string              `json:"client_name" binding:"required"`
	ClientDni     int32               `json:"client_dni"`
	ClientPhone   string              `json:"client_phone"`
	PaymentMethod string              `json:"payment_method" binding:"required"`
	PaymentStatus string              `json:"payment_status"`
	Items         []SalesNoteItemDto  `json:"items" binding:"required,min=1"`
	Discount      float64             `json:"discount"`
	TotalAmount   float64             `json:"total_amount" binding:"required,gt=0"`
	Notes         string              `json:"notes"`
}

// SalesNoteItemDto represents an item within the creation request.
type SalesNoteItemDto struct {
	Description string  `json:"description" binding:"required"`
	ItemType    string  `json:"item_type" binding:"required"` // LABOR, SPARE_PART, ACCESSORY
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" binding:"required"`
	UnitCost    float64 `json:"unit_cost"`
}

// ConvertBudgetToSaleNoteRequest defines the payload to convert an approved budget to a Sales Note.
type ConvertBudgetToSaleNoteRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required"`
	PaymentStatus string `json:"payment_status"`
	Notes         string `json:"notes"`
}

// FiscalizeSalesNoteRequest defines the payload to fiscalize an existing Sales Note in ARCA.
type FiscalizeSalesNoteRequest struct {
	DocTipo                int     `json:"doc_tipo" binding:"required"`
	DocNro                 int64   `json:"doc_nro"`
	CondicionIvaReceptorId *int    `json:"condicion_iva_receptor_id,omitempty"`
	Concepto               int     `json:"concepto" binding:"required,oneof=1 2 3"`
	FchServDesde           *string `json:"fch_serv_desde,omitempty"`
	FchServHasta           *string `json:"fch_serv_hasta,omitempty"`
	FchVtoPago             *string `json:"fch_vto_pago,omitempty"`
}
