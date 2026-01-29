package importapp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierImportRow represents a row from the supplier CSV import
type SupplierImportRow struct {
	Name            string `csv:"name"`
	Code            string `csv:"code"`
	Type            string `csv:"type"`
	ContactPerson   string `csv:"contact_person"`
	Phone           string `csv:"phone"`
	Email           string `csv:"email"`
	CreditDays      string `csv:"credit_days"`
	CreditLimit     string `csv:"credit_limit"`
	AddressProvince string `csv:"address_province"`
	AddressCity     string `csv:"address_city"`
	AddressDistrict string `csv:"address_district"`
	AddressDetail   string `csv:"address_detail"`
	BankName        string `csv:"bank_name"`
	BankAccount     string `csv:"bank_account"`
	Notes           string `csv:"notes"`
}

// SupplierImportResult represents the result of a supplier import operation
type SupplierImportResult struct {
	TotalRows    int                  `json:"total_rows"`
	ImportedRows int                  `json:"imported_rows"`
	UpdatedRows  int                  `json:"updated_rows"`
	SkippedRows  int                  `json:"skipped_rows"`
	ErrorRows    int                  `json:"error_rows"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// SupplierImportService handles supplier bulk import operations
type SupplierImportService struct {
	supplierRepo partner.SupplierRepository
	eventBus     shared.EventPublisher
	codeSeqMu    sync.Mutex
	codeSeqDate  string
	codeSeqNum   int64
}

// NewSupplierImportService creates a new SupplierImportService
func NewSupplierImportService(
	supplierRepo partner.SupplierRepository,
	eventBus shared.EventPublisher,
) *SupplierImportService {
	return &SupplierImportService{
		supplierRepo: supplierRepo,
		eventBus:     eventBus,
	}
}

// GetValidationRules returns the validation rules for supplier import
func (s *SupplierImportService) GetValidationRules() []csvimport.FieldRule {
	zero := decimal.Zero
	return []csvimport.FieldRule{
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("code").String().MaxLength(50).Unique().Build(),
		csvimport.Field("type").String().Custom(validateSupplierType).Build(),
		csvimport.Field("contact_person").String().MaxLength(100).Build(),
		csvimport.Field("phone").String().MaxLength(20).Build(),
		csvimport.Field("email").String().Email().Build(),
		csvimport.Field("credit_days").Int().MinValue(zero).Build(),
		csvimport.Field("credit_limit").Decimal().MinValue(zero).Build(),
		csvimport.Field("address_province").String().MaxLength(50).Build(),
		csvimport.Field("address_city").String().MaxLength(50).Build(),
		csvimport.Field("address_district").String().MaxLength(50).Build(),
		csvimport.Field("address_detail").String().MaxLength(200).Build(),
		csvimport.Field("bank_name").String().MaxLength(100).Build(),
		csvimport.Field("bank_account").String().MaxLength(50).Build(),
		csvimport.Field("notes").String().MaxLength(1000).Build(),
	}
}

// validateSupplierType validates the supplier type field
func validateSupplierType(value string) error {
	if value == "" {
		return nil // will use default type
	}
	lower := strings.ToLower(value)
	switch lower {
	case "manufacturer", "distributor", "retailer", "service":
		return nil
	default:
		return fmt.Errorf("type must be 'manufacturer', 'distributor', 'retailer', or 'service'")
	}
}

// normalizeSupplierType normalizes the supplier type input
func normalizeSupplierType(value string) partner.SupplierType {
	lower := strings.ToLower(value)
	switch lower {
	case "manufacturer":
		return partner.SupplierTypeManufacturer
	case "distributor":
		return partner.SupplierTypeDistributor
	case "retailer":
		return partner.SupplierTypeRetailer
	case "service":
		return partner.SupplierTypeService
	default:
		// Default to distributor if not specified
		return partner.SupplierTypeDistributor
	}
}

// LookupUnique checks if a value is unique for a given field
func (s *SupplierImportService) LookupUnique(ctx context.Context, tenantID uuid.UUID, field, value string) (bool, error) {
	if value == "" {
		return false, nil // empty is not a duplicate
	}
	switch field {
	case "code":
		return s.supplierRepo.ExistsByCode(ctx, tenantID, value)
	case "email":
		return s.supplierRepo.ExistsByEmail(ctx, tenantID, value)
	case "phone":
		return s.supplierRepo.ExistsByPhone(ctx, tenantID, value)
	default:
		return false, nil
	}
}

// Import imports suppliers from validated rows
func (s *SupplierImportService) Import(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	session *csvimport.ImportSession,
	validRows []*csvimport.Row,
	conflictMode ConflictMode,
) (*SupplierImportResult, error) {
	if session.State != csvimport.StateValidated {
		return nil, shared.NewDomainError("INVALID_STATE", "Import session must be in validated state")
	}

	if !session.IsValid() {
		return nil, shared.NewDomainError("VALIDATION_ERRORS", "Cannot import session with validation errors")
	}

	// Update session state
	session.UpdateState(csvimport.StateImporting)

	result := &SupplierImportResult{
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

// importRow imports a single supplier row
func (s *SupplierImportService) importRow(
	ctx context.Context,
	tenantID, _ uuid.UUID,
	row *csvimport.Row,
	conflictMode ConflictMode,
	result *SupplierImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Parse row data with trimming for sanitization
	code := strings.TrimSpace(row.Get("code"))
	name := strings.TrimSpace(row.Get("name"))
	supplierType := strings.TrimSpace(row.Get("type"))
	contactPerson := strings.TrimSpace(row.Get("contact_person"))
	phone := strings.TrimSpace(row.Get("phone"))
	email := strings.TrimSpace(row.Get("email"))
	creditDaysStr := strings.TrimSpace(row.Get("credit_days"))
	creditLimitStr := strings.TrimSpace(row.Get("credit_limit"))
	addressProvince := strings.TrimSpace(row.Get("address_province"))
	addressCity := strings.TrimSpace(row.Get("address_city"))
	addressDistrict := strings.TrimSpace(row.Get("address_district"))
	addressDetail := strings.TrimSpace(row.Get("address_detail"))
	bankName := strings.TrimSpace(row.Get("bank_name"))
	bankAccount := strings.TrimSpace(row.Get("bank_account"))
	notes := strings.TrimSpace(row.Get("notes"))

	// Generate code if not provided
	if code == "" {
		var err error
		code, err = s.generateCode()
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "code", csvimport.ErrCodeImportValidation, "failed to generate supplier code"))
			result.ErrorRows++
			return nil
		}
	}

	// Normalize code to uppercase
	code = strings.ToUpper(code)

	// Parse credit days (default to 0)
	var creditDays int
	if creditDaysStr != "" {
		var err error
		creditDays, err = strconv.Atoi(creditDaysStr)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "credit_days", csvimport.ErrCodeImportInvalidType, "invalid integer value"))
			result.ErrorRows++
			return nil
		}
	}

	// Parse credit limit (default to 0)
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

	// Check for existing supplier by code
	existingSupplier, err := s.supplierRepo.FindByCode(ctx, tenantID, code)
	if err != nil && err != shared.ErrNotFound {
		return fmt.Errorf("failed to check existing supplier: %w", err)
	}

	// Handle conflict
	if existingSupplier != nil {
		switch conflictMode {
		case ConflictModeSkip:
			result.SkippedRows++
			return nil
		case ConflictModeFail:
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "code", csvimport.ErrCodeImportDuplicateInDB,
				fmt.Sprintf("supplier with code '%s' already exists", code), code))
			result.ErrorRows++
			return nil
		case ConflictModeUpdate:
			return s.updateExistingSupplier(ctx, tenantID, existingSupplier, row, name, contactPerson, phone, email, creditDays, creditLimit, addressProvince, addressCity, addressDistrict, addressDetail, bankName, bankAccount, notes, result, errors)
		}
	}

	// Normalize supplier type
	normalizedType := normalizeSupplierType(supplierType)

	// Create new supplier
	supplier, err := partner.NewSupplier(tenantID, code, name, normalizedType)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Set contact info
	if contactPerson != "" || phone != "" || email != "" {
		if err := supplier.SetContact(contactPerson, phone, email); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set payment terms (credit days and limit)
	if creditDays > 0 || !creditLimit.IsZero() {
		if err := supplier.SetPaymentTerms(creditDays, creditLimit); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
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
		if err := supplier.SetAddress(fullAddress, addressCity, addressProvince, "", ""); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set bank info
	if bankName != "" || bankAccount != "" {
		if err := supplier.SetBankInfo(bankName, bankAccount); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set notes
	if notes != "" {
		supplier.SetNotes(notes)
	}

	// Save supplier
	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save supplier: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := supplier.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for supplier %s: %v", code, err)
			}
		}
		supplier.ClearDomainEvents()
	}

	result.ImportedRows++
	return nil
}

// updateExistingSupplier updates an existing supplier with import data
func (s *SupplierImportService) updateExistingSupplier(
	ctx context.Context,
	_ uuid.UUID,
	supplier *partner.Supplier,
	row *csvimport.Row,
	name, contactPerson, phone, email string,
	creditDays int,
	creditLimit decimal.Decimal,
	addressProvince, addressCity, addressDistrict, addressDetail, bankName, bankAccount, notes string,
	result *SupplierImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Update name
	if err := supplier.Update(name, ""); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "name", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update contact info
	if contactPerson != "" || phone != "" || email != "" {
		if err := supplier.SetContact(contactPerson, phone, email); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Update payment terms
	if err := supplier.SetPaymentTerms(creditDays, creditLimit); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
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
		if err := supplier.SetAddress(fullAddress, addressCity, addressProvince, "", ""); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Update bank info
	if bankName != "" || bankAccount != "" {
		if err := supplier.SetBankInfo(bankName, bankAccount); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Update notes
	if notes != "" {
		supplier.SetNotes(notes)
	}

	// Save supplier
	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save supplier: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := supplier.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for supplier %s: %v", supplier.Code, err)
			}
		}
		supplier.ClearDomainEvents()
	}

	result.UpdatedRows++
	return nil
}

// generateCode generates a unique supplier code in the format SUPP-{YYYYMMDD}-{SEQ}
// Uses a combination of date and timestamp-based sequence to ensure uniqueness
// even across service restarts
func (s *SupplierImportService) generateCode() (string, error) {
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
	return fmt.Sprintf("SUPP-%s-%06d", today, s.codeSeqNum), nil
}

// ResetCodeSequence resets the code sequence (useful for testing)
func (s *SupplierImportService) ResetCodeSequence() {
	s.codeSeqMu.Lock()
	defer s.codeSeqMu.Unlock()
	s.codeSeqDate = ""
	s.codeSeqNum = 0
}

// ValidateWithWarnings returns validation warnings (non-blocking issues)
func (s *SupplierImportService) ValidateWithWarnings(row *csvimport.Row) []string {
	var warnings []string

	// Warning: credit_limit is set very high
	creditLimitStr := row.Get("credit_limit")
	if creditLimitStr != "" {
		creditLimit, err := decimal.NewFromString(creditLimitStr)
		if err == nil && creditLimit.GreaterThan(decimal.NewFromInt(10000000)) {
			warnings = append(warnings, fmt.Sprintf("row %d: credit limit is unusually high (>10,000,000)", row.LineNumber))
		}
	}

	// Warning: credit_days is very long
	creditDaysStr := row.Get("credit_days")
	if creditDaysStr != "" {
		creditDays, err := strconv.Atoi(creditDaysStr)
		if err == nil && creditDays > 180 {
			warnings = append(warnings, fmt.Sprintf("row %d: credit days is unusually high (>180 days)", row.LineNumber))
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
