package models

import (
	"encoding/json"
	"time"

	"github.com/erp/backend/internal/domain/integration"
	"github.com/google/uuid"
)

// ProductMappingModel is the persistence model for the ProductMapping domain entity.
type ProductMappingModel struct {
	ID                  uuid.UUID                `gorm:"type:uuid;primary_key"`
	TenantID            uuid.UUID                `gorm:"type:uuid;not null;index:idx_product_mapping_tenant,priority:1"`
	LocalProductID      uuid.UUID                `gorm:"type:uuid;not null;index:idx_product_mapping_local_product,priority:1;index:idx_product_mapping_tenant_product_platform,priority:2"`
	PlatformCode        integration.PlatformCode `gorm:"type:varchar(20);not null;index:idx_product_mapping_platform,priority:1;index:idx_product_mapping_tenant_product_platform,priority:3"`
	PlatformProductID   string                   `gorm:"type:varchar(100);not null;index:idx_product_mapping_platform_product,priority:1"`
	PlatformProductName string                   `gorm:"type:varchar(255)"`
	PlatformCategoryID  string                   `gorm:"type:varchar(50)"`
	SKUMappingsJSON     string                   `gorm:"type:jsonb;column:sku_mappings"`
	IsActive            bool                     `gorm:"not null;default:true"`
	SyncEnabled         bool                     `gorm:"not null;default:true"`
	LastSyncAt          *time.Time               `gorm:"index"`
	LastSyncStatus      integration.SyncStatus   `gorm:"type:varchar(20);not null;default:'PENDING'"`
	LastSyncError       string                   `gorm:"type:text"`
	CreatedAt           time.Time                `gorm:"not null"`
	UpdatedAt           time.Time                `gorm:"not null"`
}

// TableName returns the table name for GORM
func (ProductMappingModel) TableName() string {
	return "product_mappings"
}

// ToDomain converts the persistence model to a domain ProductMapping entity.
func (m *ProductMappingModel) ToDomain() *integration.ProductMapping {
	mapping := &integration.ProductMapping{
		ID:                  m.ID,
		TenantID:            m.TenantID,
		LocalProductID:      m.LocalProductID,
		PlatformCode:        m.PlatformCode,
		PlatformProductID:   m.PlatformProductID,
		PlatformProductName: m.PlatformProductName,
		PlatformCategoryID:  m.PlatformCategoryID,
		SKUMappings:         make([]integration.SKUMapping, 0),
		IsActive:            m.IsActive,
		SyncEnabled:         m.SyncEnabled,
		LastSyncAt:          m.LastSyncAt,
		LastSyncStatus:      m.LastSyncStatus,
		LastSyncError:       m.LastSyncError,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}

	// Parse SKU mappings from JSON
	if m.SKUMappingsJSON != "" {
		var skuMappings []integration.SKUMapping
		if err := json.Unmarshal([]byte(m.SKUMappingsJSON), &skuMappings); err == nil {
			mapping.SKUMappings = skuMappings
		}
	}

	return mapping
}

// FromDomain populates the persistence model from a domain ProductMapping entity.
func (m *ProductMappingModel) FromDomain(pm *integration.ProductMapping) {
	m.ID = pm.ID
	m.TenantID = pm.TenantID
	m.LocalProductID = pm.LocalProductID
	m.PlatformCode = pm.PlatformCode
	m.PlatformProductID = pm.PlatformProductID
	m.PlatformProductName = pm.PlatformProductName
	m.PlatformCategoryID = pm.PlatformCategoryID
	m.IsActive = pm.IsActive
	m.SyncEnabled = pm.SyncEnabled
	m.LastSyncAt = pm.LastSyncAt
	m.LastSyncStatus = pm.LastSyncStatus
	m.LastSyncError = pm.LastSyncError
	m.CreatedAt = pm.CreatedAt
	m.UpdatedAt = pm.UpdatedAt

	// Serialize SKU mappings to JSON
	if len(pm.SKUMappings) > 0 {
		if jsonBytes, err := json.Marshal(pm.SKUMappings); err == nil {
			m.SKUMappingsJSON = string(jsonBytes)
		}
	} else {
		m.SKUMappingsJSON = "[]"
	}
}

// ProductMappingModelFromDomain creates a new persistence model from a domain ProductMapping entity.
func ProductMappingModelFromDomain(pm *integration.ProductMapping) *ProductMappingModel {
	m := &ProductMappingModel{}
	m.FromDomain(pm)
	return m
}
