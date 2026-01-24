package tenant

import (
	"strings"

	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantCallback provides GORM callback hooks for automatic tenant filtering
type TenantCallback struct {
	tenantColumn string
	required     bool
}

// NewTenantCallback creates a new tenant callback handler
func NewTenantCallback(tenantColumn string, required bool) *TenantCallback {
	if tenantColumn == "" {
		tenantColumn = "tenant_id"
	}
	return &TenantCallback{
		tenantColumn: tenantColumn,
		required:     required,
	}
}

// RegisterCallbacks registers tenant callbacks with GORM
func (tc *TenantCallback) RegisterCallbacks(db *gorm.DB) {
	// Register query callback - add tenant filter
	_ = db.Callback().Query().Before("gorm:query").Register("tenant:before_query", tc.beforeQuery)

	// Register update callback - ensure tenant filter
	_ = db.Callback().Update().Before("gorm:update").Register("tenant:before_update", tc.beforeUpdate)

	// Register delete callback - ensure tenant filter
	_ = db.Callback().Delete().Before("gorm:delete").Register("tenant:before_delete", tc.beforeDelete)

	// Register row query callback - add tenant filter
	_ = db.Callback().Row().Before("gorm:row").Register("tenant:before_row", tc.beforeQuery)

	// Note: Create callback is not registered because tenant_id should be set
	// explicitly by the application when creating entities
}

// beforeQuery adds tenant filter to SELECT queries
func (tc *TenantCallback) beforeQuery(db *gorm.DB) {
	tc.addTenantFilter(db)
}

// beforeUpdate adds tenant filter to UPDATE queries
func (tc *TenantCallback) beforeUpdate(db *gorm.DB) {
	tc.addTenantFilter(db)
}

// beforeDelete adds tenant filter to DELETE queries
func (tc *TenantCallback) beforeDelete(db *gorm.DB) {
	tc.addTenantFilter(db)
}

// addTenantFilter adds tenant filtering to the query
func (tc *TenantCallback) addTenantFilter(db *gorm.DB) {
	if db.Statement.Context == nil {
		return
	}

	// Skip if unscoped
	if db.Statement.Unscoped {
		return
	}

	// Skip if already has tenant condition
	if tc.hasTenantCondition(db) {
		return
	}

	// Get tenant ID from context
	tenantID := logger.GetTenantID(db.Statement.Context)
	if tenantID == "" {
		if tc.required {
			_ = db.AddError(ErrTenantIDRequired)
		}
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(tenantID); err != nil {
		_ = db.AddError(ErrInvalidTenantID)
		return
	}

	// Add tenant filter using GORM's clause
	db.Statement.AddClause(clause.Where{
		Exprs: []clause.Expression{
			clause.Eq{
				Column: clause.Column{Table: clause.CurrentTable, Name: tc.tenantColumn},
				Value:  tenantID,
			},
		},
	})
}

// hasTenantCondition checks if tenant_id condition is already present
func (tc *TenantCallback) hasTenantCondition(db *gorm.DB) bool {
	// Check if there's a manual scope applied via Unscoped
	if db.Statement.Unscoped {
		return true
	}

	// Check existing where clauses for tenant_id
	if whereClause, ok := db.Statement.Clauses["WHERE"]; ok {
		if where, ok := whereClause.Expression.(clause.Where); ok {
			for _, expr := range where.Exprs {
				if tc.exprContainsTenant(expr) {
					return true
				}
			}
		}
	}

	// Also check the built SQL if available
	sql := db.Statement.SQL.String()
	if sql != "" && strings.Contains(sql, tc.tenantColumn) {
		return true
	}

	return false
}

// exprContainsTenant checks if an expression contains tenant_id column
func (tc *TenantCallback) exprContainsTenant(expr clause.Expression) bool {
	switch e := expr.(type) {
	case clause.Eq:
		if col, ok := e.Column.(clause.Column); ok {
			return col.Name == tc.tenantColumn
		}
	case clause.IN:
		if col, ok := e.Column.(clause.Column); ok {
			return col.Name == tc.tenantColumn
		}
	case clause.AndConditions:
		for _, cond := range e.Exprs {
			if tc.exprContainsTenant(cond) {
				return true
			}
		}
	case clause.OrConditions:
		for _, cond := range e.Exprs {
			if tc.exprContainsTenant(cond) {
				return true
			}
		}
	}
	return false
}

// EnableAutoTenantFilter enables automatic tenant filtering on a GORM DB instance
// This registers callbacks that automatically add tenant_id filtering to all queries
func EnableAutoTenantFilter(db *gorm.DB, required bool) {
	tc := NewTenantCallback("tenant_id", required)
	tc.RegisterCallbacks(db)
}

// DisableAutoTenantFilter removes the tenant callbacks (not recommended in production)
func DisableAutoTenantFilter(db *gorm.DB) {
	// Note: GORM doesn't provide a clean way to remove callbacks
	// This is mainly for testing purposes
	_ = db.Callback().Query().Remove("tenant:before_query")
	_ = db.Callback().Update().Remove("tenant:before_update")
	_ = db.Callback().Delete().Remove("tenant:before_delete")
	_ = db.Callback().Row().Remove("tenant:before_row")
}
