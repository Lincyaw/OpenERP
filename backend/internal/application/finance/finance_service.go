package finance

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FinanceService provides application-level finance operations
type FinanceService struct {
	receivableRepo     finance.AccountReceivableRepository
	payableRepo        finance.AccountPayableRepository
	receiptVoucherRepo finance.ReceiptVoucherRepository
	paymentVoucherRepo finance.PaymentVoucherRepository
	reconciliationSvc  *finance.ReconciliationService
}

// FinanceServiceOption is a functional option for configuring FinanceService
type FinanceServiceOption func(*FinanceService)

// WithReconciliationStrategy sets the default reconciliation strategy type
func WithReconciliationStrategy(strategyType finance.ReconciliationStrategyType) FinanceServiceOption {
	return func(s *FinanceService) {
		// Recreate the reconciliation service with the new default strategy
		s.reconciliationSvc = finance.NewReconciliationService(
			finance.WithDefaultStrategy(strategyType),
		)
	}
}

// WithReconciliationStrategyOverride sets a function to determine strategy based on context
func WithReconciliationStrategyOverride(fn finance.StrategyOverrideFunc) FinanceServiceOption {
	return func(s *FinanceService) {
		// Recreate the reconciliation service with the override function
		currentDefault := s.reconciliationSvc.GetDefaultStrategy()
		s.reconciliationSvc = finance.NewReconciliationService(
			finance.WithDefaultStrategy(currentDefault),
			finance.WithStrategyOverride(fn),
		)
	}
}

// WithReconciliationService allows injecting a custom ReconciliationService
func WithReconciliationService(svc *finance.ReconciliationService) FinanceServiceOption {
	return func(s *FinanceService) {
		s.reconciliationSvc = svc
	}
}

// NewFinanceService creates a new FinanceService
func NewFinanceService(
	receivableRepo finance.AccountReceivableRepository,
	payableRepo finance.AccountPayableRepository,
	receiptVoucherRepo finance.ReceiptVoucherRepository,
	paymentVoucherRepo finance.PaymentVoucherRepository,
	opts ...FinanceServiceOption,
) *FinanceService {
	s := &FinanceService{
		receivableRepo:     receivableRepo,
		payableRepo:        payableRepo,
		receiptVoucherRepo: receiptVoucherRepo,
		paymentVoucherRepo: paymentVoucherRepo,
		reconciliationSvc:  finance.NewReconciliationService(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetReconciliationService returns the underlying reconciliation service for inspection
func (s *FinanceService) GetReconciliationService() *finance.ReconciliationService {
	return s.reconciliationSvc
}

// ===================== Account Receivable Operations =====================

// AccountReceivableResponse represents an account receivable in API responses
type AccountReceivableResponse struct {
	ID                uuid.UUID               `json:"id"`
	TenantID          uuid.UUID               `json:"tenant_id"`
	ReceivableNumber  string                  `json:"receivable_number"`
	CustomerID        uuid.UUID               `json:"customer_id"`
	CustomerName      string                  `json:"customer_name"`
	SourceType        string                  `json:"source_type"`
	SourceID          uuid.UUID               `json:"source_id"`
	SourceNumber      string                  `json:"source_number"`
	TotalAmount       decimal.Decimal         `json:"total_amount"`
	PaidAmount        decimal.Decimal         `json:"paid_amount"`
	OutstandingAmount decimal.Decimal         `json:"outstanding_amount"`
	Status            string                  `json:"status"`
	DueDate           *time.Time              `json:"due_date,omitempty"`
	PaymentRecords    []PaymentRecordResponse `json:"payment_records,omitempty"`
	Remark            string                  `json:"remark,omitempty"`
	PaidAt            *time.Time              `json:"paid_at,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	Version           int                     `json:"version"`
}

// PaymentRecordResponse represents a payment record in API responses
type PaymentRecordResponse struct {
	ID               uuid.UUID       `json:"id"`
	ReceiptVoucherID uuid.UUID       `json:"receipt_voucher_id"`
	Amount           decimal.Decimal `json:"amount"`
	AppliedAt        time.Time       `json:"applied_at"`
	Remark           string          `json:"remark,omitempty"`
}

// AccountReceivableListFilter defines filtering options for receivable list queries
type AccountReceivableListFilter struct {
	Search     string     `form:"search"`
	CustomerID *uuid.UUID `form:"customer_id"`
	Status     string     `form:"status"`
	SourceType string     `form:"source_type"`
	FromDate   *time.Time `form:"from_date"`
	ToDate     *time.Time `form:"to_date"`
	Overdue    *bool      `form:"overdue"`
	Page       int        `form:"page"`
	PageSize   int        `form:"page_size"`
}

// GetReceivableByID gets a receivable by ID
func (s *FinanceService) GetReceivableByID(ctx context.Context, tenantID, id uuid.UUID) (*AccountReceivableResponse, error) {
	receivable, err := s.receivableRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if receivable == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Account receivable not found")
	}
	return toReceivableResponse(receivable), nil
}

// ListReceivables lists receivables with filtering
func (s *FinanceService) ListReceivables(ctx context.Context, tenantID uuid.UUID, filter AccountReceivableListFilter) ([]AccountReceivableResponse, int64, error) {
	domainFilter := finance.AccountReceivableFilter{
		CustomerID: filter.CustomerID,
		FromDate:   filter.FromDate,
		ToDate:     filter.ToDate,
		Overdue:    filter.Overdue,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Status != "" {
		status := finance.ReceivableStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.SourceType != "" {
		sourceType := finance.SourceType(filter.SourceType)
		domainFilter.SourceType = &sourceType
	}

	receivables, err := s.receivableRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.receivableRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]AccountReceivableResponse, len(receivables))
	for i, r := range receivables {
		responses[i] = *toReceivableResponse(&r)
	}

	return responses, total, nil
}

// GetReceivableSummary gets a summary of receivables for a tenant
type ReceivableSummary struct {
	TotalOutstanding decimal.Decimal `json:"total_outstanding"`
	TotalOverdue     decimal.Decimal `json:"total_overdue"`
	PendingCount     int64           `json:"pending_count"`
	PartialCount     int64           `json:"partial_count"`
	OverdueCount     int64           `json:"overdue_count"`
}

func (s *FinanceService) GetReceivableSummary(ctx context.Context, tenantID uuid.UUID) (*ReceivableSummary, error) {
	totalOutstanding, err := s.receivableRepo.SumOutstandingForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	totalOverdue, err := s.receivableRepo.SumOverdueForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	pendingCount, err := s.receivableRepo.CountByStatus(ctx, tenantID, finance.ReceivableStatusPending)
	if err != nil {
		return nil, err
	}

	partialCount, err := s.receivableRepo.CountByStatus(ctx, tenantID, finance.ReceivableStatusPartial)
	if err != nil {
		return nil, err
	}

	overdueCount, err := s.receivableRepo.CountOverdue(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &ReceivableSummary{
		TotalOutstanding: totalOutstanding,
		TotalOverdue:     totalOverdue,
		PendingCount:     pendingCount,
		PartialCount:     partialCount,
		OverdueCount:     overdueCount,
	}, nil
}

// ===================== Account Payable Operations =====================

// AccountPayableResponse represents an account payable in API responses
type AccountPayableResponse struct {
	ID                uuid.UUID                      `json:"id"`
	TenantID          uuid.UUID                      `json:"tenant_id"`
	PayableNumber     string                         `json:"payable_number"`
	SupplierID        uuid.UUID                      `json:"supplier_id"`
	SupplierName      string                         `json:"supplier_name"`
	SourceType        string                         `json:"source_type"`
	SourceID          uuid.UUID                      `json:"source_id"`
	SourceNumber      string                         `json:"source_number"`
	TotalAmount       decimal.Decimal                `json:"total_amount"`
	PaidAmount        decimal.Decimal                `json:"paid_amount"`
	OutstandingAmount decimal.Decimal                `json:"outstanding_amount"`
	Status            string                         `json:"status"`
	DueDate           *time.Time                     `json:"due_date,omitempty"`
	PaymentRecords    []PayablePaymentRecordResponse `json:"payment_records,omitempty"`
	Remark            string                         `json:"remark,omitempty"`
	PaidAt            *time.Time                     `json:"paid_at,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	UpdatedAt         time.Time                      `json:"updated_at"`
	Version           int                            `json:"version"`
}

// PayablePaymentRecordResponse represents a payment record for payable in API responses
type PayablePaymentRecordResponse struct {
	ID               uuid.UUID       `json:"id"`
	PaymentVoucherID uuid.UUID       `json:"payment_voucher_id"`
	Amount           decimal.Decimal `json:"amount"`
	AppliedAt        time.Time       `json:"applied_at"`
	Remark           string          `json:"remark,omitempty"`
}

// AccountPayableListFilter defines filtering options for payable list queries
type AccountPayableListFilter struct {
	Search     string     `form:"search"`
	SupplierID *uuid.UUID `form:"supplier_id"`
	Status     string     `form:"status"`
	SourceType string     `form:"source_type"`
	FromDate   *time.Time `form:"from_date"`
	ToDate     *time.Time `form:"to_date"`
	Overdue    *bool      `form:"overdue"`
	Page       int        `form:"page"`
	PageSize   int        `form:"page_size"`
}

// GetPayableByID gets a payable by ID
func (s *FinanceService) GetPayableByID(ctx context.Context, tenantID, id uuid.UUID) (*AccountPayableResponse, error) {
	payable, err := s.payableRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if payable == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Account payable not found")
	}
	return toPayableResponse(payable), nil
}

// ListPayables lists payables with filtering
func (s *FinanceService) ListPayables(ctx context.Context, tenantID uuid.UUID, filter AccountPayableListFilter) ([]AccountPayableResponse, int64, error) {
	domainFilter := finance.AccountPayableFilter{
		SupplierID: filter.SupplierID,
		FromDate:   filter.FromDate,
		ToDate:     filter.ToDate,
		Overdue:    filter.Overdue,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Status != "" {
		status := finance.PayableStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.SourceType != "" {
		sourceType := finance.PayableSourceType(filter.SourceType)
		domainFilter.SourceType = &sourceType
	}

	payables, err := s.payableRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.payableRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]AccountPayableResponse, len(payables))
	for i, p := range payables {
		responses[i] = *toPayableResponse(&p)
	}

	return responses, total, nil
}

// PayableSummary represents a summary of payables
type PayableSummary struct {
	TotalOutstanding decimal.Decimal `json:"total_outstanding"`
	TotalOverdue     decimal.Decimal `json:"total_overdue"`
	PendingCount     int64           `json:"pending_count"`
	PartialCount     int64           `json:"partial_count"`
	OverdueCount     int64           `json:"overdue_count"`
}

func (s *FinanceService) GetPayableSummary(ctx context.Context, tenantID uuid.UUID) (*PayableSummary, error) {
	totalOutstanding, err := s.payableRepo.SumOutstandingForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	totalOverdue, err := s.payableRepo.SumOverdueForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	pendingCount, err := s.payableRepo.CountByStatus(ctx, tenantID, finance.PayableStatusPending)
	if err != nil {
		return nil, err
	}

	partialCount, err := s.payableRepo.CountByStatus(ctx, tenantID, finance.PayableStatusPartial)
	if err != nil {
		return nil, err
	}

	overdueCount, err := s.payableRepo.CountOverdue(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &PayableSummary{
		TotalOutstanding: totalOutstanding,
		TotalOverdue:     totalOverdue,
		PendingCount:     pendingCount,
		PartialCount:     partialCount,
		OverdueCount:     overdueCount,
	}, nil
}

// ===================== Receipt Voucher Operations =====================

// ReceiptVoucherResponse represents a receipt voucher in API responses
type ReceiptVoucherResponse struct {
	ID                uuid.UUID                      `json:"id"`
	TenantID          uuid.UUID                      `json:"tenant_id"`
	VoucherNumber     string                         `json:"voucher_number"`
	CustomerID        uuid.UUID                      `json:"customer_id"`
	CustomerName      string                         `json:"customer_name"`
	Amount            decimal.Decimal                `json:"amount"`
	AllocatedAmount   decimal.Decimal                `json:"allocated_amount"`
	UnallocatedAmount decimal.Decimal                `json:"unallocated_amount"`
	PaymentMethod     string                         `json:"payment_method"`
	PaymentReference  string                         `json:"payment_reference,omitempty"`
	Status            string                         `json:"status"`
	ReceiptDate       time.Time                      `json:"receipt_date"`
	Allocations       []ReceivableAllocationResponse `json:"allocations,omitempty"`
	Remark            string                         `json:"remark,omitempty"`
	ConfirmedAt       *time.Time                     `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	UpdatedAt         time.Time                      `json:"updated_at"`
	Version           int                            `json:"version"`
}

// ReceivableAllocationResponse represents a receivable allocation in API responses
type ReceivableAllocationResponse struct {
	ID               uuid.UUID       `json:"id"`
	ReceivableID     uuid.UUID       `json:"receivable_id"`
	ReceivableNumber string          `json:"receivable_number"`
	Amount           decimal.Decimal `json:"amount"`
	AllocatedAt      time.Time       `json:"allocated_at"`
	Remark           string          `json:"remark,omitempty"`
}

// CreateReceiptVoucherRequest represents a request to create a receipt voucher
type CreateReceiptVoucherRequest struct {
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	Amount           decimal.Decimal `json:"amount"`
	PaymentMethod    string          `json:"payment_method"`
	PaymentReference string          `json:"payment_reference"`
	ReceiptDate      time.Time       `json:"receipt_date"`
	Remark           string          `json:"remark"`
	CreatedBy        *uuid.UUID      `json:"-"` // Set from JWT context, not from request body
}

// ReceiptVoucherListFilter defines filtering options for receipt voucher list queries
type ReceiptVoucherListFilter struct {
	Search        string     `form:"search"`
	CustomerID    *uuid.UUID `form:"customer_id"`
	Status        string     `form:"status"`
	PaymentMethod string     `form:"payment_method"`
	FromDate      *time.Time `form:"from_date"`
	ToDate        *time.Time `form:"to_date"`
	Page          int        `form:"page"`
	PageSize      int        `form:"page_size"`
}

// CreateReceiptVoucher creates a new receipt voucher
func (s *FinanceService) CreateReceiptVoucher(ctx context.Context, tenantID uuid.UUID, req CreateReceiptVoucherRequest) (*ReceiptVoucherResponse, error) {
	voucherNumber, err := s.receiptVoucherRepo.GenerateVoucherNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	amount := valueobject.NewMoneyCNY(req.Amount)
	paymentMethod := finance.PaymentMethod(req.PaymentMethod)

	voucher, err := finance.NewReceiptVoucher(
		tenantID,
		voucherNumber,
		req.CustomerID,
		req.CustomerName,
		amount,
		paymentMethod,
		req.ReceiptDate,
	)
	if err != nil {
		return nil, err
	}

	if req.PaymentReference != "" {
		if err := voucher.SetPaymentReference(req.PaymentReference); err != nil {
			return nil, err
		}
	}
	if req.Remark != "" {
		if err := voucher.SetRemark(req.Remark); err != nil {
			return nil, err
		}
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		voucher.SetCreatedBy(*req.CreatedBy)
	}

	if err := s.receiptVoucherRepo.Save(ctx, voucher); err != nil {
		return nil, err
	}

	return toReceiptVoucherResponse(voucher), nil
}

// GetReceiptVoucherByID gets a receipt voucher by ID
func (s *FinanceService) GetReceiptVoucherByID(ctx context.Context, tenantID, id uuid.UUID) (*ReceiptVoucherResponse, error) {
	voucher, err := s.receiptVoucherRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Receipt voucher not found")
	}
	return toReceiptVoucherResponse(voucher), nil
}

// ListReceiptVouchers lists receipt vouchers with filtering
func (s *FinanceService) ListReceiptVouchers(ctx context.Context, tenantID uuid.UUID, filter ReceiptVoucherListFilter) ([]ReceiptVoucherResponse, int64, error) {
	domainFilter := finance.ReceiptVoucherFilter{
		CustomerID: filter.CustomerID,
		FromDate:   filter.FromDate,
		ToDate:     filter.ToDate,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Status != "" {
		status := finance.VoucherStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.PaymentMethod != "" {
		method := finance.PaymentMethod(filter.PaymentMethod)
		domainFilter.PaymentMethod = &method
	}

	vouchers, err := s.receiptVoucherRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.receiptVoucherRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]ReceiptVoucherResponse, len(vouchers))
	for i, v := range vouchers {
		responses[i] = *toReceiptVoucherResponse(&v)
	}

	return responses, total, nil
}

// ConfirmReceiptVoucher confirms a receipt voucher
func (s *FinanceService) ConfirmReceiptVoucher(ctx context.Context, tenantID, voucherID, userID uuid.UUID) (*ReceiptVoucherResponse, error) {
	voucher, err := s.receiptVoucherRepo.FindByIDForTenant(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Receipt voucher not found")
	}

	if err := voucher.Confirm(userID); err != nil {
		return nil, err
	}

	if err := s.receiptVoucherRepo.SaveWithLock(ctx, voucher); err != nil {
		return nil, err
	}

	return toReceiptVoucherResponse(voucher), nil
}

// CancelReceiptVoucher cancels a receipt voucher
func (s *FinanceService) CancelReceiptVoucher(ctx context.Context, tenantID, voucherID, userID uuid.UUID, reason string) (*ReceiptVoucherResponse, error) {
	voucher, err := s.receiptVoucherRepo.FindByIDForTenant(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Receipt voucher not found")
	}

	if err := voucher.Cancel(userID, reason); err != nil {
		return nil, err
	}

	if err := s.receiptVoucherRepo.SaveWithLock(ctx, voucher); err != nil {
		return nil, err
	}

	return toReceiptVoucherResponse(voucher), nil
}

// ===================== Payment Voucher Operations =====================

// PaymentVoucherResponse represents a payment voucher in API responses
type PaymentVoucherResponse struct {
	ID                uuid.UUID                   `json:"id"`
	TenantID          uuid.UUID                   `json:"tenant_id"`
	VoucherNumber     string                      `json:"voucher_number"`
	SupplierID        uuid.UUID                   `json:"supplier_id"`
	SupplierName      string                      `json:"supplier_name"`
	Amount            decimal.Decimal             `json:"amount"`
	AllocatedAmount   decimal.Decimal             `json:"allocated_amount"`
	UnallocatedAmount decimal.Decimal             `json:"unallocated_amount"`
	PaymentMethod     string                      `json:"payment_method"`
	PaymentReference  string                      `json:"payment_reference,omitempty"`
	Status            string                      `json:"status"`
	PaymentDate       time.Time                   `json:"payment_date"`
	Allocations       []PayableAllocationResponse `json:"allocations,omitempty"`
	Remark            string                      `json:"remark,omitempty"`
	ConfirmedAt       *time.Time                  `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	Version           int                         `json:"version"`
}

// PayableAllocationResponse represents a payable allocation in API responses
type PayableAllocationResponse struct {
	ID            uuid.UUID       `json:"id"`
	PayableID     uuid.UUID       `json:"payable_id"`
	PayableNumber string          `json:"payable_number"`
	Amount        decimal.Decimal `json:"amount"`
	AllocatedAt   time.Time       `json:"allocated_at"`
	Remark        string          `json:"remark,omitempty"`
}

// CreatePaymentVoucherRequest represents a request to create a payment voucher
type CreatePaymentVoucherRequest struct {
	SupplierID       uuid.UUID       `json:"supplier_id"`
	SupplierName     string          `json:"supplier_name"`
	Amount           decimal.Decimal `json:"amount"`
	PaymentMethod    string          `json:"payment_method"`
	PaymentReference string          `json:"payment_reference"`
	PaymentDate      time.Time       `json:"payment_date"`
	Remark           string          `json:"remark"`
	CreatedBy        *uuid.UUID      `json:"-"` // Set from JWT context, not from request body
}

// PaymentVoucherListFilter defines filtering options for payment voucher list queries
type PaymentVoucherListFilter struct {
	Search        string     `form:"search"`
	SupplierID    *uuid.UUID `form:"supplier_id"`
	Status        string     `form:"status"`
	PaymentMethod string     `form:"payment_method"`
	FromDate      *time.Time `form:"from_date"`
	ToDate        *time.Time `form:"to_date"`
	Page          int        `form:"page"`
	PageSize      int        `form:"page_size"`
}

// CreatePaymentVoucher creates a new payment voucher
func (s *FinanceService) CreatePaymentVoucher(ctx context.Context, tenantID uuid.UUID, req CreatePaymentVoucherRequest) (*PaymentVoucherResponse, error) {
	voucherNumber, err := s.paymentVoucherRepo.GenerateVoucherNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	amount := valueobject.NewMoneyCNY(req.Amount)
	paymentMethod := finance.PaymentMethod(req.PaymentMethod)

	voucher, err := finance.NewPaymentVoucher(
		tenantID,
		voucherNumber,
		req.SupplierID,
		req.SupplierName,
		amount,
		paymentMethod,
		req.PaymentDate,
	)
	if err != nil {
		return nil, err
	}

	if req.PaymentReference != "" {
		if err := voucher.SetPaymentReference(req.PaymentReference); err != nil {
			return nil, err
		}
	}
	if req.Remark != "" {
		if err := voucher.SetRemark(req.Remark); err != nil {
			return nil, err
		}
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		voucher.SetCreatedBy(*req.CreatedBy)
	}

	if err := s.paymentVoucherRepo.Save(ctx, voucher); err != nil {
		return nil, err
	}

	return toPaymentVoucherResponse(voucher), nil
}

// GetPaymentVoucherByID gets a payment voucher by ID
func (s *FinanceService) GetPaymentVoucherByID(ctx context.Context, tenantID, id uuid.UUID) (*PaymentVoucherResponse, error) {
	voucher, err := s.paymentVoucherRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Payment voucher not found")
	}
	return toPaymentVoucherResponse(voucher), nil
}

// ListPaymentVouchers lists payment vouchers with filtering
func (s *FinanceService) ListPaymentVouchers(ctx context.Context, tenantID uuid.UUID, filter PaymentVoucherListFilter) ([]PaymentVoucherResponse, int64, error) {
	domainFilter := finance.PaymentVoucherFilter{
		SupplierID: filter.SupplierID,
		FromDate:   filter.FromDate,
		ToDate:     filter.ToDate,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Status != "" {
		status := finance.VoucherStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.PaymentMethod != "" {
		method := finance.PaymentMethod(filter.PaymentMethod)
		domainFilter.PaymentMethod = &method
	}

	vouchers, err := s.paymentVoucherRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.paymentVoucherRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]PaymentVoucherResponse, len(vouchers))
	for i, v := range vouchers {
		responses[i] = *toPaymentVoucherResponse(&v)
	}

	return responses, total, nil
}

// ConfirmPaymentVoucher confirms a payment voucher
func (s *FinanceService) ConfirmPaymentVoucher(ctx context.Context, tenantID, voucherID, userID uuid.UUID) (*PaymentVoucherResponse, error) {
	voucher, err := s.paymentVoucherRepo.FindByIDForTenant(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Payment voucher not found")
	}

	if err := voucher.Confirm(userID); err != nil {
		return nil, err
	}

	if err := s.paymentVoucherRepo.SaveWithLock(ctx, voucher); err != nil {
		return nil, err
	}

	return toPaymentVoucherResponse(voucher), nil
}

// CancelPaymentVoucher cancels a payment voucher
func (s *FinanceService) CancelPaymentVoucher(ctx context.Context, tenantID, voucherID, userID uuid.UUID, reason string) (*PaymentVoucherResponse, error) {
	voucher, err := s.paymentVoucherRepo.FindByIDForTenant(ctx, tenantID, voucherID)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Payment voucher not found")
	}

	if err := voucher.Cancel(userID, reason); err != nil {
		return nil, err
	}

	if err := s.paymentVoucherRepo.SaveWithLock(ctx, voucher); err != nil {
		return nil, err
	}

	return toPaymentVoucherResponse(voucher), nil
}

// ===================== Reconciliation Operations =====================

// ReconcileReceiptRequest represents a request to reconcile a receipt voucher
type ReconcileReceiptRequest struct {
	VoucherID         uuid.UUID                 `json:"voucher_id"`
	StrategyType      string                    `json:"strategy_type"` // FIFO or MANUAL
	ManualAllocations []ManualAllocationRequest `json:"manual_allocations,omitempty"`
}

// ManualAllocationRequest represents a manual allocation request
type ManualAllocationRequest struct {
	TargetID uuid.UUID       `json:"target_id"` // Receivable or Payable ID
	Amount   decimal.Decimal `json:"amount"`
}

// ReconcileReceiptResult represents the result of reconciling a receipt voucher
type ReconcileReceiptResult struct {
	Voucher              *ReceiptVoucherResponse     `json:"voucher"`
	UpdatedReceivables   []AccountReceivableResponse `json:"updated_receivables"`
	TotalReconciled      decimal.Decimal             `json:"total_reconciled"`
	RemainingUnallocated decimal.Decimal             `json:"remaining_unallocated"`
	FullyReconciled      bool                        `json:"fully_reconciled"`
}

// ReconcileReceipt reconciles a receipt voucher to outstanding receivables
func (s *FinanceService) ReconcileReceipt(ctx context.Context, tenantID uuid.UUID, req ReconcileReceiptRequest) (*ReconcileReceiptResult, error) {
	// Wrap in profiling labels for performance analysis
	var response *ReconcileReceiptResult
	var operationErr error
	telemetry.WithProfilingLabels(ctx, telemetry.FinanceOperationLabels(telemetry.OperationReconcile, ""), func(c context.Context) {
		voucher, err := s.receiptVoucherRepo.FindByIDForTenant(c, tenantID, req.VoucherID)
		if err != nil {
			operationErr = err
			return
		}
		if voucher == nil {
			operationErr = shared.NewDomainError("NOT_FOUND", "Receipt voucher not found")
			return
		}

		// Get outstanding receivables for the customer
		receivables, err := s.receivableRepo.FindOutstanding(c, tenantID, voucher.CustomerID)
		if err != nil {
			operationErr = err
			return
		}

		strategyType := finance.ReconciliationStrategyType(req.StrategyType)
		// If no strategy specified or invalid, use the effective strategy from the service
		if !strategyType.IsValid() {
			strategyType = s.reconciliationSvc.GetEffectiveStrategy(c, tenantID)
		}

		// Convert manual allocations if provided
		var manualAllocs []finance.ManualAllocationRequest
		for _, ma := range req.ManualAllocations {
			manualAllocs = append(manualAllocs, finance.ManualAllocationRequest{
				TargetID: ma.TargetID,
				Amount:   ma.Amount,
			})
		}

		result, err := s.reconciliationSvc.ReconcileReceipt(c, finance.ReconcileReceiptRequest{
			ReceiptVoucher:    voucher,
			Receivables:       receivables,
			StrategyType:      strategyType,
			ManualAllocations: manualAllocs,
		})
		if err != nil {
			operationErr = err
			return
		}

		// Save updated voucher
		if err := s.receiptVoucherRepo.SaveWithLock(c, result.ReceiptVoucher); err != nil {
			operationErr = err
			return
		}

		// Save updated receivables
		for i := range result.UpdatedReceivables {
			if err := s.receivableRepo.SaveWithLock(c, &result.UpdatedReceivables[i]); err != nil {
				operationErr = err
				return
			}
		}

		// Convert to response
		updatedReceivables := make([]AccountReceivableResponse, len(result.UpdatedReceivables))
		for i, r := range result.UpdatedReceivables {
			updatedReceivables[i] = *toReceivableResponse(&r)
		}

		response = &ReconcileReceiptResult{
			Voucher:              toReceiptVoucherResponse(result.ReceiptVoucher),
			UpdatedReceivables:   updatedReceivables,
			TotalReconciled:      result.TotalReconciled,
			RemainingUnallocated: result.RemainingUnallocated,
			FullyReconciled:      result.FullyReconciled,
		}
	})

	return response, operationErr
}

// ReconcilePaymentRequest represents a request to reconcile a payment voucher
type ReconcilePaymentRequest struct {
	VoucherID         uuid.UUID                 `json:"voucher_id"`
	StrategyType      string                    `json:"strategy_type"` // FIFO or MANUAL
	ManualAllocations []ManualAllocationRequest `json:"manual_allocations,omitempty"`
}

// ReconcilePaymentResult represents the result of reconciling a payment voucher
type ReconcilePaymentResult struct {
	Voucher              *PaymentVoucherResponse  `json:"voucher"`
	UpdatedPayables      []AccountPayableResponse `json:"updated_payables"`
	TotalReconciled      decimal.Decimal          `json:"total_reconciled"`
	RemainingUnallocated decimal.Decimal          `json:"remaining_unallocated"`
	FullyReconciled      bool                     `json:"fully_reconciled"`
}

// ReconcilePayment reconciles a payment voucher to outstanding payables
func (s *FinanceService) ReconcilePayment(ctx context.Context, tenantID uuid.UUID, req ReconcilePaymentRequest) (*ReconcilePaymentResult, error) {
	// Wrap in profiling labels for performance analysis
	var response *ReconcilePaymentResult
	var operationErr error
	telemetry.WithProfilingLabels(ctx, telemetry.FinanceOperationLabels(telemetry.OperationReconcile, ""), func(c context.Context) {
		voucher, err := s.paymentVoucherRepo.FindByIDForTenant(c, tenantID, req.VoucherID)
		if err != nil {
			operationErr = err
			return
		}
		if voucher == nil {
			operationErr = shared.NewDomainError("NOT_FOUND", "Payment voucher not found")
			return
		}

		// Get outstanding payables for the supplier
		payables, err := s.payableRepo.FindOutstanding(c, tenantID, voucher.SupplierID)
		if err != nil {
			operationErr = err
			return
		}

		strategyType := finance.ReconciliationStrategyType(req.StrategyType)
		// If no strategy specified or invalid, use the effective strategy from the service
		if !strategyType.IsValid() {
			strategyType = s.reconciliationSvc.GetEffectiveStrategy(c, tenantID)
		}

		// Convert manual allocations if provided
		var manualAllocs []finance.ManualAllocationRequest
		for _, ma := range req.ManualAllocations {
			manualAllocs = append(manualAllocs, finance.ManualAllocationRequest{
				TargetID: ma.TargetID,
				Amount:   ma.Amount,
			})
		}

		result, err := s.reconciliationSvc.ReconcilePayment(c, finance.ReconcilePaymentRequest{
			PaymentVoucher:    voucher,
			Payables:          payables,
			StrategyType:      strategyType,
			ManualAllocations: manualAllocs,
		})
		if err != nil {
			operationErr = err
			return
		}

		// Save updated voucher
		if err := s.paymentVoucherRepo.SaveWithLock(c, result.PaymentVoucher); err != nil {
			operationErr = err
			return
		}

		// Save updated payables
		for i := range result.UpdatedPayables {
			if err := s.payableRepo.SaveWithLock(c, &result.UpdatedPayables[i]); err != nil {
				operationErr = err
				return
			}
		}

		// Convert to response
		updatedPayables := make([]AccountPayableResponse, len(result.UpdatedPayables))
		for i, p := range result.UpdatedPayables {
			updatedPayables[i] = *toPayableResponse(&p)
		}

		response = &ReconcilePaymentResult{
			Voucher:              toPaymentVoucherResponse(result.PaymentVoucher),
			UpdatedPayables:      updatedPayables,
			TotalReconciled:      result.TotalReconciled,
			RemainingUnallocated: result.RemainingUnallocated,
			FullyReconciled:      result.FullyReconciled,
		}
	})

	return response, operationErr
}

// ===================== Helper Functions =====================

func toReceivableResponse(r *finance.AccountReceivable) *AccountReceivableResponse {
	paymentRecords := make([]PaymentRecordResponse, len(r.PaymentRecords))
	for i, pr := range r.PaymentRecords {
		paymentRecords[i] = PaymentRecordResponse{
			ID:               pr.ID,
			ReceiptVoucherID: pr.ReceiptVoucherID,
			Amount:           pr.Amount,
			AppliedAt:        pr.AppliedAt,
			Remark:           pr.Remark,
		}
	}

	return &AccountReceivableResponse{
		ID:                r.ID,
		TenantID:          r.TenantID,
		ReceivableNumber:  r.ReceivableNumber,
		CustomerID:        r.CustomerID,
		CustomerName:      r.CustomerName,
		SourceType:        string(r.SourceType),
		SourceID:          r.SourceID,
		SourceNumber:      r.SourceNumber,
		TotalAmount:       r.TotalAmount,
		PaidAmount:        r.PaidAmount,
		OutstandingAmount: r.OutstandingAmount,
		Status:            string(r.Status),
		DueDate:           r.DueDate,
		PaymentRecords:    paymentRecords,
		Remark:            r.Remark,
		PaidAt:            r.PaidAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		Version:           r.Version,
	}
}

func toPayableResponse(p *finance.AccountPayable) *AccountPayableResponse {
	paymentRecords := make([]PayablePaymentRecordResponse, len(p.PaymentRecords))
	for i, pr := range p.PaymentRecords {
		paymentRecords[i] = PayablePaymentRecordResponse{
			ID:               pr.ID,
			PaymentVoucherID: pr.PaymentVoucherID,
			Amount:           pr.Amount,
			AppliedAt:        pr.AppliedAt,
			Remark:           pr.Remark,
		}
	}

	return &AccountPayableResponse{
		ID:                p.ID,
		TenantID:          p.TenantID,
		PayableNumber:     p.PayableNumber,
		SupplierID:        p.SupplierID,
		SupplierName:      p.SupplierName,
		SourceType:        string(p.SourceType),
		SourceID:          p.SourceID,
		SourceNumber:      p.SourceNumber,
		TotalAmount:       p.TotalAmount,
		PaidAmount:        p.PaidAmount,
		OutstandingAmount: p.OutstandingAmount,
		Status:            string(p.Status),
		DueDate:           p.DueDate,
		PaymentRecords:    paymentRecords,
		Remark:            p.Remark,
		PaidAt:            p.PaidAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		Version:           p.Version,
	}
}

func toReceiptVoucherResponse(v *finance.ReceiptVoucher) *ReceiptVoucherResponse {
	allocations := make([]ReceivableAllocationResponse, len(v.Allocations))
	for i, a := range v.Allocations {
		allocations[i] = ReceivableAllocationResponse{
			ID:               a.ID,
			ReceivableID:     a.ReceivableID,
			ReceivableNumber: a.ReceivableNumber,
			Amount:           a.Amount,
			AllocatedAt:      a.AllocatedAt,
			Remark:           a.Remark,
		}
	}

	return &ReceiptVoucherResponse{
		ID:                v.ID,
		TenantID:          v.TenantID,
		VoucherNumber:     v.VoucherNumber,
		CustomerID:        v.CustomerID,
		CustomerName:      v.CustomerName,
		Amount:            v.Amount,
		AllocatedAmount:   v.AllocatedAmount,
		UnallocatedAmount: v.UnallocatedAmount,
		PaymentMethod:     string(v.PaymentMethod),
		PaymentReference:  v.PaymentReference,
		Status:            string(v.Status),
		ReceiptDate:       v.ReceiptDate,
		Allocations:       allocations,
		Remark:            v.Remark,
		ConfirmedAt:       v.ConfirmedAt,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
		Version:           v.Version,
	}
}

func toPaymentVoucherResponse(v *finance.PaymentVoucher) *PaymentVoucherResponse {
	allocations := make([]PayableAllocationResponse, len(v.Allocations))
	for i, a := range v.Allocations {
		allocations[i] = PayableAllocationResponse{
			ID:            a.ID,
			PayableID:     a.PayableID,
			PayableNumber: a.PayableNumber,
			Amount:        a.Amount,
			AllocatedAt:   a.AllocatedAt,
			Remark:        a.Remark,
		}
	}

	return &PaymentVoucherResponse{
		ID:                v.ID,
		TenantID:          v.TenantID,
		VoucherNumber:     v.VoucherNumber,
		SupplierID:        v.SupplierID,
		SupplierName:      v.SupplierName,
		Amount:            v.Amount,
		AllocatedAmount:   v.AllocatedAmount,
		UnallocatedAmount: v.UnallocatedAmount,
		PaymentMethod:     string(v.PaymentMethod),
		PaymentReference:  v.PaymentReference,
		Status:            string(v.Status),
		PaymentDate:       v.PaymentDate,
		Allocations:       allocations,
		Remark:            v.Remark,
		ConfirmedAt:       v.ConfirmedAt,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
		Version:           v.Version,
	}
}
