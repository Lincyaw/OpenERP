package models

import (
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerModel is the persistence model for the Customer domain entity.
type CustomerModel struct {
	TenantAggregateModel
	Code        string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_customer_tenant_code,priority:2"`
	Name        string                 `gorm:"type:varchar(200);not null"`
	ShortName   string                 `gorm:"type:varchar(100)"`
	Type        partner.CustomerType   `gorm:"type:varchar(20);not null;default:'individual'"`
	Level       partner.CustomerLevel  `gorm:"type:varchar(20);not null;default:'normal'"`
	Status      partner.CustomerStatus `gorm:"type:varchar(20);not null;default:'active'"`
	ContactName string                 `gorm:"type:varchar(100)"`
	Phone       string                 `gorm:"type:varchar(50);index"`
	Email       string                 `gorm:"type:varchar(200);index"`
	Address     string                 `gorm:"type:text"`
	City        string                 `gorm:"type:varchar(100)"`
	Province    string                 `gorm:"type:varchar(100)"`
	PostalCode  string                 `gorm:"type:varchar(20)"`
	Country     string                 `gorm:"type:varchar(100);default:'中国'"`
	TaxID       string                 `gorm:"type:varchar(50)"`
	CreditLimit decimal.Decimal        `gorm:"type:decimal(18,4);not null;default:0"`
	Balance     decimal.Decimal        `gorm:"type:decimal(18,4);not null;default:0"`
	Notes       string                 `gorm:"type:text"`
	SortOrder   int                    `gorm:"not null;default:0"`
	Attributes  string                 `gorm:"type:jsonb"`
}

// TableName returns the table name for GORM
func (CustomerModel) TableName() string {
	return "customers"
}

// ToDomain converts the persistence model to a domain Customer entity.
func (m *CustomerModel) ToDomain() *partner.Customer {
	return &partner.Customer{
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
		Code:        m.Code,
		Name:        m.Name,
		ShortName:   m.ShortName,
		Type:        m.Type,
		Level:       m.Level,
		Status:      m.Status,
		ContactName: m.ContactName,
		Phone:       m.Phone,
		Email:       m.Email,
		Address:     m.Address,
		City:        m.City,
		Province:    m.Province,
		PostalCode:  m.PostalCode,
		Country:     m.Country,
		TaxID:       m.TaxID,
		CreditLimit: m.CreditLimit,
		Balance:     m.Balance,
		Notes:       m.Notes,
		SortOrder:   m.SortOrder,
		Attributes:  m.Attributes,
	}
}

// FromDomain populates the persistence model from a domain Customer entity.
func (m *CustomerModel) FromDomain(c *partner.Customer) {
	m.FromDomainTenantAggregateRoot(c.TenantAggregateRoot)
	m.Code = c.Code
	m.Name = c.Name
	m.ShortName = c.ShortName
	m.Type = c.Type
	m.Level = c.Level
	m.Status = c.Status
	m.ContactName = c.ContactName
	m.Phone = c.Phone
	m.Email = c.Email
	m.Address = c.Address
	m.City = c.City
	m.Province = c.Province
	m.PostalCode = c.PostalCode
	m.Country = c.Country
	m.TaxID = c.TaxID
	m.CreditLimit = c.CreditLimit
	m.Balance = c.Balance
	m.Notes = c.Notes
	m.SortOrder = c.SortOrder
	m.Attributes = c.Attributes
}

// CustomerModelFromDomain creates a new persistence model from a domain Customer entity.
func CustomerModelFromDomain(c *partner.Customer) *CustomerModel {
	m := &CustomerModel{}
	m.FromDomain(c)
	return m
}

// SupplierModel is the persistence model for the Supplier domain entity.
type SupplierModel struct {
	TenantAggregateModel
	Code        string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_supplier_tenant_code,priority:2"`
	Name        string                 `gorm:"type:varchar(200);not null"`
	ShortName   string                 `gorm:"type:varchar(100)"`
	Type        partner.SupplierType   `gorm:"type:varchar(20);not null;default:'distributor'"`
	Status      partner.SupplierStatus `gorm:"type:varchar(20);not null;default:'active'"`
	ContactName string                 `gorm:"type:varchar(100)"`
	Phone       string                 `gorm:"type:varchar(50);index"`
	Email       string                 `gorm:"type:varchar(200);index"`
	Address     string                 `gorm:"type:text"`
	City        string                 `gorm:"type:varchar(100)"`
	Province    string                 `gorm:"type:varchar(100)"`
	PostalCode  string                 `gorm:"type:varchar(20)"`
	Country     string                 `gorm:"type:varchar(100);default:'中国'"`
	TaxID       string                 `gorm:"type:varchar(50)"`
	BankName    string                 `gorm:"type:varchar(200)"`
	BankAccount string                 `gorm:"type:varchar(100)"`
	CreditDays  int                    `gorm:"not null;default:0"`
	CreditLimit decimal.Decimal        `gorm:"type:decimal(18,4);not null;default:0"`
	Balance     decimal.Decimal        `gorm:"type:decimal(18,4);not null;default:0"`
	Rating      int                    `gorm:"not null;default:0;check:rating >= 0"`
	Notes       string                 `gorm:"type:text"`
	SortOrder   int                    `gorm:"not null;default:0"`
	Attributes  string                 `gorm:"type:jsonb"`
}

// TableName returns the table name for GORM
func (SupplierModel) TableName() string {
	return "suppliers"
}

// ToDomain converts the persistence model to a domain Supplier entity.
func (m *SupplierModel) ToDomain() *partner.Supplier {
	return &partner.Supplier{
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
		Code:        m.Code,
		Name:        m.Name,
		ShortName:   m.ShortName,
		Type:        m.Type,
		Status:      m.Status,
		ContactName: m.ContactName,
		Phone:       m.Phone,
		Email:       m.Email,
		Address:     m.Address,
		City:        m.City,
		Province:    m.Province,
		PostalCode:  m.PostalCode,
		Country:     m.Country,
		TaxID:       m.TaxID,
		BankName:    m.BankName,
		BankAccount: m.BankAccount,
		CreditDays:  m.CreditDays,
		CreditLimit: m.CreditLimit,
		Balance:     m.Balance,
		Rating:      m.Rating,
		Notes:       m.Notes,
		SortOrder:   m.SortOrder,
		Attributes:  m.Attributes,
	}
}

// FromDomain populates the persistence model from a domain Supplier entity.
func (m *SupplierModel) FromDomain(s *partner.Supplier) {
	m.FromDomainTenantAggregateRoot(s.TenantAggregateRoot)
	m.Code = s.Code
	m.Name = s.Name
	m.ShortName = s.ShortName
	m.Type = s.Type
	m.Status = s.Status
	m.ContactName = s.ContactName
	m.Phone = s.Phone
	m.Email = s.Email
	m.Address = s.Address
	m.City = s.City
	m.Province = s.Province
	m.PostalCode = s.PostalCode
	m.Country = s.Country
	m.TaxID = s.TaxID
	m.BankName = s.BankName
	m.BankAccount = s.BankAccount
	m.CreditDays = s.CreditDays
	m.CreditLimit = s.CreditLimit
	m.Balance = s.Balance
	m.Rating = s.Rating
	m.Notes = s.Notes
	m.SortOrder = s.SortOrder
	m.Attributes = s.Attributes
}

// SupplierModelFromDomain creates a new persistence model from a domain Supplier entity.
func SupplierModelFromDomain(s *partner.Supplier) *SupplierModel {
	m := &SupplierModel{}
	m.FromDomain(s)
	return m
}

// WarehouseModel is the persistence model for the Warehouse domain entity.
type WarehouseModel struct {
	TenantAggregateModel
	Code        string                  `gorm:"type:varchar(50);not null;uniqueIndex:idx_warehouse_tenant_code,priority:2"`
	Name        string                  `gorm:"type:varchar(200);not null"`
	ShortName   string                  `gorm:"type:varchar(100)"`
	Type        partner.WarehouseType   `gorm:"type:varchar(20);not null;default:'physical'"`
	Status      partner.WarehouseStatus `gorm:"type:varchar(20);not null;default:'active'"`
	ContactName string                  `gorm:"type:varchar(100)"`
	Phone       string                  `gorm:"type:varchar(50);index"`
	Email       string                  `gorm:"type:varchar(200)"`
	Address     string                  `gorm:"type:text"`
	City        string                  `gorm:"type:varchar(100)"`
	Province    string                  `gorm:"type:varchar(100)"`
	PostalCode  string                  `gorm:"type:varchar(20)"`
	Country     string                  `gorm:"type:varchar(100);default:'中国'"`
	IsDefault   bool                    `gorm:"not null;default:false"`
	Capacity    int                     `gorm:"not null;default:0"`
	Notes       string                  `gorm:"type:text"`
	SortOrder   int                     `gorm:"not null;default:0"`
	Attributes  string                  `gorm:"type:jsonb"`
}

// TableName returns the table name for GORM
func (WarehouseModel) TableName() string {
	return "warehouses"
}

// ToDomain converts the persistence model to a domain Warehouse entity.
func (m *WarehouseModel) ToDomain() *partner.Warehouse {
	return &partner.Warehouse{
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
		Code:        m.Code,
		Name:        m.Name,
		ShortName:   m.ShortName,
		Type:        m.Type,
		Status:      m.Status,
		ContactName: m.ContactName,
		Phone:       m.Phone,
		Email:       m.Email,
		Address:     m.Address,
		City:        m.City,
		Province:    m.Province,
		PostalCode:  m.PostalCode,
		Country:     m.Country,
		IsDefault:   m.IsDefault,
		Capacity:    m.Capacity,
		Notes:       m.Notes,
		SortOrder:   m.SortOrder,
		Attributes:  m.Attributes,
	}
}

// FromDomain populates the persistence model from a domain Warehouse entity.
func (m *WarehouseModel) FromDomain(w *partner.Warehouse) {
	m.FromDomainTenantAggregateRoot(w.TenantAggregateRoot)
	m.Code = w.Code
	m.Name = w.Name
	m.ShortName = w.ShortName
	m.Type = w.Type
	m.Status = w.Status
	m.ContactName = w.ContactName
	m.Phone = w.Phone
	m.Email = w.Email
	m.Address = w.Address
	m.City = w.City
	m.Province = w.Province
	m.PostalCode = w.PostalCode
	m.Country = w.Country
	m.IsDefault = w.IsDefault
	m.Capacity = w.Capacity
	m.Notes = w.Notes
	m.SortOrder = w.SortOrder
	m.Attributes = w.Attributes
}

// WarehouseModelFromDomain creates a new persistence model from a domain Warehouse entity.
func WarehouseModelFromDomain(w *partner.Warehouse) *WarehouseModel {
	m := &WarehouseModel{}
	m.FromDomain(w)
	return m
}

// CustomerLevelRecordModel is the persistence model for the CustomerLevelRecord.
// This is for storing customer level definitions in the database.
type CustomerLevelRecordModel struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	Code         string          `gorm:"type:varchar(50);not null"`
	Name         string          `gorm:"type:varchar(100);not null"`
	DiscountRate decimal.Decimal `gorm:"type:decimal(5,4);not null;default:0"`
	SortOrder    int             `gorm:"not null;default:0"`
	IsDefault    bool            `gorm:"not null;default:false"`
	IsActive     bool            `gorm:"not null;default:true"`
	Description  string          `gorm:"type:text"`
	CreatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name for GORM
func (CustomerLevelRecordModel) TableName() string {
	return "customer_levels"
}

// ToDomain converts the persistence model to a domain CustomerLevelRecord.
func (m *CustomerLevelRecordModel) ToDomain() *partner.CustomerLevelRecord {
	return &partner.CustomerLevelRecord{
		ID:           m.ID,
		TenantID:     m.TenantID,
		Code:         m.Code,
		Name:         m.Name,
		DiscountRate: m.DiscountRate,
		SortOrder:    m.SortOrder,
		IsDefault:    m.IsDefault,
		IsActive:     m.IsActive,
		Description:  m.Description,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain CustomerLevelRecord.
func (m *CustomerLevelRecordModel) FromDomain(r *partner.CustomerLevelRecord) {
	m.ID = r.ID
	m.TenantID = r.TenantID
	m.Code = r.Code
	m.Name = r.Name
	m.DiscountRate = r.DiscountRate
	m.SortOrder = r.SortOrder
	m.IsDefault = r.IsDefault
	m.IsActive = r.IsActive
	m.Description = r.Description
	m.CreatedAt = r.CreatedAt
	m.UpdatedAt = r.UpdatedAt
}

// CustomerLevelRecordModelFromDomain creates a new persistence model from a domain CustomerLevelRecord.
func CustomerLevelRecordModelFromDomain(r *partner.CustomerLevelRecord) *CustomerLevelRecordModel {
	m := &CustomerLevelRecordModel{}
	m.FromDomain(r)
	return m
}

// BalanceTransactionModel is the persistence model for the BalanceTransaction entity.
type BalanceTransactionModel struct {
	BaseModel
	TenantID        uuid.UUID                            `gorm:"type:uuid;not null;index:idx_bal_tx_tenant_time,priority:1"`
	CustomerID      uuid.UUID                            `gorm:"type:uuid;not null;index:idx_bal_tx_customer"`
	TransactionType partner.BalanceTransactionType       `gorm:"type:varchar(30);not null;index:idx_bal_tx_type"`
	Amount          decimal.Decimal                      `gorm:"type:decimal(18,4);not null"`
	BalanceBefore   decimal.Decimal                      `gorm:"type:decimal(18,4);not null"`
	BalanceAfter    decimal.Decimal                      `gorm:"type:decimal(18,4);not null"`
	SourceType      partner.BalanceTransactionSourceType `gorm:"type:varchar(30);not null;index:idx_bal_tx_source"`
	SourceID        *string                              `gorm:"type:varchar(50);index:idx_bal_tx_source"`
	Reference       string                               `gorm:"type:varchar(100)"`
	Remark          string                               `gorm:"type:varchar(500)"`
	OperatorID      *uuid.UUID                           `gorm:"type:uuid"`
	TransactionDate time.Time                            `gorm:"type:timestamptz;not null;index:idx_bal_tx_tenant_time,priority:2"`
}

// TableName returns the table name for GORM
func (BalanceTransactionModel) TableName() string {
	return "balance_transactions"
}

// ToDomain converts the persistence model to a domain BalanceTransaction entity.
func (m *BalanceTransactionModel) ToDomain() *partner.BalanceTransaction {
	return &partner.BalanceTransaction{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		TenantID:        m.TenantID,
		CustomerID:      m.CustomerID,
		TransactionType: m.TransactionType,
		Amount:          m.Amount,
		BalanceBefore:   m.BalanceBefore,
		BalanceAfter:    m.BalanceAfter,
		SourceType:      m.SourceType,
		SourceID:        m.SourceID,
		Reference:       m.Reference,
		Remark:          m.Remark,
		OperatorID:      m.OperatorID,
		TransactionDate: m.TransactionDate,
	}
}

// FromDomain populates the persistence model from a domain BalanceTransaction entity.
func (m *BalanceTransactionModel) FromDomain(t *partner.BalanceTransaction) {
	m.FromDomainBaseEntity(t.BaseEntity)
	m.TenantID = t.TenantID
	m.CustomerID = t.CustomerID
	m.TransactionType = t.TransactionType
	m.Amount = t.Amount
	m.BalanceBefore = t.BalanceBefore
	m.BalanceAfter = t.BalanceAfter
	m.SourceType = t.SourceType
	m.SourceID = t.SourceID
	m.Reference = t.Reference
	m.Remark = t.Remark
	m.OperatorID = t.OperatorID
	m.TransactionDate = t.TransactionDate
}

// BalanceTransactionModelFromDomain creates a new persistence model from a domain BalanceTransaction entity.
func BalanceTransactionModelFromDomain(t *partner.BalanceTransaction) *BalanceTransactionModel {
	m := &BalanceTransactionModel{}
	m.FromDomain(t)
	return m
}
