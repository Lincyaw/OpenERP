package persistence

import (
	"context"

	appinv "github.com/erp/backend/internal/application/inventory"
	"github.com/erp/backend/internal/domain/inventory"
	"gorm.io/gorm"
)

// GormTransactionScope implements TransactionScope using GORM transactions.
// It provides atomic execution of multiple repository operations.
type GormTransactionScope struct {
	db *gorm.DB
}

// NewGormTransactionScope creates a new GormTransactionScope.
func NewGormTransactionScope(db *gorm.DB) *GormTransactionScope {
	return &GormTransactionScope{db: db}
}

// Execute runs the given function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
func (s *GormTransactionScope) Execute(ctx context.Context, fn func(repos appinv.TransactionalRepositories) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repos := &gormTransactionalRepositories{tx: tx}
		return fn(repos)
	})
}

// gormTransactionalRepositories provides access to all repositories within a transaction.
type gormTransactionalRepositories struct {
	tx *gorm.DB
}

// InventoryRepo returns the inventory item repository scoped to the current transaction.
func (r *gormTransactionalRepositories) InventoryRepo() inventory.InventoryItemRepository {
	return NewGormInventoryItemRepository(r.tx)
}

// LockRepo returns the stock lock repository scoped to the current transaction.
func (r *gormTransactionalRepositories) LockRepo() inventory.StockLockRepository {
	return NewGormStockLockRepository(r.tx)
}

// TransactionRepo returns the inventory transaction repository scoped to the current transaction.
func (r *gormTransactionalRepositories) TransactionRepo() inventory.InventoryTransactionRepository {
	return NewGormInventoryTransactionRepository(r.tx)
}

// BatchRepo returns the stock batch repository scoped to the current transaction.
func (r *gormTransactionalRepositories) BatchRepo() inventory.StockBatchRepository {
	return NewGormStockBatchRepository(r.tx)
}

// Ensure GormTransactionScope implements TransactionScope
var _ appinv.TransactionScope = (*GormTransactionScope)(nil)

// Ensure gormTransactionalRepositories implements TransactionalRepositories
var _ appinv.TransactionalRepositories = (*gormTransactionalRepositories)(nil)
