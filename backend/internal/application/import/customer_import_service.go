package importapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerImportRow represents a row from the customer CSV import
type CustomerImportRow struct {
	Name            string `csv:"name"`
	Type            string `csv:"type"`
	Code            string `csv:"code"`
	ContactPerson   string `csv:"contact_person"`
	Phone           string `csv:"phone"`
	Email           string `csv:"email"`
	LevelCode       string `csv:"level_code"`
	CreditLimit     string `csv:"credit_limit"`
	AddressProvince string `csv:"address_province"`
	AddressCity     string `csv:"address_city"`
	AddressDistrict string `csv:"address_district"`
	AddressDetail   string `csv:"address_detail"`
	Notes           string `csv:"notes"`
}

// CustomerImportResult represents the result of a customer import operation
type CustomerImportResult struct {
	TotalRows    int                  `json:"total_rows"`
	ImportedRows int                  `json:"imported_rows"`
	UpdatedRows  int                  `json:"updated_rows"`
	SkippedRows  int                  `json:"skipped_rows"`
	ErrorRows    int                  `json:"error_rows"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// CustomerImportService handles customer bulk import operations
type CustomerImportService struct {
	customerRepo      partner.CustomerRepository
	customerLevelRepo partner.CustomerLevelRepository
	eventBus          shared.EventPublisher
	codeSeqMu         sync.Mutex
	codeSeqDate       string
	codeSeqNum        int64
}

// NewCustomerImportService creates a new CustomerImportService
func NewCustomerImportService(
	customerRepo partner.CustomerRepository,
	customerLevelRepo partner.CustomerLevelRepository,
	eventBus shared.EventPublisher,
) *CustomerImportService {
	return &CustomerImportService{
		customerRepo:      customerRepo,
		customerLevelRepo: customerLevelRepo,
		eventBus:          eventBus,
	}
}

// GetValidationRules returns the validation rules for customer import
func (s *CustomerImportService) GetValidationRules() []csvimport.FieldRule {
	zero := decimal.Zero
	return []csvimport.FieldRule{
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("type").Required().String().Custom(validateCustomerType).Build(),
		csvimport.Field("code").String().MaxLength(50).Unique().Build(),
		csvimport.Field("contact_person").String().MaxLength(100).Build(),
		csvimport.Field("phone").String().MaxLength(50).Build(),
		csvimport.Field("email").String().Email().Build(),
		csvimport.Field("level_code").String().MaxLength(50).Reference("customer_level").Build(),
		csvimport.Field("credit_limit").Decimal().MinValue(zero).Build(),
		csvimport.Field("address_province").String().MaxLength(50).Build(),
		csvimport.Field("address_city").String().MaxLength(50).Build(),
		csvimport.Field("address_district").String().MaxLength(50).Build(),
		csvimport.Field("address_detail").String().MaxLength(200).Build(),
		csvimport.Field("notes").String().MaxLength(1000).Build(),
	}
}

// validateCustomerType validates the customer type field
func validateCustomerType(value string) error {
	if value == "" {
		return nil // will be caught by required check
	}
	lower := strings.ToLower(value)
	switch lower {
	case "individual", "company", "organization":
		return nil
	default:
		return fmt.Errorf("type must be 'individual' or 'company' (or 'organization')")
	}
}

// normalizeCustomerType normalizes the customer type input
func normalizeCustomerType(value string) partner.CustomerType {
	lower := strings.ToLower(value)
	switch lower {
	case "individual":
		return partner.CustomerTypeIndividual
	case "company", "organization":
		return partner.CustomerTypeOrganization
	default:
		return partner.CustomerTypeIndividual
	}
}

// LookupCustomerLevel looks up a customer level by code
func (s *CustomerImportService) LookupCustomerLevel(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	if code == "" {
		return true, nil // empty is valid (will use default level)
	}
	exists, err := s.customerLevelRepo.ExistsByCode(ctx, tenantID, code)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// LookupUnique checks if a value is unique for a given field
func (s *CustomerImportService) LookupUnique(ctx context.Context, tenantID uuid.UUID, field, value string) (bool, error) {
	if value == "" {
		return false, nil // empty is not a duplicate
	}
	switch field {
	case "code":
		return s.customerRepo.ExistsByCode(ctx, tenantID, value)
	case "email":
		return s.customerRepo.ExistsByEmail(ctx, tenantID, value)
	case "phone":
		return s.customerRepo.ExistsByPhone(ctx, tenantID, value)
	default:
		return false, nil
	}
}

// Import imports customers from validated rows
func (s *CustomerImportService) Import(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	session *csvimport.ImportSession,
	validRows []*csvimport.Row,
	conflictMode ConflictMode,
) (*CustomerImportResult, error) {
	if session.State != csvimport.StateValidated {
		return nil, shared.NewDomainError("INVALID_STATE", "Import session must be in validated state")
	}

	if !session.IsValid() {
		return nil, shared.NewDomainError("VALIDATION_ERRORS", "Cannot import session with validation errors")
	}

	// Update session state
	session.UpdateState(csvimport.StateImporting)

	result := &CustomerImportResult{
		TotalRows: len(validRows),
	}
	errors := csvimport.NewErrorCollection(100)

	// Process each row
	for _, row := range validRows {
		select {
		case <-ctx.Done():
			session.UpdateState(csvimport.StateCancelled)
			return nil, ctx.Err()
		default:
		}

		err := s.importRow(ctx, tenantID, userID, row, conflictMode, result, errors)
		if err != nil {
			// Critical error - stop import
			session.UpdateState(csvimport.StateFailed)
			return nil, err
		}
	}

	// Set errors in result
	result.Errors = errors.Errors()
	result.IsTruncated = errors.IsTruncated()
	result.TotalErrors = errors.TotalCount()

	// Update session state based on result
	if result.ErrorRows > 0 {
		session.UpdateState(csvimport.StateFailed)
	} else {
		session.UpdateState(csvimport.StateCompleted)
	}

	return result, nil
}

// importRow imports a single customer row
func (s *CustomerImportService) importRow(
	ctx context.Context,
	tenantID, _ uuid.UUID,
	row *csvimport.Row,
	conflictMode ConflictMode,
	result *CustomerImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Parse row data with trimming for sanitization
	code := strings.TrimSpace(row.Get("code"))
	name := strings.TrimSpace(row.Get("name"))
	customerType := strings.TrimSpace(row.Get("type"))
	contactPerson := strings.TrimSpace(row.Get("contact_person"))
	phone := strings.TrimSpace(row.Get("phone"))
	email := strings.TrimSpace(row.Get("email"))
	levelCode := strings.TrimSpace(row.GetOrDefault("level_code", ""))
	creditLimitStr := strings.TrimSpace(row.Get("credit_limit"))
	addressProvince := strings.TrimSpace(row.Get("address_province"))
	addressCity := strings.TrimSpace(row.Get("address_city"))
	addressDistrict := strings.TrimSpace(row.Get("address_district"))
	addressDetail := strings.TrimSpace(row.Get("address_detail"))
	notes := strings.TrimSpace(row.Get("notes"))

	// Generate code if not provided
	if code == "" {
		var err error
		code, err = s.generateCode()
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "code", csvimport.ErrCodeImportValidation, "failed to generate customer code"))
			result.ErrorRows++
			return nil
		}
	}

	// Normalize code to uppercase
	code = strings.ToUpper(code)

	// Parse credit limit
	var creditLimit decimal.Decimal
	if creditLimitStr != "" {
		var err error
		creditLimit, err = decimal.NewFromString(creditLimitStr)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "credit_limit", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
			result.ErrorRows++
			return nil
		}
	}

	// Check for existing customer by code
	existingCustomer, err := s.customerRepo.FindByCode(ctx, tenantID, code)
	if err != nil && err != shared.ErrNotFound {
		return fmt.Errorf("failed to check existing customer: %w", err)
	}

	// Handle conflict
	if existingCustomer != nil {
		switch conflictMode {
		case ConflictModeSkip:
			result.SkippedRows++
			return nil
		case ConflictModeFail:
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "code", csvimport.ErrCodeImportDuplicateInDB,
				fmt.Sprintf("customer with code '%s' already exists", code), code))
			result.ErrorRows++
			return nil
		case ConflictModeUpdate:
			return s.updateExistingCustomer(ctx, tenantID, existingCustomer, row, name, contactPerson, phone, email, levelCode, creditLimit, addressProvince, addressCity, addressDistrict, addressDetail, notes, result, errors)
		}
	}

	// Normalize customer type
	normalizedType := normalizeCustomerType(customerType)

	// Create new customer
	customer, err := partner.NewCustomer(tenantID, code, name, normalizedType)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Set contact info
	if contactPerson != "" || phone != "" || email != "" {
		if err := customer.SetContact(contactPerson, phone, email); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set customer level if provided
	if levelCode != "" {
		level, err := s.lookupCustomerLevel(ctx, tenantID, levelCode)
		if err != nil {
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "level_code", csvimport.ErrCodeImportReferenceNotFound,
				fmt.Sprintf("customer level '%s' not found", levelCode), levelCode))
			result.ErrorRows++
			return nil
		}
		if err := customer.SetLevel(level); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "level_code", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set credit limit
	if !creditLimit.IsZero() {
		if err := customer.SetCreditLimit(creditLimit); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "credit_limit", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set address info
	if addressProvince != "" || addressCity != "" || addressDetail != "" {
		// Combine district with detail for backward compatibility
		fullAddress := addressDetail
		if addressDistrict != "" {
			fullAddress = addressDistrict + " " + addressDetail
		}
		// Pass empty country to use domain default (China)
		if err := customer.SetAddress(fullAddress, addressCity, addressProvince, "", ""); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set notes
	if notes != "" {
		customer.SetNotes(notes)
	}

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save customer: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := customer.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for customer %s: %v", code, err)
			}
		}
		customer.ClearDomainEvents()
	}

	result.ImportedRows++
	return nil
}

// updateExistingCustomer updates an existing customer with import data
func (s *CustomerImportService) updateExistingCustomer(
	ctx context.Context,
	tenantID uuid.UUID,
	customer *partner.Customer,
	row *csvimport.Row,
	name, contactPerson, phone, email, levelCode string,
	creditLimit decimal.Decimal,
	addressProvince, addressCity, addressDistrict, addressDetail, notes string,
	result *CustomerImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Update name (use empty shortName for update)
	if err := customer.Update(name, ""); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "name", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update contact info
	if contactPerson != "" || phone != "" || email != "" {
		if err := customer.SetContact(contactPerson, phone, email); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Update customer level if provided
	if levelCode != "" {
		level, err := s.lookupCustomerLevel(ctx, tenantID, levelCode)
		if err != nil {
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "level_code", csvimport.ErrCodeImportReferenceNotFound,
				fmt.Sprintf("customer level '%s' not found", levelCode), levelCode))
			result.ErrorRows++
			return nil
		}
		if err := customer.SetLevel(level); err != nil {
			// Ignore "same level" errors
			if !strings.Contains(err.Error(), "same") {
				errors.Add(csvimport.NewRowError(row.LineNumber, "level_code", csvimport.ErrCodeImportValidation, err.Error()))
				result.ErrorRows++
				return nil
			}
		}
	}

	// Update credit limit
	if err := customer.SetCreditLimit(creditLimit); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "credit_limit", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update address info
	if addressProvince != "" || addressCity != "" || addressDetail != "" {
		// Combine district with detail for backward compatibility
		fullAddress := addressDetail
		if addressDistrict != "" {
			fullAddress = addressDistrict + " " + addressDetail
		}
		if err := customer.SetAddress(fullAddress, addressCity, addressProvince, "", "中国"); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Update notes
	if notes != "" {
		customer.SetNotes(notes)
	}

	// Save customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save customer: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := customer.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for customer %s: %v", customer.Code, err)
			}
		}
		customer.ClearDomainEvents()
	}

	result.UpdatedRows++
	return nil
}

// lookupCustomerLevel looks up a customer level by code and returns the CustomerLevel value object
func (s *CustomerImportService) lookupCustomerLevel(ctx context.Context, tenantID uuid.UUID, code string) (partner.CustomerLevel, error) {
	levelRecord, err := s.customerLevelRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return partner.CustomerLevel{}, err
	}
	return levelRecord.ToCustomerLevel(), nil
}

// generateCode generates a unique customer code in the format CUST-{YYYYMMDD}-{SEQ}
// Uses a combination of date and timestamp-based sequence to ensure uniqueness
// even across service restarts
func (s *CustomerImportService) generateCode() (string, error) {
	s.codeSeqMu.Lock()
	defer s.codeSeqMu.Unlock()

	today := time.Now().Format("20060102")
	if s.codeSeqDate != today {
		s.codeSeqDate = today
		// Use current time-based sequence to avoid collisions after restart
		// This gives us uniqueness at millisecond level within a day
		s.codeSeqNum = time.Now().UnixMilli() % 100000
	}

	s.codeSeqNum++
	return fmt.Sprintf("CUST-%s-%06d", today, s.codeSeqNum), nil
}

// ResetCodeSequence resets the code sequence (useful for testing)
func (s *CustomerImportService) ResetCodeSequence() {
	s.codeSeqMu.Lock()
	defer s.codeSeqMu.Unlock()
	s.codeSeqDate = ""
	s.codeSeqNum = 0
}

// ValidateWithWarnings returns validation warnings (non-blocking issues)
func (s *CustomerImportService) ValidateWithWarnings(row *csvimport.Row) []string {
	var warnings []string

	// Warning: credit_limit is set very high
	creditLimitStr := row.Get("credit_limit")
	if creditLimitStr != "" {
		creditLimit, err := decimal.NewFromString(creditLimitStr)
		if err == nil && creditLimit.GreaterThan(decimal.NewFromInt(1000000)) {
			warnings = append(warnings, fmt.Sprintf("row %d: credit limit is unusually high (>1,000,000)", row.LineNumber))
		}
	}

	// Warning: email looks like a test email
	email := row.Get("email")
	if email != "" {
		lowerEmail := strings.ToLower(email)
		if strings.Contains(lowerEmail, "test") || strings.Contains(lowerEmail, "example") {
			warnings = append(warnings, fmt.Sprintf("row %d: email appears to be a test address", row.LineNumber))
		}
	}

	return warnings
}
