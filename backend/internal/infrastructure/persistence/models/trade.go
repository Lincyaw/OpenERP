package models

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SalesOrderModel is the persistence model for the SalesOrder aggregate root.
type SalesOrderModel struct {
	TenantAggregateModel
	OrderNumber    string                `gorm:"type:varchar(50);not null;uniqueIndex:idx_sales_order_tenant_number,priority:2"`
	CustomerID     uuid.UUID             `gorm:"type:uuid;not null;index"`
	CustomerName   string                `gorm:"type:varchar(200);not null"`
	WarehouseID    *uuid.UUID            `gorm:"type:uuid;index"`
	Items          []SalesOrderItemModel `gorm:"foreignKey:OrderID;references:ID"`
	TotalAmount    decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	DiscountAmount decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	PayableAmount  decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	Status         trade.OrderStatus     `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	Remark         string                `gorm:"type:text"`
	ConfirmedAt    *time.Time            `gorm:"index"`
	ShippedAt      *time.Time            `gorm:"index"`
	CompletedAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (SalesOrderModel) TableName() string {
	return "sales_orders"
}

// ToDomain converts the persistence model to a domain SalesOrder entity.
func (m *SalesOrderModel) ToDomain() *trade.SalesOrder {
	order := &trade.SalesOrder{
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
		OrderNumber:    m.OrderNumber,
		CustomerID:     m.CustomerID,
		CustomerName:   m.CustomerName,
		WarehouseID:    m.WarehouseID,
		TotalAmount:    m.TotalAmount,
		DiscountAmount: m.DiscountAmount,
		PayableAmount:  m.PayableAmount,
		Status:         m.Status,
		Remark:         m.Remark,
		ConfirmedAt:    m.ConfirmedAt,
		ShippedAt:      m.ShippedAt,
		CompletedAt:    m.CompletedAt,
		CancelledAt:    m.CancelledAt,
		CancelReason:   m.CancelReason,
		Items:          make([]trade.SalesOrderItem, len(m.Items)),
	}
	for i, item := range m.Items {
		order.Items[i] = *item.ToDomain()
	}
	return order
}

// FromDomain populates the persistence model from a domain SalesOrder entity.
func (m *SalesOrderModel) FromDomain(o *trade.SalesOrder) {
	m.FromDomainTenantAggregateRoot(o.TenantAggregateRoot)
	m.OrderNumber = o.OrderNumber
	m.CustomerID = o.CustomerID
	m.CustomerName = o.CustomerName
	m.WarehouseID = o.WarehouseID
	m.TotalAmount = o.TotalAmount
	m.DiscountAmount = o.DiscountAmount
	m.PayableAmount = o.PayableAmount
	m.Status = o.Status
	m.Remark = o.Remark
	m.ConfirmedAt = o.ConfirmedAt
	m.ShippedAt = o.ShippedAt
	m.CompletedAt = o.CompletedAt
	m.CancelledAt = o.CancelledAt
	m.CancelReason = o.CancelReason
	m.Items = make([]SalesOrderItemModel, len(o.Items))
	for i, item := range o.Items {
		m.Items[i] = *SalesOrderItemModelFromDomain(&item)
	}
}

// SalesOrderModelFromDomain creates a new persistence model from a domain SalesOrder entity.
func SalesOrderModelFromDomain(o *trade.SalesOrder) *SalesOrderModel {
	m := &SalesOrderModel{}
	m.FromDomain(o)
	return m
}

// SalesOrderItemModel is the persistence model for the SalesOrderItem entity.
type SalesOrderItemModel struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key"`
	OrderID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID      uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName    string          `gorm:"type:varchar(200);not null"`
	ProductCode    string          `gorm:"type:varchar(50);not null"`
	Quantity       decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitPrice      decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Amount         decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Unit           string          `gorm:"type:varchar(20);not null"`
	ConversionRate decimal.Decimal `gorm:"type:decimal(18,6);not null;default:1"`
	BaseQuantity   decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	BaseUnit       string          `gorm:"type:varchar(20);not null"`
	Remark         string          `gorm:"type:varchar(500)"`
	CreatedAt      time.Time       `gorm:"not null"`
	UpdatedAt      time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (SalesOrderItemModel) TableName() string {
	return "sales_order_items"
}

// ToDomain converts the persistence model to a domain SalesOrderItem entity.
func (m *SalesOrderItemModel) ToDomain() *trade.SalesOrderItem {
	return &trade.SalesOrderItem{
		ID:             m.ID,
		OrderID:        m.OrderID,
		ProductID:      m.ProductID,
		ProductName:    m.ProductName,
		ProductCode:    m.ProductCode,
		Quantity:       m.Quantity,
		UnitPrice:      m.UnitPrice,
		Amount:         m.Amount,
		Unit:           m.Unit,
		ConversionRate: m.ConversionRate,
		BaseQuantity:   m.BaseQuantity,
		BaseUnit:       m.BaseUnit,
		Remark:         m.Remark,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain SalesOrderItem entity.
func (m *SalesOrderItemModel) FromDomain(i *trade.SalesOrderItem) {
	m.ID = i.ID
	m.OrderID = i.OrderID
	m.ProductID = i.ProductID
	m.ProductName = i.ProductName
	m.ProductCode = i.ProductCode
	m.Quantity = i.Quantity
	m.UnitPrice = i.UnitPrice
	m.Amount = i.Amount
	m.Unit = i.Unit
	m.ConversionRate = i.ConversionRate
	m.BaseQuantity = i.BaseQuantity
	m.BaseUnit = i.BaseUnit
	m.Remark = i.Remark
	m.CreatedAt = i.CreatedAt
	m.UpdatedAt = i.UpdatedAt
}

// SalesOrderItemModelFromDomain creates a new persistence model from a domain SalesOrderItem entity.
func SalesOrderItemModelFromDomain(i *trade.SalesOrderItem) *SalesOrderItemModel {
	m := &SalesOrderItemModel{}
	m.FromDomain(i)
	return m
}

// PurchaseOrderModel is the persistence model for the PurchaseOrder aggregate root.
type PurchaseOrderModel struct {
	TenantAggregateModel
	OrderNumber    string                    `gorm:"type:varchar(50);not null;uniqueIndex:idx_purchase_order_tenant_number,priority:2"`
	SupplierID     uuid.UUID                 `gorm:"type:uuid;not null;index"`
	SupplierName   string                    `gorm:"type:varchar(200);not null"`
	WarehouseID    *uuid.UUID                `gorm:"type:uuid;index"`
	Items          []PurchaseOrderItemModel  `gorm:"foreignKey:OrderID;references:ID"`
	TotalAmount    decimal.Decimal           `gorm:"type:decimal(18,4);not null;default:0"`
	DiscountAmount decimal.Decimal           `gorm:"type:decimal(18,4);not null;default:0"`
	PayableAmount  decimal.Decimal           `gorm:"type:decimal(18,4);not null;default:0"`
	Status         trade.PurchaseOrderStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	Remark         string                    `gorm:"type:text"`
	ConfirmedAt    *time.Time                `gorm:"index"`
	CompletedAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PurchaseOrderModel) TableName() string {
	return "purchase_orders"
}

// ToDomain converts the persistence model to a domain PurchaseOrder entity.
func (m *PurchaseOrderModel) ToDomain() *trade.PurchaseOrder {
	order := &trade.PurchaseOrder{
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
		OrderNumber:    m.OrderNumber,
		SupplierID:     m.SupplierID,
		SupplierName:   m.SupplierName,
		WarehouseID:    m.WarehouseID,
		TotalAmount:    m.TotalAmount,
		DiscountAmount: m.DiscountAmount,
		PayableAmount:  m.PayableAmount,
		Status:         m.Status,
		Remark:         m.Remark,
		ConfirmedAt:    m.ConfirmedAt,
		CompletedAt:    m.CompletedAt,
		CancelledAt:    m.CancelledAt,
		CancelReason:   m.CancelReason,
		Items:          make([]trade.PurchaseOrderItem, len(m.Items)),
	}
	for i, item := range m.Items {
		order.Items[i] = *item.ToDomain()
	}
	return order
}

// FromDomain populates the persistence model from a domain PurchaseOrder entity.
func (m *PurchaseOrderModel) FromDomain(o *trade.PurchaseOrder) {
	m.FromDomainTenantAggregateRoot(o.TenantAggregateRoot)
	m.OrderNumber = o.OrderNumber
	m.SupplierID = o.SupplierID
	m.SupplierName = o.SupplierName
	m.WarehouseID = o.WarehouseID
	m.TotalAmount = o.TotalAmount
	m.DiscountAmount = o.DiscountAmount
	m.PayableAmount = o.PayableAmount
	m.Status = o.Status
	m.Remark = o.Remark
	m.ConfirmedAt = o.ConfirmedAt
	m.CompletedAt = o.CompletedAt
	m.CancelledAt = o.CancelledAt
	m.CancelReason = o.CancelReason
	m.Items = make([]PurchaseOrderItemModel, len(o.Items))
	for i, item := range o.Items {
		m.Items[i] = *PurchaseOrderItemModelFromDomain(&item)
	}
}

// PurchaseOrderModelFromDomain creates a new persistence model from a domain PurchaseOrder entity.
func PurchaseOrderModelFromDomain(o *trade.PurchaseOrder) *PurchaseOrderModel {
	m := &PurchaseOrderModel{}
	m.FromDomain(o)
	return m
}

// PurchaseOrderItemModel is the persistence model for the PurchaseOrderItem entity.
type PurchaseOrderItemModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	OrderID          uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID        uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName      string          `gorm:"type:varchar(200);not null"`
	ProductCode      string          `gorm:"type:varchar(50);not null"`
	OrderedQuantity  decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	ReceivedQuantity decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitCost         decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Unit             string          `gorm:"type:varchar(20);not null"`
	ConversionRate   decimal.Decimal `gorm:"type:decimal(18,6);not null;default:1"`
	BaseQuantity     decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	BaseUnit         string          `gorm:"type:varchar(20);not null"`
	Remark           string          `gorm:"type:varchar(500)"`
	CreatedAt        time.Time       `gorm:"not null"`
	UpdatedAt        time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (PurchaseOrderItemModel) TableName() string {
	return "purchase_order_items"
}

// ToDomain converts the persistence model to a domain PurchaseOrderItem entity.
func (m *PurchaseOrderItemModel) ToDomain() *trade.PurchaseOrderItem {
	return &trade.PurchaseOrderItem{
		ID:               m.ID,
		OrderID:          m.OrderID,
		ProductID:        m.ProductID,
		ProductName:      m.ProductName,
		ProductCode:      m.ProductCode,
		OrderedQuantity:  m.OrderedQuantity,
		ReceivedQuantity: m.ReceivedQuantity,
		UnitCost:         m.UnitCost,
		Amount:           m.Amount,
		Unit:             m.Unit,
		ConversionRate:   m.ConversionRate,
		BaseQuantity:     m.BaseQuantity,
		BaseUnit:         m.BaseUnit,
		Remark:           m.Remark,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain PurchaseOrderItem entity.
func (m *PurchaseOrderItemModel) FromDomain(i *trade.PurchaseOrderItem) {
	m.ID = i.ID
	m.OrderID = i.OrderID
	m.ProductID = i.ProductID
	m.ProductName = i.ProductName
	m.ProductCode = i.ProductCode
	m.OrderedQuantity = i.OrderedQuantity
	m.ReceivedQuantity = i.ReceivedQuantity
	m.UnitCost = i.UnitCost
	m.Amount = i.Amount
	m.Unit = i.Unit
	m.ConversionRate = i.ConversionRate
	m.BaseQuantity = i.BaseQuantity
	m.BaseUnit = i.BaseUnit
	m.Remark = i.Remark
	m.CreatedAt = i.CreatedAt
	m.UpdatedAt = i.UpdatedAt
}

// PurchaseOrderItemModelFromDomain creates a new persistence model from a domain PurchaseOrderItem entity.
func PurchaseOrderItemModelFromDomain(i *trade.PurchaseOrderItem) *PurchaseOrderItemModel {
	m := &PurchaseOrderItemModel{}
	m.FromDomain(i)
	return m
}

// SalesReturnModel is the persistence model for the SalesReturn aggregate root.
type SalesReturnModel struct {
	TenantAggregateModel
	ReturnNumber     string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_sales_return_tenant_number,priority:2"`
	SalesOrderID     uuid.UUID              `gorm:"type:uuid;not null;index"`
	SalesOrderNumber string                 `gorm:"type:varchar(50);not null"`
	CustomerID       uuid.UUID              `gorm:"type:uuid;not null;index"`
	CustomerName     string                 `gorm:"type:varchar(200);not null"`
	WarehouseID      *uuid.UUID             `gorm:"type:uuid;index"`
	Items            []SalesReturnItemModel `gorm:"foreignKey:ReturnID;references:ID"`
	TotalRefund      decimal.Decimal        `gorm:"type:decimal(18,4);not null;default:0"`
	Status           trade.ReturnStatus     `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	Reason           string                 `gorm:"type:text"`
	Remark           string                 `gorm:"type:text"`
	SubmittedAt      *time.Time             `gorm:"index"`
	ApprovedAt       *time.Time             `gorm:"index"`
	ApprovedBy       *uuid.UUID             `gorm:"type:uuid"`
	ApprovalNote     string                 `gorm:"type:varchar(500)"`
	RejectedAt       *time.Time
	RejectedBy       *uuid.UUID `gorm:"type:uuid"`
	RejectionReason  string     `gorm:"type:varchar(500)"`
	ReceivedAt       *time.Time
	CompletedAt      *time.Time
	CancelledAt      *time.Time
	CancelReason     string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (SalesReturnModel) TableName() string {
	return "sales_returns"
}

// ToDomain converts the persistence model to a domain SalesReturn entity.
func (m *SalesReturnModel) ToDomain() *trade.SalesReturn {
	sr := &trade.SalesReturn{
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
		ReturnNumber:     m.ReturnNumber,
		SalesOrderID:     m.SalesOrderID,
		SalesOrderNumber: m.SalesOrderNumber,
		CustomerID:       m.CustomerID,
		CustomerName:     m.CustomerName,
		WarehouseID:      m.WarehouseID,
		TotalRefund:      m.TotalRefund,
		Status:           m.Status,
		Reason:           m.Reason,
		Remark:           m.Remark,
		SubmittedAt:      m.SubmittedAt,
		ApprovedAt:       m.ApprovedAt,
		ApprovedBy:       m.ApprovedBy,
		ApprovalNote:     m.ApprovalNote,
		RejectedAt:       m.RejectedAt,
		RejectedBy:       m.RejectedBy,
		RejectionReason:  m.RejectionReason,
		ReceivedAt:       m.ReceivedAt,
		CompletedAt:      m.CompletedAt,
		CancelledAt:      m.CancelledAt,
		CancelReason:     m.CancelReason,
		Items:            make([]trade.SalesReturnItem, len(m.Items)),
	}
	for i, item := range m.Items {
		sr.Items[i] = *item.ToDomain()
	}
	return sr
}

// FromDomain populates the persistence model from a domain SalesReturn entity.
func (m *SalesReturnModel) FromDomain(sr *trade.SalesReturn) {
	m.FromDomainTenantAggregateRoot(sr.TenantAggregateRoot)
	m.ReturnNumber = sr.ReturnNumber
	m.SalesOrderID = sr.SalesOrderID
	m.SalesOrderNumber = sr.SalesOrderNumber
	m.CustomerID = sr.CustomerID
	m.CustomerName = sr.CustomerName
	m.WarehouseID = sr.WarehouseID
	m.TotalRefund = sr.TotalRefund
	m.Status = sr.Status
	m.Reason = sr.Reason
	m.Remark = sr.Remark
	m.SubmittedAt = sr.SubmittedAt
	m.ApprovedAt = sr.ApprovedAt
	m.ApprovedBy = sr.ApprovedBy
	m.ApprovalNote = sr.ApprovalNote
	m.RejectedAt = sr.RejectedAt
	m.RejectedBy = sr.RejectedBy
	m.RejectionReason = sr.RejectionReason
	m.ReceivedAt = sr.ReceivedAt
	m.CompletedAt = sr.CompletedAt
	m.CancelledAt = sr.CancelledAt
	m.CancelReason = sr.CancelReason
	m.Items = make([]SalesReturnItemModel, len(sr.Items))
	for i, item := range sr.Items {
		m.Items[i] = *SalesReturnItemModelFromDomain(&item)
	}
}

// SalesReturnModelFromDomain creates a new persistence model from a domain SalesReturn entity.
func SalesReturnModelFromDomain(sr *trade.SalesReturn) *SalesReturnModel {
	m := &SalesReturnModel{}
	m.FromDomain(sr)
	return m
}

// SalesReturnItemModel is the persistence model for the SalesReturnItem entity.
type SalesReturnItemModel struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key"`
	ReturnID          uuid.UUID       `gorm:"type:uuid;not null;index"`
	SalesOrderItemID  uuid.UUID       `gorm:"type:uuid;not null"`
	ProductID         uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName       string          `gorm:"type:varchar(200);not null"`
	ProductCode       string          `gorm:"type:varchar(50);not null"`
	OriginalQuantity  decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	ReturnQuantity    decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitPrice         decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	RefundAmount      decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Unit              string          `gorm:"type:varchar(20);not null"`
	ConversionRate    decimal.Decimal `gorm:"type:decimal(18,6);not null;default:1"`
	BaseQuantity      decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	BaseUnit          string          `gorm:"type:varchar(20);not null"`
	Reason            string          `gorm:"type:varchar(500)"`
	ConditionOnReturn string          `gorm:"type:varchar(100)"`
	CreatedAt         time.Time       `gorm:"not null"`
	UpdatedAt         time.Time       `gorm:"not null"`
}

// TableName returns the table name for GORM
func (SalesReturnItemModel) TableName() string {
	return "sales_return_items"
}

// ToDomain converts the persistence model to a domain SalesReturnItem entity.
func (m *SalesReturnItemModel) ToDomain() *trade.SalesReturnItem {
	return &trade.SalesReturnItem{
		ID:                m.ID,
		ReturnID:          m.ReturnID,
		SalesOrderItemID:  m.SalesOrderItemID,
		ProductID:         m.ProductID,
		ProductName:       m.ProductName,
		ProductCode:       m.ProductCode,
		OriginalQuantity:  m.OriginalQuantity,
		ReturnQuantity:    m.ReturnQuantity,
		UnitPrice:         m.UnitPrice,
		RefundAmount:      m.RefundAmount,
		Unit:              m.Unit,
		ConversionRate:    m.ConversionRate,
		BaseQuantity:      m.BaseQuantity,
		BaseUnit:          m.BaseUnit,
		Reason:            m.Reason,
		ConditionOnReturn: m.ConditionOnReturn,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain SalesReturnItem entity.
func (m *SalesReturnItemModel) FromDomain(i *trade.SalesReturnItem) {
	m.ID = i.ID
	m.ReturnID = i.ReturnID
	m.SalesOrderItemID = i.SalesOrderItemID
	m.ProductID = i.ProductID
	m.ProductName = i.ProductName
	m.ProductCode = i.ProductCode
	m.OriginalQuantity = i.OriginalQuantity
	m.ReturnQuantity = i.ReturnQuantity
	m.UnitPrice = i.UnitPrice
	m.RefundAmount = i.RefundAmount
	m.Unit = i.Unit
	m.ConversionRate = i.ConversionRate
	m.BaseQuantity = i.BaseQuantity
	m.BaseUnit = i.BaseUnit
	m.Reason = i.Reason
	m.ConditionOnReturn = i.ConditionOnReturn
	m.CreatedAt = i.CreatedAt
	m.UpdatedAt = i.UpdatedAt
}

// SalesReturnItemModelFromDomain creates a new persistence model from a domain SalesReturnItem entity.
func SalesReturnItemModelFromDomain(i *trade.SalesReturnItem) *SalesReturnItemModel {
	m := &SalesReturnItemModel{}
	m.FromDomain(i)
	return m
}

// PurchaseReturnModel is the persistence model for the PurchaseReturn aggregate root.
type PurchaseReturnModel struct {
	TenantAggregateModel
	ReturnNumber        string                     `gorm:"type:varchar(50);not null;uniqueIndex:idx_purchase_return_tenant_number,priority:2"`
	PurchaseOrderID     uuid.UUID                  `gorm:"type:uuid;not null;index"`
	PurchaseOrderNumber string                     `gorm:"type:varchar(50);not null"`
	SupplierID          uuid.UUID                  `gorm:"type:uuid;not null;index"`
	SupplierName        string                     `gorm:"type:varchar(200);not null"`
	WarehouseID         *uuid.UUID                 `gorm:"type:uuid;index"`
	Items               []PurchaseReturnItemModel  `gorm:"foreignKey:ReturnID;references:ID"`
	TotalRefund         decimal.Decimal            `gorm:"type:decimal(18,4);not null;default:0"`
	Status              trade.PurchaseReturnStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	Reason              string                     `gorm:"type:text"`
	Remark              string                     `gorm:"type:text"`
	SubmittedAt         *time.Time                 `gorm:"index"`
	ApprovedAt          *time.Time                 `gorm:"index"`
	ApprovedBy          *uuid.UUID                 `gorm:"type:uuid"`
	ApprovalNote        string                     `gorm:"type:varchar(500)"`
	RejectedAt          *time.Time
	RejectedBy          *uuid.UUID `gorm:"type:uuid"`
	RejectionReason     string     `gorm:"type:varchar(500)"`
	ShippedAt           *time.Time `gorm:"index"`
	ShippedBy           *uuid.UUID `gorm:"type:uuid"`
	ShippingNote        string     `gorm:"type:varchar(500)"`
	TrackingNumber      string     `gorm:"type:varchar(100)"`
	CompletedAt         *time.Time
	CancelledAt         *time.Time
	CancelReason        string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PurchaseReturnModel) TableName() string {
	return "purchase_returns"
}

// ToDomain converts the persistence model to a domain PurchaseReturn entity.
func (m *PurchaseReturnModel) ToDomain() *trade.PurchaseReturn {
	pr := &trade.PurchaseReturn{
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
		ReturnNumber:        m.ReturnNumber,
		PurchaseOrderID:     m.PurchaseOrderID,
		PurchaseOrderNumber: m.PurchaseOrderNumber,
		SupplierID:          m.SupplierID,
		SupplierName:        m.SupplierName,
		WarehouseID:         m.WarehouseID,
		TotalRefund:         m.TotalRefund,
		Status:              m.Status,
		Reason:              m.Reason,
		Remark:              m.Remark,
		SubmittedAt:         m.SubmittedAt,
		ApprovedAt:          m.ApprovedAt,
		ApprovedBy:          m.ApprovedBy,
		ApprovalNote:        m.ApprovalNote,
		RejectedAt:          m.RejectedAt,
		RejectedBy:          m.RejectedBy,
		RejectionReason:     m.RejectionReason,
		ShippedAt:           m.ShippedAt,
		ShippedBy:           m.ShippedBy,
		ShippingNote:        m.ShippingNote,
		TrackingNumber:      m.TrackingNumber,
		CompletedAt:         m.CompletedAt,
		CancelledAt:         m.CancelledAt,
		CancelReason:        m.CancelReason,
		Items:               make([]trade.PurchaseReturnItem, len(m.Items)),
	}
	for i, item := range m.Items {
		pr.Items[i] = *item.ToDomain()
	}
	return pr
}

// FromDomain populates the persistence model from a domain PurchaseReturn entity.
func (m *PurchaseReturnModel) FromDomain(pr *trade.PurchaseReturn) {
	m.FromDomainTenantAggregateRoot(pr.TenantAggregateRoot)
	m.ReturnNumber = pr.ReturnNumber
	m.PurchaseOrderID = pr.PurchaseOrderID
	m.PurchaseOrderNumber = pr.PurchaseOrderNumber
	m.SupplierID = pr.SupplierID
	m.SupplierName = pr.SupplierName
	m.WarehouseID = pr.WarehouseID
	m.TotalRefund = pr.TotalRefund
	m.Status = pr.Status
	m.Reason = pr.Reason
	m.Remark = pr.Remark
	m.SubmittedAt = pr.SubmittedAt
	m.ApprovedAt = pr.ApprovedAt
	m.ApprovedBy = pr.ApprovedBy
	m.ApprovalNote = pr.ApprovalNote
	m.RejectedAt = pr.RejectedAt
	m.RejectedBy = pr.RejectedBy
	m.RejectionReason = pr.RejectionReason
	m.ShippedAt = pr.ShippedAt
	m.ShippedBy = pr.ShippedBy
	m.ShippingNote = pr.ShippingNote
	m.TrackingNumber = pr.TrackingNumber
	m.CompletedAt = pr.CompletedAt
	m.CancelledAt = pr.CancelledAt
	m.CancelReason = pr.CancelReason
	m.Items = make([]PurchaseReturnItemModel, len(pr.Items))
	for i, item := range pr.Items {
		m.Items[i] = *PurchaseReturnItemModelFromDomain(&item)
	}
}

// PurchaseReturnModelFromDomain creates a new persistence model from a domain PurchaseReturn entity.
func PurchaseReturnModelFromDomain(pr *trade.PurchaseReturn) *PurchaseReturnModel {
	m := &PurchaseReturnModel{}
	m.FromDomain(pr)
	return m
}

// PurchaseReturnItemModel is the persistence model for the PurchaseReturnItem entity.
type PurchaseReturnItemModel struct {
	ID                  uuid.UUID       `gorm:"type:uuid;primary_key"`
	ReturnID            uuid.UUID       `gorm:"type:uuid;not null;index"`
	PurchaseOrderItemID uuid.UUID       `gorm:"type:uuid;not null"`
	ProductID           uuid.UUID       `gorm:"type:uuid;not null"`
	ProductName         string          `gorm:"type:varchar(200);not null"`
	ProductCode         string          `gorm:"type:varchar(50);not null"`
	OriginalQuantity    decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	ReturnQuantity      decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	UnitCost            decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	RefundAmount        decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	Unit                string          `gorm:"type:varchar(20);not null"`
	Reason              string          `gorm:"type:varchar(500)"`
	ConditionOnReturn   string          `gorm:"type:varchar(100)"`
	BatchNumber         string          `gorm:"type:varchar(50)"`
	ShippedQuantity     decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	ShippedAt           *time.Time
	SupplierReceivedQty decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	SupplierReceivedAt  *time.Time
	CreatedAt           time.Time `gorm:"not null"`
	UpdatedAt           time.Time `gorm:"not null"`
}

// TableName returns the table name for GORM
func (PurchaseReturnItemModel) TableName() string {
	return "purchase_return_items"
}

// ToDomain converts the persistence model to a domain PurchaseReturnItem entity.
func (m *PurchaseReturnItemModel) ToDomain() *trade.PurchaseReturnItem {
	return &trade.PurchaseReturnItem{
		ID:                  m.ID,
		ReturnID:            m.ReturnID,
		PurchaseOrderItemID: m.PurchaseOrderItemID,
		ProductID:           m.ProductID,
		ProductName:         m.ProductName,
		ProductCode:         m.ProductCode,
		OriginalQuantity:    m.OriginalQuantity,
		ReturnQuantity:      m.ReturnQuantity,
		UnitCost:            m.UnitCost,
		RefundAmount:        m.RefundAmount,
		Unit:                m.Unit,
		Reason:              m.Reason,
		ConditionOnReturn:   m.ConditionOnReturn,
		BatchNumber:         m.BatchNumber,
		ShippedQuantity:     m.ShippedQuantity,
		ShippedAt:           m.ShippedAt,
		SupplierReceivedQty: m.SupplierReceivedQty,
		SupplierReceivedAt:  m.SupplierReceivedAt,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain PurchaseReturnItem entity.
func (m *PurchaseReturnItemModel) FromDomain(i *trade.PurchaseReturnItem) {
	m.ID = i.ID
	m.ReturnID = i.ReturnID
	m.PurchaseOrderItemID = i.PurchaseOrderItemID
	m.ProductID = i.ProductID
	m.ProductName = i.ProductName
	m.ProductCode = i.ProductCode
	m.OriginalQuantity = i.OriginalQuantity
	m.ReturnQuantity = i.ReturnQuantity
	m.UnitCost = i.UnitCost
	m.RefundAmount = i.RefundAmount
	m.Unit = i.Unit
	m.Reason = i.Reason
	m.ConditionOnReturn = i.ConditionOnReturn
	m.BatchNumber = i.BatchNumber
	m.ShippedQuantity = i.ShippedQuantity
	m.ShippedAt = i.ShippedAt
	m.SupplierReceivedQty = i.SupplierReceivedQty
	m.SupplierReceivedAt = i.SupplierReceivedAt
	m.CreatedAt = i.CreatedAt
	m.UpdatedAt = i.UpdatedAt
}

// PurchaseReturnItemModelFromDomain creates a new persistence model from a domain PurchaseReturnItem entity.
func PurchaseReturnItemModelFromDomain(i *trade.PurchaseReturnItem) *PurchaseReturnItemModel {
	m := &PurchaseReturnItemModel{}
	m.FromDomain(i)
	return m
}
