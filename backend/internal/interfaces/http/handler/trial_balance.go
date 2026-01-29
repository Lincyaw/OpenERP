package handler

import (
	"strconv"
	"time"

	financeapp "github.com/erp/backend/internal/application/finance"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
)

// TrialBalanceHandler handles trial balance API endpoints
type TrialBalanceHandler struct {
	BaseHandler
	trialBalanceService *financeapp.TrialBalanceService
}

// NewTrialBalanceHandler creates a new TrialBalanceHandler
func NewTrialBalanceHandler(trialBalanceService *financeapp.TrialBalanceService) *TrialBalanceHandler {
	return &TrialBalanceHandler{
		trialBalanceService: trialBalanceService,
	}
}

// ===================== Request/Response DTOs =====================

// TrialBalanceCheckRequest represents a request to perform a trial balance check
//
//	@Description	Request body for performing a trial balance check
type TrialBalanceCheckRequest struct {
	PeriodStart                 *string `json:"period_start" example:"2026-01-01"`
	PeriodEnd                   *string `json:"period_end" example:"2026-01-31"`
	CheckReceivables            *bool   `json:"check_receivables" example:"true"`
	CheckPayables               *bool   `json:"check_payables" example:"true"`
	CheckReceipts               *bool   `json:"check_receipts" example:"true"`
	CheckPayments               *bool   `json:"check_payments" example:"true"`
	CheckCreditMemos            *bool   `json:"check_credit_memos" example:"true"`
	CheckDebitMemos             *bool   `json:"check_debit_memos" example:"true"`
	ValidateInternalConsistency *bool   `json:"validate_internal_consistency" example:"true"`
	ValidateVoucherAllocations  *bool   `json:"validate_voucher_allocations" example:"true"`
	ValidateMemoApplications    *bool   `json:"validate_memo_applications" example:"true"`
	Notes                       string  `json:"notes" example:"Monthly balance check"`
}

// TrialBalanceResponse represents the trial balance check result
//
//	@Description	Trial balance check response
type TrialBalanceResponse struct {
	ID                  string                `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID            string                `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CheckedAt           time.Time             `json:"checked_at"`
	CheckedBy           string                `json:"checked_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status              string                `json:"status" example:"BALANCED"`
	TotalDebits         float64               `json:"total_debits" example:"50000.00"`
	TotalCredits        float64               `json:"total_credits" example:"50000.00"`
	NetBalance          float64               `json:"net_balance" example:"0.00"`
	TotalReceivables    float64               `json:"total_receivables" example:"30000.00"`
	TotalPayables       float64               `json:"total_payables" example:"20000.00"`
	TotalReceipts       float64               `json:"total_receipts" example:"10000.00"`
	TotalPayments       float64               `json:"total_payments" example:"15000.00"`
	TotalCreditMemos    float64               `json:"total_credit_memos" example:"2000.00"`
	TotalDebitMemos     float64               `json:"total_debit_memos" example:"3000.00"`
	ReceivableCount     int64                 `json:"receivable_count" example:"50"`
	PayableCount        int64                 `json:"payable_count" example:"30"`
	ReceiptCount        int64                 `json:"receipt_count" example:"20"`
	PaymentCount        int64                 `json:"payment_count" example:"25"`
	CreditMemoCount     int64                 `json:"credit_memo_count" example:"5"`
	DebitMemoCount      int64                 `json:"debit_memo_count" example:"3"`
	Discrepancies       []DiscrepancyResponse `json:"discrepancies"`
	DiscrepancyCount    int                   `json:"discrepancy_count" example:"0"`
	CriticalCount       int                   `json:"critical_count" example:"0"`
	WarningCount        int                   `json:"warning_count" example:"0"`
	PeriodStart         *time.Time            `json:"period_start,omitempty"`
	PeriodEnd           *time.Time            `json:"period_end,omitempty"`
	ExecutionDurationMs int64                 `json:"execution_duration_ms" example:"150"`
	Notes               string                `json:"notes,omitempty" example:"Monthly balance check"`
}

// DiscrepancyResponse represents a balance discrepancy in API responses
//
//	@Description	Balance discrepancy response
type DiscrepancyResponse struct {
	ID              string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type            string                  `json:"type" example:"RECEIVABLE_AMOUNT_MISMATCH"`
	EntityType      string                  `json:"entity_type" example:"AccountReceivable"`
	EntityID        string                  `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	EntityNumber    string                  `json:"entity_number" example:"AR-2026-00001"`
	ExpectedAmount  float64                 `json:"expected_amount" example:"1000.00"`
	ActualAmount    float64                 `json:"actual_amount" example:"900.00"`
	Difference      float64                 `json:"difference" example:"100.00"`
	Description     string                  `json:"description" example:"Receivable paid + outstanding doesn't equal total amount"`
	Severity        string                  `json:"severity" example:"CRITICAL"`
	DetectedAt      time.Time               `json:"detected_at"`
	RelatedEntities []RelatedEntityResponse `json:"related_entities,omitempty"`
}

// RelatedEntityResponse represents a related entity in a discrepancy
//
//	@Description	Related entity response
type RelatedEntityResponse struct {
	EntityType string `json:"entity_type" example:"ReceiptVoucher"`
	EntityID   string `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Reference  string `json:"reference" example:"RV-2026-00001"`
}

// BalanceCheckGuardResponse represents the result of a pre-operation balance check
//
//	@Description	Balance check guard response
type BalanceCheckGuardResponse struct {
	Allowed       bool                  `json:"allowed" example:"true"`
	Status        string                `json:"status" example:"BALANCED"`
	Message       string                `json:"message" example:"Trial balance is balanced, operation allowed"`
	Discrepancies []DiscrepancyResponse `json:"discrepancies,omitempty"`
}

// ReconciliationReportResponse represents a detailed reconciliation report
//
//	@Description	Reconciliation report response
type ReconciliationReportResponse struct {
	TrialBalance         TrialBalanceResponse `json:"trial_balance"`
	GeneratedAt          time.Time            `json:"generated_at"`
	GeneratedBy          string               `json:"generated_by" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReportType           string               `json:"report_type" example:"RECONCILIATION"`
	TotalEntitiesChecked int64                `json:"total_entities_checked" example:"133"`
	HealthScore          int                  `json:"health_score" example:"100"`
	Recommendations      []string             `json:"recommendations"`
}

// AuditLogResponse represents a trial balance audit log entry
//
//	@Description	Trial balance audit log response
type AuditLogResponse struct {
	ID               string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID         string     `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CheckedAt        time.Time  `json:"checked_at"`
	CheckedBy        string     `json:"checked_by" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status           string     `json:"status" example:"BALANCED"`
	TotalDebits      float64    `json:"total_debits" example:"50000.00"`
	TotalCredits     float64    `json:"total_credits" example:"50000.00"`
	NetBalance       float64    `json:"net_balance" example:"0.00"`
	DiscrepancyCount int        `json:"discrepancy_count" example:"0"`
	CriticalCount    int        `json:"critical_count" example:"0"`
	WarningCount     int        `json:"warning_count" example:"0"`
	DurationMs       int64      `json:"duration_ms" example:"150"`
	PeriodStart      *time.Time `json:"period_start,omitempty"`
	PeriodEnd        *time.Time `json:"period_end,omitempty"`
	Notes            string     `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ===================== Handler Methods =====================

// ===================== Handler Methods =====================

// PerformTrialBalanceCheck godoc
//
//	@ID				performTrialBalanceCheckFinanceTrialBalance
//	@Summary		Perform trial balance check
//	@Description	Performs a comprehensive trial balance check and returns discrepancies
//	@Tags			finance-trial-balance
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string						false	"Tenant ID (optional for dev)"
//	@Param			request		body		TrialBalanceCheckRequest	false	"Trial balance check options"
//	@Success		200			{object}	APIResponse[TrialBalanceResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/trial-balance/check [post]
func (h *TrialBalanceHandler) PerformTrialBalanceCheck(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		// User authentication is required for trial balance operations
		// to ensure proper audit trail and access control
		h.Unauthorized(c, "Authentication required for trial balance operations")
		return
	}

	var req TrialBalanceCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = TrialBalanceCheckRequest{}
	}

	// Convert request to application layer format
	appReq := financeapp.DefaultTrialBalanceCheckRequest()

	// Validate and parse period dates
	if req.PeriodStart != nil {
		t, err := time.Parse("2006-01-02", *req.PeriodStart)
		if err != nil {
			h.BadRequest(c, "Invalid period_start format. Expected YYYY-MM-DD")
			return
		}
		appReq.PeriodStart = &t
	}
	if req.PeriodEnd != nil {
		t, err := time.Parse("2006-01-02", *req.PeriodEnd)
		if err != nil {
			h.BadRequest(c, "Invalid period_end format. Expected YYYY-MM-DD")
			return
		}
		appReq.PeriodEnd = &t
	}

	if req.CheckReceivables != nil {
		appReq.CheckReceivables = *req.CheckReceivables
	}
	if req.CheckPayables != nil {
		appReq.CheckPayables = *req.CheckPayables
	}
	if req.CheckReceipts != nil {
		appReq.CheckReceipts = *req.CheckReceipts
	}
	if req.CheckPayments != nil {
		appReq.CheckPayments = *req.CheckPayments
	}
	if req.CheckCreditMemos != nil {
		appReq.CheckCreditMemos = *req.CheckCreditMemos
	}
	if req.CheckDebitMemos != nil {
		appReq.CheckDebitMemos = *req.CheckDebitMemos
	}
	if req.ValidateInternalConsistency != nil {
		appReq.ValidateInternalConsistency = *req.ValidateInternalConsistency
	}
	if req.ValidateVoucherAllocations != nil {
		appReq.ValidateVoucherAllocations = *req.ValidateVoucherAllocations
	}
	if req.ValidateMemoApplications != nil {
		appReq.ValidateMemoApplications = *req.ValidateMemoApplications
	}
	appReq.Notes = req.Notes

	result, err := h.trialBalanceService.PerformTrialBalanceCheck(c.Request.Context(), tenantID, userID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toTrialBalanceHandlerResponse(result))
}

// QuickBalanceCheck godoc
//
//	@ID				quickBalanceCheckFinanceTrialBalance
//	@Summary		Quick balance check
//	@Description	Performs a quick balance check (minimal validation) for use before operations
//	@Tags			finance-trial-balance
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[BalanceCheckGuardResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/trial-balance/quick-check [get]
func (h *TrialBalanceHandler) QuickBalanceCheck(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	result, err := h.trialBalanceService.QuickBalanceCheck(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toBalanceCheckGuardHandlerResponse(result))
}

// GetReconciliationReport godoc
//
//	@ID				getFinanceTrialBalanceReconciliationReport
//	@Summary		Get reconciliation report
//	@Description	Generates a detailed reconciliation report with health score and recommendations
//	@Tags			finance-trial-balance
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[ReconciliationReportResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/trial-balance/reconciliation-report [get]
func (h *TrialBalanceHandler) GetReconciliationReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		h.Unauthorized(c, "Authentication required for reconciliation report")
		return
	}

	result, err := h.trialBalanceService.GenerateReconciliationReport(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReconciliationReportHandlerResponse(result))
}

// GetAuditLogs godoc
//
//	@ID				getFinanceTrialBalanceAuditLogs
//	@Summary		Get trial balance audit logs
//	@Description	Retrieves the most recent trial balance audit logs
//	@Tags			finance-trial-balance
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			limit		query		int		false	"Number of logs to retrieve"	default(20)	maximum(100)
//	@Success		200			{object}	APIResponse[[]AuditLogResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/trial-balance/audit-logs [get]
func (h *TrialBalanceHandler) GetAuditLogs(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		var l int
		if _, err := parseIntWithDefault(limitStr, &l, 20); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}

	logs, err := h.trialBalanceService.GetAuditLogs(c.Request.Context(), tenantID, limit)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAuditLogHandlerResponses(logs))
}

// ===================== Response Conversion Functions =====================

func toTrialBalanceHandlerResponse(r *financeapp.TrialBalanceResponse) TrialBalanceResponse {
	discrepancies := make([]DiscrepancyResponse, len(r.Discrepancies))
	for i, d := range r.Discrepancies {
		discrepancies[i] = toDiscrepancyHandlerResponse(&d)
	}

	return TrialBalanceResponse{
		ID:                  r.ID.String(),
		TenantID:            r.TenantID.String(),
		CheckedAt:           r.CheckedAt,
		CheckedBy:           r.CheckedBy.String(),
		Status:              r.Status,
		TotalDebits:         r.TotalDebits.InexactFloat64(),
		TotalCredits:        r.TotalCredits.InexactFloat64(),
		NetBalance:          r.NetBalance.InexactFloat64(),
		TotalReceivables:    r.TotalReceivables.InexactFloat64(),
		TotalPayables:       r.TotalPayables.InexactFloat64(),
		TotalReceipts:       r.TotalReceipts.InexactFloat64(),
		TotalPayments:       r.TotalPayments.InexactFloat64(),
		TotalCreditMemos:    r.TotalCreditMemos.InexactFloat64(),
		TotalDebitMemos:     r.TotalDebitMemos.InexactFloat64(),
		ReceivableCount:     r.ReceivableCount,
		PayableCount:        r.PayableCount,
		ReceiptCount:        r.ReceiptCount,
		PaymentCount:        r.PaymentCount,
		CreditMemoCount:     r.CreditMemoCount,
		DebitMemoCount:      r.DebitMemoCount,
		Discrepancies:       discrepancies,
		DiscrepancyCount:    r.DiscrepancyCount,
		CriticalCount:       r.CriticalCount,
		WarningCount:        r.WarningCount,
		PeriodStart:         r.PeriodStart,
		PeriodEnd:           r.PeriodEnd,
		ExecutionDurationMs: r.ExecutionDurationMs,
		Notes:               r.Notes,
	}
}

func toDiscrepancyHandlerResponse(d *financeapp.DiscrepancyResponse) DiscrepancyResponse {
	relatedEntities := make([]RelatedEntityResponse, len(d.RelatedEntities))
	for i, re := range d.RelatedEntities {
		relatedEntities[i] = RelatedEntityResponse{
			EntityType: re.EntityType,
			EntityID:   re.EntityID.String(),
			Reference:  re.Reference,
		}
	}

	return DiscrepancyResponse{
		ID:              d.ID.String(),
		Type:            d.Type,
		EntityType:      d.EntityType,
		EntityID:        d.EntityID.String(),
		EntityNumber:    d.EntityNumber,
		ExpectedAmount:  d.ExpectedAmount.InexactFloat64(),
		ActualAmount:    d.ActualAmount.InexactFloat64(),
		Difference:      d.Difference.InexactFloat64(),
		Description:     d.Description,
		Severity:        d.Severity,
		DetectedAt:      d.DetectedAt,
		RelatedEntities: relatedEntities,
	}
}

func toBalanceCheckGuardHandlerResponse(r *financeapp.BalanceCheckGuardResponse) BalanceCheckGuardResponse {
	discrepancies := make([]DiscrepancyResponse, len(r.Discrepancies))
	for i, d := range r.Discrepancies {
		discrepancies[i] = toDiscrepancyHandlerResponse(&d)
	}

	return BalanceCheckGuardResponse{
		Allowed:       r.Allowed,
		Status:        r.Status,
		Message:       r.Message,
		Discrepancies: discrepancies,
	}
}

func toReconciliationReportHandlerResponse(r *financeapp.ReconciliationReportResponse) ReconciliationReportResponse {
	return ReconciliationReportResponse{
		TrialBalance:         toTrialBalanceHandlerResponse(&r.TrialBalance),
		GeneratedAt:          r.GeneratedAt,
		GeneratedBy:          r.GeneratedBy.String(),
		ReportType:           r.ReportType,
		TotalEntitiesChecked: r.TotalEntitiesChecked,
		HealthScore:          r.HealthScore,
		Recommendations:      r.Recommendations,
	}
}

func toAuditLogHandlerResponses(logs []financeapp.AuditLogResponse) []AuditLogResponse {
	responses := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = AuditLogResponse{
			ID:               log.ID.String(),
			TenantID:         log.TenantID.String(),
			CheckedAt:        log.CheckedAt,
			CheckedBy:        log.CheckedBy.String(),
			Status:           log.Status,
			TotalDebits:      log.TotalDebits.InexactFloat64(),
			TotalCredits:     log.TotalCredits.InexactFloat64(),
			NetBalance:       log.NetBalance.InexactFloat64(),
			DiscrepancyCount: log.DiscrepancyCount,
			CriticalCount:    log.CriticalCount,
			WarningCount:     log.WarningCount,
			DurationMs:       log.DurationMs,
			PeriodStart:      log.PeriodStart,
			PeriodEnd:        log.PeriodEnd,
			Notes:            log.Notes,
			CreatedAt:        log.CreatedAt,
		}
	}
	return responses
}

// parseIntWithDefault parses an int string with default value
func parseIntWithDefault(s string, target *int, defaultValue int) (int, error) {
	*target = defaultValue
	if s == "" {
		return *target, nil
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return *target, err
	}
	*target = val
	return *target, nil
}

// Suppress unused import warning
var _ = dto.Response{}
