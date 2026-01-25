// Package datascope provides data-level permission filtering for GORM queries.
//
// This package implements automatic data scope filtering based on user roles
// and their data scope configurations. It supports four scope types:
//   - ALL: Access all data within the tenant
//   - SELF: Only data created by the current user
//   - DEPARTMENT: Data within the user's department (future support)
//   - CUSTOM: Custom-defined scope values (e.g., specific regions)
//
// Usage:
//
//	filter := datascope.NewFilter(ctx, roles)
//	scopedDB := filter.Apply(db, "sales_order")
//	scopedDB.Find(&orders) // WHERE created_by = ? is auto-added for SELF scope
package datascope

import (
	"context"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataScopeContextKey is the context key for data scopes
type DataScopeContextKey string

const (
	// ScopesKey is the context key for storing user's data scopes
	ScopesKey DataScopeContextKey = "data_scopes"
	// UserRolesKey is the context key for storing user's roles
	UserRolesKey DataScopeContextKey = "user_roles"
)

// Filter applies data scope filtering to GORM queries
type Filter struct {
	ctx        context.Context
	userID     uuid.UUID
	dataScopes map[string]identity.DataScope // resource -> data scope
}

// NewFilter creates a new DataScope filter from roles
func NewFilter(ctx context.Context, roles []identity.Role) *Filter {
	userIDStr := logger.GetUserID(ctx)
	var userID uuid.UUID
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}

	// Merge data scopes from all roles
	// Higher permission level wins (ALL > DEPARTMENT > SELF)
	dataScopes := make(map[string]identity.DataScope)
	for _, role := range roles {
		if !role.IsEnabled {
			continue
		}
		for _, ds := range role.DataScopes {
			existing, exists := dataScopes[ds.Resource]
			if !exists || compareScopeLevel(ds.ScopeType, existing.ScopeType) > 0 {
				dataScopes[ds.Resource] = ds
			}
		}
	}

	return &Filter{
		ctx:        ctx,
		userID:     userID,
		dataScopes: dataScopes,
	}
}

// NewFilterFromContext creates a Filter from context if scopes are stored there
func NewFilterFromContext(ctx context.Context) *Filter {
	userIDStr := logger.GetUserID(ctx)
	var userID uuid.UUID
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}

	// Try to get data scopes from context
	dataScopes := make(map[string]identity.DataScope)
	if scopes, ok := ctx.Value(ScopesKey).(map[string]identity.DataScope); ok {
		dataScopes = scopes
	}

	return &Filter{
		ctx:        ctx,
		userID:     userID,
		dataScopes: dataScopes,
	}
}

// WithDataScopes adds data scopes to context
func WithDataScopes(ctx context.Context, roles []identity.Role) context.Context {
	dataScopes := make(map[string]identity.DataScope)
	for _, role := range roles {
		if !role.IsEnabled {
			continue
		}
		for _, ds := range role.DataScopes {
			existing, exists := dataScopes[ds.Resource]
			if !exists || compareScopeLevel(ds.ScopeType, existing.ScopeType) > 0 {
				dataScopes[ds.Resource] = ds
			}
		}
	}
	return context.WithValue(ctx, ScopesKey, dataScopes)
}

// Apply applies data scope filtering for a specific resource
func (f *Filter) Apply(db *gorm.DB, resource string) *gorm.DB {
	ds, exists := f.dataScopes[resource]
	if !exists {
		// No data scope configured for this resource, default to ALL
		return db
	}

	switch ds.ScopeType {
	case identity.DataScopeAll:
		// No additional filtering needed
		return db

	case identity.DataScopeSelf:
		// Filter to only records created by the current user
		if f.userID == uuid.Nil {
			// No user ID - return empty result (safety)
			return db.Where("1 = 0")
		}
		return db.Where("created_by = ?", f.userID)

	case identity.DataScopeDepartment:
		// TODO: Implement department filtering
		// This requires a department_id field and department membership
		// For now, fall back to SELF
		if f.userID == uuid.Nil {
			return db.Where("1 = 0")
		}
		return db.Where("created_by = ?", f.userID)

	case identity.DataScopeCustom:
		// Custom scope filtering based on scope values
		// The scope values define the allowed values for a specific field
		if len(ds.ScopeValues) == 0 {
			// No scope values defined - return empty result
			return db.Where("1 = 0")
		}
		// Custom scopes typically filter by a specific field
		// The field name is derived from the resource name (e.g., warehouse_id)
		// For now, we'll use scope values as IDs in a generic created_by-like field
		return db.Where("created_by IN ?", ds.ScopeValues)

	default:
		// Unknown scope type - fall back to ALL
		return db
	}
}

// ApplyToQuery applies data scope filtering and returns a GORM scope function
func (f *Filter) ApplyToQuery(resource string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return f.Apply(db, resource)
	}
}

// GetScopeType returns the scope type for a resource
func (f *Filter) GetScopeType(resource string) identity.DataScopeType {
	if ds, exists := f.dataScopes[resource]; exists {
		return ds.ScopeType
	}
	return identity.DataScopeAll
}

// HasScope returns true if there's a scope defined for the resource
func (f *Filter) HasScope(resource string) bool {
	_, exists := f.dataScopes[resource]
	return exists
}

// GetUserID returns the current user ID
func (f *Filter) GetUserID() uuid.UUID {
	return f.userID
}

// CanAccessAll returns true if the user has ALL scope for the resource
func (f *Filter) CanAccessAll(resource string) bool {
	ds, exists := f.dataScopes[resource]
	if !exists {
		return true // No scope = full access
	}
	return ds.ScopeType == identity.DataScopeAll
}

// IsOwner checks if the current user is the owner (creator) of a record
func (f *Filter) IsOwner(createdBy *uuid.UUID) bool {
	if createdBy == nil || f.userID == uuid.Nil {
		return false
	}
	return *createdBy == f.userID
}

// ScopeFunc is a GORM scope function type
type ScopeFunc func(*gorm.DB) *gorm.DB

// DataScopeScope creates a GORM scope for data scope filtering
func DataScopeScope(ctx context.Context, resource string, roles []identity.Role) ScopeFunc {
	filter := NewFilter(ctx, roles)
	return filter.ApplyToQuery(resource)
}

// DataScopeScopeFromContext creates a GORM scope using scopes from context
func DataScopeScopeFromContext(ctx context.Context, resource string) ScopeFunc {
	filter := NewFilterFromContext(ctx)
	return filter.ApplyToQuery(resource)
}

// compareScopeLevel compares two scope types and returns:
//
//	positive if a > b (a has more access)
//	negative if a < b (a has less access)
//	zero if equal
func compareScopeLevel(a, b identity.DataScopeType) int {
	levels := map[identity.DataScopeType]int{
		identity.DataScopeAll:        100,
		identity.DataScopeDepartment: 50,
		identity.DataScopeCustom:     40,
		identity.DataScopeSelf:       10,
	}

	levelA := levels[a]
	levelB := levels[b]

	return levelA - levelB
}

// MergeScopes merges multiple data scopes, keeping the highest permission level
func MergeScopes(scopesList ...[]identity.DataScope) map[string]identity.DataScope {
	merged := make(map[string]identity.DataScope)
	for _, scopes := range scopesList {
		for _, ds := range scopes {
			existing, exists := merged[ds.Resource]
			if !exists || compareScopeLevel(ds.ScopeType, existing.ScopeType) > 0 {
				merged[ds.Resource] = ds
			}
		}
	}
	return merged
}
