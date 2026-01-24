package partner

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BalanceTransactionFilter contains filter options for listing balance transactions
type BalanceTransactionFilter struct {
	CustomerID      *uuid.UUID
	TransactionType *BalanceTransactionType
	SourceType      *BalanceTransactionSourceType
	DateFrom        *time.Time
	DateTo          *time.Time
	Page            int
	PageSize        int
}

// BalanceTransactionRepository defines the interface for balance transaction persistence
type BalanceTransactionRepository interface {
	// Create creates a new balance transaction
	Create(ctx context.Context, transaction *BalanceTransaction) error

	// FindByID finds a balance transaction by ID within a tenant
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*BalanceTransaction, error)

	// FindByCustomerID finds all balance transactions for a customer
	FindByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID, filter BalanceTransactionFilter) ([]*BalanceTransaction, int64, error)

	// FindBySourceID finds balance transactions by source document ID
	FindBySourceID(ctx context.Context, tenantID uuid.UUID, sourceType BalanceTransactionSourceType, sourceID string) ([]*BalanceTransaction, error)

	// List lists balance transactions with filtering
	List(ctx context.Context, tenantID uuid.UUID, filter BalanceTransactionFilter) ([]*BalanceTransaction, int64, error)

	// GetLatestByCustomerID gets the latest balance transaction for a customer
	GetLatestByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) (*BalanceTransaction, error)

	// SumByCustomerIDAndType sums the amount by customer ID and transaction type within a date range
	SumByCustomerIDAndType(ctx context.Context, tenantID, customerID uuid.UUID, txType BalanceTransactionType, from, to time.Time) (totalAmount float64, err error)
}
