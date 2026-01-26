package models

import (
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductModel is the persistence model for the Product domain entity.
type ProductModel struct {
	TenantAggregateModel
	Code          string                `gorm:"type:varchar(50);not null;uniqueIndex:idx_product_tenant_code,priority:2"`
	Name          string                `gorm:"type:varchar(200);not null"`
	Description   string                `gorm:"type:text"`
	Barcode       string                `gorm:"type:varchar(50);index"`
	CategoryID    *uuid.UUID            `gorm:"type:uuid;index"`
	Unit          string                `gorm:"type:varchar(20);not null"`
	PurchasePrice decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	SellingPrice  decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	MinStock      decimal.Decimal       `gorm:"type:decimal(18,4);not null;default:0"`
	Status        catalog.ProductStatus `gorm:"type:varchar(20);not null;default:'active'"`
	SortOrder     int                   `gorm:"not null;default:0"`
	Attributes    string                `gorm:"type:jsonb"`
}

// TableName returns the table name for GORM
func (ProductModel) TableName() string {
	return "products"
}

// ToDomain converts the persistence model to a domain Product entity.
func (m *ProductModel) ToDomain() *catalog.Product {
	return &catalog.Product{
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
		Code:          m.Code,
		Name:          m.Name,
		Description:   m.Description,
		Barcode:       m.Barcode,
		CategoryID:    m.CategoryID,
		Unit:          m.Unit,
		PurchasePrice: m.PurchasePrice,
		SellingPrice:  m.SellingPrice,
		MinStock:      m.MinStock,
		Status:        m.Status,
		SortOrder:     m.SortOrder,
		Attributes:    m.Attributes,
	}
}

// FromDomain populates the persistence model from a domain Product entity.
func (m *ProductModel) FromDomain(p *catalog.Product) {
	m.FromDomainTenantAggregateRoot(p.TenantAggregateRoot)
	m.Code = p.Code
	m.Name = p.Name
	m.Description = p.Description
	m.Barcode = p.Barcode
	m.CategoryID = p.CategoryID
	m.Unit = p.Unit
	m.PurchasePrice = p.PurchasePrice
	m.SellingPrice = p.SellingPrice
	m.MinStock = p.MinStock
	m.Status = p.Status
	m.SortOrder = p.SortOrder
	m.Attributes = p.Attributes
}

// ProductModelFromDomain creates a new persistence model from a domain Product entity.
func ProductModelFromDomain(p *catalog.Product) *ProductModel {
	m := &ProductModel{}
	m.FromDomain(p)
	return m
}

// CategoryModel is the persistence model for the Category domain entity.
type CategoryModel struct {
	TenantAggregateModel
	Code        string                 `gorm:"type:varchar(50);not null;uniqueIndex:idx_category_tenant_code,priority:2"`
	Name        string                 `gorm:"type:varchar(100);not null"`
	Description string                 `gorm:"type:text"`
	ParentID    *uuid.UUID             `gorm:"type:uuid;index"`
	Path        string                 `gorm:"type:varchar(500);not null;index"`
	Level       int                    `gorm:"not null;default:0"`
	SortOrder   int                    `gorm:"not null;default:0"`
	Status      catalog.CategoryStatus `gorm:"type:varchar(20);not null;default:'active'"`
}

// TableName returns the table name for GORM
func (CategoryModel) TableName() string {
	return "categories"
}

// ToDomain converts the persistence model to a domain Category entity.
func (m *CategoryModel) ToDomain() *catalog.Category {
	return &catalog.Category{
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
		Description: m.Description,
		ParentID:    m.ParentID,
		Path:        m.Path,
		Level:       m.Level,
		SortOrder:   m.SortOrder,
		Status:      m.Status,
	}
}

// FromDomain populates the persistence model from a domain Category entity.
func (m *CategoryModel) FromDomain(c *catalog.Category) {
	m.FromDomainTenantAggregateRoot(c.TenantAggregateRoot)
	m.Code = c.Code
	m.Name = c.Name
	m.Description = c.Description
	m.ParentID = c.ParentID
	m.Path = c.Path
	m.Level = c.Level
	m.SortOrder = c.SortOrder
	m.Status = c.Status
}

// CategoryModelFromDomain creates a new persistence model from a domain Category entity.
func CategoryModelFromDomain(c *catalog.Category) *CategoryModel {
	m := &CategoryModel{}
	m.FromDomain(c)
	return m
}

// ProductUnitModel is the persistence model for the ProductUnit entity.
type ProductUnitModel struct {
	ID                    uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	TenantID              uuid.UUID       `gorm:"type:uuid;not null;index"`
	ProductID             uuid.UUID       `gorm:"type:uuid;not null;index;uniqueIndex:idx_product_unit_code,priority:2"`
	UnitCode              string          `gorm:"type:varchar(20);not null;uniqueIndex:idx_product_unit_code,priority:3"`
	UnitName              string          `gorm:"type:varchar(50);not null"`
	ConversionRate        decimal.Decimal `gorm:"type:decimal(18,6);not null"`
	DefaultPurchasePrice  decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	DefaultSellingPrice   decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"`
	IsDefaultPurchaseUnit bool            `gorm:"not null;default:false"`
	IsDefaultSalesUnit    bool            `gorm:"not null;default:false"`
	SortOrder             int             `gorm:"not null;default:0"`
	CreatedAt             time.Time       `gorm:"not null;autoCreateTime"`
	UpdatedAt             time.Time       `gorm:"not null;autoUpdateTime"`
}

// TableName returns the table name for GORM
func (ProductUnitModel) TableName() string {
	return "product_units"
}

// ToDomain converts the persistence model to a domain ProductUnit entity.
func (m *ProductUnitModel) ToDomain() *catalog.ProductUnit {
	return &catalog.ProductUnit{
		ID:                    m.ID,
		TenantID:              m.TenantID,
		ProductID:             m.ProductID,
		UnitCode:              m.UnitCode,
		UnitName:              m.UnitName,
		ConversionRate:        m.ConversionRate,
		DefaultPurchasePrice:  m.DefaultPurchasePrice,
		DefaultSellingPrice:   m.DefaultSellingPrice,
		IsDefaultPurchaseUnit: m.IsDefaultPurchaseUnit,
		IsDefaultSalesUnit:    m.IsDefaultSalesUnit,
		SortOrder:             m.SortOrder,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

// FromDomain populates the persistence model from a domain ProductUnit entity.
func (m *ProductUnitModel) FromDomain(pu *catalog.ProductUnit) {
	m.ID = pu.ID
	m.TenantID = pu.TenantID
	m.ProductID = pu.ProductID
	m.UnitCode = pu.UnitCode
	m.UnitName = pu.UnitName
	m.ConversionRate = pu.ConversionRate
	m.DefaultPurchasePrice = pu.DefaultPurchasePrice
	m.DefaultSellingPrice = pu.DefaultSellingPrice
	m.IsDefaultPurchaseUnit = pu.IsDefaultPurchaseUnit
	m.IsDefaultSalesUnit = pu.IsDefaultSalesUnit
	m.SortOrder = pu.SortOrder
	m.CreatedAt = pu.CreatedAt
	m.UpdatedAt = pu.UpdatedAt
}

// ProductUnitModelFromDomain creates a new persistence model from a domain ProductUnit entity.
func ProductUnitModelFromDomain(pu *catalog.ProductUnit) *ProductUnitModel {
	m := &ProductUnitModel{}
	m.FromDomain(pu)
	return m
}
