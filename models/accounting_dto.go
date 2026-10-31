package models

// AccountingDashboardResponse represents the aggregated financial and fiscal dashboard metrics.
type AccountingDashboardResponse struct {
	Period             PeriodSummary             `json:"period"`
	CashFlow           CashFlowSummary           `json:"cash_flow"`
	FiscalVsInternal   FiscalVsInternalSummary   `json:"fiscal_vs_internal"`
	MonotributoControl MonotributoControlSummary `json:"monotributo_control"`
	ByPaymentMethod    map[string]float64        `json:"by_payment_method"`
}

type PeriodSummary struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CashFlowSummary struct {
	GrossIncome            float64 `json:"gross_income"`
	SparePartsCost         float64 `json:"spare_parts_cost"`
	NetProfit              float64 `json:"net_profit"`
	ProfitMarginPercentage float64 `json:"profit_margin_percentage"`
}

type FiscalVsInternalSummary struct {
	FiscalInvoicedTotal     float64 `json:"fiscal_invoiced_total"`
	InternalSalesNotesTotal float64 `json:"internal_sales_notes_total"`
	FiscalPercentage        float64 `json:"fiscal_percentage"`
}

type MonotributoControlSummary struct {
	CurrentCategory             string  `json:"current_category"`
	CategoryAnnualLimit         float64 `json:"category_annual_limit"`
	AccumulatedLast12Months     float64 `json:"accumulated_last_12_months"`
	RemainingBeforeNextCategory float64 `json:"remaining_before_next_category"`
	UsagePercentage             float64 `json:"usage_percentage"`
}
