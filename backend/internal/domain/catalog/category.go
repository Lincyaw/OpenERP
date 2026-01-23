package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// MaxCategoryDepth is the maximum depth of category hierarchy
const MaxCategoryDepth = 5

// CategoryStatus represents the status of a category
type CategoryStatus string

const (
	CategoryStatusActive   CategoryStatus = "active"
	CategoryStatusInactive CategoryStatus = "inactive"
)

// Category represents a product category in the catalog
// It supports tree structure with parent-child relationships
type Category struct {
	shared.TenantAggregateRoot
	Code        string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_category_tenant_code,priority:2"`
	Name        string         `gorm:"type:varchar(100);not null"`
	Description string         `gorm:"type:text"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index"`
	Path        string         `gorm:"type:varchar(500);not null;index"` // Materialized path for tree queries
	Level       int            `gorm:"not null;default:0"`
	SortOrder   int            `gorm:"not null;default:0"`
	Status      CategoryStatus `gorm:"type:varchar(20);not null;default:'active'"`
}

// TableName returns the table name for GORM
func (Category) TableName() string {
	return "categories"
}

// NewCategory creates a new root category
func NewCategory(tenantID uuid.UUID, code, name string) (*Category, error) {
	if err := validateCategoryCode(code); err != nil {
		return nil, err
	}
	if err := validateCategoryName(name); err != nil {
		return nil, err
	}

	category := &Category{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		Status:              CategoryStatusActive,
		Level:               0,
	}
	// Root category path is just the ID
	category.Path = category.ID.String()

	category.AddDomainEvent(NewCategoryCreatedEvent(category))

	return category, nil
}

// NewChildCategory creates a new child category under a parent
func NewChildCategory(tenantID uuid.UUID, code, name string, parent *Category) (*Category, error) {
	if parent == nil {
		return nil, shared.NewDomainError("INVALID_PARENT", "Parent category is required")
	}

	if parent.Level >= MaxCategoryDepth-1 {
		return nil, shared.NewDomainError("MAX_DEPTH_EXCEEDED", fmt.Sprintf("Category depth cannot exceed %d levels", MaxCategoryDepth))
	}

	if err := validateCategoryCode(code); err != nil {
		return nil, err
	}
	if err := validateCategoryName(name); err != nil {
		return nil, err
	}

	category := &Category{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		ParentID:            &parent.ID,
		Level:               parent.Level + 1,
		Status:              CategoryStatusActive,
	}
	// Child category path is parent path + separator + child ID
	category.Path = parent.Path + "/" + category.ID.String()

	category.AddDomainEvent(NewCategoryCreatedEvent(category))

	return category, nil
}

// Update updates the category's basic information
func (c *Category) Update(name, description string) error {
	if err := validateCategoryName(name); err != nil {
		return err
	}

	c.Name = name
	c.Description = description
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCategoryUpdatedEvent(c))

	return nil
}

// UpdateCode updates the category's code
// Note: This should be used with caution as products reference category codes
func (c *Category) UpdateCode(code string) error {
	if err := validateCategoryCode(code); err != nil {
		return err
	}

	c.Code = strings.ToUpper(code)
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCategoryUpdatedEvent(c))

	return nil
}

// SetSortOrder sets the display order of the category
func (c *Category) SetSortOrder(order int) {
	c.SortOrder = order
	c.UpdatedAt = time.Now()
	c.IncrementVersion()
}

// Activate activates the category
func (c *Category) Activate() error {
	if c.Status == CategoryStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Category is already active")
	}

	c.Status = CategoryStatusActive
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCategoryStatusChangedEvent(c, CategoryStatusInactive, CategoryStatusActive))

	return nil
}

// Deactivate deactivates the category
func (c *Category) Deactivate() error {
	if c.Status == CategoryStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Category is already inactive")
	}

	c.Status = CategoryStatusInactive
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCategoryStatusChangedEvent(c, CategoryStatusActive, CategoryStatusInactive))

	return nil
}

// IsActive returns true if the category is active
func (c *Category) IsActive() bool {
	return c.Status == CategoryStatusActive
}

// IsRoot returns true if this is a root category
func (c *Category) IsRoot() bool {
	return c.ParentID == nil
}

// GetAncestorIDs returns the IDs of all ancestor categories
func (c *Category) GetAncestorIDs() []uuid.UUID {
	if c.Path == "" {
		return nil
	}

	parts := strings.Split(c.Path, "/")
	if len(parts) <= 1 {
		return nil
	}

	// Exclude the last element which is this category's ID
	ancestors := make([]uuid.UUID, 0, len(parts)-1)
	for i := 0; i < len(parts)-1; i++ {
		if id, err := uuid.Parse(parts[i]); err == nil {
			ancestors = append(ancestors, id)
		}
	}

	return ancestors
}

// IsAncestorOf returns true if this category is an ancestor of the given category
func (c *Category) IsAncestorOf(other *Category) bool {
	if other == nil || other.Path == "" {
		return false
	}
	return strings.HasPrefix(other.Path, c.Path+"/")
}

// IsDescendantOf returns true if this category is a descendant of the given category
func (c *Category) IsDescendantOf(other *Category) bool {
	if other == nil {
		return false
	}
	return other.IsAncestorOf(c)
}

// validateCategoryCode validates the category code
func validateCategoryCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Category code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Category code cannot exceed 50 characters")
	}
	// Code should be alphanumeric with underscores and hyphens
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Category code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

// validateCategoryName validates the category name
func validateCategoryName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Category name cannot be empty")
	}
	if len(name) > 100 {
		return shared.NewDomainError("INVALID_NAME", "Category name cannot exceed 100 characters")
	}
	return nil
}
