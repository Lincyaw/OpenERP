package partner

import (
	"regexp"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerStatus represents the status of a customer
type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "active"
	CustomerStatusInactive  CustomerStatus = "inactive"
	CustomerStatusSuspended CustomerStatus = "suspended" // Suspended due to credit issues
)

// CustomerType represents the type of customer
type CustomerType string

const (
	CustomerTypeIndividual   CustomerType = "individual"   // Personal customer
	CustomerTypeOrganization CustomerType = "organization" // Business/company
)

// Customer represents a customer in the partner context
// It is the aggregate root for customer-related operations
type Customer struct {
	shared.TenantAggregateRoot
	Code        string
	Name        string
	ShortName   string        // Abbreviated name
	Type        CustomerType  // individual or organization
	Level       CustomerLevel // Customer tier (stored as code)
	Status      CustomerStatus
	ContactName string // Primary contact person
	Phone       string
	Email       string
	Address     string // Full address
	City        string
	Province    string
	PostalCode  string
	Country     string
	TaxID       string // Tax identification number
	CreditLimit decimal.Decimal
	Balance     decimal.Decimal // Prepaid balance
	Notes       string
	SortOrder   int
	Attributes  string // Custom attributes
}

// NewCustomer creates a new customer with required fields
func NewCustomer(tenantID uuid.UUID, code, name string, customerType CustomerType) (*Customer, error) {
	if err := validateCustomerCode(code); err != nil {
		return nil, err
	}
	if err := validateCustomerName(name); err != nil {
		return nil, err
	}
	if err := validateCustomerType(customerType); err != nil {
		return nil, err
	}

	customer := &Customer{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		Type:                customerType,
		Level:               NormalLevel(), // Use factory function for default level
		Status:              CustomerStatusActive,
		CreditLimit:         decimal.Zero,
		Balance:             decimal.Zero,
		Country:             "中国",
		Attributes:          "{}",
	}

	customer.AddDomainEvent(NewCustomerCreatedEvent(customer))

	return customer, nil
}

// NewIndividualCustomer creates a new individual customer
func NewIndividualCustomer(tenantID uuid.UUID, code, name string) (*Customer, error) {
	return NewCustomer(tenantID, code, name, CustomerTypeIndividual)
}

// NewOrganizationCustomer creates a new organization customer
func NewOrganizationCustomer(tenantID uuid.UUID, code, name string) (*Customer, error) {
	return NewCustomer(tenantID, code, name, CustomerTypeOrganization)
}

// Update updates the customer's basic information
func (c *Customer) Update(name, shortName string) error {
	if err := validateCustomerName(name); err != nil {
		return err
	}
	if shortName != "" && len(shortName) > 100 {
		return shared.NewDomainError("INVALID_SHORT_NAME", "Short name cannot exceed 100 characters")
	}

	c.Name = name
	c.ShortName = shortName
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerUpdatedEvent(c))

	return nil
}

// UpdateCode updates the customer's code
func (c *Customer) UpdateCode(code string) error {
	if err := validateCustomerCode(code); err != nil {
		return err
	}

	c.Code = strings.ToUpper(code)
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerUpdatedEvent(c))

	return nil
}

// SetContact sets the customer's contact information
func (c *Customer) SetContact(contactName, phone, email string) error {
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

	c.ContactName = contactName
	c.Phone = phone
	c.Email = email
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	return nil
}

// SetAddress sets the customer's address information
func (c *Customer) SetAddress(address, city, province, postalCode, country string) error {
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

	c.Address = address
	c.City = city
	c.Province = province
	c.PostalCode = postalCode
	if country != "" {
		c.Country = country
	}
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	return nil
}

// SetTaxID sets the customer's tax identification number
func (c *Customer) SetTaxID(taxID string) error {
	if taxID != "" && len(taxID) > 50 {
		return shared.NewDomainError("INVALID_TAX_ID", "Tax ID cannot exceed 50 characters")
	}

	c.TaxID = taxID
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	return nil
}

// SetLevel sets the customer's tier level
func (c *Customer) SetLevel(level CustomerLevel) error {
	if err := validateCustomerLevel(level); err != nil {
		return err
	}

	oldLevel := c.Level
	c.Level = level
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerLevelChangedEvent(c, oldLevel, level))

	return nil
}

// SetCreditLimit sets the customer's credit limit
func (c *Customer) SetCreditLimit(limit decimal.Decimal) error {
	if limit.IsNegative() {
		return shared.NewDomainError("INVALID_CREDIT_LIMIT", "Credit limit cannot be negative")
	}

	c.CreditLimit = limit
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	return nil
}

// AddBalance adds to the customer's prepaid balance (deposit/recharge)
func (c *Customer) AddBalance(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be negative")
	}
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be zero")
	}

	oldBalance := c.Balance
	c.Balance = c.Balance.Add(amount)
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerBalanceChangedEvent(c, oldBalance, c.Balance, "recharge"))

	return nil
}

// DeductBalance deducts from the customer's prepaid balance
func (c *Customer) DeductBalance(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be negative")
	}
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be zero")
	}
	if c.Balance.LessThan(amount) {
		return shared.NewDomainError("INSUFFICIENT_BALANCE", "Insufficient balance")
	}

	oldBalance := c.Balance
	c.Balance = c.Balance.Sub(amount)
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerBalanceChangedEvent(c, oldBalance, c.Balance, "deduction"))

	return nil
}

// RefundBalance refunds to the customer's prepaid balance
func (c *Customer) RefundBalance(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be negative")
	}
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be zero")
	}

	oldBalance := c.Balance
	c.Balance = c.Balance.Add(amount)
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerBalanceChangedEvent(c, oldBalance, c.Balance, "refund"))

	return nil
}

// SetNotes sets the customer's notes
func (c *Customer) SetNotes(notes string) {
	c.Notes = notes
	c.UpdatedAt = time.Now()
	c.IncrementVersion()
}

// SetSortOrder sets the display order
func (c *Customer) SetSortOrder(order int) {
	c.SortOrder = order
	c.UpdatedAt = time.Now()
	c.IncrementVersion()
}

// SetAttributes sets custom attributes as JSON
func (c *Customer) SetAttributes(attributes string) error {
	if attributes == "" {
		attributes = "{}"
	}
	trimmed := strings.TrimSpace(attributes)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return shared.NewDomainError("INVALID_ATTRIBUTES", "Attributes must be valid JSON object")
	}

	c.Attributes = trimmed
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	return nil
}

// Activate activates the customer
func (c *Customer) Activate() error {
	if c.Status == CustomerStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Customer is already active")
	}

	oldStatus := c.Status
	c.Status = CustomerStatusActive
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerStatusChangedEvent(c, oldStatus, CustomerStatusActive))

	return nil
}

// Deactivate deactivates the customer
func (c *Customer) Deactivate() error {
	if c.Status == CustomerStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Customer is already inactive")
	}

	oldStatus := c.Status
	c.Status = CustomerStatusInactive
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerStatusChangedEvent(c, oldStatus, CustomerStatusInactive))

	return nil
}

// Suspend suspends the customer (e.g., due to credit issues)
func (c *Customer) Suspend() error {
	if c.Status == CustomerStatusSuspended {
		return shared.NewDomainError("ALREADY_SUSPENDED", "Customer is already suspended")
	}

	oldStatus := c.Status
	c.Status = CustomerStatusSuspended
	c.UpdatedAt = time.Now()
	c.IncrementVersion()

	c.AddDomainEvent(NewCustomerStatusChangedEvent(c, oldStatus, CustomerStatusSuspended))

	return nil
}

// IsActive returns true if the customer is active
func (c *Customer) IsActive() bool {
	return c.Status == CustomerStatusActive
}

// IsInactive returns true if the customer is inactive
func (c *Customer) IsInactive() bool {
	return c.Status == CustomerStatusInactive
}

// IsSuspended returns true if the customer is suspended
func (c *Customer) IsSuspended() bool {
	return c.Status == CustomerStatusSuspended
}

// IsIndividual returns true if customer is an individual
func (c *Customer) IsIndividual() bool {
	return c.Type == CustomerTypeIndividual
}

// IsOrganization returns true if customer is an organization
func (c *Customer) IsOrganization() bool {
	return c.Type == CustomerTypeOrganization
}

// HasCreditLimit returns true if customer has a credit limit set
func (c *Customer) HasCreditLimit() bool {
	return c.CreditLimit.GreaterThan(decimal.Zero)
}

// HasBalance returns true if customer has prepaid balance
func (c *Customer) HasBalance() bool {
	return c.Balance.GreaterThan(decimal.Zero)
}

// GetAvailableCredit returns the available credit for the customer
// (Note: This is a simplified version; real implementation would consider outstanding receivables)
func (c *Customer) GetAvailableCredit() decimal.Decimal {
	return c.CreditLimit
}

// GetFullAddress returns the formatted full address
func (c *Customer) GetFullAddress() string {
	parts := []string{}
	if c.Country != "" {
		parts = append(parts, c.Country)
	}
	if c.Province != "" {
		parts = append(parts, c.Province)
	}
	if c.City != "" {
		parts = append(parts, c.City)
	}
	if c.Address != "" {
		parts = append(parts, c.Address)
	}
	if c.PostalCode != "" {
		parts = append(parts, c.PostalCode)
	}
	return strings.Join(parts, " ")
}

// GetAddressVO returns the customer's address as an Address value object
// Returns an empty Address if no address fields are set or if validation fails
// Note: If validation fails, this may indicate data integrity issues in the database
func (c *Customer) GetAddressVO() valueobject.Address {
	if c.Province == "" && c.City == "" && c.Address == "" {
		return valueobject.EmptyAddress()
	}

	// Try to construct an Address VO; if validation fails, return empty
	// This can happen with legacy data that doesn't meet current validation rules
	addr, err := valueobject.NewAddressFull(
		c.Province,
		c.City,
		"", // District is stored as part of Address field in legacy schema
		c.Address,
		c.PostalCode,
		c.Country,
	)
	if err != nil {
		// Legacy data may not meet current validation requirements
		// Return empty address rather than failing
		return valueobject.EmptyAddress()
	}
	return addr
}

// SetAddressVO sets the customer's address from an Address value object
func (c *Customer) SetAddressVO(addr valueobject.Address) {
	if addr.IsEmpty() {
		c.Address = ""
		c.City = ""
		c.Province = ""
		c.PostalCode = ""
		// Keep default country
	} else {
		c.Province = addr.Province()
		c.City = addr.City()
		// Combine district and detail into Address field for backward compatibility
		if addr.District() != "" {
			c.Address = addr.District() + " " + addr.Detail()
		} else {
			c.Address = addr.Detail()
		}
		c.PostalCode = addr.PostalCode()
		if addr.Country() != "" {
			c.Country = addr.Country()
		}
	}
	c.UpdatedAt = time.Now()
	c.IncrementVersion()
}

// Validation functions

func validateCustomerCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Customer code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Customer code cannot exceed 50 characters")
	}
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Customer code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

func validateCustomerName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Customer name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Customer name cannot exceed 200 characters")
	}
	return nil
}

func validateCustomerType(t CustomerType) error {
	switch t {
	case CustomerTypeIndividual, CustomerTypeOrganization:
		return nil
	default:
		return shared.NewDomainError("INVALID_TYPE", "Customer type must be 'individual' or 'organization'")
	}
}

func validateCustomerLevel(level CustomerLevel) error {
	if level.IsEmpty() {
		return shared.NewDomainError("INVALID_LEVEL", "Customer level cannot be empty")
	}
	if !level.IsValid() {
		return shared.NewDomainError("INVALID_LEVEL", "Invalid customer level")
	}
	return nil
}

func validatePhone(phone string) error {
	if len(phone) > 50 {
		return shared.NewDomainError("INVALID_PHONE", "Phone number cannot exceed 50 characters")
	}
	// Basic phone validation - allow digits, spaces, hyphens, parentheses, and plus sign
	validPhone := regexp.MustCompile(`^[\d\s\-\(\)\+]+$`)
	if !validPhone.MatchString(phone) {
		return shared.NewDomainError("INVALID_PHONE", "Invalid phone number format")
	}
	return nil
}

func validateEmail(email string) error {
	if len(email) > 200 {
		return shared.NewDomainError("INVALID_EMAIL", "Email cannot exceed 200 characters")
	}
	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return shared.NewDomainError("INVALID_EMAIL", "Invalid email format")
	}
	return nil
}
