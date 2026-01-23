package catalog

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// Aggregate type constant
const AggregateTypeCategory = "Category"

// Event type constants
const (
	EventTypeCategoryCreated       = "CategoryCreated"
	EventTypeCategoryUpdated       = "CategoryUpdated"
	EventTypeCategoryStatusChanged = "CategoryStatusChanged"
	EventTypeCategoryDeleted       = "CategoryDeleted"
)

// CategoryCreatedEvent is published when a new category is created
type CategoryCreatedEvent struct {
	shared.BaseDomainEvent
	CategoryID uuid.UUID  `json:"category_id"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	Level      int        `json:"level"`
}

// NewCategoryCreatedEvent creates a new CategoryCreatedEvent
func NewCategoryCreatedEvent(category *Category) *CategoryCreatedEvent {
	return &CategoryCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCategoryCreated, AggregateTypeCategory, category.ID, category.TenantID),
		CategoryID:      category.ID,
		Code:            category.Code,
		Name:            category.Name,
		ParentID:        category.ParentID,
		Level:           category.Level,
	}
}

// CategoryUpdatedEvent is published when a category is updated
type CategoryUpdatedEvent struct {
	shared.BaseDomainEvent
	CategoryID uuid.UUID `json:"category_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
}

// NewCategoryUpdatedEvent creates a new CategoryUpdatedEvent
func NewCategoryUpdatedEvent(category *Category) *CategoryUpdatedEvent {
	return &CategoryUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCategoryUpdated, AggregateTypeCategory, category.ID, category.TenantID),
		CategoryID:      category.ID,
		Code:            category.Code,
		Name:            category.Name,
	}
}

// CategoryStatusChangedEvent is published when a category's status changes
type CategoryStatusChangedEvent struct {
	shared.BaseDomainEvent
	CategoryID uuid.UUID      `json:"category_id"`
	Code       string         `json:"code"`
	OldStatus  CategoryStatus `json:"old_status"`
	NewStatus  CategoryStatus `json:"new_status"`
}

// NewCategoryStatusChangedEvent creates a new CategoryStatusChangedEvent
func NewCategoryStatusChangedEvent(category *Category, oldStatus, newStatus CategoryStatus) *CategoryStatusChangedEvent {
	return &CategoryStatusChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCategoryStatusChanged, AggregateTypeCategory, category.ID, category.TenantID),
		CategoryID:      category.ID,
		Code:            category.Code,
		OldStatus:       oldStatus,
		NewStatus:       newStatus,
	}
}

// CategoryDeletedEvent is published when a category is deleted
type CategoryDeletedEvent struct {
	shared.BaseDomainEvent
	CategoryID uuid.UUID  `json:"category_id"`
	Code       string     `json:"code"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
}

// NewCategoryDeletedEvent creates a new CategoryDeletedEvent
func NewCategoryDeletedEvent(category *Category) *CategoryDeletedEvent {
	return &CategoryDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeCategoryDeleted, AggregateTypeCategory, category.ID, category.TenantID),
		CategoryID:      category.ID,
		Code:            category.Code,
		ParentID:        category.ParentID,
	}
}
