package shared

import (
	"time"

	"github.com/google/uuid"
)

// Entity is the base interface for all domain entities
type Entity interface {
	GetID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetID returns the entity ID
func (e *BaseEntity) GetID() uuid.UUID {
	return e.ID
}

// GetCreatedAt returns the creation timestamp
func (e *BaseEntity) GetCreatedAt() time.Time {
	return e.CreatedAt
}

// GetUpdatedAt returns the last update timestamp
func (e *BaseEntity) GetUpdatedAt() time.Time {
	return e.UpdatedAt
}

// NewBaseEntity creates a new base entity with generated ID
func NewBaseEntity() BaseEntity {
	now := time.Now()
	return BaseEntity{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
