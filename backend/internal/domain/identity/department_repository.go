package identity

import (
	"context"

	"github.com/google/uuid"
)

// DepartmentRepository defines the interface for department persistence
type DepartmentRepository interface {
	// Create saves a new department
	Create(ctx context.Context, dept *Department) error

	// Update updates an existing department
	Update(ctx context.Context, dept *Department) error

	// Delete removes a department by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID finds a department by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Department, error)

	// FindByCode finds a department by code within a tenant
	FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Department, error)

	// FindByTenantID finds all departments for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Department, error)

	// FindChildren finds all direct children of a department
	FindChildren(ctx context.Context, parentID uuid.UUID) ([]*Department, error)

	// FindDescendants finds all descendants of a department (using materialized path)
	FindDescendants(ctx context.Context, dept *Department) ([]*Department, error)

	// FindByIDs finds departments by multiple IDs
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*Department, error)

	// FindRootDepartments finds all root departments (no parent) for a tenant
	FindRootDepartments(ctx context.Context, tenantID uuid.UUID) ([]*Department, error)

	// FindByManagerID finds departments managed by a specific user
	FindByManagerID(ctx context.Context, managerID uuid.UUID) ([]*Department, error)

	// CountByTenantID counts departments for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// ExistsByCode checks if a department code exists within a tenant
	ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error)

	// GetAllDepartmentIDsInSubtree returns all department IDs in a subtree (including the root)
	// This is used for department-scoped data filtering
	GetAllDepartmentIDsInSubtree(ctx context.Context, departmentID uuid.UUID) ([]uuid.UUID, error)
}
