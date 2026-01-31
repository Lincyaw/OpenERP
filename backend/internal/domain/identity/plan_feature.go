package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FeatureKey represents a unique identifier for a feature
type FeatureKey string

// Predefined feature keys for the system
const (
	// Core features
	FeatureMultiWarehouse    FeatureKey = "multi_warehouse"
	FeatureBatchManagement   FeatureKey = "batch_management"
	FeatureSerialTracking    FeatureKey = "serial_tracking"
	FeatureMultiCurrency     FeatureKey = "multi_currency"
	FeatureAdvancedReporting FeatureKey = "advanced_reporting"
	FeatureAPIAccess         FeatureKey = "api_access"
	FeatureCustomFields      FeatureKey = "custom_fields"
	FeatureAuditLog          FeatureKey = "audit_log"
	FeatureDataExport        FeatureKey = "data_export"
	FeatureDataImport        FeatureKey = "data_import"

	// Trade features
	FeatureSalesOrders      FeatureKey = "sales_orders"
	FeaturePurchaseOrders   FeatureKey = "purchase_orders"
	FeatureSalesReturns     FeatureKey = "sales_returns"
	FeaturePurchaseReturns  FeatureKey = "purchase_returns"
	FeatureQuotations       FeatureKey = "quotations"
	FeaturePriceManagement  FeatureKey = "price_management"
	FeatureDiscountRules    FeatureKey = "discount_rules"
	FeatureCreditManagement FeatureKey = "credit_management"

	// Finance features
	FeatureReceivables      FeatureKey = "receivables"
	FeaturePayables         FeatureKey = "payables"
	FeatureReconciliation   FeatureKey = "reconciliation"
	FeatureExpenseTracking  FeatureKey = "expense_tracking"
	FeatureFinancialReports FeatureKey = "financial_reports"

	// Advanced features
	FeatureWorkflowApproval FeatureKey = "workflow_approval"
	FeatureNotifications    FeatureKey = "notifications"
	FeatureIntegrations     FeatureKey = "integrations"
	FeatureWhiteLabeling    FeatureKey = "white_labeling"
	FeaturePrioritySupport  FeatureKey = "priority_support"
	FeatureDedicatedSupport FeatureKey = "dedicated_support"
	FeatureSLA              FeatureKey = "sla"
)

// PlanFeature represents a feature mapping for a subscription plan
// It defines which features are available for each plan and their limits
type PlanFeature struct {
	ID          uuid.UUID
	PlanID      TenantPlan // The subscription plan (free, basic, pro, enterprise)
	FeatureKey  FeatureKey // Unique identifier for the feature
	Enabled     bool       // Whether the feature is enabled for this plan
	Limit       *int       // Optional limit for the feature (nil = unlimited)
	Description string     // Human-readable description of the feature
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewPlanFeature creates a new PlanFeature with the given parameters
func NewPlanFeature(planID TenantPlan, featureKey FeatureKey, enabled bool, description string) *PlanFeature {
	now := time.Now()
	return &PlanFeature{
		ID:          uuid.New(),
		PlanID:      planID,
		FeatureKey:  featureKey,
		Enabled:     enabled,
		Limit:       nil,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewPlanFeatureWithLimit creates a new PlanFeature with a limit
func NewPlanFeatureWithLimit(planID TenantPlan, featureKey FeatureKey, enabled bool, limit int, description string) *PlanFeature {
	pf := NewPlanFeature(planID, featureKey, enabled, description)
	pf.Limit = &limit
	return pf
}

// SetLimit sets the limit for this feature
func (pf *PlanFeature) SetLimit(limit int) {
	pf.Limit = &limit
	pf.UpdatedAt = time.Now()
}

// ClearLimit removes the limit for this feature (makes it unlimited)
func (pf *PlanFeature) ClearLimit() {
	pf.Limit = nil
	pf.UpdatedAt = time.Now()
}

// Enable enables this feature
func (pf *PlanFeature) Enable() {
	pf.Enabled = true
	pf.UpdatedAt = time.Now()
}

// Disable disables this feature
func (pf *PlanFeature) Disable() {
	pf.Enabled = false
	pf.UpdatedAt = time.Now()
}

// IsUnlimited returns true if the feature has no limit
func (pf *PlanFeature) IsUnlimited() bool {
	return pf.Limit == nil
}

// GetLimit returns the limit value, or -1 if unlimited
func (pf *PlanFeature) GetLimit() int {
	if pf.Limit == nil {
		return -1
	}
	return *pf.Limit
}

// PlanFeatureRepository defines the interface for plan feature persistence
type PlanFeatureRepository interface {
	// FindByID finds a plan feature by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*PlanFeature, error)

	// FindByPlan finds all features for a specific plan
	FindByPlan(ctx context.Context, planID TenantPlan) ([]PlanFeature, error)

	// FindByPlanAndFeature finds a specific feature for a plan
	FindByPlanAndFeature(ctx context.Context, planID TenantPlan, featureKey FeatureKey) (*PlanFeature, error)

	// FindEnabledByPlan finds all enabled features for a plan
	FindEnabledByPlan(ctx context.Context, planID TenantPlan) ([]PlanFeature, error)

	// HasFeature checks if a plan has a specific feature enabled
	HasFeature(ctx context.Context, planID TenantPlan, featureKey FeatureKey) (bool, error)

	// GetFeatureLimit returns the limit for a feature in a plan (nil if unlimited or not found)
	GetFeatureLimit(ctx context.Context, planID TenantPlan, featureKey FeatureKey) (*int, error)

	// Save creates or updates a plan feature
	Save(ctx context.Context, feature *PlanFeature) error

	// SaveBatch creates or updates multiple plan features
	SaveBatch(ctx context.Context, features []PlanFeature) error

	// Delete deletes a plan feature
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByPlan deletes all features for a plan
	DeleteByPlan(ctx context.Context, planID TenantPlan) error
}

// DefaultPlanFeatures returns the default feature set for a given plan
// This defines which features are available for each subscription tier
func DefaultPlanFeatures(plan TenantPlan) []PlanFeature {
	switch plan {
	case TenantPlanFree:
		return defaultFreePlanFeatures()
	case TenantPlanBasic:
		return defaultBasicPlanFeatures()
	case TenantPlanPro:
		return defaultProPlanFeatures()
	case TenantPlanEnterprise:
		return defaultEnterprisePlanFeatures()
	default:
		return defaultFreePlanFeatures()
	}
}

// defaultFreePlanFeatures returns features for the free plan
func defaultFreePlanFeatures() []PlanFeature {
	plan := TenantPlanFree
	features := []PlanFeature{
		// Core features - limited
		*NewPlanFeature(plan, FeatureMultiWarehouse, false, "Multiple warehouse management"),
		*NewPlanFeature(plan, FeatureBatchManagement, false, "Batch/lot tracking"),
		*NewPlanFeature(plan, FeatureSerialTracking, false, "Serial number tracking"),
		*NewPlanFeature(plan, FeatureMultiCurrency, false, "Multi-currency support"),
		*NewPlanFeature(plan, FeatureAdvancedReporting, false, "Advanced analytics and reports"),
		*NewPlanFeature(plan, FeatureAPIAccess, false, "API access for integrations"),
		*NewPlanFeature(plan, FeatureCustomFields, false, "Custom fields on entities"),
		*NewPlanFeature(plan, FeatureAuditLog, false, "Audit log tracking"),
		*NewPlanFeature(plan, FeatureDataExport, true, "Export data to CSV/Excel"),
		*NewPlanFeatureWithLimit(plan, FeatureDataImport, true, 100, "Import data from CSV (100 rows/import)"),

		// Trade features - basic only
		*NewPlanFeature(plan, FeatureSalesOrders, true, "Create and manage sales orders"),
		*NewPlanFeature(plan, FeaturePurchaseOrders, true, "Create and manage purchase orders"),
		*NewPlanFeature(plan, FeatureSalesReturns, true, "Process sales returns"),
		*NewPlanFeature(plan, FeaturePurchaseReturns, true, "Process purchase returns"),
		*NewPlanFeature(plan, FeatureQuotations, false, "Create quotations"),
		*NewPlanFeature(plan, FeaturePriceManagement, false, "Advanced price management"),
		*NewPlanFeature(plan, FeatureDiscountRules, false, "Discount rules engine"),
		*NewPlanFeature(plan, FeatureCreditManagement, false, "Customer credit management"),

		// Finance features - basic only
		*NewPlanFeature(plan, FeatureReceivables, true, "Accounts receivable tracking"),
		*NewPlanFeature(plan, FeaturePayables, true, "Accounts payable tracking"),
		*NewPlanFeature(plan, FeatureReconciliation, false, "Account reconciliation"),
		*NewPlanFeature(plan, FeatureExpenseTracking, false, "Expense tracking"),
		*NewPlanFeature(plan, FeatureFinancialReports, false, "Financial reports"),

		// Advanced features - none
		*NewPlanFeature(plan, FeatureWorkflowApproval, false, "Workflow approval system"),
		*NewPlanFeature(plan, FeatureNotifications, false, "Email/SMS notifications"),
		*NewPlanFeature(plan, FeatureIntegrations, false, "Third-party integrations"),
		*NewPlanFeature(plan, FeatureWhiteLabeling, false, "White-label branding"),
		*NewPlanFeature(plan, FeaturePrioritySupport, false, "Priority support"),
		*NewPlanFeature(plan, FeatureDedicatedSupport, false, "Dedicated support manager"),
		*NewPlanFeature(plan, FeatureSLA, false, "Service level agreement"),
	}
	return features
}

// defaultBasicPlanFeatures returns features for the basic plan
func defaultBasicPlanFeatures() []PlanFeature {
	plan := TenantPlanBasic
	features := []PlanFeature{
		// Core features - some enabled
		*NewPlanFeature(plan, FeatureMultiWarehouse, true, "Multiple warehouse management"),
		*NewPlanFeature(plan, FeatureBatchManagement, true, "Batch/lot tracking"),
		*NewPlanFeature(plan, FeatureSerialTracking, false, "Serial number tracking"),
		*NewPlanFeature(plan, FeatureMultiCurrency, false, "Multi-currency support"),
		*NewPlanFeature(plan, FeatureAdvancedReporting, false, "Advanced analytics and reports"),
		*NewPlanFeature(plan, FeatureAPIAccess, false, "API access for integrations"),
		*NewPlanFeature(plan, FeatureCustomFields, false, "Custom fields on entities"),
		*NewPlanFeature(plan, FeatureAuditLog, true, "Audit log tracking"),
		*NewPlanFeature(plan, FeatureDataExport, true, "Export data to CSV/Excel"),
		*NewPlanFeatureWithLimit(plan, FeatureDataImport, true, 1000, "Import data from CSV (1000 rows/import)"),

		// Trade features - most enabled
		*NewPlanFeature(plan, FeatureSalesOrders, true, "Create and manage sales orders"),
		*NewPlanFeature(plan, FeaturePurchaseOrders, true, "Create and manage purchase orders"),
		*NewPlanFeature(plan, FeatureSalesReturns, true, "Process sales returns"),
		*NewPlanFeature(plan, FeaturePurchaseReturns, true, "Process purchase returns"),
		*NewPlanFeature(plan, FeatureQuotations, true, "Create quotations"),
		*NewPlanFeature(plan, FeaturePriceManagement, true, "Advanced price management"),
		*NewPlanFeature(plan, FeatureDiscountRules, false, "Discount rules engine"),
		*NewPlanFeature(plan, FeatureCreditManagement, true, "Customer credit management"),

		// Finance features - most enabled
		*NewPlanFeature(plan, FeatureReceivables, true, "Accounts receivable tracking"),
		*NewPlanFeature(plan, FeaturePayables, true, "Accounts payable tracking"),
		*NewPlanFeature(plan, FeatureReconciliation, true, "Account reconciliation"),
		*NewPlanFeature(plan, FeatureExpenseTracking, true, "Expense tracking"),
		*NewPlanFeature(plan, FeatureFinancialReports, false, "Financial reports"),

		// Advanced features - limited
		*NewPlanFeature(plan, FeatureWorkflowApproval, false, "Workflow approval system"),
		*NewPlanFeature(plan, FeatureNotifications, true, "Email/SMS notifications"),
		*NewPlanFeature(plan, FeatureIntegrations, false, "Third-party integrations"),
		*NewPlanFeature(plan, FeatureWhiteLabeling, false, "White-label branding"),
		*NewPlanFeature(plan, FeaturePrioritySupport, false, "Priority support"),
		*NewPlanFeature(plan, FeatureDedicatedSupport, false, "Dedicated support manager"),
		*NewPlanFeature(plan, FeatureSLA, false, "Service level agreement"),
	}
	return features
}

// defaultProPlanFeatures returns features for the pro plan
func defaultProPlanFeatures() []PlanFeature {
	plan := TenantPlanPro
	features := []PlanFeature{
		// Core features - most enabled
		*NewPlanFeature(plan, FeatureMultiWarehouse, true, "Multiple warehouse management"),
		*NewPlanFeature(plan, FeatureBatchManagement, true, "Batch/lot tracking"),
		*NewPlanFeature(plan, FeatureSerialTracking, true, "Serial number tracking"),
		*NewPlanFeature(plan, FeatureMultiCurrency, true, "Multi-currency support"),
		*NewPlanFeature(plan, FeatureAdvancedReporting, true, "Advanced analytics and reports"),
		*NewPlanFeature(plan, FeatureAPIAccess, true, "API access for integrations"),
		*NewPlanFeature(plan, FeatureCustomFields, true, "Custom fields on entities"),
		*NewPlanFeature(plan, FeatureAuditLog, true, "Audit log tracking"),
		*NewPlanFeature(plan, FeatureDataExport, true, "Export data to CSV/Excel"),
		*NewPlanFeatureWithLimit(plan, FeatureDataImport, true, 10000, "Import data from CSV (10000 rows/import)"),

		// Trade features - all enabled
		*NewPlanFeature(plan, FeatureSalesOrders, true, "Create and manage sales orders"),
		*NewPlanFeature(plan, FeaturePurchaseOrders, true, "Create and manage purchase orders"),
		*NewPlanFeature(plan, FeatureSalesReturns, true, "Process sales returns"),
		*NewPlanFeature(plan, FeaturePurchaseReturns, true, "Process purchase returns"),
		*NewPlanFeature(plan, FeatureQuotations, true, "Create quotations"),
		*NewPlanFeature(plan, FeaturePriceManagement, true, "Advanced price management"),
		*NewPlanFeature(plan, FeatureDiscountRules, true, "Discount rules engine"),
		*NewPlanFeature(plan, FeatureCreditManagement, true, "Customer credit management"),

		// Finance features - all enabled
		*NewPlanFeature(plan, FeatureReceivables, true, "Accounts receivable tracking"),
		*NewPlanFeature(plan, FeaturePayables, true, "Accounts payable tracking"),
		*NewPlanFeature(plan, FeatureReconciliation, true, "Account reconciliation"),
		*NewPlanFeature(plan, FeatureExpenseTracking, true, "Expense tracking"),
		*NewPlanFeature(plan, FeatureFinancialReports, true, "Financial reports"),

		// Advanced features - most enabled
		*NewPlanFeature(plan, FeatureWorkflowApproval, true, "Workflow approval system"),
		*NewPlanFeature(plan, FeatureNotifications, true, "Email/SMS notifications"),
		*NewPlanFeature(plan, FeatureIntegrations, true, "Third-party integrations"),
		*NewPlanFeature(plan, FeatureWhiteLabeling, false, "White-label branding"),
		*NewPlanFeature(plan, FeaturePrioritySupport, true, "Priority support"),
		*NewPlanFeature(plan, FeatureDedicatedSupport, false, "Dedicated support manager"),
		*NewPlanFeature(plan, FeatureSLA, false, "Service level agreement"),
	}
	return features
}

// defaultEnterprisePlanFeatures returns features for the enterprise plan
func defaultEnterprisePlanFeatures() []PlanFeature {
	plan := TenantPlanEnterprise
	features := []PlanFeature{
		// Core features - all enabled, unlimited
		*NewPlanFeature(plan, FeatureMultiWarehouse, true, "Multiple warehouse management"),
		*NewPlanFeature(plan, FeatureBatchManagement, true, "Batch/lot tracking"),
		*NewPlanFeature(plan, FeatureSerialTracking, true, "Serial number tracking"),
		*NewPlanFeature(plan, FeatureMultiCurrency, true, "Multi-currency support"),
		*NewPlanFeature(plan, FeatureAdvancedReporting, true, "Advanced analytics and reports"),
		*NewPlanFeature(plan, FeatureAPIAccess, true, "API access for integrations"),
		*NewPlanFeature(plan, FeatureCustomFields, true, "Custom fields on entities"),
		*NewPlanFeature(plan, FeatureAuditLog, true, "Audit log tracking"),
		*NewPlanFeature(plan, FeatureDataExport, true, "Export data to CSV/Excel"),
		*NewPlanFeature(plan, FeatureDataImport, true, "Import data from CSV (unlimited)"),

		// Trade features - all enabled
		*NewPlanFeature(plan, FeatureSalesOrders, true, "Create and manage sales orders"),
		*NewPlanFeature(plan, FeaturePurchaseOrders, true, "Create and manage purchase orders"),
		*NewPlanFeature(plan, FeatureSalesReturns, true, "Process sales returns"),
		*NewPlanFeature(plan, FeaturePurchaseReturns, true, "Process purchase returns"),
		*NewPlanFeature(plan, FeatureQuotations, true, "Create quotations"),
		*NewPlanFeature(plan, FeaturePriceManagement, true, "Advanced price management"),
		*NewPlanFeature(plan, FeatureDiscountRules, true, "Discount rules engine"),
		*NewPlanFeature(plan, FeatureCreditManagement, true, "Customer credit management"),

		// Finance features - all enabled
		*NewPlanFeature(plan, FeatureReceivables, true, "Accounts receivable tracking"),
		*NewPlanFeature(plan, FeaturePayables, true, "Accounts payable tracking"),
		*NewPlanFeature(plan, FeatureReconciliation, true, "Account reconciliation"),
		*NewPlanFeature(plan, FeatureExpenseTracking, true, "Expense tracking"),
		*NewPlanFeature(plan, FeatureFinancialReports, true, "Financial reports"),

		// Advanced features - all enabled
		*NewPlanFeature(plan, FeatureWorkflowApproval, true, "Workflow approval system"),
		*NewPlanFeature(plan, FeatureNotifications, true, "Email/SMS notifications"),
		*NewPlanFeature(plan, FeatureIntegrations, true, "Third-party integrations"),
		*NewPlanFeature(plan, FeatureWhiteLabeling, true, "White-label branding"),
		*NewPlanFeature(plan, FeaturePrioritySupport, true, "Priority support"),
		*NewPlanFeature(plan, FeatureDedicatedSupport, true, "Dedicated support manager"),
		*NewPlanFeature(plan, FeatureSLA, true, "Service level agreement"),
	}
	return features
}

// GetAllFeatureKeys returns all defined feature keys
func GetAllFeatureKeys() []FeatureKey {
	return []FeatureKey{
		FeatureMultiWarehouse,
		FeatureBatchManagement,
		FeatureSerialTracking,
		FeatureMultiCurrency,
		FeatureAdvancedReporting,
		FeatureAPIAccess,
		FeatureCustomFields,
		FeatureAuditLog,
		FeatureDataExport,
		FeatureDataImport,
		FeatureSalesOrders,
		FeaturePurchaseOrders,
		FeatureSalesReturns,
		FeaturePurchaseReturns,
		FeatureQuotations,
		FeaturePriceManagement,
		FeatureDiscountRules,
		FeatureCreditManagement,
		FeatureReceivables,
		FeaturePayables,
		FeatureReconciliation,
		FeatureExpenseTracking,
		FeatureFinancialReports,
		FeatureWorkflowApproval,
		FeatureNotifications,
		FeatureIntegrations,
		FeatureWhiteLabeling,
		FeaturePrioritySupport,
		FeatureDedicatedSupport,
		FeatureSLA,
	}
}

// IsValidFeatureKey checks if a feature key is valid
func IsValidFeatureKey(key FeatureKey) bool {
	for _, k := range GetAllFeatureKeys() {
		if k == key {
			return true
		}
	}
	return false
}

// PlanHasFeature is a helper function to check if a plan has a specific feature enabled
// based on the default feature definitions
func PlanHasFeature(plan TenantPlan, featureKey FeatureKey) bool {
	features := DefaultPlanFeatures(plan)
	for _, f := range features {
		if f.FeatureKey == featureKey {
			return f.Enabled
		}
	}
	return false
}

// GetPlanFeatureLimit returns the limit for a feature in a plan based on default definitions
// Returns nil if the feature is unlimited or not found
func GetPlanFeatureLimit(plan TenantPlan, featureKey FeatureKey) *int {
	features := DefaultPlanFeatures(plan)
	for _, f := range features {
		if f.FeatureKey == featureKey {
			return f.Limit
		}
	}
	return nil
}
