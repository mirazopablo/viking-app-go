package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
	"github.com/mirazopablo/viking-app-go/services/afip"
	"gorm.io/gorm"
)

var (
	ErrSalesNoteNotFound      = errors.New("sales note not found")
	ErrSalesNoteAlreadyFiscal = errors.New("sales note is already fiscalized")
	ErrBudgetNotApproved      = errors.New("budget is not approved for conversion")
	ErrBudgetAlreadyConverted = errors.New("budget has already been converted to a sales note")
)

// AccountingService manages internal sales notes, conversions, and financial dashboards.
type AccountingService struct {
	wsfeService *afip.WSFEService
}

// NewAccountingService creates a new AccountingService instance.
func NewAccountingService() *AccountingService {
	return &AccountingService{
		wsfeService: afip.GetWSFEService(),
	}
}

// generateSalesNoteNumber creates the next sequential number (e.g. NV-0001-00000001).
func (s *AccountingService) generateSalesNoteNumber(tx *gorm.DB) (string, error) {
	var count int64
	if err := tx.Model(&models.SalesNote{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("NV-0001-%08d", count+1), nil
}

// CreateSalesNote creates a direct internal Sales Note (Letra [ X ]).
func (s *AccountingService) CreateSalesNote(req models.CreateSalesNoteRequest, staffID *string) (*models.SalesNote, error) {
	if req.TotalAmount <= 0 {
		return nil, errors.New("total_amount must be greater than 0")
	}

	var calculatedCost float64
	var items []models.SalesNoteItem
	for _, it := range req.Items {
		totalPrice := it.UnitPrice * it.Quantity
		totalCost := it.UnitCost * it.Quantity
		calculatedCost += totalCost

		items = append(items, models.SalesNoteItem{
			Description: it.Description,
			ItemType:    it.ItemType,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			UnitCost:    it.UnitCost,
			TotalPrice:  totalPrice,
		})
	}

	netProfit := req.TotalAmount - calculatedCost

	paymentStatus := req.PaymentStatus
	if paymentStatus == "" {
		paymentStatus = models.PaymentStatusPaid
	}

	var createdNote *models.SalesNote

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		number, err := s.generateSalesNoteNumber(tx)
		if err != nil {
			return err
		}

		note := &models.SalesNote{
			Number:           number,
			WorkOrderID:      req.WorkOrderID,
			BudgetID:         req.BudgetID,
			ClientID:         req.ClientID,
			CreatedByStaffID: staffID,
			ClientName:       req.ClientName,
			ClientDni:        req.ClientDni,
			ClientPhone:      req.ClientPhone,
			Subtotal:         req.TotalAmount + req.Discount,
			Discount:         req.Discount,
			TotalAmount:      req.TotalAmount,
			CostAmount:       calculatedCost,
			NetProfit:        netProfit,
			PaymentMethod:    req.PaymentMethod,
			PaymentStatus:    paymentStatus,
			IsFiscal:         false,
			Notes:            req.Notes,
			Items:            items,
		}

		if err := tx.Create(note).Error; err != nil {
			return err
		}

		createdNote = note
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create sales note: %w", err)
	}

	return createdNote, nil
}

// ConvertBudgetToSaleNote converts an approved budget into a sales note without re-entering items.
func (s *AccountingService) ConvertBudgetToSaleNote(budgetID string, req models.ConvertBudgetToSaleNoteRequest, staffID *string) (*models.SalesNote, error) {
	var budget models.Budget
	if err := config.DB.Preload("WorkOrder").First(&budget, "id = ?", budgetID).Error; err != nil {
		return nil, errors.New("budget not found")
	}

	if budget.Status == models.BudgetStatusConverted {
		return nil, ErrBudgetAlreadyConverted
	}
	if budget.Status != models.BudgetStatusApproved {
		return nil, fmt.Errorf("%w: current status is %s", ErrBudgetNotApproved, budget.Status)
	}

	// Parse sections from dynamic budget JSON
	type SectionItem struct {
		Name      string  `json:"name"`
		Quantity  float64 `json:"quantity"`
		UnitPrice float64 `json:"unitPrice"`
		UnitCost  float64 `json:"unitCost"`
		Total     float64 `json:"total"`
	}
	type BudgetSection struct {
		Title string        `json:"title"`
		Items []SectionItem `json:"items"`
	}
	type DynamicBudgetData struct {
		Sections []BudgetSection `json:"sections"`
	}

	var data DynamicBudgetData
	var items []models.SalesNoteItem
	var calculatedCost float64

	if err := json.Unmarshal(budget.BudgetData, &data); err == nil {
		for _, sec := range data.Sections {
			for _, it := range sec.Items {
				unitCost := it.UnitCost
				totalCost := unitCost * it.Quantity
				calculatedCost += totalCost

				items = append(items, models.SalesNoteItem{
					Description: it.Name,
					ItemType:    "SPARE_PART",
					Quantity:    it.Quantity,
					UnitPrice:   it.UnitPrice,
					UnitCost:    unitCost,
					TotalPrice:  it.Total,
				})
			}
		}
	}

	if len(items) == 0 {
		items = append(items, models.SalesNoteItem{
			Description: budget.Title,
			ItemType:    "LABOR",
			Quantity:    1,
			UnitPrice:   budget.GrandTotal,
			UnitCost:    budget.SparePartsCost,
			TotalPrice:  budget.GrandTotal,
		})
		calculatedCost = budget.SparePartsCost
	}

	paymentStatus := req.PaymentStatus
	if paymentStatus == "" {
		paymentStatus = models.PaymentStatusPaid
	}

	var createdNote *models.SalesNote

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update Budget status to CONVERTED
		if err := tx.Model(&budget).Update("status", models.BudgetStatusConverted).Error; err != nil {
			return err
		}

		// 2. Generate sequential number
		number, err := s.generateSalesNoteNumber(tx)
		if err != nil {
			return err
		}

		clientName := budget.WorkOrder.ClientNameSnapshot
		clientDni := budget.WorkOrder.ClientDniSnapshot
		clientPhone := budget.WorkOrder.ClientPhoneSnapshot

		note := &models.SalesNote{
			Number:           number,
			WorkOrderID:      &budget.WorkOrderID,
			BudgetID:         &budget.ID,
			ClientID:         budget.WorkOrder.ClientID,
			CreatedByStaffID: staffID,
			ClientName:       clientName,
			ClientDni:        clientDni,
			ClientPhone:      clientPhone,
			Subtotal:         budget.GrandTotal,
			Discount:         0,
			TotalAmount:      budget.GrandTotal,
			CostAmount:       calculatedCost,
			NetProfit:        budget.GrandTotal - calculatedCost,
			PaymentMethod:    req.PaymentMethod,
			PaymentStatus:    paymentStatus,
			IsFiscal:         false,
			Notes:            req.Notes,
			Items:            items,
		}

		if err := tx.Create(note).Error; err != nil {
			return err
		}

		createdNote = note
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to convert budget to sales note: %w", err)
	}

	return createdNote, nil
}

// FiscalizeSalesNote issues an official Factura C via ARCA for an existing sales note.
func (s *AccountingService) FiscalizeSalesNote(salesNoteID uint, req models.FiscalizeSalesNoteRequest, userID *uint) (*models.SalesNote, error) {
	var note models.SalesNote
	if err := config.DB.Preload("Items").First(&note, salesNoteID).Error; err != nil {
		return nil, ErrSalesNoteNotFound
	}

	if note.IsFiscal || note.InvoiceID != nil {
		return nil, ErrSalesNoteAlreadyFiscal
	}

	invoiceReq := models.CreateInvoiceRequest{
		Concepto:               req.Concepto,
		DocTipo:                req.DocTipo,
		DocNro:                 req.DocNro,
		CondicionIvaReceptorId: req.CondicionIvaReceptorId,
		ImporteTotal:           note.TotalAmount,
		FchServDesde:           req.FchServDesde,
		FchServHasta:           req.FchServHasta,
		FchVtoPago:             req.FchVtoPago,
	}

	invoice, err := s.wsfeService.RequestCAE(invoiceReq)
	if err != nil {
		return nil, fmt.Errorf("ARCA error: %w", err)
	}

	invoice.UserID = userID

	// Clone SalesNote items into Invoice items
	var invoiceItems []models.InvoiceItem
	for _, it := range note.Items {
		invoiceItems = append(invoiceItems, models.InvoiceItem{
			Description: it.Description,
			ItemType:    it.ItemType,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			TotalPrice:  it.TotalPrice,
		})
	}
	invoice.Items = invoiceItems

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}

		// Update sales note with fiscal reference
		note.IsFiscal = (invoice.Status == models.InvoiceStatusApproved)
		note.InvoiceID = &invoice.ID
		if err := tx.Save(&note).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to persist fiscalized invoice: %w", err)
	}

	note.Invoice = invoice
	return &note, nil
}

// GetSalesNotes retrieves a paginated list of sales notes.
func (s *AccountingService) GetSalesNotes(limit, offset int, paymentMethod string) ([]models.SalesNote, int64, error) {
	var notes []models.SalesNote
	var total int64

	query := config.DB.Model(&models.SalesNote{})
	if paymentMethod != "" {
		query = query.Where("payment_method = ?", paymentMethod)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("id desc").Limit(limit).Offset(offset).Preload("Invoice").Preload("Items").Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// GetSalesNoteByID retrieves a single sales note by ID.
func (s *AccountingService) GetSalesNoteByID(id uint) (*models.SalesNote, error) {
	var note models.SalesNote
	if err := config.DB.Preload("Invoice").Preload("Items").First(&note, id).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

// GetDashboardMetrics calculates unified financial and fiscal metrics for a given date range.
func (s *AccountingService) GetDashboardMetrics(fromStr, toStr string) (*models.AccountingDashboardResponse, error) {
	now := time.Now()
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}
	} else {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			to = now
		} else {
			to = to.Add(24*time.Hour - time.Nanosecond)
		}
	} else {
		to = now
	}

	// 1. Sales metrics from SalesNotes in period
	type SalesAgg struct {
		GrossIncome    float64
		SparePartsCost float64
		NetProfit      float64
		FiscalTotal    float64
		InternalTotal  float64
	}
	var agg SalesAgg

	config.DB.Model(&models.SalesNote{}).
		Select(`
			COALESCE(SUM(total_amount), 0) as gross_income,
			COALESCE(SUM(cost_amount), 0) as spare_parts_cost,
			COALESCE(SUM(net_profit), 0) as net_profit,
			COALESCE(SUM(CASE WHEN is_fiscal = true THEN total_amount ELSE 0 END), 0) as fiscal_total,
			COALESCE(SUM(CASE WHEN is_fiscal = false THEN total_amount ELSE 0 END), 0) as internal_total
		`).
		Where("created_at BETWEEN ? AND ?", from, to).
		Scan(&agg)

	// 2. Metrics by Payment Method
	type PMRow struct {
		PaymentMethod string
		Total         float64
	}
	var pmRows []PMRow
	config.DB.Model(&models.SalesNote{}).
		Select("payment_method, COALESCE(SUM(total_amount), 0) as total").
		Where("created_at BETWEEN ? AND ?", from, to).
		Group("payment_method").
		Scan(&pmRows)

	byPM := make(map[string]float64)
	for _, r := range pmRows {
		byPM[r.PaymentMethod] = r.Total
	}

	// 3. Monotributo Control (Rolling 12 months fiscal invoices)
	twelveMonthsAgo := now.AddDate(-1, 0, 0)
	var accumulatedFiscal float64
	config.DB.Model(&models.Invoice{}).
		Select("COALESCE(SUM(importe_total), 0)").
		Where("status = ? AND created_at >= ?", models.InvoiceStatusApproved, twelveMonthsAgo).
		Scan(&accumulatedFiscal)

	// Standard category A annual ceiling reference in Argentina (Monotributo 2026)
	categoryLimit := 6450000.00
	remaining := categoryLimit - accumulatedFiscal
	if remaining < 0 {
		remaining = 0
	}
	usagePct := (accumulatedFiscal / categoryLimit) * 100.0

	var fiscalPct float64
	if agg.GrossIncome > 0 {
		fiscalPct = (agg.FiscalTotal / agg.GrossIncome) * 100.0
	}

	var profitMargin float64
	if agg.GrossIncome > 0 {
		profitMargin = (agg.NetProfit / agg.GrossIncome) * 100.0
	}

	return &models.AccountingDashboardResponse{
		Period: models.PeriodSummary{
			From: from.Format("2006-01-02"),
			To:   to.Format("2006-01-02"),
		},
		CashFlow: models.CashFlowSummary{
			GrossIncome:            agg.GrossIncome,
			SparePartsCost:         agg.SparePartsCost,
			NetProfit:              agg.NetProfit,
			ProfitMarginPercentage: profitMargin,
		},
		FiscalVsInternal: models.FiscalVsInternalSummary{
			FiscalInvoicedTotal:     agg.FiscalTotal,
			InternalSalesNotesTotal: agg.InternalTotal,
			FiscalPercentage:        fiscalPct,
		},
		MonotributoControl: models.MonotributoControlSummary{
			CurrentCategory:             "A",
			CategoryAnnualLimit:         categoryLimit,
			AccumulatedLast12Months:     accumulatedFiscal,
			RemainingBeforeNextCategory: remaining,
			UsagePercentage:             usagePct,
		},
		ByPaymentMethod: byPM,
	}, nil
}
