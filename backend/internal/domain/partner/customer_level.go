package partner

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Standard customer level codes
const (
	CustomerLevelCodeNormal   = "normal"
	CustomerLevelCodeSilver   = "silver"
	CustomerLevelCodeGold     = "gold"
	CustomerLevelCodePlatinum = "platinum"
	CustomerLevelCodeVIP      = "vip"
)

// CustomerLevel is a Value Object representing a customer's tier/grade.
// It encapsulates the level code, display name, and discount rate.
// CustomerLevel is immutable - all operations return new instances.
type CustomerLevel struct {
	code         string
	name         string
	discountRate decimal.Decimal
}

// NewCustomerLevel creates a new CustomerLevel with the specified code, name, and discount rate.
// The discount rate should be a decimal between 0 and 1 (e.g., 0.05 for 5% discount).
func NewCustomerLevel(code, name string, discountRate decimal.Decimal) (CustomerLevel, error) {
	if code == "" {
		return CustomerLevel{}, errors.New("customer level code cannot be empty")
	}
	if name == "" {
		return CustomerLevel{}, errors.New("customer level name cannot be empty")
	}
	if discountRate.IsNegative() {
		return CustomerLevel{}, errors.New("discount rate cannot be negative")
	}
	if discountRate.GreaterThan(decimal.NewFromInt(1)) {
		return CustomerLevel{}, errors.New("discount rate cannot exceed 1 (100%)")
	}

	return CustomerLevel{
		code:         code,
		name:         name,
		discountRate: discountRate,
	}, nil
}

// MustNewCustomerLevel creates a new CustomerLevel, panicking if validation fails.
// Use this only for static initialization where values are known to be valid.
func MustNewCustomerLevel(code, name string, discountRate decimal.Decimal) CustomerLevel {
	level, err := NewCustomerLevel(code, name, discountRate)
	if err != nil {
		panic(fmt.Sprintf("invalid customer level: %v", err))
	}
	return level
}

// NewCustomerLevelFromCode creates a CustomerLevel from just a code.
// This is useful when loading from legacy data where only the code is stored.
// The name will be derived from the code and discount rate will be zero.
//
// IMPORTANT: This creates a PARTIAL CustomerLevel that should be enriched
// with full details from the customer_levels table using WithDetails() method.
// Using this level for discount calculations without enrichment will result
// in 0% discount being applied.
func NewCustomerLevelFromCode(code string) (CustomerLevel, error) {
	if code == "" {
		return CustomerLevel{}, errors.New("customer level code cannot be empty")
	}
	return CustomerLevel{
		code:         code,
		name:         code, // Use code as name when name is unknown
		discountRate: decimal.Zero,
	}, nil
}

// Predefined standard customer levels with default discount rates

// NormalLevel returns the standard "normal" customer level (0% discount)
func NormalLevel() CustomerLevel {
	return CustomerLevel{
		code:         CustomerLevelCodeNormal,
		name:         "普通会员",
		discountRate: decimal.Zero,
	}
}

// SilverLevel returns the standard "silver" customer level (3% discount)
func SilverLevel() CustomerLevel {
	return CustomerLevel{
		code:         CustomerLevelCodeSilver,
		name:         "银卡会员",
		discountRate: decimal.NewFromFloat(0.03),
	}
}

// GoldLevel returns the standard "gold" customer level (5% discount)
func GoldLevel() CustomerLevel {
	return CustomerLevel{
		code:         CustomerLevelCodeGold,
		name:         "金卡会员",
		discountRate: decimal.NewFromFloat(0.05),
	}
}

// PlatinumLevel returns the standard "platinum" customer level (8% discount)
func PlatinumLevel() CustomerLevel {
	return CustomerLevel{
		code:         CustomerLevelCodePlatinum,
		name:         "白金会员",
		discountRate: decimal.NewFromFloat(0.08),
	}
}

// VIPLevel returns the standard "VIP" customer level (10% discount)
func VIPLevel() CustomerLevel {
	return CustomerLevel{
		code:         CustomerLevelCodeVIP,
		name:         "VIP会员",
		discountRate: decimal.NewFromFloat(0.10),
	}
}

// DefaultLevels returns all standard customer levels in order
func DefaultLevels() []CustomerLevel {
	return []CustomerLevel{
		NormalLevel(),
		SilverLevel(),
		GoldLevel(),
		PlatinumLevel(),
		VIPLevel(),
	}
}

// Getters - CustomerLevel is immutable, so we only provide read access

// Code returns the customer level code
func (cl CustomerLevel) Code() string {
	return cl.code
}

// Name returns the customer level display name
func (cl CustomerLevel) Name() string {
	return cl.name
}

// DiscountRate returns the discount rate as a decimal (e.g., 0.05 for 5%)
func (cl CustomerLevel) DiscountRate() decimal.Decimal {
	return cl.discountRate
}

// DiscountPercent returns the discount rate as a percentage (e.g., 5 for 5%)
func (cl CustomerLevel) DiscountPercent() decimal.Decimal {
	return cl.discountRate.Mul(decimal.NewFromInt(100))
}

// Comparison methods

// Equals returns true if two CustomerLevel values are equal
func (cl CustomerLevel) Equals(other CustomerLevel) bool {
	return cl.code == other.code &&
		cl.name == other.name &&
		cl.discountRate.Equal(other.discountRate)
}

// CodeEquals returns true if the codes match (ignoring name and discount rate)
func (cl CustomerLevel) CodeEquals(other CustomerLevel) bool {
	return cl.code == other.code
}

// IsHigherThan returns true if this level has a higher discount rate than the other
func (cl CustomerLevel) IsHigherThan(other CustomerLevel) bool {
	return cl.discountRate.GreaterThan(other.discountRate)
}

// IsLowerThan returns true if this level has a lower discount rate than the other
func (cl CustomerLevel) IsLowerThan(other CustomerLevel) bool {
	return cl.discountRate.LessThan(other.discountRate)
}

// HasDiscount returns true if this level has any discount (discount rate > 0)
func (cl CustomerLevel) HasDiscount() bool {
	return cl.discountRate.GreaterThan(decimal.Zero)
}

// Validation methods

// IsValid returns true if the CustomerLevel has valid data
func (cl CustomerLevel) IsValid() bool {
	return cl.code != "" &&
		cl.name != "" &&
		!cl.discountRate.IsNegative() &&
		cl.discountRate.LessThanOrEqual(decimal.NewFromInt(1))
}

// IsEmpty returns true if this is an empty/zero CustomerLevel
func (cl CustomerLevel) IsEmpty() bool {
	return cl.code == "" && cl.name == "" && cl.discountRate.IsZero()
}

// IsStandardLevel returns true if this is one of the predefined standard levels
func (cl CustomerLevel) IsStandardLevel() bool {
	switch cl.code {
	case CustomerLevelCodeNormal, CustomerLevelCodeSilver, CustomerLevelCodeGold,
		CustomerLevelCodePlatinum, CustomerLevelCodeVIP:
		return true
	default:
		return false
	}
}

// String returns a string representation of the CustomerLevel
func (cl CustomerLevel) String() string {
	return fmt.Sprintf("%s (%s, %.1f%% discount)", cl.name, cl.code, cl.DiscountPercent().InexactFloat64())
}

// ApplyDiscount applies the customer level discount to a price
func (cl CustomerLevel) ApplyDiscount(price decimal.Decimal) decimal.Decimal {
	if cl.discountRate.IsZero() {
		return price
	}
	// discountedPrice = price * (1 - discountRate)
	return price.Mul(decimal.NewFromInt(1).Sub(cl.discountRate)).Round(4)
}

// CalculateDiscountAmount calculates the discount amount for a given price
func (cl CustomerLevel) CalculateDiscountAmount(price decimal.Decimal) decimal.Decimal {
	return price.Mul(cl.discountRate).Round(4)
}

// JSON serialization

// customerLevelJSON is the JSON representation of CustomerLevel
type customerLevelJSON struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	DiscountRate string `json:"discount_rate"`
}

// MarshalJSON implements json.Marshaler
func (cl CustomerLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(customerLevelJSON{
		Code:         cl.code,
		Name:         cl.name,
		DiscountRate: cl.discountRate.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (cl *CustomerLevel) UnmarshalJSON(data []byte) error {
	var v customerLevelJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	discountRate, err := decimal.NewFromString(v.DiscountRate)
	if err != nil {
		return fmt.Errorf("invalid discount rate: %w", err)
	}

	cl.code = v.Code
	cl.name = v.Name
	cl.discountRate = discountRate
	return nil
}

// Database serialization (GORM)
// CustomerLevel is stored as the code string in the database for backwards compatibility.
// The full level details (name, discount_rate) are stored in the customer_levels table.

// Value implements driver.Valuer for database storage
// Stores only the code as a string for backwards compatibility
func (cl CustomerLevel) Value() (driver.Value, error) {
	if cl.code == "" {
		return nil, nil
	}
	return cl.code, nil
}

// Scan implements sql.Scanner for database retrieval
// Reads the code string and creates a CustomerLevel with just the code
func (cl *CustomerLevel) Scan(value any) error {
	if value == nil {
		cl.code = ""
		cl.name = ""
		cl.discountRate = decimal.Zero
		return nil
	}

	var code string
	switch v := value.(type) {
	case string:
		code = v
	case []byte:
		code = string(v)
	default:
		return fmt.Errorf("cannot scan %T into CustomerLevel", value)
	}

	// When scanning from DB, we only have the code
	// The full details should be loaded from customer_levels table
	cl.code = code
	cl.name = code // Placeholder; should be enriched from customer_levels table
	cl.discountRate = decimal.Zero
	return nil
}

// WithDetails returns a new CustomerLevel with updated name and discount rate
// This is used to enrich a CustomerLevel loaded from the legacy column with
// full details from the customer_levels table
func (cl CustomerLevel) WithDetails(name string, discountRate decimal.Decimal) CustomerLevel {
	return CustomerLevel{
		code:         cl.code,
		name:         name,
		discountRate: discountRate,
	}
}

// CustomerLevelRecord represents a row in the customer_levels database table
// This is used for GORM persistence of customer level definitions
type CustomerLevelRecord struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Code         string
	Name         string
	DiscountRate decimal.Decimal
	SortOrder    int
	IsDefault    bool
	IsActive     bool
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ToCustomerLevel converts a database record to a CustomerLevel value object
func (r *CustomerLevelRecord) ToCustomerLevel() CustomerLevel {
	return CustomerLevel{
		code:         r.Code,
		name:         r.Name,
		discountRate: r.DiscountRate,
	}
}

// NewCustomerLevelRecord creates a new CustomerLevelRecord for database insertion
func NewCustomerLevelRecord(tenantID uuid.UUID, level CustomerLevel, sortOrder int, isDefault bool) *CustomerLevelRecord {
	return &CustomerLevelRecord{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Code:         level.Code(),
		Name:         level.Name(),
		DiscountRate: level.DiscountRate(),
		SortOrder:    sortOrder,
		IsDefault:    isDefault,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// DefaultCustomerLevelRecords returns the default customer level records for a new tenant
func DefaultCustomerLevelRecords(tenantID uuid.UUID) []*CustomerLevelRecord {
	levels := DefaultLevels()
	records := make([]*CustomerLevelRecord, len(levels))
	for i, level := range levels {
		records[i] = NewCustomerLevelRecord(tenantID, level, i, i == 0) // First level (normal) is default
	}
	return records
}
