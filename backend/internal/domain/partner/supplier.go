package partner

import (
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierStatus represents the status of a supplier
type SupplierStatus string

const (
	SupplierStatusActive   SupplierStatus = "active"
	SupplierStatusInactive SupplierStatus = "inactive"
	SupplierStatusBlocked  SupplierStatus = "blocked" // Blocked due to quality/payment issues
)

// SupplierType represents the type of supplier
type SupplierType string

const (
	SupplierTypeManufacturer SupplierType = "manufacturer" // Direct manufacturer
	SupplierTypeDistributor  SupplierType = "distributor"  // Distributor/Wholesaler
	SupplierTypeRetailer     SupplierType = "retailer"     // Retailer
	SupplierTypeService      SupplierType = "service"      // Service provider
)

// Supplier represents a supplier in the partner context
// It is the aggregate root for supplier-related operations
type Supplier struct {
	shared.TenantAggregateRoot
	Code        string          `gorm:"type:varchar(50);not null;uniqueIndex:idx_supplier_tenant_code,priority:2"`
	Name        string          `gorm:"type:varchar(200);not null"`
	ShortName   string          `gorm:"type:varchar(100)"` // Abbreviated name
	Type        SupplierType    `gorm:"type:varchar(20);not null;default:'distributor'"`
	Status      SupplierStatus  `gorm:"type:varchar(20);not null;default:'active'"`
	ContactName string          `gorm:"type:varchar(100)"` // Primary contact person
	Phone       string          `gorm:"type:varchar(50);index"`
	Email       string          `gorm:"type:varchar(200);index"`
	Address     string          `gorm:"type:text"` // Full address
	City        string          `gorm:"type:varchar(100)"`
	Province    string          `gorm:"type:varchar(100)"`
	PostalCode  string          `gorm:"type:varchar(20)"`
	Country     string          `gorm:"type:varchar(100);default:'中国'"`
	TaxID       string          `gorm:"type:varchar(50)"` // Tax identification number
	BankName    string          `gorm:"type:varchar(200)"`
	BankAccount string          `gorm:"type:varchar(100)"`
	CreditDays  int             `gorm:"not null;default:0"`                    // Payment terms: days until payment due
	CreditLimit decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Maximum credit allowed
	Balance     decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0"` // Current accounts payable balance
	Rating      int             `gorm:"not null;default:0;check:rating >= 0"`  // Supplier rating (0-5)
	Notes       string          `gorm:"type:text"`
	SortOrder   int             `gorm:"not null;default:0"`
	Attributes  string          `gorm:"type:jsonb"` // Custom attributes
}

// TableName returns the table name for GORM
func (Supplier) TableName() string {
	return "suppliers"
}

// NewSupplier creates a new supplier with required fields
func NewSupplier(tenantID uuid.UUID, code, name string, supplierType SupplierType) (*Supplier, error) {
	if err := validateSupplierCode(code); err != nil {
		return nil, err
	}
	if err := validateSupplierName(name); err != nil {
		return nil, err
	}
	if err := validateSupplierType(supplierType); err != nil {
		return nil, err
	}

	supplier := &Supplier{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Code:                strings.ToUpper(code),
		Name:                name,
		Type:                supplierType,
		Status:              SupplierStatusActive,
		CreditDays:          0,
		CreditLimit:         decimal.Zero,
		Balance:             decimal.Zero,
		Rating:              0,
		Country:             "中国",
		Attributes:          "{}",
	}

	supplier.AddDomainEvent(NewSupplierCreatedEvent(supplier))

	return supplier, nil
}

// NewManufacturerSupplier creates a new manufacturer supplier
func NewManufacturerSupplier(tenantID uuid.UUID, code, name string) (*Supplier, error) {
	return NewSupplier(tenantID, code, name, SupplierTypeManufacturer)
}

// NewDistributorSupplier creates a new distributor supplier
func NewDistributorSupplier(tenantID uuid.UUID, code, name string) (*Supplier, error) {
	return NewSupplier(tenantID, code, name, SupplierTypeDistributor)
}

// Update updates the supplier's basic information
func (s *Supplier) Update(name, shortName string) error {
	if err := validateSupplierName(name); err != nil {
		return err
	}
	if shortName != "" && len(shortName) > 100 {
		return shared.NewDomainError("INVALID_SHORT_NAME", "Short name cannot exceed 100 characters")
	}

	s.Name = name
	s.ShortName = shortName
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierUpdatedEvent(s))

	return nil
}

// UpdateCode updates the supplier's code
func (s *Supplier) UpdateCode(code string) error {
	if err := validateSupplierCode(code); err != nil {
		return err
	}

	s.Code = strings.ToUpper(code)
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierUpdatedEvent(s))

	return nil
}

// SetContact sets the supplier's contact information
func (s *Supplier) SetContact(contactName, phone, email string) error {
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

	s.ContactName = contactName
	s.Phone = phone
	s.Email = email
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// SetAddress sets the supplier's address information
func (s *Supplier) SetAddress(address, city, province, postalCode, country string) error {
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

	s.Address = address
	s.City = city
	s.Province = province
	s.PostalCode = postalCode
	if country != "" {
		s.Country = country
	}
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// SetTaxID sets the supplier's tax identification number
func (s *Supplier) SetTaxID(taxID string) error {
	if taxID != "" && len(taxID) > 50 {
		return shared.NewDomainError("INVALID_TAX_ID", "Tax ID cannot exceed 50 characters")
	}

	s.TaxID = taxID
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// SetBankInfo sets the supplier's bank information
func (s *Supplier) SetBankInfo(bankName, bankAccount string) error {
	if bankName != "" && len(bankName) > 200 {
		return shared.NewDomainError("INVALID_BANK_NAME", "Bank name cannot exceed 200 characters")
	}
	if bankAccount != "" && len(bankAccount) > 100 {
		return shared.NewDomainError("INVALID_BANK_ACCOUNT", "Bank account cannot exceed 100 characters")
	}

	s.BankName = bankName
	s.BankAccount = bankAccount
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// SetPaymentTerms sets the supplier's payment terms (credit days and limit)
func (s *Supplier) SetPaymentTerms(creditDays int, creditLimit decimal.Decimal) error {
	if creditDays < 0 {
		return shared.NewDomainError("INVALID_CREDIT_DAYS", "Credit days cannot be negative")
	}
	if creditDays > 365 {
		return shared.NewDomainError("INVALID_CREDIT_DAYS", "Credit days cannot exceed 365")
	}
	if creditLimit.IsNegative() {
		return shared.NewDomainError("INVALID_CREDIT_LIMIT", "Credit limit cannot be negative")
	}

	oldCreditDays := s.CreditDays
	oldCreditLimit := s.CreditLimit
	s.CreditDays = creditDays
	s.CreditLimit = creditLimit
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierPaymentTermsChangedEvent(s, oldCreditDays, creditDays, oldCreditLimit, creditLimit))

	return nil
}

// SetCreditDays sets the supplier's credit days
func (s *Supplier) SetCreditDays(days int) error {
	return s.SetPaymentTerms(days, s.CreditLimit)
}

// SetCreditLimit sets the supplier's credit limit
func (s *Supplier) SetCreditLimit(limit decimal.Decimal) error {
	return s.SetPaymentTerms(s.CreditDays, limit)
}

// AddBalance adds to the supplier's accounts payable balance (when we receive goods)
func (s *Supplier) AddBalance(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be negative")
	}
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be zero")
	}

	oldBalance := s.Balance
	s.Balance = s.Balance.Add(amount)
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierBalanceChangedEvent(s, oldBalance, s.Balance, "purchase"))

	return nil
}

// DeductBalance deducts from the supplier's accounts payable balance (when we pay)
func (s *Supplier) DeductBalance(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be negative")
	}
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount cannot be zero")
	}
	if s.Balance.LessThan(amount) {
		return shared.NewDomainError("INVALID_AMOUNT", "Amount exceeds current balance")
	}

	oldBalance := s.Balance
	s.Balance = s.Balance.Sub(amount)
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierBalanceChangedEvent(s, oldBalance, s.Balance, "payment"))

	return nil
}

// AdjustBalance adjusts the supplier's balance (for corrections)
func (s *Supplier) AdjustBalance(amount decimal.Decimal, reason string) error {
	if amount.IsZero() {
		return shared.NewDomainError("INVALID_AMOUNT", "Adjustment amount cannot be zero")
	}
	newBalance := s.Balance.Add(amount)
	if newBalance.IsNegative() {
		return shared.NewDomainError("INVALID_AMOUNT", "Adjustment would result in negative balance")
	}

	oldBalance := s.Balance
	s.Balance = newBalance
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierBalanceChangedEvent(s, oldBalance, s.Balance, reason))

	return nil
}

// SetRating sets the supplier's rating (0-5)
func (s *Supplier) SetRating(rating int) error {
	if rating < 0 || rating > 5 {
		return shared.NewDomainError("INVALID_RATING", "Rating must be between 0 and 5")
	}

	s.Rating = rating
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// SetNotes sets the supplier's notes
func (s *Supplier) SetNotes(notes string) {
	s.Notes = notes
	s.UpdatedAt = time.Now()
	s.IncrementVersion()
}

// SetSortOrder sets the display order
func (s *Supplier) SetSortOrder(order int) {
	s.SortOrder = order
	s.UpdatedAt = time.Now()
	s.IncrementVersion()
}

// SetAttributes sets custom attributes as JSON
func (s *Supplier) SetAttributes(attributes string) error {
	if attributes == "" {
		attributes = "{}"
	}
	trimmed := strings.TrimSpace(attributes)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return shared.NewDomainError("INVALID_ATTRIBUTES", "Attributes must be valid JSON object")
	}

	s.Attributes = trimmed
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	return nil
}

// Activate activates the supplier
func (s *Supplier) Activate() error {
	if s.Status == SupplierStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "Supplier is already active")
	}

	oldStatus := s.Status
	s.Status = SupplierStatusActive
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierStatusChangedEvent(s, oldStatus, SupplierStatusActive))

	return nil
}

// Deactivate deactivates the supplier
func (s *Supplier) Deactivate() error {
	if s.Status == SupplierStatusInactive {
		return shared.NewDomainError("ALREADY_INACTIVE", "Supplier is already inactive")
	}

	oldStatus := s.Status
	s.Status = SupplierStatusInactive
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierStatusChangedEvent(s, oldStatus, SupplierStatusInactive))

	return nil
}

// Block blocks the supplier (e.g., due to quality or payment issues)
func (s *Supplier) Block() error {
	if s.Status == SupplierStatusBlocked {
		return shared.NewDomainError("ALREADY_BLOCKED", "Supplier is already blocked")
	}

	oldStatus := s.Status
	s.Status = SupplierStatusBlocked
	s.UpdatedAt = time.Now()
	s.IncrementVersion()

	s.AddDomainEvent(NewSupplierStatusChangedEvent(s, oldStatus, SupplierStatusBlocked))

	return nil
}

// IsActive returns true if the supplier is active
func (s *Supplier) IsActive() bool {
	return s.Status == SupplierStatusActive
}

// IsInactive returns true if the supplier is inactive
func (s *Supplier) IsInactive() bool {
	return s.Status == SupplierStatusInactive
}

// IsBlocked returns true if the supplier is blocked
func (s *Supplier) IsBlocked() bool {
	return s.Status == SupplierStatusBlocked
}

// IsManufacturer returns true if supplier is a manufacturer
func (s *Supplier) IsManufacturer() bool {
	return s.Type == SupplierTypeManufacturer
}

// IsDistributor returns true if supplier is a distributor
func (s *Supplier) IsDistributor() bool {
	return s.Type == SupplierTypeDistributor
}

// HasCreditTerms returns true if supplier has credit terms configured
func (s *Supplier) HasCreditTerms() bool {
	return s.CreditDays > 0 || s.CreditLimit.GreaterThan(decimal.Zero)
}

// HasBalance returns true if supplier has outstanding balance
func (s *Supplier) HasBalance() bool {
	return s.Balance.GreaterThan(decimal.Zero)
}

// GetAvailableCredit returns the available credit for new purchases
func (s *Supplier) GetAvailableCredit() decimal.Decimal {
	if s.CreditLimit.IsZero() {
		return decimal.Zero
	}
	available := s.CreditLimit.Sub(s.Balance)
	if available.IsNegative() {
		return decimal.Zero
	}
	return available
}

// IsOverCreditLimit returns true if current balance exceeds credit limit
func (s *Supplier) IsOverCreditLimit() bool {
	if s.CreditLimit.IsZero() {
		return false // No limit set
	}
	return s.Balance.GreaterThan(s.CreditLimit)
}

// GetFullAddress returns the formatted full address
func (s *Supplier) GetFullAddress() string {
	parts := []string{}
	if s.Country != "" {
		parts = append(parts, s.Country)
	}
	if s.Province != "" {
		parts = append(parts, s.Province)
	}
	if s.City != "" {
		parts = append(parts, s.City)
	}
	if s.Address != "" {
		parts = append(parts, s.Address)
	}
	if s.PostalCode != "" {
		parts = append(parts, s.PostalCode)
	}
	return strings.Join(parts, " ")
}

// Validation functions

func validateSupplierCode(code string) error {
	if code == "" {
		return shared.NewDomainError("INVALID_CODE", "Supplier code cannot be empty")
	}
	if len(code) > 50 {
		return shared.NewDomainError("INVALID_CODE", "Supplier code cannot exceed 50 characters")
	}
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return shared.NewDomainError("INVALID_CODE", "Supplier code can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

func validateSupplierName(name string) error {
	if name == "" {
		return shared.NewDomainError("INVALID_NAME", "Supplier name cannot be empty")
	}
	if len(name) > 200 {
		return shared.NewDomainError("INVALID_NAME", "Supplier name cannot exceed 200 characters")
	}
	return nil
}

func validateSupplierType(t SupplierType) error {
	switch t {
	case SupplierTypeManufacturer, SupplierTypeDistributor, SupplierTypeRetailer, SupplierTypeService:
		return nil
	default:
		return shared.NewDomainError("INVALID_TYPE", "Invalid supplier type")
	}
}

// validatePhone and validateEmail are defined in customer.go
// They use the same validation logic so we reuse them
