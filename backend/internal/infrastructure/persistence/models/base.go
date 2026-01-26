package models

import (
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// BaseModel provides common persistence fields for all models.
// It maps to the domain's BaseEntity.
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// ToDomain converts BaseModel to domain BaseEntity
func (m *BaseModel) ToDomain() shared.BaseEntity {
	return shared.BaseEntity{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// FromDomainBaseEntity populates BaseModel from domain BaseEntity
func (m *BaseModel) FromDomainBaseEntity(e shared.BaseEntity) {
	m.ID = e.ID
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
}

// AggregateModel provides common persistence fields for aggregate roots.
// It extends BaseModel with version for optimistic locking.
type AggregateModel struct {
	BaseModel
	Version int `gorm:"not null;default:1"`
}

// FromDomainAggregateRoot populates AggregateModel from domain BaseAggregateRoot
func (m *AggregateModel) FromDomainAggregateRoot(a shared.BaseAggregateRoot) {
	m.FromDomainBaseEntity(a.BaseEntity)
	m.Version = a.Version
}

// TenantAggregateModel provides common persistence fields for tenant-scoped aggregate roots.
// It extends AggregateModel with tenant ID and creator info.
type TenantAggregateModel struct {
	AggregateModel
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	CreatedBy *uuid.UUID `gorm:"type:uuid;index"`
}

// FromDomainTenantAggregateRoot populates TenantAggregateModel from domain TenantAggregateRoot
func (m *TenantAggregateModel) FromDomainTenantAggregateRoot(t shared.TenantAggregateRoot) {
	m.FromDomainAggregateRoot(t.BaseAggregateRoot)
	m.TenantID = t.TenantID
	m.CreatedBy = t.CreatedBy
}

// PopulateTenantAggregateRoot populates a domain TenantAggregateRoot from persistence model
func (m *TenantAggregateModel) PopulateTenantAggregateRoot(t *shared.TenantAggregateRoot) {
	t.BaseAggregateRoot.BaseEntity.ID = m.ID
	t.BaseAggregateRoot.BaseEntity.CreatedAt = m.CreatedAt
	t.BaseAggregateRoot.BaseEntity.UpdatedAt = m.UpdatedAt
	t.BaseAggregateRoot.Version = m.Version
	t.TenantID = m.TenantID
	t.CreatedBy = m.CreatedBy
}
