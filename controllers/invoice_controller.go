package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// InvoiceController handles HTTP requests related to electronic invoices.
type InvoiceController struct {
	invoiceService *services.InvoiceService
}

// NewInvoiceController creates a new InvoiceController.
func NewInvoiceController() *InvoiceController {
	return &InvoiceController{
		invoiceService: services.NewInvoiceService(),
	}
}

// CreateInvoice godoc
// @Summary Emitir Factura C electrónica en ARCA
// @Description Emite un comprobante fiscal Tipo 11 (Factura C) ante ARCA y devuelve el CAE obtenido
// @Tags Invoices
// @Accept json
// @Produce json
// @Param invoice body models.CreateInvoiceRequest true "Datos de la factura"
// @Success 201 {object} models.InvoiceResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/invoices [post]
func (c *InvoiceController) CreateInvoice(ctx *gin.Context) {
	var req models.CreateInvoiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Extract optional authenticated user ID from context
	var userIDPtr *uint
	if val, exists := ctx.Get("userID"); exists {
		if uid, ok := val.(uint); ok {
			userIDPtr = &uid
		}
	}

	invoice, err := c.invoiceService.CreateInvoice(req, userIDPtr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := models.InvoiceResponse{
		ID:           invoice.ID,
		PuntoVenta:   invoice.PuntoVenta,
		TipoCmp:      invoice.TipoCmp,
		NumeroCmp:    invoice.NumeroCmp,
		FechaEmision: invoice.FechaEmision,
		DocTipo:      invoice.DocTipo,
		DocNro:       invoice.DocNro,
		ImporteTotal: invoice.ImporteTotal,
		CAE:          invoice.CAE,
		CAEVto:       invoice.CAEVto,
		Resultado:    invoice.Resultado,
		Status:       string(invoice.Status),
		Observaciones: invoice.Observaciones,
	}

	ctx.JSON(http.StatusCreated, resp)
}

// ImportManualInvoice godoc
// @Summary Registrar factura emitida externamente en ARCA (RCEL / Facturador)
// @Description Permite cargar manualmente una Factura emitida por fuera del sistema para cuadrar la contabilidad
// @Tags Invoices
// @Accept json
// @Produce json
// @Param invoice body models.ImportManualInvoiceRequest true "Datos del comprobante emitido en ARCA"
// @Success 201 {object} models.Invoice
// @Failure 400 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/invoices/import-manual [post]
func (c *InvoiceController) ImportManualInvoice(ctx *gin.Context) {
	var req models.ImportManualInvoiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	var userIDPtr *uint
	if val, exists := ctx.Get("userID"); exists {
		if uid, ok := val.(uint); ok {
			userIDPtr = &uid
		}
	}

	invoice, err := c.invoiceService.ImportManualInvoice(req, userIDPtr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, invoice)
}

// GetInvoices godoc
// @Summary Listar facturas emitidas
// @Description Obtiene un listado paginado de las facturas registradas localmente
// @Tags Invoices
// @Produce json
// @Param limit query int false "Límite de resultados" default(10)
// @Param offset query int false "Desplazamiento" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/invoices [get]
func (c *InvoiceController) GetInvoices(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	invoices, total, err := c.invoiceService.GetInvoices(limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error al recuperar facturas: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":  invoices,
		"total": total,
	})
}

// GetInvoiceByID godoc
// @Summary Obtener factura por ID
// @Description Obtiene el detalle de un comprobante emitido
// @Tags Invoices
// @Produce json
// @Param id path int true "ID de la factura"
// @Success 200 {object} models.Invoice
// @Failure 404 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/invoices/{id} [get]
func (c *InvoiceController) GetInvoiceByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	invoice, err := c.invoiceService.GetInvoiceByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Factura no encontrada"})
		return
	}

	ctx.JSON(http.StatusOK, invoice)
}
