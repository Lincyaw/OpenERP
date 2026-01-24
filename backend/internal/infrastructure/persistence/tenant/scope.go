// Package tenant provides multi-tenant database scoping for GORM.
//
// This package implements automatic tenant_id filtering to prevent cross-tenant
// data access at the repository layer. It extracts the tenant ID from the request
// context and automatically applies WHERE tenant_id = ? conditions to all queries.
//
// Usage:
//
//	db := tenant.NewTenantDB(gormDB)
//	scopedDB := db.WithContext(ctx) // automatically applies tenant filtering
//	scopedDB.Find(&products) // WHERE tenant_id = 'xxx' is auto-added
package tenant

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrTenantIDRequired is returned when tenant_id is required but not found
var ErrTenantIDRequired = errors.New("tenant_id is required but not found in context")

// ErrInvalidTenantID is returned when tenant_id format is invalid
var ErrInvalidTenantID = errors.New("invalid tenant_id format")

// TenantScope applies tenant filtering to GORM queries
func TenantScope(tenantID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// TenantScopeString applies tenant filtering using string tenant ID
func TenantScopeString(tenantID string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant_id = ?", tenantID)
	}
}

// TenantCreateScope sets tenant_id on create operations
func TenantCreateScope(tenantID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Set("tenant_id", tenantID)
	}
}

// TenantDB wraps GORM DB with automatic tenant scoping
type TenantDB struct {
	db           *gorm.DB
	tenantColumn string
	required     bool
}

// Config holds configuration for TenantDB
type Config struct {
	// TenantColumn is the name of the tenant ID column (default: "tenant_id")
	TenantColumn string
	// Required determines if tenant_id is mandatory (default: true)
	Required bool
}

// DefaultConfig returns default TenantDB configuration
func DefaultConfig() Config {
	return Config{
		TenantColumn: "tenant_id",
		Required:     true,
	}
}

// NewTenantDB creates a new TenantDB with default configuration
func NewTenantDB(db *gorm.DB) *TenantDB {
	return NewTenantDBWithConfig(db, DefaultConfig())
}

// NewTenantDBWithConfig creates a new TenantDB with custom configuration
func NewTenantDBWithConfig(db *gorm.DB, cfg Config) *TenantDB {
	if cfg.TenantColumn == "" {
		cfg.TenantColumn = "tenant_id"
	}
	return &TenantDB{
		db:           db,
		tenantColumn: cfg.TenantColumn,
		required:     cfg.Required,
	}
}

// DB returns the underlying GORM DB without tenant scoping
// Use with caution - this bypasses tenant isolation
func (t *TenantDB) DB() *gorm.DB {
	return t.db
}

// WithContext returns a GORM DB scoped to the tenant from context.
// It extracts tenant_id from the context (set by tenant middleware)
// and automatically applies the tenant filter to all queries.
//
// If tenant_id is not found in context and Required is true, it returns
// a DB that will error on any operation.
func (t *TenantDB) WithContext(ctx context.Context) *gorm.DB {
	tenantID := logger.GetTenantID(ctx)

	if tenantID == "" {
		if t.required {
			// Return a DB that will error on execution
			db := t.db.WithContext(ctx)
			_ = db.AddError(ErrTenantIDRequired)
			return db
		}
		// If not required, return DB without tenant scope
		return t.db.WithContext(ctx)
	}

	// Validate UUID format
	if _, err := uuid.Parse(tenantID); err != nil {
		db := t.db.WithContext(ctx)
		_ = db.AddError(ErrInvalidTenantID)
		return db
	}

	// Apply tenant scope
	return t.db.WithContext(ctx).Scopes(TenantScopeString(tenantID))
}

// WithTenant returns a GORM DB scoped to a specific tenant ID.
// Use this when you have the tenant ID directly rather than from context.
func (t *TenantDB) WithTenant(tenantID uuid.UUID) *gorm.DB {
	if tenantID == uuid.Nil {
		if t.required {
			db := t.db
			_ = db.AddError(ErrTenantIDRequired)
			return db
		}
		return t.db
	}
	return t.db.Scopes(TenantScope(tenantID))
}

// WithTenantString returns a GORM DB scoped to a specific tenant ID string.
func (t *TenantDB) WithTenantString(tenantID string) *gorm.DB {
	if tenantID == "" {
		if t.required {
			db := t.db
			_ = db.AddError(ErrTenantIDRequired)
			return db
		}
		return t.db
	}

	// Validate UUID format
	if _, err := uuid.Parse(tenantID); err != nil {
		db := t.db
		_ = db.AddError(ErrInvalidTenantID)
		return db
	}

	return t.db.Scopes(TenantScopeString(tenantID))
}

// ForTenant creates a new TenantDB instance scoped to a specific context.
// This is useful for creating a scoped DB that can be passed around.
func (t *TenantDB) ForTenant(ctx context.Context, tenantID uuid.UUID) *gorm.DB {
	return t.db.WithContext(ctx).Scopes(TenantScope(tenantID))
}

// Transaction executes a function within a database transaction with tenant scope
func (t *TenantDB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tenantID := logger.GetTenantID(ctx)

	if tenantID == "" && t.required {
		return ErrTenantIDRequired
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tenantID != "" {
			tx = tx.Scopes(TenantScopeString(tenantID))
		}
		return fn(tx)
	})
}

// Unscoped returns the underlying DB without any tenant scoping.
// WARNING: Use this with extreme caution as it bypasses tenant isolation.
// This should only be used for system-level operations or migrations.
func (t *TenantDB) Unscoped() *gorm.DB {
	return t.db
}

// SetRequired changes whether tenant_id is required
func (t *TenantDB) SetRequired(required bool) *TenantDB {
	return &TenantDB{
		db:           t.db,
		tenantColumn: t.tenantColumn,
		required:     required,
	}
}
