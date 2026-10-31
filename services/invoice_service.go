package services

import (
	"errors"
	"fmt"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services/afip"
)

// InvoiceService defines operations for fiscal electronic invoices.
type InvoiceService struct {
	wsfeService *afip.WSFEService
}

// NewInvoiceService creates a new instance of InvoiceService.
func NewInvoiceService() *InvoiceService {
	return &InvoiceService{
		wsfeService: afip.GetWSFEService(),
	}
}

// CreateInvoice validates payload, requests CAE from ARCA, and persists the record.
func (s *InvoiceService) CreateInvoice(req models.CreateInvoiceRequest, userID *uint) (*models.Invoice, error) {
	// 1. Validation of business rules
	if req.ImporteTotal <= 0 {
		return nil, errors.New("el importe total debe ser mayor a 0")
	}

	// Consumidor Final (DocTipo 99): Si DocNro no se envía o es 0, lo aseguramos en 0
	if req.DocTipo == 99 {
		req.DocNro = 0
	} else if req.DocNro <= 0 {
		return nil, errors.New("para tipos de documento distintos a Consumidor Final (99), debe proveerse un número de documento válido")
	}

	// 2. Request CAE from ARCA WSFEV1
	invoice, err := s.wsfeService.RequestCAE(req)
	if err != nil {
		return nil, fmt.Errorf("error al solicitar autorización en ARCA: %w", err)
	}

	invoice.UserID = userID

	// 3. Persist invoice to database
	if err := config.DB.Create(invoice).Error; err != nil {
		// Log warning: Invoice was approved by ARCA but failed to persist locally
		return invoice, fmt.Errorf("comprobante emitido con CAE %s pero ocurrió un error al persistir: %w", invoice.CAE, err)
	}

	return invoice, nil
}

// ImportManualInvoice registers an invoice issued externally (e.g. from ARCA RCEL Punto 1) without calling WSFE.
func (s *InvoiceService) ImportManualInvoice(req models.ImportManualInvoiceRequest, userID *uint) (*models.Invoice, error) {
	if req.ImporteTotal <= 0 {
		return nil, errors.New("el importe total debe ser mayor a 0")
	}

	condicionIVA := 5 // Default Consumidor Final
	if req.CondicionIvaReceptorId != nil && *req.CondicionIvaReceptorId > 0 {
		condicionIVA = *req.CondicionIvaReceptorId
	} else if req.DocTipo == 80 {
		condicionIVA = 1
	}

	docNro := req.DocNro
	if req.DocTipo == 99 {
		docNro = 0
	}

	invoice := &models.Invoice{
		UserID:                 userID,
		PuntoVenta:             req.PuntoVenta,
		TipoCmp:                req.TipoCmp,
		NumeroCmp:              req.NumeroCmp,
		FechaEmision:           req.FechaEmision,
		Concepto:               req.Concepto,
		DocTipo:                req.DocTipo,
		DocNro:                 docNro,
		CondicionIvaReceptorId: condicionIVA,
		ImporteTotal:           req.ImporteTotal,
		CAE:                    req.CAE,
		CAEVto:                 req.CAEVto,
		Resultado:              "A",
		Status:                 models.InvoiceStatusApproved,
		Observaciones:          "[Comprobante importado manualmente desde ARCA] " + req.Observaciones,
	}

	if err := config.DB.Create(invoice).Error; err != nil {
		return nil, fmt.Errorf("error al guardar comprobante manual: %w", err)
	}

	return invoice, nil
}

// GetInvoices retrieves a paginated list of issued invoices.
func (s *InvoiceService) GetInvoices(limit, offset int) ([]models.Invoice, int64, error) {
	var invoices []models.Invoice
	var total int64

	if err := config.DB.Model(&models.Invoice{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := config.DB.Order("id desc").Limit(limit).Offset(offset).Find(&invoices).Error; err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// GetInvoiceByID retrieves a single invoice by its local ID.
func (s *InvoiceService) GetInvoiceByID(id uint) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := config.DB.First(&invoice, id).Error; err != nil {
		return nil, err
	}
	return &invoice, nil
}
