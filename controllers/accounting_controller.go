package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services"
)

// AccountingController handles REST endpoints for Sales Notes, budget conversions, and accounting dashboards.
type AccountingController struct {
	accountingService *services.AccountingService
}

// NewAccountingController instantiates a new AccountingController.
func NewAccountingController() *AccountingController {
	return &AccountingController{
		accountingService: services.NewAccountingService(),
	}
}

// CreateSalesNote godoc
// @Summary Crear Nota de Venta interna
// @Description Emite una Nota de Venta interna (Letra [ X ] no fiscal) para control de caja
// @Tags Accounting
// @Accept json
// @Produce json
// @Param payload body models.CreateSalesNoteRequest true "Datos de la venta"
// @Success 201 {object} models.SalesNote
// @Failure 400 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/sales-notes [post]
func (c *AccountingController) CreateSalesNote(ctx *gin.Context) {
	var req models.CreateSalesNoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	var staffIDPtr *string
	if val, exists := ctx.Get("userID"); exists {
		if uidStr, ok := val.(string); ok {
			staffIDPtr = &uidStr
		}
	}

	note, err := c.accountingService.CreateSalesNote(req, staffIDPtr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, note)
}

// ConvertBudgetToSaleNote godoc
// @Summary Convertir Presupuesto aprobado a Nota de Venta
// @Description Transforma una cotización aprobada en una Nota de Venta interna en caja
// @Tags Budget
// @Accept json
// @Produce json
// @Param id path string true "ID del Presupuesto (UUID)"
// @Param payload body models.ConvertBudgetToSaleNoteRequest true "Datos de cobro"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security bearer-jwt
// @Router /api/budget/{id}/convert-to-sale-note [post]
func (c *AccountingController) ConvertBudgetToSaleNote(ctx *gin.Context) {
	budgetID := ctx.Param("id")

	var req models.ConvertBudgetToSaleNoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	var staffIDPtr *string
	if val, exists := ctx.Get("userID"); exists {
		if uidStr, ok := val.(string); ok {
			staffIDPtr = &uidStr
		}
	}

	note, err := c.accountingService.ConvertBudgetToSaleNote(budgetID, req, staffIDPtr)
	if err != nil {
		if err == services.ErrBudgetAlreadyConverted {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message":   "Presupuesto convertido exitosamente a Nota de Venta",
		"sale_note": note,
	})
}

// FiscalizeSalesNote godoc
// @Summary Fiscalizar Nota de Venta existente en ARCA
// @Description Emite Factura C oficial ante ARCA para una Nota de Venta previa y le asigna el CAE
// @Tags Accounting
// @Accept json
// @Produce json
// @Param id path int true "ID de la Nota de Venta"
// @Param payload body models.FiscalizeSalesNoteRequest true "Datos fiscales del receptor"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/sales-notes/{id}/fiscalize [post]
func (c *AccountingController) FiscalizeSalesNote(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID de Nota de Venta inválido"})
		return
	}

	var req models.FiscalizeSalesNoteRequest
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

	note, err := c.accountingService.FiscalizeSalesNote(uint(id), req, userIDPtr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":      "Nota de Venta fiscalizada exitosamente ante ARCA",
		"sale_note_id": note.ID,
		"is_fiscal":    note.IsFiscal,
		"invoice":      note.Invoice,
	})
}

// GetSalesNotes godoc
// @Summary Listar Notas de Venta
// @Description Obtiene un listado paginado de comprobantes de venta internos
// @Tags Accounting
// @Produce json
// @Param limit query int false "Límite" default(10)
// @Param offset query int false "Desplazamiento" default(0)
// @Param payment_method query string false "Método de Pago (CASH, TRANSFER, MERCADOPAGO)"
// @Success 200 {object} map[string]interface{}
// @Security bearer-jwt
// @Router /api/v1/sales-notes [get]
func (c *AccountingController) GetSalesNotes(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	pm := ctx.Query("payment_method")

	notes, total, err := c.accountingService.GetSalesNotes(limit, offset, pm)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar notas de venta: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total": total,
		"data":  notes,
	})
}

// GetSalesNoteByID godoc
// @Summary Obtener Nota de Venta por ID
// @Tags Accounting
// @Produce json
// @Param id path int true "ID de la Nota de Venta"
// @Success 200 {object} models.SalesNote
// @Failure 404 {object} map[string]string
// @Security bearer-jwt
// @Router /api/v1/sales-notes/{id} [get]
func (c *AccountingController) GetSalesNoteByID(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	note, err := c.accountingService.GetSalesNoteByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Nota de Venta no encontrada"})
		return
	}

	ctx.JSON(http.StatusOK, note)
}

// GetDashboardMetrics godoc
// @Summary Obtener métricas del Dashboard Contable
// @Description Consolida ingresos reales, desglose fiscal vs interno y control de Monotributo
// @Tags Accounting
// @Produce json
// @Param from query string false "Fecha desde (YYYY-MM-DD)"
// @Param to query string false "Fecha hasta (YYYY-MM-DD)"
// @Success 200 {object} models.AccountingDashboardResponse
// @Security bearer-jwt
// @Router /api/v1/accounting/dashboard [get]
func (c *AccountingController) GetDashboardMetrics(ctx *gin.Context) {
	from := ctx.Query("from")
	to := ctx.Query("to")

	metrics, err := c.accountingService.GetDashboardMetrics(from, to)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error al calcular métricas: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, metrics)
}
