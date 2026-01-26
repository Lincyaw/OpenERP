package models

import (
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountReceivableModel is the persistence model for the AccountReceivable aggregate root.
type AccountReceivableModel struct {
	TenantAggregateModel
	ReceivableNumber  string                   `gorm:"type:varchar(50);not null;uniqueIndex:idx_receivable_tenant_number,priority:2"`
	CustomerID        uuid.UUID                `gorm:"type:uuid;not null;index"`
	CustomerName      string                   `gorm:"type:varchar(200);not null"`
	SourceType        finance.SourceType       `gorm:"type:varchar(30);not null;index"`
	SourceID          uuid.UUID                `gorm:"type:uuid;not null;index"`
	SourceNumber      string                   `gorm:"type:varchar(50);not null"`
	TotalAmount       decimal.Decimal          `gorm:"type:decimal(18,4);not null"`
	PaidAmount        decimal.Decimal          `gorm:"type:decimal(18,4);not null"`
	OutstandingAmount decimal.Decimal          `gorm:"type:decimal(18,4);not null;index"`
	Status            finance.ReceivableStatus `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	DueDate           *time.Time               `gorm:"index"`
	PaymentRecords    finance.PaymentRecords   `gorm:"type:jsonb;default:'[]'"`
	Remark            string                   `gorm:"type:text"`
	PaidAt            *time.Time
	ReversedAt        *time.Time
	ReversalReason    string `gorm:"type:varchar(500)"`
	CancelledAt       *time.Time
	CancelReason      string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (AccountReceivableModel) TableName() string {
	return "account_receivables"
}

// ToDomain converts the persistence model to a domain AccountReceivable entity.
func (m *AccountReceivableModel) ToDomain() *finance.AccountReceivable {
	return &finance.AccountReceivable{
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
		ReceivableNumber:  m.ReceivableNumber,
		CustomerID:        m.CustomerID,
		CustomerName:      m.CustomerName,
		SourceType:        m.SourceType,
		SourceID:          m.SourceID,
		SourceNumber:      m.SourceNumber,
		TotalAmount:       m.TotalAmount,
		PaidAmount:        m.PaidAmount,
		OutstandingAmount: m.OutstandingAmount,
		Status:            m.Status,
		DueDate:           m.DueDate,
		PaymentRecords:    m.PaymentRecords,
		Remark:            m.Remark,
		PaidAt:            m.PaidAt,
		ReversedAt:        m.ReversedAt,
		ReversalReason:    m.ReversalReason,
		CancelledAt:       m.CancelledAt,
		CancelReason:      m.CancelReason,
	}
}

// FromDomain populates the persistence model from a domain AccountReceivable entity.
func (m *AccountReceivableModel) FromDomain(ar *finance.AccountReceivable) {
	m.FromDomainTenantAggregateRoot(ar.TenantAggregateRoot)
	m.ReceivableNumber = ar.ReceivableNumber
	m.CustomerID = ar.CustomerID
	m.CustomerName = ar.CustomerName
	m.SourceType = ar.SourceType
	m.SourceID = ar.SourceID
	m.SourceNumber = ar.SourceNumber
	m.TotalAmount = ar.TotalAmount
	m.PaidAmount = ar.PaidAmount
	m.OutstandingAmount = ar.OutstandingAmount
	m.Status = ar.Status
	m.DueDate = ar.DueDate
	m.PaymentRecords = ar.PaymentRecords
	m.Remark = ar.Remark
	m.PaidAt = ar.PaidAt
	m.ReversedAt = ar.ReversedAt
	m.ReversalReason = ar.ReversalReason
	m.CancelledAt = ar.CancelledAt
	m.CancelReason = ar.CancelReason
}

// AccountReceivableModelFromDomain creates a new persistence model from a domain AccountReceivable.
func AccountReceivableModelFromDomain(ar *finance.AccountReceivable) *AccountReceivableModel {
	m := &AccountReceivableModel{}
	m.FromDomain(ar)
	return m
}

// AccountPayableModel is the persistence model for the AccountPayable aggregate root.
type AccountPayableModel struct {
	TenantAggregateModel
	PayableNumber     string                      `gorm:"type:varchar(50);not null;uniqueIndex:idx_payable_tenant_number,priority:2"`
	SupplierID        uuid.UUID                   `gorm:"type:uuid;not null;index"`
	SupplierName      string                      `gorm:"type:varchar(200);not null"`
	SourceType        finance.PayableSourceType   `gorm:"type:varchar(30);not null;index"`
	SourceID          uuid.UUID                   `gorm:"type:uuid;not null;index"`
	SourceNumber      string                      `gorm:"type:varchar(50);not null"`
	TotalAmount       decimal.Decimal             `gorm:"type:decimal(18,4);not null"`
	PaidAmount        decimal.Decimal             `gorm:"type:decimal(18,4);not null"`
	OutstandingAmount decimal.Decimal             `gorm:"type:decimal(18,4);not null;index"`
	Status            finance.PayableStatus       `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	DueDate           *time.Time                  `gorm:"index"`
	PaymentRecords    []PayablePaymentRecordModel `gorm:"foreignKey:PayableID;references:ID"`
	Remark            string                      `gorm:"type:text"`
	PaidAt            *time.Time
	ReversedAt        *time.Time
	ReversalReason    string `gorm:"type:varchar(500)"`
	CancelledAt       *time.Time
	CancelReason      string `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (AccountPayableModel) TableName() string {
	return "account_payables"
}

// ToDomain converts the persistence model to a domain AccountPayable entity.
func (m *AccountPayableModel) ToDomain() *finance.AccountPayable {
	ap := &finance.AccountPayable{
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
		PayableNumber:     m.PayableNumber,
		SupplierID:        m.SupplierID,
		SupplierName:      m.SupplierName,
		SourceType:        m.SourceType,
		SourceID:          m.SourceID,
		SourceNumber:      m.SourceNumber,
		TotalAmount:       m.TotalAmount,
		PaidAmount:        m.PaidAmount,
		OutstandingAmount: m.OutstandingAmount,
		Status:            m.Status,
		DueDate:           m.DueDate,
		Remark:            m.Remark,
		PaidAt:            m.PaidAt,
		ReversedAt:        m.ReversedAt,
		ReversalReason:    m.ReversalReason,
		CancelledAt:       m.CancelledAt,
		CancelReason:      m.CancelReason,
		PaymentRecords:    make([]finance.PayablePaymentRecord, len(m.PaymentRecords)),
	}
	for i, pr := range m.PaymentRecords {
		ap.PaymentRecords[i] = *pr.ToDomain()
	}
	return ap
}

// FromDomain populates the persistence model from a domain AccountPayable entity.
func (m *AccountPayableModel) FromDomain(ap *finance.AccountPayable) {
	m.FromDomainTenantAggregateRoot(ap.TenantAggregateRoot)
	m.PayableNumber = ap.PayableNumber
	m.SupplierID = ap.SupplierID
	m.SupplierName = ap.SupplierName
	m.SourceType = ap.SourceType
	m.SourceID = ap.SourceID
	m.SourceNumber = ap.SourceNumber
	m.TotalAmount = ap.TotalAmount
	m.PaidAmount = ap.PaidAmount
	m.OutstandingAmount = ap.OutstandingAmount
	m.Status = ap.Status
	m.DueDate = ap.DueDate
	m.Remark = ap.Remark
	m.PaidAt = ap.PaidAt
	m.ReversedAt = ap.ReversedAt
	m.ReversalReason = ap.ReversalReason
	m.CancelledAt = ap.CancelledAt
	m.CancelReason = ap.CancelReason
	m.PaymentRecords = make([]PayablePaymentRecordModel, len(ap.PaymentRecords))
	for i, pr := range ap.PaymentRecords {
		m.PaymentRecords[i] = *PayablePaymentRecordModelFromDomain(&pr)
	}
}

// AccountPayableModelFromDomain creates a new persistence model from a domain AccountPayable.
func AccountPayableModelFromDomain(ap *finance.AccountPayable) *AccountPayableModel {
	m := &AccountPayableModel{}
	m.FromDomain(ap)
	return m
}

// PayablePaymentRecordModel is the persistence model for PayablePaymentRecord.
type PayablePaymentRecordModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	PayableID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	PaymentVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"`
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AppliedAt        time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PayablePaymentRecordModel) TableName() string {
	return "payable_payment_records"
}

// ToDomain converts the persistence model to a domain PayablePaymentRecord.
func (m *PayablePaymentRecordModel) ToDomain() *finance.PayablePaymentRecord {
	return &finance.PayablePaymentRecord{
		ID:               m.ID,
		PayableID:        m.PayableID,
		PaymentVoucherID: m.PaymentVoucherID,
		Amount:           m.Amount,
		AppliedAt:        m.AppliedAt,
		Remark:           m.Remark,
	}
}

// FromDomain populates the persistence model from a domain PayablePaymentRecord.
func (m *PayablePaymentRecordModel) FromDomain(pr *finance.PayablePaymentRecord) {
	m.ID = pr.ID
	m.PayableID = pr.PayableID
	m.PaymentVoucherID = pr.PaymentVoucherID
	m.Amount = pr.Amount
	m.AppliedAt = pr.AppliedAt
	m.Remark = pr.Remark
}

// PayablePaymentRecordModelFromDomain creates a new persistence model from domain.
func PayablePaymentRecordModelFromDomain(pr *finance.PayablePaymentRecord) *PayablePaymentRecordModel {
	m := &PayablePaymentRecordModel{}
	m.FromDomain(pr)
	return m
}

// ReceiptVoucherModel is the persistence model for the ReceiptVoucher aggregate root.
type ReceiptVoucherModel struct {
	TenantAggregateModel
	VoucherNumber     string                      `gorm:"type:varchar(50);not null;uniqueIndex:idx_receipt_tenant_number,priority:2"`
	CustomerID        uuid.UUID                   `gorm:"type:uuid;not null;index"`
	CustomerName      string                      `gorm:"type:varchar(200);not null"`
	Amount            decimal.Decimal             `gorm:"type:decimal(18,4);not null"`
	AllocatedAmount   decimal.Decimal             `gorm:"type:decimal(18,4);not null"`
	UnallocatedAmount decimal.Decimal             `gorm:"type:decimal(18,4);not null"`
	PaymentMethod     finance.PaymentMethod       `gorm:"type:varchar(30);not null"`
	PaymentReference  string                      `gorm:"type:varchar(100)"`
	Status            finance.VoucherStatus       `gorm:"type:varchar(20);not null;default:'DRAFT';index"`
	ReceiptDate       time.Time                   `gorm:"not null"`
	Allocations       []ReceivableAllocationModel `gorm:"foreignKey:ReceiptVoucherID;references:ID"`
	Remark            string                      `gorm:"type:text"`
	ConfirmedAt       *time.Time
	ConfirmedBy       *uuid.UUID `gorm:"type:uuid"`
	CancelledAt       *time.Time
	CancelledBy       *uuid.UUID `gorm:"type:uuid"`
	CancelReason      string     `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (ReceiptVoucherModel) TableName() string {
	return "receipt_vouchers"
}

// ToDomain converts the persistence model to a domain ReceiptVoucher entity.
func (m *ReceiptVoucherModel) ToDomain() *finance.ReceiptVoucher {
	rv := &finance.ReceiptVoucher{
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
		VoucherNumber:     m.VoucherNumber,
		CustomerID:        m.CustomerID,
		CustomerName:      m.CustomerName,
		Amount:            m.Amount,
		AllocatedAmount:   m.AllocatedAmount,
		UnallocatedAmount: m.UnallocatedAmount,
		PaymentMethod:     m.PaymentMethod,
		PaymentReference:  m.PaymentReference,
		Status:            m.Status,
		ReceiptDate:       m.ReceiptDate,
		Remark:            m.Remark,
		ConfirmedAt:       m.ConfirmedAt,
		ConfirmedBy:       m.ConfirmedBy,
		CancelledAt:       m.CancelledAt,
		CancelledBy:       m.CancelledBy,
		CancelReason:      m.CancelReason,
		Allocations:       make([]finance.ReceivableAllocation, len(m.Allocations)),
	}
	for i, alloc := range m.Allocations {
		rv.Allocations[i] = *alloc.ToDomain()
	}
	return rv
}

// FromDomain populates the persistence model from a domain ReceiptVoucher entity.
func (m *ReceiptVoucherModel) FromDomain(rv *finance.ReceiptVoucher) {
	m.FromDomainTenantAggregateRoot(rv.TenantAggregateRoot)
	m.VoucherNumber = rv.VoucherNumber
	m.CustomerID = rv.CustomerID
	m.CustomerName = rv.CustomerName
	m.Amount = rv.Amount
	m.AllocatedAmount = rv.AllocatedAmount
	m.UnallocatedAmount = rv.UnallocatedAmount
	m.PaymentMethod = rv.PaymentMethod
	m.PaymentReference = rv.PaymentReference
	m.Status = rv.Status
	m.ReceiptDate = rv.ReceiptDate
	m.Remark = rv.Remark
	m.ConfirmedAt = rv.ConfirmedAt
	m.ConfirmedBy = rv.ConfirmedBy
	m.CancelledAt = rv.CancelledAt
	m.CancelledBy = rv.CancelledBy
	m.CancelReason = rv.CancelReason
	m.Allocations = make([]ReceivableAllocationModel, len(rv.Allocations))
	for i, alloc := range rv.Allocations {
		m.Allocations[i] = *ReceivableAllocationModelFromDomain(&alloc)
	}
}

// ReceiptVoucherModelFromDomain creates a new persistence model from domain.
func ReceiptVoucherModelFromDomain(rv *finance.ReceiptVoucher) *ReceiptVoucherModel {
	m := &ReceiptVoucherModel{}
	m.FromDomain(rv)
	return m
}

// ReceivableAllocationModel is the persistence model for ReceivableAllocation.
type ReceivableAllocationModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	ReceiptVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"`
	ReceivableID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	ReceivableNumber string          `gorm:"type:varchar(50);not null"`
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AllocatedAt      time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (ReceivableAllocationModel) TableName() string {
	return "receivable_allocations"
}

// ToDomain converts the persistence model to a domain ReceivableAllocation.
func (m *ReceivableAllocationModel) ToDomain() *finance.ReceivableAllocation {
	return &finance.ReceivableAllocation{
		ID:               m.ID,
		ReceiptVoucherID: m.ReceiptVoucherID,
		ReceivableID:     m.ReceivableID,
		ReceivableNumber: m.ReceivableNumber,
		Amount:           m.Amount,
		AllocatedAt:      m.AllocatedAt,
		Remark:           m.Remark,
	}
}

// FromDomain populates the persistence model from a domain ReceivableAllocation.
func (m *ReceivableAllocationModel) FromDomain(ra *finance.ReceivableAllocation) {
	m.ID = ra.ID
	m.ReceiptVoucherID = ra.ReceiptVoucherID
	m.ReceivableID = ra.ReceivableID
	m.ReceivableNumber = ra.ReceivableNumber
	m.Amount = ra.Amount
	m.AllocatedAt = ra.AllocatedAt
	m.Remark = ra.Remark
}

// ReceivableAllocationModelFromDomain creates a new persistence model from domain.
func ReceivableAllocationModelFromDomain(ra *finance.ReceivableAllocation) *ReceivableAllocationModel {
	m := &ReceivableAllocationModel{}
	m.FromDomain(ra)
	return m
}

// PaymentVoucherModel is the persistence model for the PaymentVoucher aggregate root.
type PaymentVoucherModel struct {
	TenantAggregateModel
	VoucherNumber     string                   `gorm:"type:varchar(50);not null;uniqueIndex:idx_payment_tenant_number,priority:2"`
	SupplierID        uuid.UUID                `gorm:"type:uuid;not null;index"`
	SupplierName      string                   `gorm:"type:varchar(200);not null"`
	Amount            decimal.Decimal          `gorm:"type:decimal(18,4);not null"`
	AllocatedAmount   decimal.Decimal          `gorm:"type:decimal(18,4);not null"`
	UnallocatedAmount decimal.Decimal          `gorm:"type:decimal(18,4);not null"`
	PaymentMethod     finance.PaymentMethod    `gorm:"type:varchar(30);not null"`
	PaymentReference  string                   `gorm:"type:varchar(100)"`
	Status            finance.VoucherStatus    `gorm:"type:varchar(20);not null;default:'DRAFT';index"`
	PaymentDate       time.Time                `gorm:"not null"`
	Allocations       []PayableAllocationModel `gorm:"foreignKey:PaymentVoucherID;references:ID"`
	Remark            string                   `gorm:"type:text"`
	ConfirmedAt       *time.Time
	ConfirmedBy       *uuid.UUID `gorm:"type:uuid"`
	CancelledAt       *time.Time
	CancelledBy       *uuid.UUID `gorm:"type:uuid"`
	CancelReason      string     `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PaymentVoucherModel) TableName() string {
	return "payment_vouchers"
}

// ToDomain converts the persistence model to a domain PaymentVoucher entity.
func (m *PaymentVoucherModel) ToDomain() *finance.PaymentVoucher {
	pv := &finance.PaymentVoucher{
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
		VoucherNumber:     m.VoucherNumber,
		SupplierID:        m.SupplierID,
		SupplierName:      m.SupplierName,
		Amount:            m.Amount,
		AllocatedAmount:   m.AllocatedAmount,
		UnallocatedAmount: m.UnallocatedAmount,
		PaymentMethod:     m.PaymentMethod,
		PaymentReference:  m.PaymentReference,
		Status:            m.Status,
		PaymentDate:       m.PaymentDate,
		Remark:            m.Remark,
		ConfirmedAt:       m.ConfirmedAt,
		ConfirmedBy:       m.ConfirmedBy,
		CancelledAt:       m.CancelledAt,
		CancelledBy:       m.CancelledBy,
		CancelReason:      m.CancelReason,
		Allocations:       make([]finance.PayableAllocation, len(m.Allocations)),
	}
	for i, alloc := range m.Allocations {
		pv.Allocations[i] = *alloc.ToDomain()
	}
	return pv
}

// FromDomain populates the persistence model from a domain PaymentVoucher entity.
func (m *PaymentVoucherModel) FromDomain(pv *finance.PaymentVoucher) {
	m.FromDomainTenantAggregateRoot(pv.TenantAggregateRoot)
	m.VoucherNumber = pv.VoucherNumber
	m.SupplierID = pv.SupplierID
	m.SupplierName = pv.SupplierName
	m.Amount = pv.Amount
	m.AllocatedAmount = pv.AllocatedAmount
	m.UnallocatedAmount = pv.UnallocatedAmount
	m.PaymentMethod = pv.PaymentMethod
	m.PaymentReference = pv.PaymentReference
	m.Status = pv.Status
	m.PaymentDate = pv.PaymentDate
	m.Remark = pv.Remark
	m.ConfirmedAt = pv.ConfirmedAt
	m.ConfirmedBy = pv.ConfirmedBy
	m.CancelledAt = pv.CancelledAt
	m.CancelledBy = pv.CancelledBy
	m.CancelReason = pv.CancelReason
	m.Allocations = make([]PayableAllocationModel, len(pv.Allocations))
	for i, alloc := range pv.Allocations {
		m.Allocations[i] = *PayableAllocationModelFromDomain(&alloc)
	}
}

// PaymentVoucherModelFromDomain creates a new persistence model from domain.
func PaymentVoucherModelFromDomain(pv *finance.PaymentVoucher) *PaymentVoucherModel {
	m := &PaymentVoucherModel{}
	m.FromDomain(pv)
	return m
}

// PayableAllocationModel is the persistence model for PayableAllocation.
type PayableAllocationModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key"`
	PaymentVoucherID uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayableID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayableNumber    string          `gorm:"type:varchar(50);not null"`
	Amount           decimal.Decimal `gorm:"type:decimal(18,4);not null"`
	AllocatedAt      time.Time       `gorm:"not null"`
	Remark           string          `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (PayableAllocationModel) TableName() string {
	return "payable_allocations"
}

// ToDomain converts the persistence model to a domain PayableAllocation.
func (m *PayableAllocationModel) ToDomain() *finance.PayableAllocation {
	return &finance.PayableAllocation{
		ID:               m.ID,
		PaymentVoucherID: m.PaymentVoucherID,
		PayableID:        m.PayableID,
		PayableNumber:    m.PayableNumber,
		Amount:           m.Amount,
		AllocatedAt:      m.AllocatedAt,
		Remark:           m.Remark,
	}
}

// FromDomain populates the persistence model from a domain PayableAllocation.
func (m *PayableAllocationModel) FromDomain(pa *finance.PayableAllocation) {
	m.ID = pa.ID
	m.PaymentVoucherID = pa.PaymentVoucherID
	m.PayableID = pa.PayableID
	m.PayableNumber = pa.PayableNumber
	m.Amount = pa.Amount
	m.AllocatedAt = pa.AllocatedAt
	m.Remark = pa.Remark
}

// PayableAllocationModelFromDomain creates a new persistence model from domain.
func PayableAllocationModelFromDomain(pa *finance.PayableAllocation) *PayableAllocationModel {
	m := &PayableAllocationModel{}
	m.FromDomain(pa)
	return m
}

// ExpenseRecordModel is the persistence model for the ExpenseRecord aggregate root.
type ExpenseRecordModel struct {
	TenantAggregateModel
	ExpenseNumber   string                  `gorm:"type:varchar(50);not null;uniqueIndex:idx_expense_tenant_number,priority:2"`
	Category        finance.ExpenseCategory `gorm:"type:varchar(30);not null;index"`
	Amount          decimal.Decimal         `gorm:"type:decimal(18,4);not null"`
	Description     string                  `gorm:"type:varchar(500);not null"`
	IncurredAt      time.Time               `gorm:"not null;index"`
	Status          finance.ExpenseStatus   `gorm:"type:varchar(20);not null;default:'DRAFT';index"`
	PaymentStatus   finance.PaymentStatus   `gorm:"type:varchar(20);not null;default:'UNPAID';index"`
	PaymentMethod   *finance.PaymentMethod  `gorm:"type:varchar(30)"`
	PaidAt          *time.Time
	Remark          string `gorm:"type:text"`
	AttachmentURLs  string `gorm:"type:text"`
	SubmittedAt     *time.Time
	SubmittedBy     *uuid.UUID `gorm:"type:uuid"`
	ApprovedAt      *time.Time
	ApprovedBy      *uuid.UUID `gorm:"type:uuid"`
	ApprovalRemark  string     `gorm:"type:varchar(500)"`
	RejectedAt      *time.Time
	RejectedBy      *uuid.UUID `gorm:"type:uuid"`
	RejectionReason string     `gorm:"type:varchar(500)"`
	CancelledAt     *time.Time
	CancelledBy     *uuid.UUID `gorm:"type:uuid"`
	CancelReason    string     `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (ExpenseRecordModel) TableName() string {
	return "expense_records"
}

// ToDomain converts the persistence model to a domain ExpenseRecord entity.
func (m *ExpenseRecordModel) ToDomain() *finance.ExpenseRecord {
	return &finance.ExpenseRecord{
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
		ExpenseNumber:   m.ExpenseNumber,
		Category:        m.Category,
		Amount:          m.Amount,
		Description:     m.Description,
		IncurredAt:      m.IncurredAt,
		Status:          m.Status,
		PaymentStatus:   m.PaymentStatus,
		PaymentMethod:   m.PaymentMethod,
		PaidAt:          m.PaidAt,
		Remark:          m.Remark,
		AttachmentURLs:  m.AttachmentURLs,
		SubmittedAt:     m.SubmittedAt,
		SubmittedBy:     m.SubmittedBy,
		ApprovedAt:      m.ApprovedAt,
		ApprovedBy:      m.ApprovedBy,
		ApprovalRemark:  m.ApprovalRemark,
		RejectedAt:      m.RejectedAt,
		RejectedBy:      m.RejectedBy,
		RejectionReason: m.RejectionReason,
		CancelledAt:     m.CancelledAt,
		CancelledBy:     m.CancelledBy,
		CancelReason:    m.CancelReason,
	}
}

// FromDomain populates the persistence model from a domain ExpenseRecord entity.
func (m *ExpenseRecordModel) FromDomain(er *finance.ExpenseRecord) {
	m.FromDomainTenantAggregateRoot(er.TenantAggregateRoot)
	m.ExpenseNumber = er.ExpenseNumber
	m.Category = er.Category
	m.Amount = er.Amount
	m.Description = er.Description
	m.IncurredAt = er.IncurredAt
	m.Status = er.Status
	m.PaymentStatus = er.PaymentStatus
	m.PaymentMethod = er.PaymentMethod
	m.PaidAt = er.PaidAt
	m.Remark = er.Remark
	m.AttachmentURLs = er.AttachmentURLs
	m.SubmittedAt = er.SubmittedAt
	m.SubmittedBy = er.SubmittedBy
	m.ApprovedAt = er.ApprovedAt
	m.ApprovedBy = er.ApprovedBy
	m.ApprovalRemark = er.ApprovalRemark
	m.RejectedAt = er.RejectedAt
	m.RejectedBy = er.RejectedBy
	m.RejectionReason = er.RejectionReason
	m.CancelledAt = er.CancelledAt
	m.CancelledBy = er.CancelledBy
	m.CancelReason = er.CancelReason
}

// ExpenseRecordModelFromDomain creates a new persistence model from domain.
func ExpenseRecordModelFromDomain(er *finance.ExpenseRecord) *ExpenseRecordModel {
	m := &ExpenseRecordModel{}
	m.FromDomain(er)
	return m
}

// OtherIncomeRecordModel is the persistence model for the OtherIncomeRecord aggregate root.
type OtherIncomeRecordModel struct {
	TenantAggregateModel
	IncomeNumber   string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_income_tenant_number,priority:2"`
	Category       finance.IncomeCategory `gorm:"type:varchar(30);not null;index"`
	Amount         decimal.Decimal        `gorm:"type:decimal(18,4);not null"`
	Description    string                 `gorm:"type:varchar(500);not null"`
	ReceivedAt     time.Time              `gorm:"not null;index"`
	Status         finance.IncomeStatus   `gorm:"type:varchar(20);not null;default:'DRAFT';index"`
	ReceiptStatus  finance.ReceiptStatus  `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	PaymentMethod  *finance.PaymentMethod `gorm:"type:varchar(30)"`
	ActualReceived *time.Time
	Remark         string `gorm:"type:text"`
	AttachmentURLs string `gorm:"type:text"`
	ConfirmedAt    *time.Time
	ConfirmedBy    *uuid.UUID `gorm:"type:uuid"`
	CancelledAt    *time.Time
	CancelledBy    *uuid.UUID `gorm:"type:uuid"`
	CancelReason   string     `gorm:"type:varchar(500)"`
}

// TableName returns the table name for GORM
func (OtherIncomeRecordModel) TableName() string {
	return "other_income_records"
}

// ToDomain converts the persistence model to a domain OtherIncomeRecord entity.
func (m *OtherIncomeRecordModel) ToDomain() *finance.OtherIncomeRecord {
	return &finance.OtherIncomeRecord{
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
		IncomeNumber:   m.IncomeNumber,
		Category:       m.Category,
		Amount:         m.Amount,
		Description:    m.Description,
		ReceivedAt:     m.ReceivedAt,
		Status:         m.Status,
		ReceiptStatus:  m.ReceiptStatus,
		PaymentMethod:  m.PaymentMethod,
		ActualReceived: m.ActualReceived,
		Remark:         m.Remark,
		AttachmentURLs: m.AttachmentURLs,
		ConfirmedAt:    m.ConfirmedAt,
		ConfirmedBy:    m.ConfirmedBy,
		CancelledAt:    m.CancelledAt,
		CancelledBy:    m.CancelledBy,
		CancelReason:   m.CancelReason,
	}
}

// FromDomain populates the persistence model from a domain OtherIncomeRecord entity.
func (m *OtherIncomeRecordModel) FromDomain(oir *finance.OtherIncomeRecord) {
	m.FromDomainTenantAggregateRoot(oir.TenantAggregateRoot)
	m.IncomeNumber = oir.IncomeNumber
	m.Category = oir.Category
	m.Amount = oir.Amount
	m.Description = oir.Description
	m.ReceivedAt = oir.ReceivedAt
	m.Status = oir.Status
	m.ReceiptStatus = oir.ReceiptStatus
	m.PaymentMethod = oir.PaymentMethod
	m.ActualReceived = oir.ActualReceived
	m.Remark = oir.Remark
	m.AttachmentURLs = oir.AttachmentURLs
	m.ConfirmedAt = oir.ConfirmedAt
	m.ConfirmedBy = oir.ConfirmedBy
	m.CancelledAt = oir.CancelledAt
	m.CancelledBy = oir.CancelledBy
	m.CancelReason = oir.CancelReason
}

// OtherIncomeRecordModelFromDomain creates a new persistence model from domain.
func OtherIncomeRecordModelFromDomain(oir *finance.OtherIncomeRecord) *OtherIncomeRecordModel {
	m := &OtherIncomeRecordModel{}
	m.FromDomain(oir)
	return m
}

// TrialBalanceAuditLogModel is the persistence model for TrialBalanceAuditLog.
type TrialBalanceAuditLogModel struct {
	ID               uuid.UUID                  `gorm:"type:uuid;primary_key"`
	TenantID         uuid.UUID                  `gorm:"type:uuid;not null;index"`
	CheckedAt        time.Time                  `gorm:"not null;index"`
	CheckedBy        uuid.UUID                  `gorm:"type:uuid;not null;index"`
	Status           finance.TrialBalanceStatus `gorm:"type:varchar(20);not null;index"`
	TotalDebits      decimal.Decimal            `gorm:"type:decimal(18,4);not null"`
	TotalCredits     decimal.Decimal            `gorm:"type:decimal(18,4);not null"`
	NetBalance       decimal.Decimal            `gorm:"type:decimal(18,4);not null"`
	DiscrepancyCount int                        `gorm:"not null"`
	CriticalCount    int                        `gorm:"not null"`
	WarningCount     int                        `gorm:"not null"`
	DurationMs       int64                      `gorm:"not null"`
	PeriodStart      *time.Time                 `gorm:"index"`
	PeriodEnd        *time.Time                 `gorm:"index"`
	Notes            string                     `gorm:"type:text"`
	DetailsJSON      string                     `gorm:"type:text"`
	CreatedAt        time.Time                  `gorm:"autoCreateTime"`
}

// TableName returns the table name for GORM
func (TrialBalanceAuditLogModel) TableName() string {
	return "trial_balance_audit_logs"
}

// ToDomain converts the persistence model to a domain TrialBalanceAuditLog.
func (m *TrialBalanceAuditLogModel) ToDomain() *finance.TrialBalanceAuditLog {
	return &finance.TrialBalanceAuditLog{
		ID:               m.ID,
		TenantID:         m.TenantID,
		CheckedAt:        m.CheckedAt,
		CheckedBy:        m.CheckedBy,
		Status:           m.Status,
		TotalDebits:      m.TotalDebits,
		TotalCredits:     m.TotalCredits,
		NetBalance:       m.NetBalance,
		DiscrepancyCount: m.DiscrepancyCount,
		CriticalCount:    m.CriticalCount,
		WarningCount:     m.WarningCount,
		DurationMs:       m.DurationMs,
		PeriodStart:      m.PeriodStart,
		PeriodEnd:        m.PeriodEnd,
		Notes:            m.Notes,
		DetailsJSON:      m.DetailsJSON,
		CreatedAt:        m.CreatedAt,
	}
}

// FromDomain populates the persistence model from a domain TrialBalanceAuditLog.
func (m *TrialBalanceAuditLogModel) FromDomain(al *finance.TrialBalanceAuditLog) {
	m.ID = al.ID
	m.TenantID = al.TenantID
	m.CheckedAt = al.CheckedAt
	m.CheckedBy = al.CheckedBy
	m.Status = al.Status
	m.TotalDebits = al.TotalDebits
	m.TotalCredits = al.TotalCredits
	m.NetBalance = al.NetBalance
	m.DiscrepancyCount = al.DiscrepancyCount
	m.CriticalCount = al.CriticalCount
	m.WarningCount = al.WarningCount
	m.DurationMs = al.DurationMs
	m.PeriodStart = al.PeriodStart
	m.PeriodEnd = al.PeriodEnd
	m.Notes = al.Notes
	m.DetailsJSON = al.DetailsJSON
	m.CreatedAt = al.CreatedAt
}

// TrialBalanceAuditLogModelFromDomain creates a new persistence model from domain.
func TrialBalanceAuditLogModelFromDomain(al *finance.TrialBalanceAuditLog) *TrialBalanceAuditLogModel {
	m := &TrialBalanceAuditLogModel{}
	m.FromDomain(al)
	return m
}
