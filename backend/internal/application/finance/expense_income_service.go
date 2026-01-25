package finance

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseIncomeService provides application-level expense and income operations
type ExpenseIncomeService struct {
	expenseRepo finance.ExpenseRecordRepository
	incomeRepo  finance.OtherIncomeRecordRepository
}

// NewExpenseIncomeService creates a new ExpenseIncomeService
func NewExpenseIncomeService(
	expenseRepo finance.ExpenseRecordRepository,
	incomeRepo finance.OtherIncomeRecordRepository,
) *ExpenseIncomeService {
	return &ExpenseIncomeService{
		expenseRepo: expenseRepo,
		incomeRepo:  incomeRepo,
	}
}

// ===================== Expense Record Operations =====================

// ExpenseRecordResponse represents an expense record in API responses
type ExpenseRecordResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	ExpenseNumber   string          `json:"expense_number"`
	Category        string          `json:"category"`
	CategoryName    string          `json:"category_name"`
	Amount          decimal.Decimal `json:"amount"`
	Description     string          `json:"description"`
	IncurredAt      time.Time       `json:"incurred_at"`
	Status          string          `json:"status"`
	PaymentStatus   string          `json:"payment_status"`
	PaymentMethod   *string         `json:"payment_method,omitempty"`
	PaidAt          *time.Time      `json:"paid_at,omitempty"`
	Remark          string          `json:"remark,omitempty"`
	AttachmentURLs  string          `json:"attachment_urls,omitempty"`
	SubmittedAt     *time.Time      `json:"submitted_at,omitempty"`
	SubmittedBy     *uuid.UUID      `json:"submitted_by,omitempty"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty"`
	ApprovedBy      *uuid.UUID      `json:"approved_by,omitempty"`
	ApprovalRemark  string          `json:"approval_remark,omitempty"`
	RejectedAt      *time.Time      `json:"rejected_at,omitempty"`
	RejectedBy      *uuid.UUID      `json:"rejected_by,omitempty"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Version         int             `json:"version"`
}

// CreateExpenseRecordRequest represents a request to create an expense record
type CreateExpenseRecordRequest struct {
	Category       string          `json:"category" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	Description    string          `json:"description" binding:"required"`
	IncurredAt     time.Time       `json:"incurred_at" binding:"required"`
	Remark         string          `json:"remark"`
	AttachmentURLs string          `json:"attachment_urls"`
	CreatedBy      *uuid.UUID      `json:"-"` // Set from JWT context, not from request body
}

// UpdateExpenseRecordRequest represents a request to update an expense record
type UpdateExpenseRecordRequest struct {
	Category       string          `json:"category" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	Description    string          `json:"description" binding:"required"`
	IncurredAt     time.Time       `json:"incurred_at" binding:"required"`
	Remark         string          `json:"remark"`
	AttachmentURLs string          `json:"attachment_urls"`
}

// ExpenseRecordListFilter defines filtering options for expense record list queries
type ExpenseRecordListFilter struct {
	Search        string     `form:"search"`
	Category      string     `form:"category"`
	Status        string     `form:"status"`
	PaymentStatus string     `form:"payment_status"`
	FromDate      *time.Time `form:"from_date"`
	ToDate        *time.Time `form:"to_date"`
	Page          int        `form:"page"`
	PageSize      int        `form:"page_size"`
}

// CreateExpenseRecord creates a new expense record
func (s *ExpenseIncomeService) CreateExpenseRecord(ctx context.Context, tenantID uuid.UUID, req CreateExpenseRecordRequest) (*ExpenseRecordResponse, error) {
	expenseNumber, err := s.expenseRepo.GenerateExpenseNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	category := finance.ExpenseCategory(req.Category)
	amount := valueobject.NewMoneyCNY(req.Amount)

	expense, err := finance.NewExpenseRecord(
		tenantID,
		expenseNumber,
		category,
		amount,
		req.Description,
		req.IncurredAt,
	)
	if err != nil {
		return nil, err
	}

	if req.Remark != "" {
		expense.SetRemark(req.Remark)
	}
	if req.AttachmentURLs != "" {
		expense.SetAttachmentURLs(req.AttachmentURLs)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		expense.SetCreatedBy(*req.CreatedBy)
	}

	if err := s.expenseRepo.Save(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// GetExpenseRecordByID gets an expense record by ID
func (s *ExpenseIncomeService) GetExpenseRecordByID(ctx context.Context, tenantID, id uuid.UUID) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}
	return toExpenseRecordResponse(expense), nil
}

// UpdateExpenseRecord updates an expense record (only draft status)
func (s *ExpenseIncomeService) UpdateExpenseRecord(ctx context.Context, tenantID, id uuid.UUID, req UpdateExpenseRecordRequest) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	category := finance.ExpenseCategory(req.Category)
	amount := valueobject.NewMoneyCNY(req.Amount)

	if err := expense.Update(category, amount, req.Description, req.IncurredAt); err != nil {
		return nil, err
	}

	expense.SetRemark(req.Remark)
	expense.SetAttachmentURLs(req.AttachmentURLs)

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// ListExpenseRecords lists expense records with filtering
func (s *ExpenseIncomeService) ListExpenseRecords(ctx context.Context, tenantID uuid.UUID, filter ExpenseRecordListFilter) ([]ExpenseRecordResponse, int64, error) {
	domainFilter := finance.ExpenseRecordFilter{
		FromDate: filter.FromDate,
		ToDate:   filter.ToDate,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Category != "" {
		category := finance.ExpenseCategory(filter.Category)
		domainFilter.Category = &category
	}
	if filter.Status != "" {
		status := finance.ExpenseStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.PaymentStatus != "" {
		paymentStatus := finance.PaymentStatus(filter.PaymentStatus)
		domainFilter.PaymentStatus = &paymentStatus
	}

	expenses, err := s.expenseRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.expenseRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]ExpenseRecordResponse, len(expenses))
	for i, e := range expenses {
		responses[i] = *toExpenseRecordResponse(&e)
	}

	return responses, total, nil
}

// SubmitExpenseRecord submits an expense record for approval
func (s *ExpenseIncomeService) SubmitExpenseRecord(ctx context.Context, tenantID, expenseID, userID uuid.UUID) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	if err := expense.Submit(userID); err != nil {
		return nil, err
	}

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// ApproveExpenseRecord approves an expense record
func (s *ExpenseIncomeService) ApproveExpenseRecord(ctx context.Context, tenantID, expenseID, userID uuid.UUID, remark string) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	if err := expense.Approve(userID, remark); err != nil {
		return nil, err
	}

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// RejectExpenseRecord rejects an expense record
func (s *ExpenseIncomeService) RejectExpenseRecord(ctx context.Context, tenantID, expenseID, userID uuid.UUID, reason string) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	if err := expense.Reject(userID, reason); err != nil {
		return nil, err
	}

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// CancelExpenseRecord cancels an expense record
func (s *ExpenseIncomeService) CancelExpenseRecord(ctx context.Context, tenantID, expenseID, userID uuid.UUID, reason string) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	if err := expense.Cancel(userID, reason); err != nil {
		return nil, err
	}

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// MarkExpenseAsPaid marks an expense as paid
func (s *ExpenseIncomeService) MarkExpenseAsPaid(ctx context.Context, tenantID, expenseID uuid.UUID, paymentMethod string) (*ExpenseRecordResponse, error) {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	method := finance.PaymentMethod(paymentMethod)
	if err := expense.MarkAsPaid(method); err != nil {
		return nil, err
	}

	if err := s.expenseRepo.SaveWithLock(ctx, expense); err != nil {
		return nil, err
	}

	return toExpenseRecordResponse(expense), nil
}

// DeleteExpenseRecord deletes an expense record (soft delete)
func (s *ExpenseIncomeService) DeleteExpenseRecord(ctx context.Context, tenantID, expenseID uuid.UUID) error {
	expense, err := s.expenseRepo.FindByIDForTenant(ctx, tenantID, expenseID)
	if err != nil {
		return err
	}
	if expense == nil {
		return shared.NewDomainError("NOT_FOUND", "Expense record not found")
	}

	// Only allow deletion of draft expenses
	if !expense.IsDraft() {
		return shared.NewDomainError("INVALID_STATE", "Can only delete expense in draft status")
	}

	return s.expenseRepo.DeleteForTenant(ctx, tenantID, expenseID)
}

// ExpenseSummary represents a summary of expenses
type ExpenseSummary struct {
	TotalApproved decimal.Decimal            `json:"total_approved"`
	TotalPending  int64                      `json:"total_pending"`
	ByCategory    map[string]decimal.Decimal `json:"by_category"`
}

// GetExpenseSummary gets a summary of expenses for a date range
func (s *ExpenseIncomeService) GetExpenseSummary(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*ExpenseSummary, error) {
	totalApproved, err := s.expenseRepo.SumApprovedForTenant(ctx, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	pendingStatus := finance.ExpenseStatusPending
	totalPending, err := s.expenseRepo.CountByStatus(ctx, tenantID, pendingStatus)
	if err != nil {
		return nil, err
	}

	// Get sum by category
	byCategory := make(map[string]decimal.Decimal)
	categories := []finance.ExpenseCategory{
		finance.ExpenseCategoryRent,
		finance.ExpenseCategoryUtilities,
		finance.ExpenseCategorySalary,
		finance.ExpenseCategoryOffice,
		finance.ExpenseCategoryTravel,
		finance.ExpenseCategoryMarketing,
		finance.ExpenseCategoryEquipment,
		finance.ExpenseCategoryMaintenance,
		finance.ExpenseCategoryInsurance,
		finance.ExpenseCategoryTax,
		finance.ExpenseCategoryOther,
	}
	for _, cat := range categories {
		sum, err := s.expenseRepo.SumByCategory(ctx, tenantID, cat, from, to)
		if err != nil {
			return nil, err
		}
		if !sum.IsZero() {
			byCategory[string(cat)] = sum
		}
	}

	return &ExpenseSummary{
		TotalApproved: totalApproved,
		TotalPending:  totalPending,
		ByCategory:    byCategory,
	}, nil
}

// ===================== Other Income Record Operations =====================

// OtherIncomeRecordResponse represents an other income record in API responses
type OtherIncomeRecordResponse struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	IncomeNumber   string          `json:"income_number"`
	Category       string          `json:"category"`
	CategoryName   string          `json:"category_name"`
	Amount         decimal.Decimal `json:"amount"`
	Description    string          `json:"description"`
	ReceivedAt     time.Time       `json:"received_at"`
	Status         string          `json:"status"`
	ReceiptStatus  string          `json:"receipt_status"`
	PaymentMethod  *string         `json:"payment_method,omitempty"`
	ActualReceived *time.Time      `json:"actual_received,omitempty"`
	Remark         string          `json:"remark,omitempty"`
	AttachmentURLs string          `json:"attachment_urls,omitempty"`
	ConfirmedAt    *time.Time      `json:"confirmed_at,omitempty"`
	ConfirmedBy    *uuid.UUID      `json:"confirmed_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Version        int             `json:"version"`
}

// CreateOtherIncomeRecordRequest represents a request to create an other income record
type CreateOtherIncomeRecordRequest struct {
	Category       string          `json:"category" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	Description    string          `json:"description" binding:"required"`
	ReceivedAt     time.Time       `json:"received_at" binding:"required"`
	Remark         string          `json:"remark"`
	AttachmentURLs string          `json:"attachment_urls"`
	CreatedBy      *uuid.UUID      `json:"-"` // Set from JWT context, not from request body
}

// UpdateOtherIncomeRecordRequest represents a request to update an other income record
type UpdateOtherIncomeRecordRequest struct {
	Category       string          `json:"category" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	Description    string          `json:"description" binding:"required"`
	ReceivedAt     time.Time       `json:"received_at" binding:"required"`
	Remark         string          `json:"remark"`
	AttachmentURLs string          `json:"attachment_urls"`
}

// OtherIncomeRecordListFilter defines filtering options for other income record list queries
type OtherIncomeRecordListFilter struct {
	Search        string     `form:"search"`
	Category      string     `form:"category"`
	Status        string     `form:"status"`
	ReceiptStatus string     `form:"receipt_status"`
	FromDate      *time.Time `form:"from_date"`
	ToDate        *time.Time `form:"to_date"`
	Page          int        `form:"page"`
	PageSize      int        `form:"page_size"`
}

// CreateOtherIncomeRecord creates a new other income record
func (s *ExpenseIncomeService) CreateOtherIncomeRecord(ctx context.Context, tenantID uuid.UUID, req CreateOtherIncomeRecordRequest) (*OtherIncomeRecordResponse, error) {
	incomeNumber, err := s.incomeRepo.GenerateIncomeNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	category := finance.IncomeCategory(req.Category)
	amount := valueobject.NewMoneyCNY(req.Amount)

	income, err := finance.NewOtherIncomeRecord(
		tenantID,
		incomeNumber,
		category,
		amount,
		req.Description,
		req.ReceivedAt,
	)
	if err != nil {
		return nil, err
	}

	if req.Remark != "" {
		income.SetRemark(req.Remark)
	}
	if req.AttachmentURLs != "" {
		income.SetAttachmentURLs(req.AttachmentURLs)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		income.SetCreatedBy(*req.CreatedBy)
	}

	if err := s.incomeRepo.Save(ctx, income); err != nil {
		return nil, err
	}

	return toOtherIncomeRecordResponse(income), nil
}

// GetOtherIncomeRecordByID gets an other income record by ID
func (s *ExpenseIncomeService) GetOtherIncomeRecordByID(ctx context.Context, tenantID, id uuid.UUID) (*OtherIncomeRecordResponse, error) {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if income == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}
	return toOtherIncomeRecordResponse(income), nil
}

// UpdateOtherIncomeRecord updates an other income record (only draft status)
func (s *ExpenseIncomeService) UpdateOtherIncomeRecord(ctx context.Context, tenantID, id uuid.UUID, req UpdateOtherIncomeRecordRequest) (*OtherIncomeRecordResponse, error) {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if income == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}

	category := finance.IncomeCategory(req.Category)
	amount := valueobject.NewMoneyCNY(req.Amount)

	if err := income.Update(category, amount, req.Description, req.ReceivedAt); err != nil {
		return nil, err
	}

	income.SetRemark(req.Remark)
	income.SetAttachmentURLs(req.AttachmentURLs)

	if err := s.incomeRepo.SaveWithLock(ctx, income); err != nil {
		return nil, err
	}

	return toOtherIncomeRecordResponse(income), nil
}

// ListOtherIncomeRecords lists other income records with filtering
func (s *ExpenseIncomeService) ListOtherIncomeRecords(ctx context.Context, tenantID uuid.UUID, filter OtherIncomeRecordListFilter) ([]OtherIncomeRecordResponse, int64, error) {
	domainFilter := finance.OtherIncomeRecordFilter{
		FromDate: filter.FromDate,
		ToDate:   filter.ToDate,
	}
	domainFilter.Page = filter.Page
	domainFilter.PageSize = filter.PageSize
	domainFilter.Search = filter.Search

	if filter.Category != "" {
		category := finance.IncomeCategory(filter.Category)
		domainFilter.Category = &category
	}
	if filter.Status != "" {
		status := finance.IncomeStatus(filter.Status)
		domainFilter.Status = &status
	}
	if filter.ReceiptStatus != "" {
		receiptStatus := finance.ReceiptStatus(filter.ReceiptStatus)
		domainFilter.ReceiptStatus = &receiptStatus
	}

	incomes, err := s.incomeRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.incomeRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]OtherIncomeRecordResponse, len(incomes))
	for i, inc := range incomes {
		responses[i] = *toOtherIncomeRecordResponse(&inc)
	}

	return responses, total, nil
}

// ConfirmOtherIncomeRecord confirms an other income record
func (s *ExpenseIncomeService) ConfirmOtherIncomeRecord(ctx context.Context, tenantID, incomeID, userID uuid.UUID) (*OtherIncomeRecordResponse, error) {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, incomeID)
	if err != nil {
		return nil, err
	}
	if income == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}

	if err := income.Confirm(userID); err != nil {
		return nil, err
	}

	if err := s.incomeRepo.SaveWithLock(ctx, income); err != nil {
		return nil, err
	}

	return toOtherIncomeRecordResponse(income), nil
}

// CancelOtherIncomeRecord cancels an other income record
func (s *ExpenseIncomeService) CancelOtherIncomeRecord(ctx context.Context, tenantID, incomeID, userID uuid.UUID, reason string) (*OtherIncomeRecordResponse, error) {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, incomeID)
	if err != nil {
		return nil, err
	}
	if income == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}

	if err := income.Cancel(userID, reason); err != nil {
		return nil, err
	}

	if err := s.incomeRepo.SaveWithLock(ctx, income); err != nil {
		return nil, err
	}

	return toOtherIncomeRecordResponse(income), nil
}

// MarkIncomeAsReceived marks an income as received
func (s *ExpenseIncomeService) MarkIncomeAsReceived(ctx context.Context, tenantID, incomeID uuid.UUID, paymentMethod string) (*OtherIncomeRecordResponse, error) {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, incomeID)
	if err != nil {
		return nil, err
	}
	if income == nil {
		return nil, shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}

	method := finance.PaymentMethod(paymentMethod)
	if err := income.MarkAsReceived(method); err != nil {
		return nil, err
	}

	if err := s.incomeRepo.SaveWithLock(ctx, income); err != nil {
		return nil, err
	}

	return toOtherIncomeRecordResponse(income), nil
}

// DeleteOtherIncomeRecord deletes an other income record (soft delete)
func (s *ExpenseIncomeService) DeleteOtherIncomeRecord(ctx context.Context, tenantID, incomeID uuid.UUID) error {
	income, err := s.incomeRepo.FindByIDForTenant(ctx, tenantID, incomeID)
	if err != nil {
		return err
	}
	if income == nil {
		return shared.NewDomainError("NOT_FOUND", "Other income record not found")
	}

	// Only allow deletion of draft incomes
	if !income.IsDraft() {
		return shared.NewDomainError("INVALID_STATE", "Can only delete income in draft status")
	}

	return s.incomeRepo.DeleteForTenant(ctx, tenantID, incomeID)
}

// IncomeSummary represents a summary of other income
type IncomeSummary struct {
	TotalConfirmed decimal.Decimal            `json:"total_confirmed"`
	TotalDraft     int64                      `json:"total_draft"`
	ByCategory     map[string]decimal.Decimal `json:"by_category"`
}

// GetIncomeSummary gets a summary of other income for a date range
func (s *ExpenseIncomeService) GetIncomeSummary(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*IncomeSummary, error) {
	totalConfirmed, err := s.incomeRepo.SumConfirmedForTenant(ctx, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	draftStatus := finance.IncomeStatusDraft
	totalDraft, err := s.incomeRepo.CountByStatus(ctx, tenantID, draftStatus)
	if err != nil {
		return nil, err
	}

	// Get sum by category
	byCategory := make(map[string]decimal.Decimal)
	categories := []finance.IncomeCategory{
		finance.IncomeCategoryInvestment,
		finance.IncomeCategorySubsidy,
		finance.IncomeCategoryInterest,
		finance.IncomeCategoryRental,
		finance.IncomeCategoryRefund,
		finance.IncomeCategoryCompensation,
		finance.IncomeCategoryAssetDisposal,
		finance.IncomeCategoryOther,
	}
	for _, cat := range categories {
		sum, err := s.incomeRepo.SumByCategory(ctx, tenantID, cat, from, to)
		if err != nil {
			return nil, err
		}
		if !sum.IsZero() {
			byCategory[string(cat)] = sum
		}
	}

	return &IncomeSummary{
		TotalConfirmed: totalConfirmed,
		TotalDraft:     totalDraft,
		ByCategory:     byCategory,
	}, nil
}

// ===================== Cash Flow Operations =====================

// CashFlowItem represents a single cash flow item
type CashFlowItem struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"` // EXPENSE, INCOME, RECEIPT, PAYMENT
	Category    string          `json:"category"`
	Number      string          `json:"number"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	Date        time.Time       `json:"date"`
	Direction   string          `json:"direction"` // INFLOW or OUTFLOW
}

// CashFlowSummary represents a summary of cash flow
type CashFlowSummary struct {
	TotalInflow  decimal.Decimal `json:"total_inflow"`
	TotalOutflow decimal.Decimal `json:"total_outflow"`
	NetCashFlow  decimal.Decimal `json:"net_cash_flow"`
	ExpenseTotal decimal.Decimal `json:"expense_total"`
	IncomeTotal  decimal.Decimal `json:"income_total"`
	Items        []CashFlowItem  `json:"items,omitempty"`
}

// GetCashFlowSummary gets a summary of cash flow for a date range
func (s *ExpenseIncomeService) GetCashFlowSummary(ctx context.Context, tenantID uuid.UUID, from, to time.Time, includeItems bool) (*CashFlowSummary, error) {
	// Get approved expense total
	expenseTotal, err := s.expenseRepo.SumApprovedForTenant(ctx, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	// Get confirmed income total
	incomeTotal, err := s.incomeRepo.SumConfirmedForTenant(ctx, tenantID, from, to)
	if err != nil {
		return nil, err
	}

	// For now, we only have expense and income
	// In a full implementation, we would also include ReceiptVoucher and PaymentVoucher
	totalInflow := incomeTotal
	totalOutflow := expenseTotal
	netCashFlow := totalInflow.Sub(totalOutflow)

	summary := &CashFlowSummary{
		TotalInflow:  totalInflow,
		TotalOutflow: totalOutflow,
		NetCashFlow:  netCashFlow,
		ExpenseTotal: expenseTotal,
		IncomeTotal:  incomeTotal,
	}

	if includeItems {
		var items []CashFlowItem

		// Get approved expenses
		approvedStatus := finance.ExpenseStatusApproved
		expenseFilter := finance.ExpenseRecordFilter{
			Status:   &approvedStatus,
			FromDate: &from,
			ToDate:   &to,
		}
		expenses, err := s.expenseRepo.FindAllForTenant(ctx, tenantID, expenseFilter)
		if err != nil {
			return nil, err
		}
		for _, e := range expenses {
			items = append(items, CashFlowItem{
				ID:          e.ID,
				Type:        "EXPENSE",
				Category:    string(e.Category),
				Number:      e.ExpenseNumber,
				Description: e.Description,
				Amount:      e.Amount,
				Date:        e.IncurredAt,
				Direction:   "OUTFLOW",
			})
		}

		// Get confirmed incomes
		confirmedStatus := finance.IncomeStatusConfirmed
		incomeFilter := finance.OtherIncomeRecordFilter{
			Status:   &confirmedStatus,
			FromDate: &from,
			ToDate:   &to,
		}
		incomes, err := s.incomeRepo.FindAllForTenant(ctx, tenantID, incomeFilter)
		if err != nil {
			return nil, err
		}
		for _, i := range incomes {
			items = append(items, CashFlowItem{
				ID:          i.ID,
				Type:        "INCOME",
				Category:    string(i.Category),
				Number:      i.IncomeNumber,
				Description: i.Description,
				Amount:      i.Amount,
				Date:        i.ReceivedAt,
				Direction:   "INFLOW",
			})
		}

		summary.Items = items
	}

	return summary, nil
}

// ===================== Helper Functions =====================

func toExpenseRecordResponse(e *finance.ExpenseRecord) *ExpenseRecordResponse {
	var paymentMethod *string
	if e.PaymentMethod != nil {
		pm := string(*e.PaymentMethod)
		paymentMethod = &pm
	}

	return &ExpenseRecordResponse{
		ID:              e.ID,
		TenantID:        e.TenantID,
		ExpenseNumber:   e.ExpenseNumber,
		Category:        string(e.Category),
		CategoryName:    e.Category.DisplayName(),
		Amount:          e.Amount,
		Description:     e.Description,
		IncurredAt:      e.IncurredAt,
		Status:          string(e.Status),
		PaymentStatus:   string(e.PaymentStatus),
		PaymentMethod:   paymentMethod,
		PaidAt:          e.PaidAt,
		Remark:          e.Remark,
		AttachmentURLs:  e.AttachmentURLs,
		SubmittedAt:     e.SubmittedAt,
		SubmittedBy:     e.SubmittedBy,
		ApprovedAt:      e.ApprovedAt,
		ApprovedBy:      e.ApprovedBy,
		ApprovalRemark:  e.ApprovalRemark,
		RejectedAt:      e.RejectedAt,
		RejectedBy:      e.RejectedBy,
		RejectionReason: e.RejectionReason,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
		Version:         e.Version,
	}
}

func toOtherIncomeRecordResponse(i *finance.OtherIncomeRecord) *OtherIncomeRecordResponse {
	var paymentMethod *string
	if i.PaymentMethod != nil {
		pm := string(*i.PaymentMethod)
		paymentMethod = &pm
	}

	return &OtherIncomeRecordResponse{
		ID:             i.ID,
		TenantID:       i.TenantID,
		IncomeNumber:   i.IncomeNumber,
		Category:       string(i.Category),
		CategoryName:   i.Category.DisplayName(),
		Amount:         i.Amount,
		Description:    i.Description,
		ReceivedAt:     i.ReceivedAt,
		Status:         string(i.Status),
		ReceiptStatus:  string(i.ReceiptStatus),
		PaymentMethod:  paymentMethod,
		ActualReceived: i.ActualReceived,
		Remark:         i.Remark,
		AttachmentURLs: i.AttachmentURLs,
		ConfirmedAt:    i.ConfirmedAt,
		ConfirmedBy:    i.ConfirmedBy,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
		Version:        i.Version,
	}
}
