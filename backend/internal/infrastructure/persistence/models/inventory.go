package models

import (
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryItemModel is the persistence model for the InventoryItem aggregate root.
type InventoryItemModel struct {
	TenantAggregateModel
	WarehouseID       uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_item_warehouse_product,priority:2"`
	ProductID         uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_inventory_item_warehouse_product,priority:3"`
	AvailableQuantity decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	LockedQuantity    decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	UnitCost          decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	MinQuantity       decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	MaxQuantity       decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	// Associations
	Batches []StockBatchModel `gorm:"foreignKey:InventoryItemID;references:ID"`
	Locks   []StockLockModel  `gorm:"foreignKey:InventoryItemID;references:ID"`
}

// TableName returns the table name for GORM
func (InventoryItemModel) TableName() string {
	return "inventory_items"
}

// ToDomain converts the persistence model to a domain InventoryItem entity.
func (m *InventoryItemModel) ToDomain() *inventory.InventoryItem {
	item := &inventory.InventoryItem{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: m.CreatedBy,
		},
		WarehouseID:       m.WarehouseID,
		ProductID:         m.ProductID,
		AvailableQuantity: inventory.MustNewInventoryQuantity(m.AvailableQuantity),
		LockedQuantity:    inventory.MustNewInventoryQuantity(m.LockedQuantity),
		UnitCost:          m.UnitCost,
		MinQuantity:       inventory.MustNewInventoryQuantity(m.MinQuantity),
		MaxQuantity:       inventory.MustNewInventoryQuantity(m.MaxQuantity),
		Batches:           make([]inventory.StockBatch, len(m.Batches)),
		Locks:             make([]inventory.StockLock, len(m.Locks)),
	}
	// Convert nested models to domain
	for i, batch := range m.Batches {
		item.Batches[i] = *batch.ToDomain()
	}
	for i, lock := range m.Locks {
		item.Locks[i] = *lock.ToDomain()
	}
	return item
}

// FromDomain populates the persistence model from a domain InventoryItem entity.
func (m *InventoryItemModel) FromDomain(i *inventory.InventoryItem) {
	m.FromDomainTenantAggregateRoot(i.TenantAggregateRoot)
	m.WarehouseID = i.WarehouseID
	m.ProductID = i.ProductID
	m.AvailableQuantity = i.AvailableQuantity.Amount()
	m.LockedQuantity = i.LockedQuantity.Amount()
	m.UnitCost = i.UnitCost
	m.MinQuantity = i.MinQuantity.Amount()
	m.MaxQuantity = i.MaxQuantity.Amount()
	// Convert nested domain entities to models
	m.Batches = make([]StockBatchModel, len(i.Batches))
	for idx, batch := range i.Batches {
		m.Batches[idx] = *StockBatchModelFromDomain(&batch)
	}
	m.Locks = make([]StockLockModel, len(i.Locks))
	for idx, lock := range i.Locks {
		m.Locks[idx] = *StockLockModelFromDomain(&lock)
	}
}

// InventoryItemModelFromDomain creates a new persistence model from a domain InventoryItem entity.
func InventoryItemModelFromDomain(i *inventory.InventoryItem) *InventoryItemModel {
	m := &InventoryItemModel{}
	m.FromDomain(i)
	return m
}

// StockBatchModel is the persistence model for the StockBatch entity.
type StockBatchModel struct {
	BaseModel
	InventoryItemID uuid.UUID       `gorm:"type:uuid;not null;index"`
	BatchNumber     string          `gorm:"type:varchar(50);not null;index"`
	ProductionDate  *time.Time      `gorm:"type:date"`
	ExpiryDate      *time.Time      `gorm:"type:date;index"`
	Quantity        decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitCost        decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Consumed        bool            `gorm:"not null;default:false"`
}

// TableName returns the table name for GORM
func (StockBatchModel) TableName() string {
	return "stock_batches"
}

// ToDomain converts the persistence model to a domain StockBatch entity.
func (m *StockBatchModel) ToDomain() *inventory.StockBatch {
	return &inventory.StockBatch{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		InventoryItemID: m.InventoryItemID,
		BatchNumber:     m.BatchNumber,
		ProductionDate:  m.ProductionDate,
		ExpiryDate:      m.ExpiryDate,
		Quantity:        m.Quantity,
		UnitCost:        m.UnitCost,
		Consumed:        m.Consumed,
	}
}

// FromDomain populates the persistence model from a domain StockBatch entity.
func (m *StockBatchModel) FromDomain(b *inventory.StockBatch) {
	m.FromDomainBaseEntity(b.BaseEntity)
	m.InventoryItemID = b.InventoryItemID
	m.BatchNumber = b.BatchNumber
	m.ProductionDate = b.ProductionDate
	m.ExpiryDate = b.ExpiryDate
	m.Quantity = b.Quantity
	m.UnitCost = b.UnitCost
	m.Consumed = b.Consumed
}

// StockBatchModelFromDomain creates a new persistence model from a domain StockBatch entity.
func StockBatchModelFromDomain(b *inventory.StockBatch) *StockBatchModel {
	m := &StockBatchModel{}
	m.FromDomain(b)
	return m
}

// StockLockModel is the persistence model for the StockLock entity.
type StockLockModel struct {
	BaseModel
	InventoryItemID uuid.UUID       `gorm:"type:uuid;not null;index"`
	Quantity        decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	SourceType      string          `gorm:"type:varchar(50);not null;index:idx_lock_src"`
	SourceID        string          `gorm:"type:varchar(100);not null;index:idx_lock_src"`
	ExpireAt        time.Time       `gorm:"not null;index"`
	Released        bool            `gorm:"not null;default:false"`
	Consumed        bool            `gorm:"not null;default:false"`
	ReleasedAt      *time.Time      `gorm:"type:timestamp"`
}

// TableName returns the table name for GORM
func (StockLockModel) TableName() string {
	return "stock_locks"
}

// ToDomain converts the persistence model to a domain StockLock entity.
func (m *StockLockModel) ToDomain() *inventory.StockLock {
	return &inventory.StockLock{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		InventoryItemID: m.InventoryItemID,
		Quantity:        m.Quantity,
		SourceType:      m.SourceType,
		SourceID:        m.SourceID,
		ExpireAt:        m.ExpireAt,
		Released:        m.Released,
		Consumed:        m.Consumed,
		ReleasedAt:      m.ReleasedAt,
	}
}

// FromDomain populates the persistence model from a domain StockLock entity.
func (m *StockLockModel) FromDomain(l *inventory.StockLock) {
	m.FromDomainBaseEntity(l.BaseEntity)
	m.InventoryItemID = l.InventoryItemID
	m.Quantity = l.Quantity
	m.SourceType = l.SourceType
	m.SourceID = l.SourceID
	m.ExpireAt = l.ExpireAt
	m.Released = l.Released
	m.Consumed = l.Consumed
	m.ReleasedAt = l.ReleasedAt
}

// StockLockModelFromDomain creates a new persistence model from a domain StockLock entity.
func StockLockModelFromDomain(l *inventory.StockLock) *StockLockModel {
	m := &StockLockModel{}
	m.FromDomain(l)
	return m
}

// InventoryTransactionModel is the persistence model for the InventoryTransaction entity.
type InventoryTransactionModel struct {
	BaseModel
	TenantID        uuid.UUID                 `gorm:"type:uuid;not null;index:idx_inv_tx_tenant_time,priority:1"`
	InventoryItemID uuid.UUID                 `gorm:"type:uuid;not null;index:idx_inv_tx_item"`
	WarehouseID     uuid.UUID                 `gorm:"type:uuid;not null;index:idx_inv_tx_warehouse"`
	ProductID       uuid.UUID                 `gorm:"type:uuid;not null;index:idx_inv_tx_product"`
	TransactionType inventory.TransactionType `gorm:"type:varchar(30);not null;index:idx_inv_tx_type"`
	Quantity        decimal.Decimal           `gorm:"type:decimal(18,4);not null"`
	UnitCost        decimal.Decimal           `gorm:"type:decimal(18,4);not null"`
	TotalCost       decimal.Decimal           `gorm:"type:decimal(18,4);not null"`
	BalanceBefore   decimal.Decimal           `gorm:"type:decimal(18,4);not null"`
	BalanceAfter    decimal.Decimal           `gorm:"type:decimal(18,4);not null"`
	SourceType      inventory.SourceType      `gorm:"type:varchar(30);not null;index:idx_inv_tx_source"`
	SourceID        string                    `gorm:"type:varchar(50);not null;index:idx_inv_tx_source"`
	SourceLineID    string                    `gorm:"type:varchar(50)"`
	BatchID         *uuid.UUID                `gorm:"type:uuid;index"`
	LockID          *uuid.UUID                `gorm:"type:uuid;index"`
	Reference       string                    `gorm:"type:varchar(100)"`
	Reason          string                    `gorm:"type:varchar(255)"`
	CostMethod      string                    `gorm:"type:varchar(30)"`
	OperatorID      *uuid.UUID                `gorm:"type:uuid"`
	TransactionDate time.Time                 `gorm:"type:timestamptz;not null;index:idx_inv_tx_tenant_time,priority:2"`
}

// TableName returns the table name for GORM
func (InventoryTransactionModel) TableName() string {
	return "inventory_transactions"
}

// ToDomain converts the persistence model to a domain InventoryTransaction entity.
func (m *InventoryTransactionModel) ToDomain() *inventory.InventoryTransaction {
	return &inventory.InventoryTransaction{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		TenantID:        m.TenantID,
		InventoryItemID: m.InventoryItemID,
		WarehouseID:     m.WarehouseID,
		ProductID:       m.ProductID,
		TransactionType: m.TransactionType,
		Quantity:        m.Quantity,
		UnitCost:        m.UnitCost,
		TotalCost:       m.TotalCost,
		BalanceBefore:   m.BalanceBefore,
		BalanceAfter:    m.BalanceAfter,
		SourceType:      m.SourceType,
		SourceID:        m.SourceID,
		SourceLineID:    m.SourceLineID,
		BatchID:         m.BatchID,
		LockID:          m.LockID,
		Reference:       m.Reference,
		Reason:          m.Reason,
		CostMethod:      m.CostMethod,
		OperatorID:      m.OperatorID,
		TransactionDate: m.TransactionDate,
	}
}

// FromDomain populates the persistence model from a domain InventoryTransaction entity.
func (m *InventoryTransactionModel) FromDomain(t *inventory.InventoryTransaction) {
	m.FromDomainBaseEntity(t.BaseEntity)
	m.TenantID = t.TenantID
	m.InventoryItemID = t.InventoryItemID
	m.WarehouseID = t.WarehouseID
	m.ProductID = t.ProductID
	m.TransactionType = t.TransactionType
	m.Quantity = t.Quantity
	m.UnitCost = t.UnitCost
	m.TotalCost = t.TotalCost
	m.BalanceBefore = t.BalanceBefore
	m.BalanceAfter = t.BalanceAfter
	m.SourceType = t.SourceType
	m.SourceID = t.SourceID
	m.SourceLineID = t.SourceLineID
	m.BatchID = t.BatchID
	m.LockID = t.LockID
	m.Reference = t.Reference
	m.Reason = t.Reason
	m.CostMethod = t.CostMethod
	m.OperatorID = t.OperatorID
	m.TransactionDate = t.TransactionDate
}

// InventoryTransactionModelFromDomain creates a new persistence model from a domain InventoryTransaction entity.
func InventoryTransactionModelFromDomain(t *inventory.InventoryTransaction) *InventoryTransactionModel {
	m := &InventoryTransactionModel{}
	m.FromDomain(t)
	return m
}

// StockTakingModel is the persistence model for the StockTaking aggregate root.
type StockTakingModel struct {
	TenantAggregateModel
	TakingNumber    string                      `gorm:"type:varchar(50);not null;uniqueIndex:idx_stock_taking_number_tenant,priority:2"`
	WarehouseID     uuid.UUID                   `gorm:"type:uuid;not null;index"`
	WarehouseName   string                      `gorm:"type:varchar(100);not null"`
	Status          inventory.StockTakingStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	TakingDate      time.Time                   `gorm:"not null"`
	StartedAt       *time.Time                  `gorm:""`
	CompletedAt     *time.Time                  `gorm:""`
	ApprovedAt      *time.Time                  `gorm:""`
	ApprovedByID    *uuid.UUID                  `gorm:"type:uuid"`
	ApprovedByName  string                      `gorm:"type:varchar(100)"`
	CreatedByID     uuid.UUID                   `gorm:"type:uuid;not null"`
	CreatedByName   string                      `gorm:"type:varchar(100);not null"`
	TotalItems      int                         `gorm:"not null;default:0"`
	CountedItems    int                         `gorm:"not null;default:0"`
	DifferenceItems int                         `gorm:"not null;default:0"`
	TotalDifference decimal.Decimal             `gorm:"type:decimal(18,4);not null;default:0"`
	ApprovalNote    string                      `gorm:"type:varchar(500)"`
	Remark          string                      `gorm:"type:varchar(500)"`
	Items           []StockTakingItemModel      `gorm:"foreignKey:StockTakingID;references:ID"`
}

// TableName returns the table name for GORM
func (StockTakingModel) TableName() string {
	return "stock_takings"
}

// ToDomain converts the persistence model to a domain StockTaking entity.
func (m *StockTakingModel) ToDomain() *inventory.StockTaking {
	st := &inventory.StockTaking{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID:  m.TenantID,
			CreatedBy: m.CreatedBy,
		},
		TakingNumber:    m.TakingNumber,
		WarehouseID:     m.WarehouseID,
		WarehouseName:   m.WarehouseName,
		Status:          m.Status,
		TakingDate:      m.TakingDate,
		StartedAt:       m.StartedAt,
		CompletedAt:     m.CompletedAt,
		ApprovedAt:      m.ApprovedAt,
		ApprovedByID:    m.ApprovedByID,
		ApprovedByName:  m.ApprovedByName,
		CreatedByID:     m.CreatedByID,
		CreatedByName:   m.CreatedByName,
		TotalItems:      m.TotalItems,
		CountedItems:    m.CountedItems,
		DifferenceItems: m.DifferenceItems,
		TotalDifference: m.TotalDifference,
		ApprovalNote:    m.ApprovalNote,
		Remark:          m.Remark,
		Items:           make([]inventory.StockTakingItem, len(m.Items)),
	}
	for i, item := range m.Items {
		st.Items[i] = *item.ToDomain()
	}
	return st
}

// FromDomain populates the persistence model from a domain StockTaking entity.
func (m *StockTakingModel) FromDomain(st *inventory.StockTaking) {
	m.FromDomainTenantAggregateRoot(st.TenantAggregateRoot)
	m.TakingNumber = st.TakingNumber
	m.WarehouseID = st.WarehouseID
	m.WarehouseName = st.WarehouseName
	m.Status = st.Status
	m.TakingDate = st.TakingDate
	m.StartedAt = st.StartedAt
	m.CompletedAt = st.CompletedAt
	m.ApprovedAt = st.ApprovedAt
	m.ApprovedByID = st.ApprovedByID
	m.ApprovedByName = st.ApprovedByName
	m.CreatedByID = st.CreatedByID
	m.CreatedByName = st.CreatedByName
	m.TotalItems = st.TotalItems
	m.CountedItems = st.CountedItems
	m.DifferenceItems = st.DifferenceItems
	m.TotalDifference = st.TotalDifference
	m.ApprovalNote = st.ApprovalNote
	m.Remark = st.Remark
	m.Items = make([]StockTakingItemModel, len(st.Items))
	for i, item := range st.Items {
		m.Items[i] = *StockTakingItemModelFromDomain(&item)
	}
}

// StockTakingModelFromDomain creates a new persistence model from a domain StockTaking entity.
func StockTakingModelFromDomain(st *inventory.StockTaking) *StockTakingModel {
	m := &StockTakingModel{}
	m.FromDomain(st)
	return m
}

// StockTakingItemModel is the persistence model for the StockTakingItem entity.
type StockTakingItemModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	StockTakingID    uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID        uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName      string          `gorm:"type:varchar(200);not null"`
	ProductCode      string          `gorm:"type:varchar(50);not null"`
	Unit             string          `gorm:"type:varchar(20);not null"`
	SystemQuantity   decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	ActualQuantity   decimal.Decimal `gorm:"type:decimal(18,4)"`
	DifferenceQty    decimal.Decimal `gorm:"type:decimal(18,4)"`
	UnitCost         decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	DifferenceAmount decimal.Decimal `gorm:"type:decimal(18,4)"`
	Counted          bool            `gorm:"not null;default:false"`
	Remark           string          `gorm:"type:varchar(500)"`
	CreatedAt        time.Time       `gorm:"not null"`
	UpdatedAt        time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (StockTakingItemModel) TableName() string {
	return "stock_taking_items"
}

// ToDomain converts the persistence model to a domain StockTakingItem entity.
func (m *StockTakingItemModel) ToDomain() *inventory.StockTakingItem {
	return &inventory.StockTakingItem{
		ID:               m.ID,
		StockTakingID:    m.StockTakingID,
		ProductID:        m.ProductID,
		ProductName:      m.ProductName,
		ProductCode:      m.ProductCode,
		Unit:             m.Unit,
		SystemQuantity:   m.SystemQuantity,
		ActualQuantity:   m.ActualQuantity,
		DifferenceQty:    m.DifferenceQty,
		UnitCost:         m.UnitCost,
		DifferenceAmount: m.DifferenceAmount,
		Counted:          m.Counted,
		Remark:           m.Remark,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain StockTakingItem entity.
func (m *StockTakingItemModel) FromDomain(i *inventory.StockTakingItem) {
	m.ID = i.ID
	m.StockTakingID = i.StockTakingID
	m.ProductID = i.ProductID
	m.ProductName = i.ProductName
	m.ProductCode = i.ProductCode
	m.Unit = i.Unit
	m.SystemQuantity = i.SystemQuantity
	m.ActualQuantity = i.ActualQuantity
	m.DifferenceQty = i.DifferenceQty
	m.UnitCost = i.UnitCost
	m.DifferenceAmount = i.DifferenceAmount
	m.Counted = i.Counted
	m.Remark = i.Remark
	m.CreatedAt = i.CreatedAt
	m.UpdatedAt = i.UpdatedAt
}

// StockTakingItemModelFromDomain creates a new persistence model from a domain StockTakingItem entity.
func StockTakingItemModelFromDomain(i *inventory.StockTakingItem) *StockTakingItemModel {
	m := &StockTakingItemModel{}
	m.FromDomain(i)
	return m
}
