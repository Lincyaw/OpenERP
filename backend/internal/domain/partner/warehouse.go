package partner

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// WarehouseStatus represents the status of a warehouse
type WarehouseStatus string

const (
	WarehouseStatusActive   WarehouseStatus = "active"
	WarehouseStatusInactive WarehouseStatus = "inactive"
)

// WarehouseType represents the type of warehouse
type WarehouseType string

const (
	WarehouseTypePhysical WarehouseType = "physical" // Physical warehouse
	WarehouseTypeVirtual  WarehouseType = "virtual"  // Virtual/logical warehouse
	WarehouseTypeConsign  WarehouseType = "consign"  // Consignment warehouse
	WarehouseTypeTransit  WarehouseType = "transit"  // Transit/in-transit warehouse
)

// Warehouse represents a warehouse in the partner context
// It is the aggregate root for warehouse-related operations
type Warehouse struct {
	shared.TenantAggregateRoot
	Code        string          `gorm:"type:varchar(50);not null;uniqueIndex:idx_warehouse_tenant_code,priority:2"`
	Name        string          `gorm:"type:varchar(200);not null"`
	ShortName   string          `gorm:"type:varchar(100)"` // Abbreviated name
	Type        WarehouseType   `gorm:"type:varchar(20);not null;default:'physical'"`
	Status      WarehouseStatus `gorm:"type:varchar(20);not null;default:'active'"`
	ContactName string          `gorm:"type:varchar(100)"` // Warehouse manager/contact
	Phone       string          `gorm:"type:varchar(50);index"`
	Email       string          `gorm:"type:varchar(200)"`
	Address     string          `gorm:"type:text"` // Full address
	City        string          `gorm:"type:varchar(100)"`
	Province    string          `gorm:"type:varchar(100)"`
	PostalCode  string          `gorm:"type:varchar(20)"`
	Country     string          `gorm:"type:varchar(100);default:'中国'"`
	IsDefault   bool            `gorm:"not null;default:false"` // Default warehouse for operations
	Capacity    int             `gorm:"not null;default:0"`     // Storage capacity (in units)
	Notes       string          `gorm:"type:text"`
	SortOrder   int             `gorm:"not null;default:0"`
	Attributes  string          `gorm:"type:jsonb"` // Custom attributes
}

// TableName returns the table name for GORM
func (Warehouse) TableName() string {
	return "warehouses"
}

// NewWarehouse creates a new warehouse with required fields
func NewWarehouse(tenantID uuid.UUID, code, name string, warehouseType WarehouseType) (*Warehouse, error) {
	if err := validateWarehouseCode(code); err != nil {
		return nil, err
	}
	if err := validateWarehouseName(name); err != nil {
		return nil, err
	}
	if err := validateWarehouseType(warehouseType); err != nil {
		return nil, err
	}

	warehouse := &Warehouse{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		Type:                warehouseType,
		Status:              WarehouseStatusActive,
		IsDefault:           false,
		Capacity:            0,
		Country:             "中国",
		Attributes:          "{}",
	}

	warehouse.AddDomainEvent(NewWarehouseCreatedEvent(warehouse))

	return warehouse, nil
}

// NewPhysicalWarehouse creates a new physical warehouse
func NewPhysicalWarehouse(tenantID uuid.UUID, code, name string) (*Warehouse, error) {
	return NewWarehouse(tenantID, code, name, WarehouseTypePhysical)
}

// NewVirtualWarehouse creates a new virtual warehouse
func NewVirtualWarehouse(tenantID uuid.UUID, code, name string) (*Warehouse, error) {
	return NewWarehouse(tenantID, code, name, WarehouseTypeVirtual)
}

// NewConsignWarehouse creates a new consignment warehouse
func NewConsignWarehouse(tenantID uuid.UUID, code, name string) (*Warehouse, error) {
	return NewWarehouse(tenantID, code, name, WarehouseTypeConsign)
}

// NewTransitWarehouse creates a new transit warehouse
func NewTransitWarehouse(tenantID uuid.UUID, code, name string) (*Warehouse, error) {
	return NewWarehouse(tenantID, code, name, WarehouseTypeTransit)
}

// Update updates the warehouse's basic information
func (w *Warehouse) Update(name, shortName string) error {
	if err := validateWarehouseName(name); err != nil {
		return err
	}
	if shortName != "" && len(shortName) > 100 {
		return shared.NewDomainError("INVALID_SHORT_NAME", "Short name cannot exceed 100 characters")
	}

	w.Name = name
	w.ShortName = shortName
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	w.AddDomainEvent(NewWarehouseUpdatedEvent(w))

	return nil
}

// UpdateCode updates the warehouse's code
func (w *Warehouse) UpdateCode(code string) error {
	if err := validateWarehouseCode(code); err != nil {
		return err
	}

	w.Code = strings.ToUpper(code)
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	w.AddDomainEvent(NewWarehouseUpdatedEvent(w))

	return nil
}

// SetContact sets the warehouse's contact information
func (w *Warehouse) SetContact(contactName, phone, email string) error {
	if contactName != "" && len(contactName) > 100 {
		return shared.NewDomainError("INVALID_CONTACT_NAME", "Contact name cannot exceed 100 characters")
	}
	if phone != "" {
		if err := validatePhone(phone); err != nil {
			return err
		}
	}
	if email != "" {
		if err := validateEmail(email); err != nil {
			return err
		}
	}

	w.ContactName = contactName
	w.Phone = phone
	w.Email = email
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	return nil
}

// SetAddress sets the warehouse's address information
func (w *Warehouse) SetAddress(address, city, province, postalCode, country string) error {
	if address != "" && len(address) > 500 {
		return shared.NewDomainError("INVALID_ADDRESS", "Address cannot exceed 500 characters")
	}
	if city != "" && len(city) > 100 {
		return shared.NewDomainError("INVALID_CITY", "City cannot exceed 100 characters")
	}
	if province != "" && len(province) > 100 {
		return shared.NewDomainError("INVALID_PROVINCE", "Province cannot exceed 100 characters")
	}
	if postalCode != "" && len(postalCode) > 20 {
		return shared.NewDomainError("INVALID_POSTAL_CODE", "Postal code cannot exceed 20 characters")
	}
	if country != "" && len(country) > 100 {
		return shared.NewDomainError("INVALID_COUNTRY", "Country cannot exceed 100 characters")
	}

	w.Address = address
	w.City = city
	w.Province = province
	w.PostalCode = postalCode
	if country != "" {
		w.Country = country
	}
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	return nil
}

// SetDefault marks this warehouse as the default warehouse
func (w *Warehouse) SetDefault(isDefault bool) {
	w.IsDefault = isDefault
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	if isDefault {
		w.AddDomainEvent(NewWarehouseSetAsDefaultEvent(w))
	}
}

// SetCapacity sets the warehouse's storage capacity
func (w *Warehouse) SetCapacity(capacity int) error {
	if capacity < 0 {
		return shared.NewDomainError("INVALID_CAPACITY", "Capacity cannot be negative")
	}

	w.Capacity = capacity
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	return nil
}

// SetNotes sets the warehouse's notes
func (w *Warehouse) SetNotes(notes string) {
	w.Notes = notes
	w.UpdatedAt = time.Now()
	w.IncrementVersion()
}

// SetSortOrder sets the display order
func (w *Warehouse) SetSortOrder(order int) {
	w.SortOrder = order
	w.UpdatedAt = time.Now()
	w.IncrementVersion()
}

// SetAttributes sets custom attributes as JSON
func (w *Warehouse) SetAttributes(attributes string) error {
	if attributes == "" {
		attributes = "{}"
	}
	trimmed := strings.TrimSpace(attributes)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return shared.NewDomainError("INVALID_ATTRIBUTES", "Attributes must be valid JSON object")
	}

	w.Attributes = trimmed
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	return nil
}

// Enable enables the warehouse (makes it active)
func (w *Warehouse) Enable() error {
	if w.Status == WarehouseStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Warehouse is already active")
	}

	oldStatus := w.Status
	w.Status = WarehouseStatusActive
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	w.AddDomainEvent(NewWarehouseStatusChangedEvent(w, oldStatus, WarehouseStatusActive))

	return nil
}

// Disable disables the warehouse (makes it inactive)
func (w *Warehouse) Disable() error {
	if w.Status == WarehouseStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Warehouse is already inactive")
	}

	// Cannot disable a default warehouse
	if w.IsDefault {
		return shared.NewDomainError("CANNOT_DISABLE_DEFAULT", "Cannot disable the default warehouse")
	}

	oldStatus := w.Status
	w.Status = WarehouseStatusInactive
	w.UpdatedAt = time.Now()
	w.IncrementVersion()

	w.AddDomainEvent(NewWarehouseStatusChangedEvent(w, oldStatus, WarehouseStatusInactive))

	return nil
}

// IsActive returns true if the warehouse is active
func (w *Warehouse) IsActive() bool {
	return w.Status == WarehouseStatusActive
}

// IsInactive returns true if the warehouse is inactive
func (w *Warehouse) IsInactive() bool {
	return w.Status == WarehouseStatusInactive
}

// IsPhysical returns true if warehouse is a physical warehouse
func (w *Warehouse) IsPhysical() bool {
	return w.Type == WarehouseTypePhysical
}

// IsVirtual returns true if warehouse is a virtual warehouse
func (w *Warehouse) IsVirtual() bool {
	return w.Type == WarehouseTypeVirtual
}

// IsConsign returns true if warehouse is a consignment warehouse
func (w *Warehouse) IsConsign() bool {
	return w.Type == WarehouseTypeConsign
}

// IsTransit returns true if warehouse is a transit warehouse
func (w *Warehouse) IsTransit() bool {
	return w.Type == WarehouseTypeTransit
}

// HasCapacity returns true if warehouse has capacity configured
func (w *Warehouse) HasCapacity() bool {
	return w.Capacity > 0
}

// GetFullAddress returns the formatted full address
func (w *Warehouse) GetFullAddress() string {
	parts := []string{}
	if w.Country != "" {
		parts = append(parts, w.Country)
	}
	if w.Province != "" {
		parts = append(parts, w.Province)
	}
	if w.City != "" {
		parts = append(parts, w.City)
	}
	if w.Address != "" {
		parts = append(parts, w.Address)
	}
	if w.PostalCode != "" {
		parts = append(parts, w.PostalCode)
	}
	return strings.Join(parts, " ")
}

// Validation functions

func validateWarehouseCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Warehouse code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Warehouse code cannot exceed 50 characters")
	}
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Warehouse code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

func validateWarehouseName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Warehouse name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Warehouse name cannot exceed 200 characters")
	}
	return nil
}

func validateWarehouseType(t WarehouseType) error {
	switch t {
	case WarehouseTypePhysical, WarehouseTypeVirtual, WarehouseTypeConsign, WarehouseTypeTransit:
		return nil
	default:
		return shared.NewDomainError("INVALID_TYPE", "Invalid warehouse type")
	}
}
