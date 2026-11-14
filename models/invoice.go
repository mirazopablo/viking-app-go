package models

import (
	"time"
)

// InvoiceStatus represents the authorization status of an invoice.
type InvoiceStatus string

const (
	InvoiceStatusApproved InvoiceStatus = "APPROVED"
	InvoiceStatusRejected InvoiceStatus = "REJECTED"
	InvoiceStatusPending  InvoiceStatus = "PENDING"
)

// Invoice represents a fiscal electronic invoice (Factura C - Tipo 011).
type Invoice struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	UserID         *uint         `gorm:"index" json:"user_id,omitempty"`
	PuntoVenta     int           `gorm:"not null" json:"punto_venta"`
	TipoCmp        int           `gorm:"not null" json:"tipo_cmp"` // 11 = Factura C
	NumeroCmp      int64         `gorm:"not null;index" json:"numero_cmp"`
	FechaEmision   string        `gorm:"size:8;not null" json:"fecha_emision"` // YYYYMMDD
	Concepto       int           `gorm:"not null" json:"concepto"`            // 1: Productos, 2: Servicios, 3: Ambos
	DocTipo                int           `gorm:"not null" json:"doc_tipo"`            // 99: Consumidor Final, 80: CUIT, 96: DNI
	DocNro                 int64         `gorm:"not null" json:"doc_nro"`
	CondicionIvaReceptorId int           `gorm:"default:5" json:"condicion_iva_receptor_id"` // RG 5616: 5 = Consumidor Final
	ImporteTotal           float64       `gorm:"type:numeric(12,2);not null" json:"importe_total"`
	CAE                    string        `gorm:"size:20;index" json:"cae"`
	CAEVto                 string        `gorm:"size:8" json:"cae_vto"` // YYYYMMDD
	Resultado              string        `gorm:"size:10" json:"resultado"` // A: Aprobado, R: Rechazado
	Observaciones          string        `gorm:"type:text" json:"observaciones,omitempty"`
	Status                 InvoiceStatus `gorm:"size:20;default:'PENDING'" json:"status"`
	Items                  []InvoiceItem `gorm:"foreignKey:InvoiceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
}

// InvoiceItem represents an individual line item/content within an invoice.
type InvoiceItem struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	InvoiceID   uint    `gorm:"not null;index" json:"invoice_id"`
	Description string  `gorm:"size:255;not null" json:"description"`
	ItemType    string  `gorm:"size:20;default:'SERVICE'" json:"item_type,omitempty"` // LABOR, SPARE_PART, ACCESSORY, SERVICE
	Quantity    float64 `gorm:"type:numeric(10,2);not null;default:1.00" json:"quantity"`
	UnitPrice   float64 `gorm:"type:numeric(12,2);not null" json:"unit_price"`
	TotalPrice  float64 `gorm:"type:numeric(12,2);not null" json:"total_price"`
}

// InvoiceItemDto represents an item in creation/import requests.
type InvoiceItemDto struct {
	Description string  `json:"description" binding:"required"`
	ItemType    string  `json:"item_type,omitempty"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" binding:"required"`
	TotalPrice  float64 `json:"total_price"`
}

// CreateInvoiceRequest represents the payload received to issue an invoice.
type CreateInvoiceRequest struct {
	Concepto               int              `json:"concepto" binding:"required,oneof=1 2 3"`
	DocTipo                int              `json:"doc_tipo" binding:"required"`
	DocNro                 int64            `json:"doc_nro"`
	CondicionIvaReceptorId *int             `json:"condicion_iva_receptor_id,omitempty"` // RG 5616: default 5 (Consumidor Final)
	ImporteTotal           float64          `json:"importe_total" binding:"required,gt=0"`
	Items                  []InvoiceItemDto `json:"items,omitempty"`
	// Campos requeridos si Concepto == 2 o 3 (Servicios)
	FchServDesde *string `json:"fch_serv_desde,omitempty"` // YYYYMMDD
	FchServHasta *string `json:"fch_serv_hasta,omitempty"` // YYYYMMDD
	FchVtoPago   *string `json:"fch_vto_pago,omitempty"`   // YYYYMMDD
}

// InvoiceResponse represents the standardized output after issuing an invoice.
type InvoiceResponse struct {
	ID            uint          `json:"id"`
	PuntoVenta    int           `json:"punto_venta"`
	TipoCmp       int           `json:"tipo_cmp"`
	NumeroCmp     int64         `json:"numero_cmp"`
	FechaEmision  string        `json:"fecha_emision"`
	DocTipo       int           `json:"doc_tipo"`
	DocNro        int64         `json:"doc_nro"`
	ImporteTotal  float64       `json:"importe_total"`
	CAE           string        `json:"cae"`
	CAEVto        string        `json:"cae_vto"`
	Resultado     string        `json:"resultado"`
	Status        string        `json:"status"`
	Observaciones string        `json:"observaciones,omitempty"`
	Items         []InvoiceItem `json:"items,omitempty"`
}

// ImportManualInvoiceRequest represents the payload to register an invoice issued externally via ARCA RCEL / Facturador.
type ImportManualInvoiceRequest struct {
	PuntoVenta             int              `json:"punto_venta" binding:"required"`
	TipoCmp                int              `json:"tipo_cmp" binding:"required"` // 11 = Factura C
	NumeroCmp              int64            `json:"numero_cmp" binding:"required,gt=0"`
	FechaEmision           string           `json:"fecha_emision" binding:"required"` // YYYYMMDD
	Concepto               int              `json:"concepto" binding:"required,oneof=1 2 3"`
	DocTipo                int              `json:"doc_tipo" binding:"required"`
	DocNro                 int64            `json:"doc_nro"`
	CondicionIvaReceptorId *int             `json:"condicion_iva_receptor_id,omitempty"`
	ImporteTotal           float64          `json:"importe_total" binding:"required,gt=0"`
	CAE                    string           `json:"cae" binding:"required"`
	CAEVto                 string           `json:"cae_vto" binding:"required"`
	Observaciones          string           `json:"observaciones,omitempty"`
	Items                  []InvoiceItemDto `json:"items,omitempty"`
}
