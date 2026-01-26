package finance

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TrialBalanceStatus represents the result status of a trial balance check
type TrialBalanceStatus string

const (
	TrialBalanceStatusBalanced   TrialBalanceStatus = "BALANCED"   // Debit equals Credit
	TrialBalanceStatusUnbalanced TrialBalanceStatus = "UNBALANCED" // Debit does not equal Credit
)

// IsValid checks if the status is a valid TrialBalanceStatus
func (s TrialBalanceStatus) IsValid() bool {
	return s == TrialBalanceStatusBalanced || s == TrialBalanceStatusUnbalanced
}

// String returns the string representation
func (s TrialBalanceStatus) String() string {
	return string(s)
}

// IsBalanced returns true if the trial balance is balanced
func (s TrialBalanceStatus) IsBalanced() bool {
	return s == TrialBalanceStatusBalanced
}

// BalanceDiscrepancyType represents the type of balance discrepancy
type BalanceDiscrepancyType string

const (
	// ReceivablePaymentMismatch occurs when receivable payments don't match receipt voucher allocations
	ReceivablePaymentMismatch BalanceDiscrepancyType = "RECEIVABLE_PAYMENT_MISMATCH"
	// PayablePaymentMismatch occurs when payable payments don't match payment voucher allocations
	PayablePaymentMismatch BalanceDiscrepancyType = "PAYABLE_PAYMENT_MISMATCH"
	// VoucherAllocationMismatch occurs when voucher allocations don't match total amount
	VoucherAllocationMismatch BalanceDiscrepancyType = "VOUCHER_ALLOCATION_MISMATCH"
	// CreditMemoImbalance occurs when credit memo applications exceed total credit
	CreditMemoImbalance BalanceDiscrepancyType = "CREDIT_MEMO_IMBALANCE"
	// DebitMemoImbalance occurs when debit memo applications exceed total debit
	DebitMemoImbalance BalanceDiscrepancyType = "DEBIT_MEMO_IMBALANCE"
	// ReceivableAmountMismatch occurs when paid + outstanding != total
	ReceivableAmountMismatch BalanceDiscrepancyType = "RECEIVABLE_AMOUNT_MISMATCH"
	// PayableAmountMismatch occurs when paid + outstanding != total
	PayableAmountMismatch BalanceDiscrepancyType = "PAYABLE_AMOUNT_MISMATCH"
)

// IsValid checks if the discrepancy type is valid
func (t BalanceDiscrepancyType) IsValid() bool {
	switch t {
	case ReceivablePaymentMismatch, PayablePaymentMismatch, VoucherAllocationMismatch,
		CreditMemoImbalance, DebitMemoImbalance, ReceivableAmountMismatch, PayableAmountMismatch:
		return true
	}
	return false
}

// Description returns a human-readable description of the discrepancy type
func (t BalanceDiscrepancyType) Description() string {
	switch t {
	case ReceivablePaymentMismatch:
		return "Receivable payment records don't match receipt voucher allocations"
	case PayablePaymentMismatch:
		return "Payable payment records don't match payment voucher allocations"
	case VoucherAllocationMismatch:
		return "Voucher allocated amount doesn't match sum of allocations"
	case CreditMemoImbalance:
		return "Credit memo applied amount exceeds total credit"
	case DebitMemoImbalance:
		return "Debit memo applied amount exceeds total debit"
	case ReceivableAmountMismatch:
		return "Receivable paid + outstanding doesn't equal total amount"
	case PayableAmountMismatch:
		return "Payable paid + outstanding doesn't equal total amount"
	default:
		return "Unknown discrepancy"
	}
}

// BalanceDiscrepancy represents a specific balance discrepancy found during trial balance check
type BalanceDiscrepancy struct {
	ID              uuid.UUID                  `json:"id"`
	Type            BalanceDiscrepancyType     `json:"type"`
	EntityType      string                     `json:"entity_type"`                // "AccountReceivable", "ReceiptVoucher", etc.
	EntityID        uuid.UUID                  `json:"entity_id"`                  // ID of the entity with the discrepancy
	EntityNumber    string                     `json:"entity_number"`              // Business number of the entity
	ExpectedAmount  decimal.Decimal            `json:"expected_amount"`            // What the amount should be
	ActualAmount    decimal.Decimal            `json:"actual_amount"`              // What the amount actually is
	Difference      decimal.Decimal            `json:"difference"`                 // ExpectedAmount - ActualAmount
	Description     string                     `json:"description"`                // Human-readable description
	Severity        string                     `json:"severity"`                   // "CRITICAL", "WARNING", "INFO"
	DetectedAt      time.Time                  `json:"detected_at"`                // When the discrepancy was detected
	RelatedEntities []DiscrepancyRelatedEntity `json:"related_entities,omitempty"` // Related entities for context
}

// DiscrepancyRelatedEntity provides context about related entities
type DiscrepancyRelatedEntity struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Reference  string    `json:"reference"` // e.g., "ReceiptVoucher: RV-2026-00001"
}

// NewBalanceDiscrepancy creates a new balance discrepancy
func NewBalanceDiscrepancy(
	discrepancyType BalanceDiscrepancyType,
	entityType string,
	entityID uuid.UUID,
	entityNumber string,
	expected, actual decimal.Decimal,
) *BalanceDiscrepancy {
	diff := expected.Sub(actual)
	severity := "WARNING"
	if diff.Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		severity = "CRITICAL"
	}

	return &BalanceDiscrepancy{
		ID:              uuid.New(),
		Type:            discrepancyType,
		EntityType:      entityType,
		EntityID:        entityID,
		EntityNumber:    entityNumber,
		ExpectedAmount:  expected,
		ActualAmount:    actual,
		Difference:      diff,
		Description:     discrepancyType.Description(),
		Severity:        severity,
		DetectedAt:      time.Now(),
		RelatedEntities: make([]DiscrepancyRelatedEntity, 0),
	}
}

// AddRelatedEntity adds a related entity for context
func (d *BalanceDiscrepancy) AddRelatedEntity(entityType string, entityID uuid.UUID, reference string) {
	d.RelatedEntities = append(d.RelatedEntities, DiscrepancyRelatedEntity{
		EntityType: entityType,
		EntityID:   entityID,
		Reference:  reference,
	})
}

// TrialBalanceResult represents the result of a trial balance check
type TrialBalanceResult struct {
	ID           uuid.UUID          `json:"id"`
	TenantID     uuid.UUID          `json:"tenant_id"`
	CheckedAt    time.Time          `json:"checked_at"`
	CheckedBy    uuid.UUID          `json:"checked_by"` // User who initiated the check
	Status       TrialBalanceStatus `json:"status"`
	TotalDebits  decimal.Decimal    `json:"total_debits"`  // Total debit amounts
	TotalCredits decimal.Decimal    `json:"total_credits"` // Total credit amounts
	NetBalance   decimal.Decimal    `json:"net_balance"`   // Debits - Credits (should be 0)

	// Breakdown by category
	TotalReceivables decimal.Decimal `json:"total_receivables"`  // Outstanding receivables
	TotalPayables    decimal.Decimal `json:"total_payables"`     // Outstanding payables
	TotalReceipts    decimal.Decimal `json:"total_receipts"`     // Unallocated receipt amounts
	TotalPayments    decimal.Decimal `json:"total_payments"`     // Unallocated payment amounts
	TotalCreditMemos decimal.Decimal `json:"total_credit_memos"` // Remaining credit memo amounts
	TotalDebitMemos  decimal.Decimal `json:"total_debit_memos"`  // Remaining debit memo amounts

	// Counts
	ReceivableCount int64 `json:"receivable_count"`
	PayableCount    int64 `json:"payable_count"`
	ReceiptCount    int64 `json:"receipt_count"`
	PaymentCount    int64 `json:"payment_count"`
	CreditMemoCount int64 `json:"credit_memo_count"`
	DebitMemoCount  int64 `json:"debit_memo_count"`

	// Discrepancies found
	Discrepancies    []BalanceDiscrepancy `json:"discrepancies"`
	DiscrepancyCount int                  `json:"discrepancy_count"`
	CriticalCount    int                  `json:"critical_count"`
	WarningCount     int                  `json:"warning_count"`

	// Period filter (if applicable)
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`

	// Execution metadata
	ExecutionDurationMs int64  `json:"execution_duration_ms"`
	Notes               string `json:"notes,omitempty"`
}

// NewTrialBalanceResult creates a new trial balance result
func NewTrialBalanceResult(tenantID, checkedBy uuid.UUID) *TrialBalanceResult {
	return &TrialBalanceResult{
		ID:            uuid.New(),
		TenantID:      tenantID,
		CheckedAt:     time.Now(),
		CheckedBy:     checkedBy,
		Status:        TrialBalanceStatusBalanced,
		TotalDebits:   decimal.Zero,
		TotalCredits:  decimal.Zero,
		NetBalance:    decimal.Zero,
		Discrepancies: make([]BalanceDiscrepancy, 0),
	}
}

// AddDiscrepancy adds a discrepancy to the result
func (r *TrialBalanceResult) AddDiscrepancy(d *BalanceDiscrepancy) {
	r.Discrepancies = append(r.Discrepancies, *d)
	r.DiscrepancyCount++
	if d.Severity == "CRITICAL" {
		r.CriticalCount++
	} else if d.Severity == "WARNING" {
		r.WarningCount++
	}
	// If any discrepancy, mark as unbalanced
	r.Status = TrialBalanceStatusUnbalanced
}

// SetTotals sets the debit and credit totals and calculates net balance
func (r *TrialBalanceResult) SetTotals(debits, credits decimal.Decimal) {
	r.TotalDebits = debits
	r.TotalCredits = credits
	r.NetBalance = debits.Sub(credits)
}

// SetPeriod sets the period filter for the trial balance check
func (r *TrialBalanceResult) SetPeriod(start, end *time.Time) {
	r.PeriodStart = start
	r.PeriodEnd = end
}

// SetExecutionDuration sets the execution duration in milliseconds
func (r *TrialBalanceResult) SetExecutionDuration(durationMs int64) {
	r.ExecutionDurationMs = durationMs
}

// IsBalanced returns true if the trial balance is balanced (no discrepancies and net balance is 0)
func (r *TrialBalanceResult) IsBalanced() bool {
	return r.Status.IsBalanced() && r.DiscrepancyCount == 0 && r.NetBalance.IsZero()
}

// HasCriticalDiscrepancies returns true if there are critical discrepancies
func (r *TrialBalanceResult) HasCriticalDiscrepancies() bool {
	return r.CriticalCount > 0
}

// GetCriticalDiscrepancies returns only critical discrepancies
func (r *TrialBalanceResult) GetCriticalDiscrepancies() []BalanceDiscrepancy {
	var critical []BalanceDiscrepancy
	for _, d := range r.Discrepancies {
		if d.Severity == "CRITICAL" {
			critical = append(critical, d)
		}
	}
	return critical
}

// TrialBalanceCheckOptions configures the trial balance check
type TrialBalanceCheckOptions struct {
	// Period filter
	PeriodStart *time.Time
	PeriodEnd   *time.Time

	// What to check
	CheckReceivables bool
	CheckPayables    bool
	CheckReceipts    bool
	CheckPayments    bool
	CheckCreditMemos bool
	CheckDebitMemos  bool

	// Validation options
	ValidateInternalConsistency bool // Check that paid + outstanding = total
	ValidateVoucherAllocations  bool // Check that allocations match voucher amounts
	ValidateMemoApplications    bool // Check that memo applications are valid

	// Tolerance for float comparison (default: 0.01)
	Tolerance decimal.Decimal
}

// DefaultTrialBalanceCheckOptions returns default options with all checks enabled
func DefaultTrialBalanceCheckOptions() TrialBalanceCheckOptions {
	return TrialBalanceCheckOptions{
		CheckReceivables:            true,
		CheckPayables:               true,
		CheckReceipts:               true,
		CheckPayments:               true,
		CheckCreditMemos:            true,
		CheckDebitMemos:             true,
		ValidateInternalConsistency: true,
		ValidateVoucherAllocations:  true,
		ValidateMemoApplications:    true,
		Tolerance:                   decimal.NewFromFloat(0.01),
	}
}

// TrialBalanceAuditLog represents an audit log entry for trial balance checks
type TrialBalanceAuditLog struct {
	ID               uuid.UUID          `json:"id" gorm:"type:uuid;primary_key"`
	TenantID         uuid.UUID          `json:"tenant_id" gorm:"type:uuid;not null;index"`
	CheckedAt        time.Time          `json:"checked_at" gorm:"not null;index"`
	CheckedBy        uuid.UUID          `json:"checked_by" gorm:"type:uuid;not null;index"`
	Status           TrialBalanceStatus `json:"status" gorm:"type:varchar(20);not null;index"`
	TotalDebits      decimal.Decimal    `json:"total_debits" gorm:"type:decimal(18,4);not null"`
	TotalCredits     decimal.Decimal    `json:"total_credits" gorm:"type:decimal(18,4);not null"`
	NetBalance       decimal.Decimal    `json:"net_balance" gorm:"type:decimal(18,4);not null"`
	DiscrepancyCount int                `json:"discrepancy_count" gorm:"not null"`
	CriticalCount    int                `json:"critical_count" gorm:"not null"`
	WarningCount     int                `json:"warning_count" gorm:"not null"`
	DurationMs       int64              `json:"duration_ms" gorm:"not null"`
	PeriodStart      *time.Time         `json:"period_start" gorm:"index"`
	PeriodEnd        *time.Time         `json:"period_end" gorm:"index"`
	Notes            string             `json:"notes" gorm:"type:text"`
	DetailsJSON      string             `json:"details_json" gorm:"type:text"` // JSON of discrepancies for audit
	CreatedAt        time.Time          `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the table name for GORM
func (TrialBalanceAuditLog) TableName() string {
	return "trial_balance_audit_logs"
}

// NewTrialBalanceAuditLog creates a new audit log entry from a trial balance result
func NewTrialBalanceAuditLog(result *TrialBalanceResult) *TrialBalanceAuditLog {
	return &TrialBalanceAuditLog{
		ID:               uuid.New(),
		TenantID:         result.TenantID,
		CheckedAt:        result.CheckedAt,
		CheckedBy:        result.CheckedBy,
		Status:           result.Status,
		TotalDebits:      result.TotalDebits,
		TotalCredits:     result.TotalCredits,
		NetBalance:       result.NetBalance,
		DiscrepancyCount: result.DiscrepancyCount,
		CriticalCount:    result.CriticalCount,
		WarningCount:     result.WarningCount,
		DurationMs:       result.ExecutionDurationMs,
		PeriodStart:      result.PeriodStart,
		PeriodEnd:        result.PeriodEnd,
		Notes:            result.Notes,
	}
}

// BalanceCheckGuardResult represents the result of a pre-operation balance check
type BalanceCheckGuardResult struct {
	Allowed       bool                 `json:"allowed"`
	Status        TrialBalanceStatus   `json:"status"`
	Message       string               `json:"message"`
	Discrepancies []BalanceDiscrepancy `json:"discrepancies,omitempty"`
}

// NewAllowedGuardResult creates a result indicating the operation is allowed
func NewAllowedGuardResult() *BalanceCheckGuardResult {
	return &BalanceCheckGuardResult{
		Allowed: true,
		Status:  TrialBalanceStatusBalanced,
		Message: "Trial balance is balanced, operation allowed",
	}
}

// NewBlockedGuardResult creates a result indicating the operation is blocked
func NewBlockedGuardResult(discrepancies []BalanceDiscrepancy) *BalanceCheckGuardResult {
	return &BalanceCheckGuardResult{
		Allowed:       false,
		Status:        TrialBalanceStatusUnbalanced,
		Message:       "Trial balance check failed, operation blocked due to balance discrepancies",
		Discrepancies: discrepancies,
	}
}
