package identity

import (
	"regexp"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// DataScopeType represents the type of data scope for data-level authorization
type DataScopeType string

const (
	DataScopeAll        DataScopeType = "all"        // Access all data
	DataScopeSelf       DataScopeType = "self"       // Only data created by self
	DataScopeDepartment DataScopeType = "department" // Data within the same department
	DataScopeCustom     DataScopeType = "custom"     // Custom scope defined by scope values
	DataScopeWarehouse  DataScopeType = "warehouse"  // Data within assigned warehouses (warehouse_id based)
)

// Permission represents a functional permission (resource:action pattern)
// It is a value object
type Permission struct {
	Code        string // e.g., "product:create"
	Resource    string // e.g., "product"
	Action      string // e.g., "create"
	Description string // Optional description
}

// NewPermission creates a new Permission value object
func NewPermission(resource, action string) (*Permission, error) {
	if err := validatePermissionResource(resource); err != nil {
		return nil, err
	}
	if err := validatePermissionAction(action); err != nil {
		return nil, err
	}

	resource = strings.ToLower(strings.TrimSpace(resource))
	action = strings.ToLower(strings.TrimSpace(action))

	return &Permission{
		Code:     resource + ":" + action,
		Resource: resource,
		Action:   action,
	}, nil
}

// NewPermissionWithDescription creates a new Permission with description
func NewPermissionWithDescription(resource, action, description string) (*Permission, error) {
	perm, err := NewPermission(resource, action)
	if err != nil {
		return nil, err
	}
	perm.Description = description
	return perm, nil
}

// NewPermissionFromCode creates a Permission from a code string (e.g., "product:create")
func NewPermissionFromCode(code string) (*Permission, error) {
	parts := strings.SplitN(code, ":", 2)
	if len(parts) != 2 {
		return nil, shared.NewDomainError("INVALID_PERMISSION_CODE", "Permission code must be in format 'resource:action'")
	}
	return NewPermission(parts[0], parts[1])
}

// Equals checks if two permissions are equal
func (p Permission) Equals(other Permission) bool {
	return p.Code == other.Code
}

// IsEmpty returns true if the permission is empty
func (p Permission) IsEmpty() bool {
	return p.Code == ""
}

// DataScope represents a data-level permission scope
// It is a value object
type DataScope struct {
	Resource    string        // e.g., "sales_order"
	ScopeType   DataScopeType // Type of scope
	ScopeField  string        // Field to filter on (e.g., "warehouse_id", "region_id")
	ScopeValues []string      // For custom scope: specific IDs or conditions
	Description string        // Optional description
}

// NewDataScope creates a new DataScope value object
func NewDataScope(resource string, scopeType DataScopeType) (*DataScope, error) {
	if err := validateDataScopeResource(resource); err != nil {
		return nil, err
	}
	if err := validateDataScopeType(scopeType); err != nil {
		return nil, err
	}

	return &DataScope{
		Resource:    strings.ToLower(strings.TrimSpace(resource)),
		ScopeType:   scopeType,
		ScopeField:  "",
		ScopeValues: make([]string, 0),
	}, nil
}

// NewCustomDataScope creates a DataScope with custom scope values
func NewCustomDataScope(resource string, scopeValues []string) (*DataScope, error) {
	ds, err := NewDataScope(resource, DataScopeCustom)
	if err != nil {
		return nil, err
	}

	if len(scopeValues) == 0 {
		return nil, shared.NewDomainError("INVALID_SCOPE_VALUES", "Custom data scope must have at least one scope value")
	}

	ds.ScopeValues = make([]string, len(scopeValues))
	copy(ds.ScopeValues, scopeValues)

	return ds, nil
}

// NewCustomDataScopeWithField creates a DataScope with custom scope values and a specific field
func NewCustomDataScopeWithField(resource, scopeField string, scopeValues []string) (*DataScope, error) {
	ds, err := NewCustomDataScope(resource, scopeValues)
	if err != nil {
		return nil, err
	}

	scopeField = strings.TrimSpace(scopeField)
	if scopeField == "" {
		return nil, shared.NewDomainError("INVALID_SCOPE_FIELD", "Scope field cannot be empty for custom data scope with field")
	}

	ds.ScopeField = scopeField
	return ds, nil
}

// NewWarehouseDataScope creates a DataScope for warehouse-level access
// This is specifically designed for WAREHOUSE role users who can only access
// inventory and related data within their assigned warehouses
func NewWarehouseDataScope(resource string, warehouseIDs []string) (*DataScope, error) {
	if err := validateDataScopeResource(resource); err != nil {
		return nil, err
	}

	if len(warehouseIDs) == 0 {
		return nil, shared.NewDomainError("INVALID_WAREHOUSE_IDS", "Warehouse data scope must have at least one warehouse ID")
	}

	return &DataScope{
		Resource:    strings.ToLower(strings.TrimSpace(resource)),
		ScopeType:   DataScopeWarehouse,
		ScopeField:  "warehouse_id",                      // Fixed field for warehouse scoping
		ScopeValues: append([]string{}, warehouseIDs...), // Defensive copy
	}, nil
}

// SetDescription sets the description for the data scope
func (ds *DataScope) SetDescription(description string) {
	ds.Description = description
}

// Equals checks if two data scopes are equal
func (ds DataScope) Equals(other DataScope) bool {
	if ds.Resource != other.Resource || ds.ScopeType != other.ScopeType || ds.ScopeField != other.ScopeField {
		return false
	}
	if len(ds.ScopeValues) != len(other.ScopeValues) {
		return false
	}
	// Check scope values (order matters)
	for i, v := range ds.ScopeValues {
		if v != other.ScopeValues[i] {
			return false
		}
	}
	return true
}

// IsEmpty returns true if the data scope is empty
func (ds DataScope) IsEmpty() bool {
	return ds.Resource == ""
}

// Role represents a role in the RBAC system
// It is the aggregate root for role-related operations
type Role struct {
	shared.TenantAggregateRoot
	Code         string
	Name         string
	Description  string
	IsSystemRole bool // System roles cannot be deleted
	IsEnabled    bool
	SortOrder    int          // For display ordering
	Permissions  []Permission // Stored in separate table
	DataScopes   []DataScope  // Stored in separate table
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID      uuid.UUID
	TenantID    uuid.UUID
	Code        string
	Resource    string
	Action      string
	Description string
	CreatedAt   time.Time
}

// RoleDataScope represents the data scope configuration for a role
type RoleDataScope struct {
	RoleID      uuid.UUID
	TenantID    uuid.UUID
	Resource    string
	ScopeType   DataScopeType
	ScopeField  string // Field to filter on (e.g., "warehouse_id")
	ScopeValues string // JSON array for custom scopes
	Description string
	CreatedAt   time.Time
}

// NewRole creates a new role with required fields
func NewRole(tenantID uuid.UUID, code, name string) (*Role, error) {
	if err := validateRoleCode(code); err != nil {
		return nil, err
	}
	if err := validateRoleName(name); err != nil {
		return nil, err
	}

	role := &Role{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(strings.TrimSpace(code)),
		Name:                strings.TrimSpace(name),
		IsSystemRole:        false,
		IsEnabled:           true,
		Permissions:         make([]Permission, 0),
		DataScopes:          make([]DataScope, 0),
	}

	role.AddDomainEvent(NewRoleCreatedEvent(role))

	return role, nil
}

// NewSystemRole creates a new system role (cannot be deleted)
func NewSystemRole(tenantID uuid.UUID, code, name string) (*Role, error) {
	role, err := NewRole(tenantID, code, name)
	if err != nil {
		return nil, err
	}

	role.IsSystemRole = true
	return role, nil
}

// SetName sets the role name
func (r *Role) SetName(name string) error {
	if err := validateRoleName(name); err != nil {
		return err
	}

	r.Name = strings.TrimSpace(name)
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return nil
}

// SetDescription sets the role description
func (r *Role) SetDescription(description string) {
	r.Description = description
	r.UpdatedAt = time.Now()
	r.IncrementVersion()
}

// SetSortOrder sets the sort order for display
func (r *Role) SetSortOrder(order int) {
	r.SortOrder = order
	r.UpdatedAt = time.Now()
	r.IncrementVersion()
}

// Enable enables the role
func (r *Role) Enable() error {
	if r.IsEnabled {
		return shared.NewDomainError("ALREADY_ENABLED", "Role is already enabled")
	}

	r.IsEnabled = true
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	r.AddDomainEvent(NewRoleEnabledEvent(r))

	return nil
}

// Disable disables the role
func (r *Role) Disable() error {
	if !r.IsEnabled {
		return shared.NewDomainError("ALREADY_DISABLED", "Role is already disabled")
	}

	r.IsEnabled = false
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	r.AddDomainEvent(NewRoleDisabledEvent(r))

	return nil
}

// GrantPermission grants a permission to the role
func (r *Role) GrantPermission(perm Permission) error {
	if perm.IsEmpty() {
		return shared.NewDomainError("INVALID_PERMISSION", "Permission cannot be empty")
	}

	// Check if already has the permission
	for _, p := range r.Permissions {
		if p.Equals(perm) {
			return shared.NewDomainError("PERMISSION_ALREADY_GRANTED", "Role already has this permission")
		}
	}

	r.Permissions = append(r.Permissions, perm)
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	r.AddDomainEvent(NewRolePermissionGrantedEvent(r, perm))

	return nil
}

// GrantPermissionByCode grants a permission by code string
func (r *Role) GrantPermissionByCode(code string) error {
	perm, err := NewPermissionFromCode(code)
	if err != nil {
		return err
	}
	return r.GrantPermission(*perm)
}

// RevokePermission revokes a permission from the role
func (r *Role) RevokePermission(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_PERMISSION_CODE", "Permission code cannot be empty")
	}

	found := false
	newPermissions := make([]Permission, 0, len(r.Permissions))
	var revokedPerm Permission

	for _, p := range r.Permissions {
		if p.Code != code {
			newPermissions = append(newPermissions, p)
		} else {
			found = true
			revokedPerm = p
		}
	}

	if !found {
		return shared.NewDomainError("PERMISSION_NOT_FOUND", "Role does not have this permission")
	}

	r.Permissions = newPermissions
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	r.AddDomainEvent(NewRolePermissionRevokedEvent(r, revokedPerm))

	return nil
}

// SetPermissions sets all permissions for the role (replaces existing)
func (r *Role) SetPermissions(permissions []Permission) error {
	// Validate all permissions
	for _, p := range permissions {
		if p.IsEmpty() {
			return shared.NewDomainError("INVALID_PERMISSION", "Permission cannot be empty")
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	uniquePerms := make([]Permission, 0, len(permissions))
	for _, p := range permissions {
		if !seen[p.Code] {
			seen[p.Code] = true
			uniquePerms = append(uniquePerms, p)
		}
	}

	r.Permissions = uniquePerms
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return nil
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(code string) bool {
	for _, p := range r.Permissions {
		if p.Code == code {
			return true
		}
	}
	return false
}

// HasPermissionForResource checks if the role has any permission for a resource
func (r *Role) HasPermissionForResource(resource string) bool {
	resource = strings.ToLower(strings.TrimSpace(resource))
	for _, p := range r.Permissions {
		if p.Resource == resource {
			return true
		}
	}
	return false
}

// GetPermissionsForResource returns all permissions for a specific resource
func (r *Role) GetPermissionsForResource(resource string) []Permission {
	resource = strings.ToLower(strings.TrimSpace(resource))
	result := make([]Permission, 0)
	for _, p := range r.Permissions {
		if p.Resource == resource {
			result = append(result, p)
		}
	}
	return result
}

// SetDataScope sets a data scope for a resource (replaces if exists)
func (r *Role) SetDataScope(ds DataScope) error {
	if ds.IsEmpty() {
		return shared.NewDomainError("INVALID_DATA_SCOPE", "Data scope cannot be empty")
	}

	// Remove existing data scope for the same resource
	newScopes := make([]DataScope, 0, len(r.DataScopes))
	for _, s := range r.DataScopes {
		if s.Resource != ds.Resource {
			newScopes = append(newScopes, s)
		}
	}

	newScopes = append(newScopes, ds)
	r.DataScopes = newScopes
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	r.AddDomainEvent(NewRoleDataScopeChangedEvent(r, ds))

	return nil
}

// RemoveDataScope removes a data scope for a resource
func (r *Role) RemoveDataScope(resource string) error {
	resource = strings.ToLower(strings.TrimSpace(resource))

	found := false
	newScopes := make([]DataScope, 0, len(r.DataScopes))
	for _, s := range r.DataScopes {
		if s.Resource != resource {
			newScopes = append(newScopes, s)
		} else {
			found = true
		}
	}

	if !found {
		return shared.NewDomainError("DATA_SCOPE_NOT_FOUND", "Role does not have data scope for this resource")
	}

	r.DataScopes = newScopes
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return nil
}

// SetDataScopes sets all data scopes for the role (replaces existing)
func (r *Role) SetDataScopes(scopes []DataScope) error {
	// Validate and deduplicate by resource
	seen := make(map[string]bool)
	uniqueScopes := make([]DataScope, 0, len(scopes))
	for _, s := range scopes {
		if s.IsEmpty() {
			return shared.NewDomainError("INVALID_DATA_SCOPE", "Data scope cannot be empty")
		}
		if !seen[s.Resource] {
			seen[s.Resource] = true
			uniqueScopes = append(uniqueScopes, s)
		}
	}

	r.DataScopes = uniqueScopes
	r.UpdatedAt = time.Now()
	r.IncrementVersion()

	return nil
}

// GetDataScope returns the data scope for a specific resource
func (r *Role) GetDataScope(resource string) *DataScope {
	resource = strings.ToLower(strings.TrimSpace(resource))
	for _, s := range r.DataScopes {
		if s.Resource == resource {
			return &s
		}
	}
	return nil
}

// HasDataScope checks if the role has a data scope for a resource
func (r *Role) HasDataScope(resource string) bool {
	return r.GetDataScope(resource) != nil
}

// CanDelete returns true if the role can be deleted
func (r *Role) CanDelete() bool {
	return !r.IsSystemRole
}

// Update updates the role's basic information
func (r *Role) Update(name, description string) error {
	if err := r.SetName(name); err != nil {
		return err
	}
	r.SetDescription(description)

	r.AddDomainEvent(NewRoleUpdatedEvent(r))

	return nil
}

// Validation functions

func validateRoleCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return shared.NewDomainError("INVALID_ROLE_CODE", "Role code cannot be empty")
	}
	if len(code) < 2 {
		return shared.NewDomainError("INVALID_ROLE_CODE", "Role code must be at least 2 characters")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_ROLE_CODE", "Role code cannot exceed 50 characters")
	}

	// Allow alphanumeric and underscore only
	codeRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !codeRegex.MatchString(code) {
		return shared.NewDomainError("INVALID_ROLE_CODE", "Role code must start with a letter and contain only letters, numbers, and underscores")
	}

	return nil
}

func validateRoleName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return shared.NewDomainError("INVALID_ROLE_NAME", "Role name cannot be empty")
	}
	if len(name) > 100 {
		return shared.NewDomainError("INVALID_ROLE_NAME", "Role name cannot exceed 100 characters")
	}
	return nil
}

func validatePermissionResource(resource string) error {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return shared.NewDomainError("INVALID_PERMISSION_RESOURCE", "Permission resource cannot be empty")
	}
	if len(resource) > 50 {
		return shared.NewDomainError("INVALID_PERMISSION_RESOURCE", "Permission resource cannot exceed 50 characters")
	}

	resourceRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !resourceRegex.MatchString(strings.ToLower(resource)) {
		return shared.NewDomainError("INVALID_PERMISSION_RESOURCE", "Permission resource must start with a letter and contain only lowercase letters, numbers, and underscores")
	}

	return nil
}

func validatePermissionAction(action string) error {
	action = strings.TrimSpace(action)
	if action == "" {
		return shared.NewDomainError("INVALID_PERMISSION_ACTION", "Permission action cannot be empty")
	}
	if len(action) > 50 {
		return shared.NewDomainError("INVALID_PERMISSION_ACTION", "Permission action cannot exceed 50 characters")
	}

	actionRegex := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !actionRegex.MatchString(strings.ToLower(action)) {
		return shared.NewDomainError("INVALID_PERMISSION_ACTION", "Permission action must start with a letter and contain only lowercase letters, numbers, and underscores")
	}

	return nil
}

func validateDataScopeResource(resource string) error {
	return validatePermissionResource(resource) // Same rules as permission resource
}

func validateDataScopeType(scopeType DataScopeType) error {
	switch scopeType {
	case DataScopeAll, DataScopeSelf, DataScopeDepartment, DataScopeCustom, DataScopeWarehouse:
		return nil
	default:
		return shared.NewDomainError("INVALID_DATA_SCOPE_TYPE", "Invalid data scope type")
	}
}

// Predefined system role codes
const (
	RoleCodeAdmin      = "ADMIN"
	RoleCodeOwner      = "OWNER"
	RoleCodeManager    = "MANAGER"
	RoleCodeSales      = "SALES"
	RoleCodePurchaser  = "PURCHASER"
	RoleCodeWarehouse  = "WAREHOUSE"
	RoleCodeCashier    = "CASHIER"
	RoleCodeAccountant = "ACCOUNTANT"
)

// Predefined resources
const (
	ResourceProduct           = "product"
	ResourceCategory          = "category"
	ResourceCustomer          = "customer"
	ResourceSupplier          = "supplier"
	ResourceWarehouse         = "warehouse"
	ResourceInventory         = "inventory"
	ResourceSalesOrder        = "sales_order"
	ResourcePurchaseOrder     = "purchase_order"
	ResourceSalesReturn       = "sales_return"
	ResourcePurchaseReturn    = "purchase_return"
	ResourceAccountReceivable = "account_receivable"
	ResourceAccountPayable    = "account_payable"
	ResourceReceipt           = "receipt"
	ResourcePayment           = "payment"
	ResourceExpense           = "expense"
	ResourceIncome            = "income"
	ResourceReport            = "report"
	ResourceUser              = "user"
	ResourceRole              = "role"
	ResourceTenant            = "tenant"
)

// Predefined actions
const (
	ActionCreate     = "create"
	ActionRead       = "read"
	ActionUpdate     = "update"
	ActionDelete     = "delete"
	ActionEnable     = "enable"
	ActionDisable    = "disable"
	ActionConfirm    = "confirm"
	ActionCancel     = "cancel"
	ActionShip       = "ship"
	ActionReceive    = "receive"
	ActionApprove    = "approve"
	ActionReject     = "reject"
	ActionAdjust     = "adjust"
	ActionLock       = "lock"
	ActionUnlock     = "unlock"
	ActionReconcile  = "reconcile"
	ActionExport     = "export"
	ActionImport     = "import"
	ActionAssignRole = "assign_role"
	ActionViewAll    = "view_all"
)
