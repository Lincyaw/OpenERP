package identity

import (
	"regexp"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// DepartmentStatus represents the status of a department
type DepartmentStatus string

const (
	DepartmentStatusActive   DepartmentStatus = "active"
	DepartmentStatusInactive DepartmentStatus = "inactive"
)

// Department represents an organizational unit in the system
// It is an aggregate root for department-related operations
type Department struct {
	shared.TenantAggregateRoot
	Code        string            // Unique code within tenant (e.g., "SALES", "HR")
	Name        string            // Display name
	Description string            // Optional description
	ParentID    *uuid.UUID        // Parent department ID (nil for root departments)
	Path        string            // Materialized path for efficient hierarchy queries (e.g., "/root-id/parent-id/this-id")
	Level       int               // Depth level in hierarchy (0 = root)
	SortOrder   int               // Display order within same parent
	ManagerID   *uuid.UUID        // ID of the department manager (User ID)
	Status      DepartmentStatus  // Active or inactive
	Metadata    map[string]string // Additional metadata
}

// NewDepartment creates a new department with required fields
func NewDepartment(tenantID uuid.UUID, code, name string) (*Department, error) {
	if err := validateDepartmentCode(code); err != nil {
		return nil, err
	}
	if err := validateDepartmentName(name); err != nil {
		return nil, err
	}

	dept := &Department{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(strings.TrimSpace(code)),
		Name:                strings.TrimSpace(name),
		Status:              DepartmentStatusActive,
		Level:               0,
		Path:                "", // Will be set after save with ID
		Metadata:            make(map[string]string),
	}

	dept.AddDomainEvent(NewDepartmentCreatedEvent(dept))

	return dept, nil
}

// SetParent sets the parent department
func (d *Department) SetParent(parentID *uuid.UUID, parentPath string, parentLevel int) error {
	if parentID != nil && *parentID == d.ID {
		return shared.NewDomainError("INVALID_PARENT", "Department cannot be its own parent")
	}

	d.ParentID = parentID
	if parentID == nil {
		d.Path = "/" + d.ID.String()
		d.Level = 0
	} else {
		d.Path = parentPath + "/" + d.ID.String()
		d.Level = parentLevel + 1
	}

	d.UpdatedAt = time.Now()
	d.IncrementVersion()

	return nil
}

// UpdatePath updates the materialized path after initial creation
// This should be called after the department is saved and has an ID
func (d *Department) UpdatePath(parentPath string) {
	if d.ParentID == nil {
		d.Path = "/" + d.ID.String()
	} else {
		d.Path = parentPath + "/" + d.ID.String()
	}
}

// SetName sets the department name
func (d *Department) SetName(name string) error {
	if err := validateDepartmentName(name); err != nil {
		return err
	}

	d.Name = strings.TrimSpace(name)
	d.UpdatedAt = time.Now()
	d.IncrementVersion()

	return nil
}

// SetDescription sets the department description
func (d *Department) SetDescription(description string) {
	d.Description = description
	d.UpdatedAt = time.Now()
	d.IncrementVersion()
}

// SetManager sets the department manager
func (d *Department) SetManager(managerID *uuid.UUID) {
	d.ManagerID = managerID
	d.UpdatedAt = time.Now()
	d.IncrementVersion()

	d.AddDomainEvent(NewDepartmentManagerChangedEvent(d))
}

// SetSortOrder sets the display order
func (d *Department) SetSortOrder(order int) {
	d.SortOrder = order
	d.UpdatedAt = time.Now()
	d.IncrementVersion()
}

// Activate activates the department
func (d *Department) Activate() error {
	if d.Status == DepartmentStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Department is already active")
	}

	d.Status = DepartmentStatusActive
	d.UpdatedAt = time.Now()
	d.IncrementVersion()

	return nil
}

// Deactivate deactivates the department
func (d *Department) Deactivate() error {
	if d.Status == DepartmentStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Department is already inactive")
	}

	d.Status = DepartmentStatusInactive
	d.UpdatedAt = time.Now()
	d.IncrementVersion()

	return nil
}

// IsActive returns true if department is active
func (d *Department) IsActive() bool {
	return d.Status == DepartmentStatusActive
}

// IsRoot returns true if this is a root department (no parent)
func (d *Department) IsRoot() bool {
	return d.ParentID == nil
}

// IsAncestorOf checks if this department is an ancestor of another department
// by comparing materialized paths
func (d *Department) IsAncestorOf(otherPath string) bool {
	return strings.HasPrefix(otherPath, d.Path+"/")
}

// IsDescendantOf checks if this department is a descendant of another department
func (d *Department) IsDescendantOf(ancestorPath string) bool {
	return strings.HasPrefix(d.Path, ancestorPath+"/")
}

// GetAncestorIDs extracts all ancestor IDs from the path
func (d *Department) GetAncestorIDs() []uuid.UUID {
	if d.Path == "" {
		return nil
	}

	parts := strings.Split(strings.Trim(d.Path, "/"), "/")
	// Last element is self, exclude it
	if len(parts) <= 1 {
		return nil
	}

	ancestors := make([]uuid.UUID, 0, len(parts)-1)
	for i := 0; i < len(parts)-1; i++ {
		if id, err := uuid.Parse(parts[i]); err == nil {
			ancestors = append(ancestors, id)
		}
	}

	return ancestors
}

// SetMetadata sets a metadata key-value pair
func (d *Department) SetMetadata(key, value string) {
	if d.Metadata == nil {
		d.Metadata = make(map[string]string)
	}
	d.Metadata[key] = value
	d.UpdatedAt = time.Now()
	d.IncrementVersion()
}

// GetMetadata gets a metadata value by key
func (d *Department) GetMetadata(key string) (string, bool) {
	if d.Metadata == nil {
		return "", false
	}
	v, ok := d.Metadata[key]
	return v, ok
}

// Update updates the department's basic information
func (d *Department) Update(name, description string) error {
	if err := d.SetName(name); err != nil {
		return err
	}
	d.SetDescription(description)

	d.AddDomainEvent(NewDepartmentUpdatedEvent(d))

	return nil
}

// Validation functions

func validateDepartmentCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return shared.NewDomainError("INVALID_DEPARTMENT_CODE", "Department code cannot be empty")
	}
	if len(code) < 2 {
		return shared.NewDomainError("INVALID_DEPARTMENT_CODE", "Department code must be at least 2 characters")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_DEPARTMENT_CODE", "Department code cannot exceed 50 characters")
	}

	// Allow alphanumeric, underscore, and hyphen
	codeRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	if !codeRegex.MatchString(code) {
		return shared.NewDomainError("INVALID_DEPARTMENT_CODE", "Department code must start with a letter and contain only letters, numbers, underscores, and hyphens")
	}

	return nil
}

func validateDepartmentName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return shared.NewDomainError("INVALID_DEPARTMENT_NAME", "Department name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_DEPARTMENT_NAME", "Department name cannot exceed 200 characters")
	}
	return nil
}
