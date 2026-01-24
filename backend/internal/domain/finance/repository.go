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
	CustomerID    *uuid.UUID      // Filter by customer
	Status        *VoucherStatus  // Filter by status
	PaymentMethod *PaymentMethod  // Filter by payment method
	FromDate      *time.Time      // Filter by receipt date range start
	ToDate        *time.Time      // Filter by receipt date range end
	MinAmount     *decimal.Decimal // Filter by minimum amount
	MaxAmount     *decimal.Decimal // Filter by maximum amount
	HasUnallocated *bool          // Filter vouchers with unallocated amount
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
