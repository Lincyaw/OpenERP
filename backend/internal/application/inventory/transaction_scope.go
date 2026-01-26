package inventory

import (
	"context"

	"github.com/erp/backend/internal/domain/inventory"
)

// TransactionScope provides transactional access to inventory repositories.
// When a function is executed within a transaction scope, all repository operations
// will be part of the same database transaction and will be committed or rolled back atomically.
type TransactionScope interface {
	// Execute runs the given function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function succeeds, the transaction is committed.
	Execute(ctx context.Context, fn func(repos TransactionalRepositories) error) error
}

// TransactionalRepositories provides access to all inventory repositories within a transaction.
// All repositories returned share the same underlying database transaction.
//
// DDD Aggregate Boundary Notes:
//   - InventoryRepo: Repository for the InventoryItem aggregate root. All inventory state changes
//     (including batches and locks) should go through this repository.
//   - LockRepo: Used for cross-aggregate lock queries (FindExpired, FindBySource) and persisting
//     individual lock state changes. Locks are child entities of InventoryItem, but have separate
//     storage for query performance.
//   - TransactionRepo: Append-only repository for inventory transaction records.
//
// Note: StockBatch is a child entity within the InventoryItem aggregate and does NOT have
// independent repository access. Batches are persisted automatically via GORM's association
// handling when the aggregate root is saved.
type TransactionalRepositories interface {
	// InventoryRepo returns the inventory item repository scoped to the current transaction
	InventoryRepo() inventory.InventoryItemRepository
	// LockRepo returns the stock lock repository scoped to the current transaction
	LockRepo() inventory.StockLockRepository
	// TransactionRepo returns the inventory transaction repository scoped to the current transaction
	TransactionRepo() inventory.InventoryTransactionRepository
}

// NoOpTransactionScope is a transaction scope that doesn't actually use transactions.
// This is useful for testing or when transaction support is not required.
type NoOpTransactionScope struct {
	inventoryRepo   inventory.InventoryItemRepository
	lockRepo        inventory.StockLockRepository
	transactionRepo inventory.InventoryTransactionRepository
}

// NewNoOpTransactionScope creates a NoOpTransactionScope with the given repositories.
func NewNoOpTransactionScope(
	inventoryRepo inventory.InventoryItemRepository,
	lockRepo inventory.StockLockRepository,
	transactionRepo inventory.InventoryTransactionRepository,
) *NoOpTransactionScope {
	return &NoOpTransactionScope{
		inventoryRepo:   inventoryRepo,
		lockRepo:        lockRepo,
		transactionRepo: transactionRepo,
	}
}

// Execute runs the function without a real transaction (for testing/compatibility).
func (s *NoOpTransactionScope) Execute(_ context.Context, fn func(repos TransactionalRepositories) error) error {
	return fn(s)
}

// InventoryRepo returns the inventory item repository.
func (s *NoOpTransactionScope) InventoryRepo() inventory.InventoryItemRepository {
	return s.inventoryRepo
}

// LockRepo returns the stock lock repository.
func (s *NoOpTransactionScope) LockRepo() inventory.StockLockRepository {
	return s.lockRepo
}

// TransactionRepo returns the inventory transaction repository.
func (s *NoOpTransactionScope) TransactionRepo() inventory.InventoryTransactionRepository {
	return s.transactionRepo
}

// Ensure NoOpTransactionScope implements both interfaces
var _ TransactionScope = (*NoOpTransactionScope)(nil)
var _ TransactionalRepositories = (*NoOpTransactionScope)(nil)
