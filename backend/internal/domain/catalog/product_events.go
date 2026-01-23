package catalog

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant
const AggregateTypeProduct = "Product"

// Event type constants
const (
	EventTypeProductCreated       = "ProductCreated"
	EventTypeProductUpdated       = "ProductUpdated"
	EventTypeProductStatusChanged = "ProductStatusChanged"
	EventTypeProductPriceChanged  = "ProductPriceChanged"
	EventTypeProductDeleted       = "ProductDeleted"
)

// ProductCreatedEvent is published when a new product is created
type ProductCreatedEvent struct {
	shared.BaseDomainEvent
	ProductID  uuid.UUID  `json:"product_id"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Unit       string     `json:"unit"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
}

// NewProductCreatedEvent creates a new ProductCreatedEvent
func NewProductCreatedEvent(product *Product) *ProductCreatedEvent {
	return &ProductCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeProductCreated, AggregateTypeProduct, product.ID, product.TenantID),
		ProductID:       product.ID,
		Code:            product.Code,
		Name:            product.Name,
		Unit:            product.Unit,
		CategoryID:      product.CategoryID,
	}
}

// ProductUpdatedEvent is published when a product is updated
type ProductUpdatedEvent struct {
	shared.BaseDomainEvent
	ProductID   uuid.UUID  `json:"product_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
}

// NewProductUpdatedEvent creates a new ProductUpdatedEvent
func NewProductUpdatedEvent(product *Product) *ProductUpdatedEvent {
	return &ProductUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeProductUpdated, AggregateTypeProduct, product.ID, product.TenantID),
		ProductID:       product.ID,
		Code:            product.Code,
		Name:            product.Name,
		Description:     product.Description,
		CategoryID:      product.CategoryID,
	}
}

// ProductStatusChangedEvent is published when a product's status changes
type ProductStatusChangedEvent struct {
	shared.BaseDomainEvent
	ProductID uuid.UUID     `json:"product_id"`
	Code      string        `json:"code"`
	OldStatus ProductStatus `json:"old_status"`
	NewStatus ProductStatus `json:"new_status"`
}

// NewProductStatusChangedEvent creates a new ProductStatusChangedEvent
func NewProductStatusChangedEvent(product *Product, oldStatus, newStatus ProductStatus) *ProductStatusChangedEvent {
	return &ProductStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeProductStatusChanged, AggregateTypeProduct, product.ID, product.TenantID),
		ProductID:       product.ID,
		Code:            product.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// ProductPriceChangedEvent is published when a product's prices change
type ProductPriceChangedEvent struct {
	shared.BaseDomainEvent
	ProductID        uuid.UUID       `json:"product_id"`
	Code             string          `json:"code"`
	OldPurchasePrice decimal.Decimal `json:"old_purchase_price"`
	NewPurchasePrice decimal.Decimal `json:"new_purchase_price"`
	OldSellingPrice  decimal.Decimal `json:"old_selling_price"`
	NewSellingPrice  decimal.Decimal `json:"new_selling_price"`
}

// NewProductPriceChangedEvent creates a new ProductPriceChangedEvent
func NewProductPriceChangedEvent(product *Product, oldPurchasePrice, oldSellingPrice decimal.Decimal) *ProductPriceChangedEvent {
	return &ProductPriceChangedEvent{
		BaseDomainEvent:  shared.NewBaseDomainEvent(EventTypeProductPriceChanged, AggregateTypeProduct, product.ID, product.TenantID),
		ProductID:        product.ID,
		Code:             product.Code,
		OldPurchasePrice: oldPurchasePrice,
		NewPurchasePrice: product.PurchasePrice,
		OldSellingPrice:  oldSellingPrice,
		NewSellingPrice:  product.SellingPrice,
	}
}

// ProductDeletedEvent is published when a product is deleted
type ProductDeletedEvent struct {
	shared.BaseDomainEvent
	ProductID  uuid.UUID  `json:"product_id"`
	Code       string     `json:"code"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
}

// NewProductDeletedEvent creates a new ProductDeletedEvent
func NewProductDeletedEvent(product *Product) *ProductDeletedEvent {
	return &ProductDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeProductDeleted, AggregateTypeProduct, product.ID, product.TenantID),
		ProductID:       product.ID,
		Code:            product.Code,
		CategoryID:      product.CategoryID,
	}
}
