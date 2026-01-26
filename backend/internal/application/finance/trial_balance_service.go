package finance

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TrialBalanceService provides application-level trial balance operations
type TrialBalanceService struct {
	domainService *finance.TrialBalanceService
}

// NewTrialBalanceService creates a new TrialBalanceService
func NewTrialBalanceService(
	receivableRepo finance.AccountReceivableRepository,
	payableRepo finance.AccountPayableRepository,
	receiptVoucherRepo finance.ReceiptVoucherRepository,
	paymentVoucherRepo finance.PaymentVoucherRepository,
	creditMemoRepo finance.CreditMemoRepository,
	debitMemoRepo finance.DebitMemoRepository,
	auditLogRepo finance.TrialBalanceAuditLogRepository,
) *TrialBalanceService {
	domainSvc := finance.NewTrialBalanceService(
		receivableRepo,
		payableRepo,
		receiptVoucherRepo,
		paymentVoucherRepo,
		creditMemoRepo,
		debitMemoRepo,
		finance.WithAuditLogRepository(auditLogRepo),
	)

	return &TrialBalanceService{
		domainService: domainSvc,
	}
}

// ===================== Response DTOs =====================

// TrialBalanceResponse represents the trial balance check result
type TrialBalanceResponse struct {
	ID                  uuid.UUID             `json:"id"`
	TenantID            uuid.UUID             `json:"tenant_id"`
	CheckedAt           time.Time             `json:"checked_at"`
	CheckedBy           uuid.UUID             `json:"checked_by"`
	Status              string                `json:"status"`
	TotalDebits         decimal.Decimal       `json:"total_debits"`
	TotalCredits        decimal.Decimal       `json:"total_credits"`
	NetBalance          decimal.Decimal       `json:"net_balance"`
	TotalReceivables    decimal.Decimal       `json:"total_receivables"`
	TotalPayables       decimal.Decimal       `json:"total_payables"`
	TotalReceipts       decimal.Decimal       `json:"total_receipts"`
	TotalPayments       decimal.Decimal       `json:"total_payments"`
	TotalCreditMemos    decimal.Decimal       `json:"total_credit_memos"`
	TotalDebitMemos     decimal.Decimal       `json:"total_debit_memos"`
	ReceivableCount     int64                 `json:"receivable_count"`
	PayableCount        int64                 `json:"payable_count"`
	ReceiptCount        int64                 `json:"receipt_count"`
	PaymentCount        int64                 `json:"payment_count"`
	CreditMemoCount     int64                 `json:"credit_memo_count"`
	DebitMemoCount      int64                 `json:"debit_memo_count"`
	Discrepancies       []DiscrepancyResponse `json:"discrepancies"`
	DiscrepancyCount    int                   `json:"discrepancy_count"`
	CriticalCount       int                   `json:"critical_count"`
	WarningCount        int                   `json:"warning_count"`
	PeriodStart         *time.Time            `json:"period_start,omitempty"`
	PeriodEnd           *time.Time            `json:"period_end,omitempty"`
	ExecutionDurationMs int64                 `json:"execution_duration_ms"`
	Notes               string                `json:"notes,omitempty"`
}

// DiscrepancyResponse represents a balance discrepancy in API responses
type DiscrepancyResponse struct {
	ID              uuid.UUID               `json:"id"`
	Type            string                  `json:"type"`
	EntityType      string                  `json:"entity_type"`
	EntityID        uuid.UUID               `json:"entity_id"`
	EntityNumber    string                  `json:"entity_number"`
	ExpectedAmount  decimal.Decimal         `json:"expected_amount"`
	ActualAmount    decimal.Decimal         `json:"actual_amount"`
	Difference      decimal.Decimal         `json:"difference"`
	Description     string                  `json:"description"`
	Severity        string                  `json:"severity"`
	DetectedAt      time.Time               `json:"detected_at"`
	RelatedEntities []RelatedEntityResponse `json:"related_entities,omitempty"`
}

// RelatedEntityResponse represents a related entity in a discrepancy
type RelatedEntityResponse struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Reference  string    `json:"reference"`
}

// BalanceCheckGuardResponse represents the result of a pre-operation balance check
type BalanceCheckGuardResponse struct {
	Allowed       bool                  `json:"allowed"`
	Status        string                `json:"status"`
	Message       string                `json:"message"`
	Discrepancies []DiscrepancyResponse `json:"discrepancies,omitempty"`
}

// AuditLogResponse represents a trial balance audit log entry
type AuditLogResponse struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	CheckedAt        time.Time       `json:"checked_at"`
	CheckedBy        uuid.UUID       `json:"checked_by"`
	Status           string          `json:"status"`
	TotalDebits      decimal.Decimal `json:"total_debits"`
	TotalCredits     decimal.Decimal `json:"total_credits"`
	NetBalance       decimal.Decimal `json:"net_balance"`
	DiscrepancyCount int             `json:"discrepancy_count"`
	CriticalCount    int             `json:"critical_count"`
	WarningCount     int             `json:"warning_count"`
	DurationMs       int64           `json:"duration_ms"`
	PeriodStart      *time.Time      `json:"period_start,omitempty"`
	PeriodEnd        *time.Time      `json:"period_end,omitempty"`
	Notes            string          `json:"notes,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ===================== Request DTOs =====================

// TrialBalanceCheckRequest represents a request to perform a trial balance check
type TrialBalanceCheckRequest struct {
	// Period filter
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`

	// What to check (default: all)
	CheckReceivables bool `json:"check_receivables"`
	CheckPayables    bool `json:"check_payables"`
	CheckReceipts    bool `json:"check_receipts"`
	CheckPayments    bool `json:"check_payments"`
	CheckCreditMemos bool `json:"check_credit_memos"`
	CheckDebitMemos  bool `json:"check_debit_memos"`

	// Validation options (default: all enabled)
	ValidateInternalConsistency bool `json:"validate_internal_consistency"`
	ValidateVoucherAllocations  bool `json:"validate_voucher_allocations"`
	ValidateMemoApplications    bool `json:"validate_memo_applications"`

	// Notes for the audit log
	Notes string `json:"notes,omitempty"`
}

// DefaultTrialBalanceCheckRequest returns a request with all checks enabled
func DefaultTrialBalanceCheckRequest() TrialBalanceCheckRequest {
	return TrialBalanceCheckRequest{
		CheckReceivables:            true,
		CheckPayables:               true,
		CheckReceipts:               true,
		CheckPayments:               true,
		CheckCreditMemos:            true,
		CheckDebitMemos:             true,
		ValidateInternalConsistency: true,
		ValidateVoucherAllocations:  true,
		ValidateMemoApplications:    true,
	}
}

// ===================== Service Methods =====================

// PerformTrialBalanceCheck performs a comprehensive trial balance check
func (s *TrialBalanceService) PerformTrialBalanceCheck(
	ctx context.Context,
	tenantID uuid.UUID,
	checkedBy uuid.UUID,
	req TrialBalanceCheckRequest,
) (*TrialBalanceResponse, error) {
	// Convert request to domain options
	opts := finance.TrialBalanceCheckOptions{
		PeriodStart:                 req.PeriodStart,
		PeriodEnd:                   req.PeriodEnd,
		CheckReceivables:            req.CheckReceivables,
		CheckPayables:               req.CheckPayables,
		CheckReceipts:               req.CheckReceipts,
		CheckPayments:               req.CheckPayments,
		CheckCreditMemos:            req.CheckCreditMemos,
		CheckDebitMemos:             req.CheckDebitMemos,
		ValidateInternalConsistency: req.ValidateInternalConsistency,
		ValidateVoucherAllocations:  req.ValidateVoucherAllocations,
		ValidateMemoApplications:    req.ValidateMemoApplications,
		Tolerance:                   decimal.NewFromFloat(0.01),
	}

	// If no checks specified, use defaults
	if !anyCheckEnabled(req) {
		opts = finance.DefaultTrialBalanceCheckOptions()
		opts.PeriodStart = req.PeriodStart
		opts.PeriodEnd = req.PeriodEnd
	}

	result, err := s.domainService.PerformTrialBalanceCheck(ctx, tenantID, checkedBy, opts)
	if err != nil {
		return nil, err
	}

	// Add notes if provided
	if req.Notes != "" {
		result.Notes = req.Notes
	}

	return toTrialBalanceResponse(result), nil
}

// QuickBalanceCheck performs a quick balance check (minimal validation)
func (s *TrialBalanceService) QuickBalanceCheck(
	ctx context.Context,
	tenantID uuid.UUID,
) (*BalanceCheckGuardResponse, error) {
	result, err := s.domainService.CheckBalanceBeforeOperation(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return toBalanceCheckGuardResponse(result), nil
}

// EnforceBalanceCheck enforces balance check - returns error if unbalanced
func (s *TrialBalanceService) EnforceBalanceCheck(
	ctx context.Context,
	tenantID uuid.UUID,
) error {
	return s.domainService.EnforceBalanceCheck(ctx, tenantID)
}

// GetAuditLogs retrieves the most recent trial balance audit logs
func (s *TrialBalanceService) GetAuditLogs(
	ctx context.Context,
	tenantID uuid.UUID,
	limit int,
) ([]AuditLogResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	logs, err := s.domainService.GetLatestAuditLogs(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}

	return toAuditLogResponses(logs), nil
}

// GenerateReconciliationReport generates a detailed reconciliation report
func (s *TrialBalanceService) GenerateReconciliationReport(
	ctx context.Context,
	tenantID uuid.UUID,
	checkedBy uuid.UUID,
) (*ReconciliationReportResponse, error) {
	// Perform full check
	req := DefaultTrialBalanceCheckRequest()
	req.ValidateInternalConsistency = true
	req.ValidateVoucherAllocations = true
	req.ValidateMemoApplications = true
	req.Notes = "Reconciliation report generation"

	result, err := s.PerformTrialBalanceCheck(ctx, tenantID, checkedBy, req)
	if err != nil {
		return nil, err
	}

	return &ReconciliationReportResponse{
		TrialBalance:         *result,
		GeneratedAt:          time.Now(),
		GeneratedBy:          checkedBy,
		ReportType:           "RECONCILIATION",
		TotalEntitiesChecked: result.ReceivableCount + result.PayableCount + result.ReceiptCount + result.PaymentCount + result.CreditMemoCount + result.DebitMemoCount,
		HealthScore:          calculateHealthScore(result),
		Recommendations:      generateRecommendations(result),
	}, nil
}

// ReconciliationReportResponse represents a detailed reconciliation report
type ReconciliationReportResponse struct {
	TrialBalance         TrialBalanceResponse `json:"trial_balance"`
	GeneratedAt          time.Time            `json:"generated_at"`
	GeneratedBy          uuid.UUID            `json:"generated_by"`
	ReportType           string               `json:"report_type"`
	TotalEntitiesChecked int64                `json:"total_entities_checked"`
	HealthScore          int                  `json:"health_score"` // 0-100
	Recommendations      []string             `json:"recommendations"`
}

// ===================== Helper Functions =====================

func anyCheckEnabled(req TrialBalanceCheckRequest) bool {
	return req.CheckReceivables || req.CheckPayables || req.CheckReceipts ||
		req.CheckPayments || req.CheckCreditMemos || req.CheckDebitMemos
}

func toTrialBalanceResponse(r *finance.TrialBalanceResult) *TrialBalanceResponse {
	discrepancies := make([]DiscrepancyResponse, len(r.Discrepancies))
	for i, d := range r.Discrepancies {
		discrepancies[i] = toDiscrepancyResponse(&d)
	}

	return &TrialBalanceResponse{
		ID:                  r.ID,
		TenantID:            r.TenantID,
		CheckedAt:           r.CheckedAt,
		CheckedBy:           r.CheckedBy,
		Status:              string(r.Status),
		TotalDebits:         r.TotalDebits,
		TotalCredits:        r.TotalCredits,
		NetBalance:          r.NetBalance,
		TotalReceivables:    r.TotalReceivables,
		TotalPayables:       r.TotalPayables,
		TotalReceipts:       r.TotalReceipts,
		TotalPayments:       r.TotalPayments,
		TotalCreditMemos:    r.TotalCreditMemos,
		TotalDebitMemos:     r.TotalDebitMemos,
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

func toDiscrepancyResponse(d *finance.BalanceDiscrepancy) DiscrepancyResponse {
	relatedEntities := make([]RelatedEntityResponse, len(d.RelatedEntities))
	for i, re := range d.RelatedEntities {
		relatedEntities[i] = RelatedEntityResponse{
			EntityType: re.EntityType,
			EntityID:   re.EntityID,
			Reference:  re.Reference,
		}
	}

	return DiscrepancyResponse{
		ID:              d.ID,
		Type:            string(d.Type),
		EntityType:      d.EntityType,
		EntityID:        d.EntityID,
		EntityNumber:    d.EntityNumber,
		ExpectedAmount:  d.ExpectedAmount,
		ActualAmount:    d.ActualAmount,
		Difference:      d.Difference,
		Description:     d.Description,
		Severity:        d.Severity,
		DetectedAt:      d.DetectedAt,
		RelatedEntities: relatedEntities,
	}
}

func toBalanceCheckGuardResponse(r *finance.BalanceCheckGuardResult) *BalanceCheckGuardResponse {
	discrepancies := make([]DiscrepancyResponse, len(r.Discrepancies))
	for i, d := range r.Discrepancies {
		discrepancies[i] = toDiscrepancyResponse(&d)
	}

	return &BalanceCheckGuardResponse{
		Allowed:       r.Allowed,
		Status:        string(r.Status),
		Message:       r.Message,
		Discrepancies: discrepancies,
	}
}

func toAuditLogResponses(logs []finance.TrialBalanceAuditLog) []AuditLogResponse {
	responses := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = AuditLogResponse{
			ID:               log.ID,
			TenantID:         log.TenantID,
			CheckedAt:        log.CheckedAt,
			CheckedBy:        log.CheckedBy,
			Status:           string(log.Status),
			TotalDebits:      log.TotalDebits,
			TotalCredits:     log.TotalCredits,
			NetBalance:       log.NetBalance,
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

func calculateHealthScore(result *TrialBalanceResponse) int {
	// Base score is 100
	score := 100

	// Deduct for discrepancies
	// Critical discrepancies: -20 each (max -60)
	criticalDeduction := result.CriticalCount * 20
	if criticalDeduction > 60 {
		criticalDeduction = 60
	}
	score -= criticalDeduction

	// Warning discrepancies: -5 each (max -30)
	warningDeduction := result.WarningCount * 5
	if warningDeduction > 30 {
		warningDeduction = 30
	}
	score -= warningDeduction

	// Deduct for unbalanced net balance
	if !result.NetBalance.IsZero() {
		score -= 10
	}

	// Ensure score is non-negative
	if score < 0 {
		score = 0
	}

	return score
}

func generateRecommendations(result *TrialBalanceResponse) []string {
	var recommendations []string

	if result.CriticalCount > 0 {
		recommendations = append(recommendations, "CRITICAL: Investigate and resolve critical discrepancies immediately")
	}

	if result.WarningCount > 0 {
		recommendations = append(recommendations, "Review warning-level discrepancies for potential data issues")
	}

	if !result.NetBalance.IsZero() {
		recommendations = append(recommendations, "Net balance is non-zero - review all financial transactions")
	}

	// Check for specific discrepancy types
	for _, d := range result.Discrepancies {
		switch d.Type {
		case string(finance.ReceivableAmountMismatch):
			recommendations = append(recommendations, "Receivable amount mismatch detected - verify payment records")
		case string(finance.PayableAmountMismatch):
			recommendations = append(recommendations, "Payable amount mismatch detected - verify payment vouchers")
		case string(finance.VoucherAllocationMismatch):
			recommendations = append(recommendations, "Voucher allocation mismatch - review reconciliation records")
		case string(finance.CreditMemoImbalance):
			recommendations = append(recommendations, "Credit memo imbalance - verify credit memo applications")
		case string(finance.DebitMemoImbalance):
			recommendations = append(recommendations, "Debit memo imbalance - verify debit memo applications")
		}
	}

	// Deduplicate recommendations
	seen := make(map[string]bool)
	uniqueRecs := []string{}
	for _, rec := range recommendations {
		if !seen[rec] {
			seen[rec] = true
			uniqueRecs = append(uniqueRecs, rec)
		}
	}

	if len(uniqueRecs) == 0 {
		uniqueRecs = append(uniqueRecs, "All financial records are balanced - no action required")
	}

	return uniqueRecs
}
