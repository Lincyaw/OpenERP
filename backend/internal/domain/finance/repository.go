package finance

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountReceivableFilter defines filtering options for receivable queries
type AccountReceivableFilter struct {
	shared.Filter
	CustomerID *uuid.UUID        // Filter by customer
	Status     *ReceivableStatus // Filter by status
	SourceType *SourceType       // Filter by source type
	SourceID   *uuid.UUID        // Filter by source document
	FromDate   *time.Time        // Filter by creation date range start
	ToDate     *time.Time        // Filter by creation date range end
	DueFrom    *time.Time        // Filter by due date range start
	DueTo      *time.Time        // Filter by due date range end
	Overdue    *bool             // Filter only overdue receivables
	MinAmount  *decimal.Decimal  // Filter by minimum outstanding amount
	MaxAmount  *decimal.Decimal  // Filter by maximum outstanding amount
}

// AccountReceivableRepository defines the interface for account receivable persistence
type AccountReceivableRepository interface {
	// FindByID finds an account receivable by ID
	FindByID(ctx context.Context, id uuid.UUID) (*AccountReceivable, error)

	// FindByIDForTenant finds an account receivable by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*AccountReceivable, error)

	// FindByReceivableNumber finds by receivable number for a tenant
	FindByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (*AccountReceivable, error)

	// FindBySource finds by source document (e.g., sales order)
	FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType SourceType, sourceID uuid.UUID) (*AccountReceivable, error)

	// FindAllForTenant finds all account receivables for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter AccountReceivableFilter) ([]AccountReceivable, error)

	// FindByCustomer finds account receivables for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter AccountReceivableFilter) ([]AccountReceivable, error)

	// FindByStatus finds account receivables by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status ReceivableStatus, filter AccountReceivableFilter) ([]AccountReceivable, error)

	// FindOutstanding finds all outstanding (pending or partial) receivables for a customer
	FindOutstanding(ctx context.Context, tenantID, customerID uuid.UUID) ([]AccountReceivable, error)

	// FindOverdue finds all overdue receivables for a tenant
	FindOverdue(ctx context.Context, tenantID uuid.UUID, filter AccountReceivableFilter) ([]AccountReceivable, error)

	// Save creates or updates an account receivable
	Save(ctx context.Context, receivable *AccountReceivable) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, receivable *AccountReceivable) error

	// Delete soft deletes an account receivable
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes an account receivable for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts account receivables for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter AccountReceivableFilter) (int64, error)

	// CountByStatus counts account receivables by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status ReceivableStatus) (int64, error)

	// CountByCustomer counts account receivables for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// CountOutstandingByCustomer counts unsettled (PENDING or PARTIAL) receivables for a customer
	// Used for validation before customer deletion
	CountOutstandingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// CountOverdue counts overdue receivables for a tenant
	CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// SumOutstandingByCustomer calculates total outstanding amount for a customer
	SumOutstandingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// SumOutstandingForTenant calculates total outstanding amount for a tenant
	SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// SumOverdueForTenant calculates total overdue amount for a tenant
	SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// ExistsByReceivableNumber checks if a receivable number exists for a tenant
	ExistsByReceivableNumber(ctx context.Context, tenantID uuid.UUID, receivableNumber string) (bool, error)

	// ExistsBySource checks if a receivable exists for the given source document
	ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType SourceType, sourceID uuid.UUID) (bool, error)

	// GenerateReceivableNumber generates a unique receivable number for a tenant
	GenerateReceivableNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// AccountPayableFilter defines filtering options for payable queries
type AccountPayableFilter struct {
	shared.Filter
	SupplierID *uuid.UUID         // Filter by supplier
	Status     *PayableStatus     // Filter by status
	SourceType *PayableSourceType // Filter by source type
	SourceID   *uuid.UUID         // Filter by source document
	FromDate   *time.Time         // Filter by creation date range start
	ToDate     *time.Time         // Filter by creation date range end
	DueFrom    *time.Time         // Filter by due date range start
	DueTo      *time.Time         // Filter by due date range end
	Overdue    *bool              // Filter only overdue payables
	MinAmount  *decimal.Decimal   // Filter by minimum outstanding amount
	MaxAmount  *decimal.Decimal   // Filter by maximum outstanding amount
}

// AccountPayableRepository defines the interface for account payable persistence
type AccountPayableRepository interface {
	// FindByID finds an account payable by ID
	FindByID(ctx context.Context, id uuid.UUID) (*AccountPayable, error)

	// FindByIDForTenant finds an account payable by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*AccountPayable, error)

	// FindByPayableNumber finds by payable number for a tenant
	FindByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (*AccountPayable, error)

	// FindBySource finds by source document (e.g., purchase order)
	FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType PayableSourceType, sourceID uuid.UUID) (*AccountPayable, error)

	// FindAllForTenant finds all account payables for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter AccountPayableFilter) ([]AccountPayable, error)

	// FindBySupplier finds account payables for a supplier
	FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter AccountPayableFilter) ([]AccountPayable, error)

	// FindByStatus finds account payables by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status PayableStatus, filter AccountPayableFilter) ([]AccountPayable, error)

	// FindOutstanding finds all outstanding (pending or partial) payables for a supplier
	FindOutstanding(ctx context.Context, tenantID, supplierID uuid.UUID) ([]AccountPayable, error)

	// FindOverdue finds all overdue payables for a tenant
	FindOverdue(ctx context.Context, tenantID uuid.UUID, filter AccountPayableFilter) ([]AccountPayable, error)

	// Save creates or updates an account payable
	Save(ctx context.Context, payable *AccountPayable) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, payable *AccountPayable) error

	// Delete soft deletes an account payable
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes an account payable for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts account payables for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter AccountPayableFilter) (int64, error)

	// CountByStatus counts account payables by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status PayableStatus) (int64, error)

	// CountBySupplier counts account payables for a supplier
	CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// CountOverdue counts overdue payables for a tenant
	CountOverdue(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountOutstandingBySupplier counts unsettled (PENDING or PARTIAL) payables for a supplier
	// Used for validation before supplier deletion
	CountOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// SumOutstandingBySupplier calculates total outstanding amount for a supplier
	SumOutstandingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error)

	// SumOutstandingForTenant calculates total outstanding amount for a tenant
	SumOutstandingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// SumOverdueForTenant calculates total overdue amount for a tenant
	SumOverdueForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// ExistsByPayableNumber checks if a payable number exists for a tenant
	ExistsByPayableNumber(ctx context.Context, tenantID uuid.UUID, payableNumber string) (bool, error)

	// ExistsBySource checks if a payable exists for the given source document
	ExistsBySource(ctx context.Context, tenantID uuid.UUID, sourceType PayableSourceType, sourceID uuid.UUID) (bool, error)

	// GeneratePayableNumber generates a unique payable number for a tenant
	GeneratePayableNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// ReceiptVoucherFilter defines filtering options for receipt voucher queries
type ReceiptVoucherFilter struct {
	shared.Filter
	CustomerID     *uuid.UUID       // Filter by customer
	Status         *VoucherStatus   // Filter by status
	PaymentMethod  *PaymentMethod   // Filter by payment method
	FromDate       *time.Time       // Filter by receipt date range start
	ToDate         *time.Time       // Filter by receipt date range end
	MinAmount      *decimal.Decimal // Filter by minimum amount
	MaxAmount      *decimal.Decimal // Filter by maximum amount
	HasUnallocated *bool            // Filter vouchers with unallocated amount
}

// ReceiptVoucherRepository defines the interface for receipt voucher persistence
type ReceiptVoucherRepository interface {
	// FindByID finds a receipt voucher by ID
	FindByID(ctx context.Context, id uuid.UUID) (*ReceiptVoucher, error)

	// FindByIDForTenant finds a receipt voucher by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*ReceiptVoucher, error)

	// FindByVoucherNumber finds by voucher number for a tenant
	FindByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (*ReceiptVoucher, error)

	// FindAllForTenant finds all receipt vouchers for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter ReceiptVoucherFilter) ([]ReceiptVoucher, error)

	// FindByCustomer finds receipt vouchers for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter ReceiptVoucherFilter) ([]ReceiptVoucher, error)

	// FindByStatus finds receipt vouchers by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status VoucherStatus, filter ReceiptVoucherFilter) ([]ReceiptVoucher, error)

	// FindWithUnallocatedAmount finds vouchers that have unallocated amount
	FindWithUnallocatedAmount(ctx context.Context, tenantID, customerID uuid.UUID) ([]ReceiptVoucher, error)

	// Save creates or updates a receipt voucher
	Save(ctx context.Context, voucher *ReceiptVoucher) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, voucher *ReceiptVoucher) error

	// Delete soft deletes a receipt voucher
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes a receipt voucher for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts receipt vouchers for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter ReceiptVoucherFilter) (int64, error)

	// CountByStatus counts receipt vouchers by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status VoucherStatus) (int64, error)

	// CountByCustomer counts receipt vouchers for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// SumByCustomer calculates total receipt amount for a customer
	SumByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// SumForTenant calculates total receipt amount for a tenant
	SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// SumUnallocatedByCustomer calculates total unallocated amount for a customer
	SumUnallocatedByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// ExistsByVoucherNumber checks if a voucher number exists for a tenant
	ExistsByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (bool, error)

	// GenerateVoucherNumber generates a unique voucher number for a tenant
	GenerateVoucherNumber(ctx context.Context, tenantID uuid.UUID) (string, error)

	// FindByPaymentReference finds a receipt voucher by payment reference (e.g., gateway order number)
	// This is used for payment callback processing to locate the voucher by the order number
	// sent to the payment gateway
	FindByPaymentReference(ctx context.Context, paymentReference string) (*ReceiptVoucher, error)
}

// PaymentVoucherFilter defines filtering options for payment voucher queries
type PaymentVoucherFilter struct {
	shared.Filter
	SupplierID     *uuid.UUID       // Filter by supplier
	Status         *VoucherStatus   // Filter by status
	PaymentMethod  *PaymentMethod   // Filter by payment method
	FromDate       *time.Time       // Filter by payment date range start
	ToDate         *time.Time       // Filter by payment date range end
	MinAmount      *decimal.Decimal // Filter by minimum amount
	MaxAmount      *decimal.Decimal // Filter by maximum amount
	HasUnallocated *bool            // Filter vouchers with unallocated amount
}

// PaymentVoucherRepository defines the interface for payment voucher persistence
type PaymentVoucherRepository interface {
	// FindByID finds a payment voucher by ID
	FindByID(ctx context.Context, id uuid.UUID) (*PaymentVoucher, error)

	// FindByIDForTenant finds a payment voucher by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*PaymentVoucher, error)

	// FindByVoucherNumber finds by voucher number for a tenant
	FindByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (*PaymentVoucher, error)

	// FindAllForTenant finds all payment vouchers for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter PaymentVoucherFilter) ([]PaymentVoucher, error)

	// FindBySupplier finds payment vouchers for a supplier
	FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter PaymentVoucherFilter) ([]PaymentVoucher, error)

	// FindByStatus finds payment vouchers by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status VoucherStatus, filter PaymentVoucherFilter) ([]PaymentVoucher, error)

	// FindWithUnallocatedAmount finds vouchers that have unallocated amount
	FindWithUnallocatedAmount(ctx context.Context, tenantID, supplierID uuid.UUID) ([]PaymentVoucher, error)

	// Save creates or updates a payment voucher
	Save(ctx context.Context, voucher *PaymentVoucher) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, voucher *PaymentVoucher) error

	// Delete soft deletes a payment voucher
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes a payment voucher for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts payment vouchers for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter PaymentVoucherFilter) (int64, error)

	// CountByStatus counts payment vouchers by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status VoucherStatus) (int64, error)

	// CountBySupplier counts payment vouchers for a supplier
	CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// SumBySupplier calculates total payment amount for a supplier
	SumBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error)

	// SumForTenant calculates total payment amount for a tenant
	SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// SumUnallocatedBySupplier calculates total unallocated amount for a supplier
	SumUnallocatedBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error)

	// ExistsByVoucherNumber checks if a voucher number exists for a tenant
	ExistsByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (bool, error)

	// GenerateVoucherNumber generates a unique voucher number for a tenant
	GenerateVoucherNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// ExpenseRecordFilter defines filtering options for expense record queries
type ExpenseRecordFilter struct {
	shared.Filter
	Category      *ExpenseCategory // Filter by category
	Status        *ExpenseStatus   // Filter by status
	PaymentStatus *PaymentStatus   // Filter by payment status
	FromDate      *time.Time       // Filter by incurred date range start
	ToDate        *time.Time       // Filter by incurred date range end
	MinAmount     *decimal.Decimal // Filter by minimum amount
	MaxAmount     *decimal.Decimal // Filter by maximum amount
}

// ExpenseRecordRepository defines the interface for expense record persistence
type ExpenseRecordRepository interface {
	// FindByID finds an expense record by ID
	FindByID(ctx context.Context, id uuid.UUID) (*ExpenseRecord, error)

	// FindByIDForTenant finds an expense record by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*ExpenseRecord, error)

	// FindByExpenseNumber finds by expense number for a tenant
	FindByExpenseNumber(ctx context.Context, tenantID uuid.UUID, expenseNumber string) (*ExpenseRecord, error)

	// FindAllForTenant finds all expense records for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter ExpenseRecordFilter) ([]ExpenseRecord, error)

	// FindByCategory finds expense records by category for a tenant
	FindByCategory(ctx context.Context, tenantID uuid.UUID, category ExpenseCategory, filter ExpenseRecordFilter) ([]ExpenseRecord, error)

	// FindByStatus finds expense records by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status ExpenseStatus, filter ExpenseRecordFilter) ([]ExpenseRecord, error)

	// FindPendingApproval finds all expenses pending approval for a tenant
	FindPendingApproval(ctx context.Context, tenantID uuid.UUID) ([]ExpenseRecord, error)

	// Save creates or updates an expense record
	Save(ctx context.Context, expense *ExpenseRecord) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, expense *ExpenseRecord) error

	// Delete soft deletes an expense record
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes an expense record for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts expense records for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter ExpenseRecordFilter) (int64, error)

	// CountByStatus counts expense records by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status ExpenseStatus) (int64, error)

	// CountByCategory counts expense records by category for a tenant
	CountByCategory(ctx context.Context, tenantID uuid.UUID, category ExpenseCategory) (int64, error)

	// SumByCategory calculates total amount by category for a tenant within a date range
	SumByCategory(ctx context.Context, tenantID uuid.UUID, category ExpenseCategory, from, to time.Time) (decimal.Decimal, error)

	// SumForTenant calculates total expense amount for a tenant within a date range
	SumForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error)

	// SumApprovedForTenant calculates total approved expense amount for a tenant within a date range
	SumApprovedForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error)

	// ExistsByExpenseNumber checks if an expense number exists for a tenant
	ExistsByExpenseNumber(ctx context.Context, tenantID uuid.UUID, expenseNumber string) (bool, error)

	// GenerateExpenseNumber generates a unique expense number for a tenant
	GenerateExpenseNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// OtherIncomeRecordFilter defines filtering options for other income record queries
type OtherIncomeRecordFilter struct {
	shared.Filter
	Category      *IncomeCategory  // Filter by category
	Status        *IncomeStatus    // Filter by status
	ReceiptStatus *ReceiptStatus   // Filter by receipt status
	FromDate      *time.Time       // Filter by received date range start
	ToDate        *time.Time       // Filter by received date range end
	MinAmount     *decimal.Decimal // Filter by minimum amount
	MaxAmount     *decimal.Decimal // Filter by maximum amount
}

// OtherIncomeRecordRepository defines the interface for other income record persistence
type OtherIncomeRecordRepository interface {
	// FindByID finds an income record by ID
	FindByID(ctx context.Context, id uuid.UUID) (*OtherIncomeRecord, error)

	// FindByIDForTenant finds an income record by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*OtherIncomeRecord, error)

	// FindByIncomeNumber finds by income number for a tenant
	FindByIncomeNumber(ctx context.Context, tenantID uuid.UUID, incomeNumber string) (*OtherIncomeRecord, error)

	// FindAllForTenant finds all income records for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter OtherIncomeRecordFilter) ([]OtherIncomeRecord, error)

	// FindByCategory finds income records by category for a tenant
	FindByCategory(ctx context.Context, tenantID uuid.UUID, category IncomeCategory, filter OtherIncomeRecordFilter) ([]OtherIncomeRecord, error)

	// FindByStatus finds income records by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status IncomeStatus, filter OtherIncomeRecordFilter) ([]OtherIncomeRecord, error)

	// Save creates or updates an income record
	Save(ctx context.Context, income *OtherIncomeRecord) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, income *OtherIncomeRecord) error

	// Delete soft deletes an income record
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes an income record for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts income records for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter OtherIncomeRecordFilter) (int64, error)

	// CountByStatus counts income records by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status IncomeStatus) (int64, error)

	// CountByCategory counts income records by category for a tenant
	CountByCategory(ctx context.Context, tenantID uuid.UUID, category IncomeCategory) (int64, error)

	// SumByCategory calculates total amount by category for a tenant within a date range
	SumByCategory(ctx context.Context, tenantID uuid.UUID, category IncomeCategory, from, to time.Time) (decimal.Decimal, error)

	// SumForTenant calculates total income amount for a tenant within a date range
	SumForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error)

	// SumConfirmedForTenant calculates total confirmed income amount for a tenant within a date range
	SumConfirmedForTenant(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (decimal.Decimal, error)

	// ExistsByIncomeNumber checks if an income number exists for a tenant
	ExistsByIncomeNumber(ctx context.Context, tenantID uuid.UUID, incomeNumber string) (bool, error)

	// GenerateIncomeNumber generates a unique income number for a tenant
	GenerateIncomeNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// CreditMemoFilter defines filtering options for credit memo queries
type CreditMemoFilter struct {
	shared.Filter
	SalesReturnID *uuid.UUID        // Filter by sales return
	SalesOrderID  *uuid.UUID        // Filter by original sales order
	CustomerID    *uuid.UUID        // Filter by customer
	Status        *CreditMemoStatus // Filter by status
	FromDate      *time.Time        // Filter by creation date range start
	ToDate        *time.Time        // Filter by creation date range end
	MinAmount     *decimal.Decimal  // Filter by minimum total credit
	MaxAmount     *decimal.Decimal  // Filter by maximum total credit
	HasRemaining  *bool             // Filter memos with remaining amount
}

// CreditMemoRepository defines the interface for credit memo persistence
type CreditMemoRepository interface {
	// FindByID finds a credit memo by ID
	FindByID(ctx context.Context, id uuid.UUID) (*CreditMemo, error)

	// FindByIDForTenant finds a credit memo by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*CreditMemo, error)

	// FindByMemoNumber finds by memo number for a tenant
	FindByMemoNumber(ctx context.Context, tenantID uuid.UUID, memoNumber string) (*CreditMemo, error)

	// FindBySalesReturn finds by sales return ID
	FindBySalesReturn(ctx context.Context, tenantID, salesReturnID uuid.UUID) (*CreditMemo, error)

	// FindAllForTenant finds all credit memos for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter CreditMemoFilter) ([]CreditMemo, error)

	// FindByCustomer finds credit memos for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter CreditMemoFilter) ([]CreditMemo, error)

	// FindByStatus finds credit memos by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status CreditMemoStatus, filter CreditMemoFilter) ([]CreditMemo, error)

	// FindWithRemainingCredit finds memos with remaining credit for a customer
	FindWithRemainingCredit(ctx context.Context, tenantID, customerID uuid.UUID) ([]CreditMemo, error)

	// Save creates or updates a credit memo
	Save(ctx context.Context, memo *CreditMemo) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, memo *CreditMemo) error

	// Delete soft deletes a credit memo
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes a credit memo for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts credit memos for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter CreditMemoFilter) (int64, error)

	// CountByStatus counts credit memos by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status CreditMemoStatus) (int64, error)

	// CountByCustomer counts credit memos for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// SumRemainingByCustomer calculates total remaining credit for a customer
	SumRemainingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// SumRemainingForTenant calculates total remaining credit for a tenant
	SumRemainingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// ExistsByMemoNumber checks if a memo number exists for a tenant
	ExistsByMemoNumber(ctx context.Context, tenantID uuid.UUID, memoNumber string) (bool, error)

	// ExistsBySalesReturn checks if a credit memo exists for the given sales return
	ExistsBySalesReturn(ctx context.Context, tenantID, salesReturnID uuid.UUID) (bool, error)

	// GenerateMemoNumber generates a unique memo number for a tenant
	GenerateMemoNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// DebitMemoFilter defines filtering options for debit memo queries
type DebitMemoFilter struct {
	shared.Filter
	PurchaseReturnID *uuid.UUID       // Filter by purchase return
	PurchaseOrderID  *uuid.UUID       // Filter by original purchase order
	SupplierID       *uuid.UUID       // Filter by supplier
	Status           *DebitMemoStatus // Filter by status
	FromDate         *time.Time       // Filter by creation date range start
	ToDate           *time.Time       // Filter by creation date range end
	MinAmount        *decimal.Decimal // Filter by minimum total debit
	MaxAmount        *decimal.Decimal // Filter by maximum total debit
	HasRemaining     *bool            // Filter memos with remaining amount
}

// DebitMemoRepository defines the interface for debit memo persistence
type DebitMemoRepository interface {
	// FindByID finds a debit memo by ID
	FindByID(ctx context.Context, id uuid.UUID) (*DebitMemo, error)

	// FindByIDForTenant finds a debit memo by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*DebitMemo, error)

	// FindByMemoNumber finds by memo number for a tenant
	FindByMemoNumber(ctx context.Context, tenantID uuid.UUID, memoNumber string) (*DebitMemo, error)

	// FindByPurchaseReturn finds by purchase return ID
	FindByPurchaseReturn(ctx context.Context, tenantID, purchaseReturnID uuid.UUID) (*DebitMemo, error)

	// FindAllForTenant finds all debit memos for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter DebitMemoFilter) ([]DebitMemo, error)

	// FindBySupplier finds debit memos for a supplier
	FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter DebitMemoFilter) ([]DebitMemo, error)

	// FindByStatus finds debit memos by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status DebitMemoStatus, filter DebitMemoFilter) ([]DebitMemo, error)

	// FindWithRemainingDebit finds memos with remaining debit for a supplier
	FindWithRemainingDebit(ctx context.Context, tenantID, supplierID uuid.UUID) ([]DebitMemo, error)

	// Save creates or updates a debit memo
	Save(ctx context.Context, memo *DebitMemo) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, memo *DebitMemo) error

	// Delete soft deletes a debit memo
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes a debit memo for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts debit memos for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter DebitMemoFilter) (int64, error)

	// CountByStatus counts debit memos by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status DebitMemoStatus) (int64, error)

	// CountBySupplier counts debit memos for a supplier
	CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error)

	// SumRemainingBySupplier calculates total remaining debit for a supplier
	SumRemainingBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error)

	// SumRemainingForTenant calculates total remaining debit for a tenant
	SumRemainingForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// ExistsByMemoNumber checks if a memo number exists for a tenant
	ExistsByMemoNumber(ctx context.Context, tenantID uuid.UUID, memoNumber string) (bool, error)

	// ExistsByPurchaseReturn checks if a debit memo exists for the given purchase return
	ExistsByPurchaseReturn(ctx context.Context, tenantID, purchaseReturnID uuid.UUID) (bool, error)

	// GenerateMemoNumber generates a unique memo number for a tenant
	GenerateMemoNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// RefundRecordFilter defines filtering options for refund record queries
type RefundRecordFilter struct {
	shared.Filter
	CustomerID  *uuid.UUID          // Filter by customer
	Status      *RefundRecordStatus // Filter by status
	SourceType  *RefundSourceType   // Filter by source type
	SourceID    *uuid.UUID          // Filter by source document
	GatewayType *PaymentGatewayType // Filter by payment gateway
	FromDate    *time.Time          // Filter by creation date range start
	ToDate      *time.Time          // Filter by creation date range end
	MinAmount   *decimal.Decimal    // Filter by minimum refund amount
	MaxAmount   *decimal.Decimal    // Filter by maximum refund amount
}

// RefundRecordRepository defines the interface for refund record persistence
type RefundRecordRepository interface {
	// FindByID finds a refund record by ID
	FindByID(ctx context.Context, id uuid.UUID) (*RefundRecord, error)

	// FindByIDForTenant finds a refund record by ID for a specific tenant
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*RefundRecord, error)

	// FindByRefundNumber finds by refund number for a tenant
	FindByRefundNumber(ctx context.Context, tenantID uuid.UUID, refundNumber string) (*RefundRecord, error)

	// FindByGatewayRefundID finds by gateway refund ID
	FindByGatewayRefundID(ctx context.Context, gatewayType PaymentGatewayType, gatewayRefundID string) (*RefundRecord, error)

	// FindBySource finds by source document (e.g., sales return, credit memo)
	FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType RefundSourceType, sourceID uuid.UUID) ([]RefundRecord, error)

	// FindAllForTenant finds all refund records for a tenant with filtering
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter RefundRecordFilter) ([]RefundRecord, error)

	// FindByCustomer finds refund records for a customer
	FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter RefundRecordFilter) ([]RefundRecord, error)

	// FindByStatus finds refund records by status for a tenant
	FindByStatus(ctx context.Context, tenantID uuid.UUID, status RefundRecordStatus, filter RefundRecordFilter) ([]RefundRecord, error)

	// FindPending finds all pending refund records for a tenant
	FindPending(ctx context.Context, tenantID uuid.UUID) ([]RefundRecord, error)

	// Save creates or updates a refund record
	Save(ctx context.Context, record *RefundRecord) error

	// SaveWithLock saves with optimistic locking (version check)
	SaveWithLock(ctx context.Context, record *RefundRecord) error

	// Delete soft deletes a refund record
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteForTenant soft deletes a refund record for a tenant
	DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error

	// CountForTenant counts refund records for a tenant with optional filters
	CountForTenant(ctx context.Context, tenantID uuid.UUID, filter RefundRecordFilter) (int64, error)

	// CountByStatus counts refund records by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status RefundRecordStatus) (int64, error)

	// CountByCustomer counts refund records for a customer
	CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error)

	// SumByCustomer calculates total refund amount for a customer
	SumByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// SumForTenant calculates total refund amount for a tenant
	SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// SumSuccessfulByCustomer calculates total successful refund amount for a customer
	SumSuccessfulByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error)

	// ExistsByRefundNumber checks if a refund number exists for a tenant
	ExistsByRefundNumber(ctx context.Context, tenantID uuid.UUID, refundNumber string) (bool, error)

	// ExistsByGatewayRefundID checks if a refund exists for the given gateway refund ID
	ExistsByGatewayRefundID(ctx context.Context, gatewayType PaymentGatewayType, gatewayRefundID string) (bool, error)

	// GenerateRefundNumber generates a unique refund number for a tenant
	GenerateRefundNumber(ctx context.Context, tenantID uuid.UUID) (string, error)
}
