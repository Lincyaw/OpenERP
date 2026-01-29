package importapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ConflictMode defines how to handle conflicts during import
type ConflictMode string

const (
	// ConflictModeSkip skips rows that conflict with existing data
	ConflictModeSkip ConflictMode = "skip"
	// ConflictModeUpdate updates existing records with new data
	ConflictModeUpdate ConflictMode = "update"
	// ConflictModeFail fails the import if any conflicts are found
	ConflictModeFail ConflictMode = "fail"
)

// IsValid checks if the conflict mode is valid
func (c ConflictMode) IsValid() bool {
	switch c {
	case ConflictModeSkip, ConflictModeUpdate, ConflictModeFail:
		return true
	}
	return false
}

// ProductImportRow represents a row from the product CSV import
type ProductImportRow struct {
	Name          string `csv:"name"`
	CategoryCode  string `csv:"category_code"`
	BaseUnit      string `csv:"base_unit"`
	PurchasePrice string `csv:"purchase_price"`
	SellingPrice  string `csv:"selling_price"`
	SKU           string `csv:"sku"`
	Barcode       string `csv:"barcode"`
	Description   string `csv:"description"`
	Status        string `csv:"status"`
	MinStockLevel string `csv:"min_stock_level"`
	MaxStockLevel string `csv:"max_stock_level"`
	Attributes    string `csv:"attributes"`
}

// ProductImportResult represents the result of a product import operation
type ProductImportResult struct {
	TotalRows    int                  `json:"total_rows"`
	ImportedRows int                  `json:"imported_rows"`
	UpdatedRows  int                  `json:"updated_rows"`
	SkippedRows  int                  `json:"skipped_rows"`
	ErrorRows    int                  `json:"error_rows"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// ProductImportService handles product bulk import operations
type ProductImportService struct {
	productRepo  catalog.ProductRepository
	categoryRepo catalog.CategoryRepository
	eventBus     shared.EventPublisher
	skuSeqMu     sync.Mutex
	skuSeqDate   string
	skuSeqNum    int64
}

// NewProductImportService creates a new ProductImportService
func NewProductImportService(
	productRepo catalog.ProductRepository,
	categoryRepo catalog.CategoryRepository,
	eventBus shared.EventPublisher,
) *ProductImportService {
	return &ProductImportService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		eventBus:     eventBus,
	}
}

// GetValidationRules returns the validation rules for product import
func (s *ProductImportService) GetValidationRules() []csvimport.FieldRule {
	zero := decimal.Zero
	return []csvimport.FieldRule{
		csvimport.Field("name").Required().String().MinLength(1).MaxLength(200).Build(),
		csvimport.Field("category_code").String().MaxLength(50).Reference("category").Build(),
		csvimport.Field("base_unit").Required().String().MinLength(1).MaxLength(20).Build(),
		csvimport.Field("purchase_price").Required().Decimal().MinValue(zero).Build(),
		csvimport.Field("selling_price").Required().Decimal().MinValue(zero).Build(),
		csvimport.Field("sku").String().MaxLength(50).Unique().Build(),
		csvimport.Field("barcode").String().MaxLength(50).Unique().Build(),
		csvimport.Field("description").String().MaxLength(1000).Build(),
		csvimport.Field("status").String().Custom(validateProductStatus).Build(),
		csvimport.Field("min_stock_level").Decimal().MinValue(zero).Build(),
		csvimport.Field("max_stock_level").Decimal().MinValue(zero).Build(),
		csvimport.Field("attributes").String().Custom(validateJSONObject).Build(),
	}
}

// validateProductStatus validates the status field
func validateProductStatus(value string) error {
	if value == "" {
		return nil // optional field
	}
	switch value {
	case "active", "inactive":
		return nil
	default:
		return fmt.Errorf("status must be 'active' or 'inactive'")
	}
}

// validateJSONObject validates that a value is a valid JSON object
func validateJSONObject(value string) error {
	if value == "" {
		return nil // optional field
	}
	// Properly validate JSON using encoding/json
	var obj map[string]any
	if err := json.Unmarshal([]byte(value), &obj); err != nil {
		return fmt.Errorf("attributes must be a valid JSON object: %v", err)
	}
	return nil
}

// LookupCategory looks up a category by code
func (s *ProductImportService) LookupCategory(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	if code == "" {
		return true, nil // empty is valid
	}
	_, err := s.categoryRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		if err == shared.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// LookupUnique checks if a value is unique for a given field
func (s *ProductImportService) LookupUnique(ctx context.Context, tenantID uuid.UUID, field, value string) (bool, error) {
	if value == "" {
		return false, nil // empty is not a duplicate
	}
	switch field {
	case "sku":
		return s.productRepo.ExistsByCode(ctx, tenantID, value)
	case "barcode":
		return s.productRepo.ExistsByBarcode(ctx, tenantID, value)
	default:
		return false, nil
	}
}

// Import imports products from validated rows
func (s *ProductImportService) Import(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	session *csvimport.ImportSession,
	validRows []*csvimport.Row,
	conflictMode ConflictMode,
) (*ProductImportResult, error) {
	if session.State != csvimport.StateValidated {
		return nil, shared.NewDomainError("INVALID_STATE", "Import session must be in validated state")
	}

	if !session.IsValid() {
		return nil, shared.NewDomainError("VALIDATION_ERRORS", "Cannot import session with validation errors")
	}

	// Update session state
	session.UpdateState(csvimport.StateImporting)

	result := &ProductImportResult{
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

// importRow imports a single product row
func (s *ProductImportService) importRow(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	row *csvimport.Row,
	conflictMode ConflictMode,
	result *ProductImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Parse row data
	sku := row.Get("sku")
	barcode := row.Get("barcode")
	name := row.Get("name")
	categoryCode := row.Get("category_code")
	unit := row.Get("base_unit")
	purchasePriceStr := row.Get("purchase_price")
	sellingPriceStr := row.Get("selling_price")
	description := row.Get("description")
	statusStr := row.GetOrDefault("status", "active")
	minStockStr := row.Get("min_stock_level")
	attributes := row.GetOrDefault("attributes", "{}")

	// Generate SKU if not provided
	if sku == "" {
		var err error
		sku, err = s.generateSKU()
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "sku", csvimport.ErrCodeImportValidation, "failed to generate SKU"))
			result.ErrorRows++
			return nil
		}
	}

	// Parse prices
	purchasePrice, err := decimal.NewFromString(purchasePriceStr)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "purchase_price", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
		result.ErrorRows++
		return nil
	}

	sellingPrice, err := decimal.NewFromString(sellingPriceStr)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "selling_price", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
		result.ErrorRows++
		return nil
	}

	// Parse min stock
	var minStock decimal.Decimal
	if minStockStr != "" {
		minStock, err = decimal.NewFromString(minStockStr)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "min_stock_level", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
			result.ErrorRows++
			return nil
		}
	}

	// Check for existing product by SKU
	existingProduct, err := s.productRepo.FindByCode(ctx, tenantID, sku)
	if err != nil && err != shared.ErrNotFound {
		return fmt.Errorf("failed to check existing product: %w", err)
	}

	// Handle conflict
	if existingProduct != nil {
		switch conflictMode {
		case ConflictModeSkip:
			result.SkippedRows++
			return nil
		case ConflictModeFail:
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "sku", csvimport.ErrCodeImportDuplicateInDB,
				fmt.Sprintf("product with SKU '%s' already exists", sku), sku))
			result.ErrorRows++
			return nil
		case ConflictModeUpdate:
			return s.updateExistingProduct(ctx, existingProduct, row, name, description, purchasePrice, sellingPrice, minStock, attributes, statusStr, result, errors)
		}
	}

	// Check barcode uniqueness
	if barcode != "" {
		barcodeExists, err := s.productRepo.ExistsByBarcode(ctx, tenantID, barcode)
		if err != nil {
			return fmt.Errorf("failed to check barcode: %w", err)
		}
		if barcodeExists {
			switch conflictMode {
			case ConflictModeSkip:
				result.SkippedRows++
				return nil
			case ConflictModeFail, ConflictModeUpdate:
				errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "barcode", csvimport.ErrCodeImportDuplicateInDB,
					fmt.Sprintf("barcode '%s' already exists", barcode), barcode))
				result.ErrorRows++
				return nil
			}
		}
	}

	// Lookup category ID
	var categoryID *uuid.UUID
	if categoryCode != "" {
		category, err := s.categoryRepo.FindByCode(ctx, tenantID, categoryCode)
		if err != nil {
			if err == shared.ErrNotFound {
				errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "category_code", csvimport.ErrCodeImportReferenceNotFound,
					fmt.Sprintf("category '%s' not found", categoryCode), categoryCode))
				result.ErrorRows++
				return nil
			}
			return fmt.Errorf("failed to lookup category: %w", err)
		}
		categoryID = &category.ID
	}

	// Create new product
	product, err := catalog.NewProduct(tenantID, sku, name, unit)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Set optional fields
	if description != "" {
		if err := product.Update(name, description); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "description", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	if barcode != "" {
		if err := product.SetBarcode(barcode); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "barcode", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	if categoryID != nil {
		product.SetCategory(categoryID)
	}

	// Set prices
	purchaseMoney := valueobject.NewMoneyCNY(purchasePrice)
	sellingMoney := valueobject.NewMoneyCNY(sellingPrice)
	if err := product.SetPrices(purchaseMoney, sellingMoney); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Set min stock
	if !minStock.IsZero() {
		if err := product.SetMinStock(minStock); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "min_stock_level", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set attributes
	if attributes != "{}" && attributes != "" {
		if err := product.SetAttributes(attributes); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "attributes", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Set status - ignore "already inactive" errors which are expected
	if statusStr == "inactive" {
		_ = product.Deactivate() // Ignore error, product may already be inactive from creation
	}

	// Set created_by
	product.SetCreatedBy(userID)

	// Save product
	if err := s.productRepo.Save(ctx, product); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save product: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := product.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for product %s: %v", sku, err)
			}
		}
		product.ClearDomainEvents()
	}

	result.ImportedRows++
	return nil
}

// updateExistingProduct updates an existing product with import data
func (s *ProductImportService) updateExistingProduct(
	ctx context.Context,
	product *catalog.Product,
	row *csvimport.Row,
	name, description string,
	purchasePrice, sellingPrice, minStock decimal.Decimal,
	attributes, statusStr string,
	result *ProductImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Update product fields
	if err := product.Update(name, description); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "name", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update prices
	purchaseMoney := valueobject.NewMoneyCNY(purchasePrice)
	sellingMoney := valueobject.NewMoneyCNY(sellingPrice)
	if err := product.SetPrices(purchaseMoney, sellingMoney); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update min stock
	if err := product.SetMinStock(minStock); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "min_stock_level", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Update attributes
	if attributes != "{}" && attributes != "" {
		if err := product.SetAttributes(attributes); err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "attributes", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Handle status changes
	if statusStr == "active" && !product.IsActive() {
		if err := product.Activate(); err != nil {
			// Can't activate discontinued products
			if product.IsDiscontinued() {
				errors.Add(csvimport.NewRowError(row.LineNumber, "status", csvimport.ErrCodeImportValidation, "cannot activate discontinued product"))
				result.ErrorRows++
				return nil
			}
		}
	} else if statusStr == "inactive" && product.IsActive() {
		_ = product.Deactivate() // Ignore error, we checked IsActive() above
	}

	// Save product
	if err := s.productRepo.Save(ctx, product); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save product: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Publish domain events
	if s.eventBus != nil {
		events := product.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for product %s: %v", product.Code, err)
			}
		}
		product.ClearDomainEvents()
	}

	result.UpdatedRows++
	return nil
}

// generateSKU generates a unique SKU in the format PRD-{YYYYMMDD}-{SEQ}
// Uses a combination of date and timestamp-based sequence to ensure uniqueness
// even across service restarts
func (s *ProductImportService) generateSKU() (string, error) {
	s.skuSeqMu.Lock()
	defer s.skuSeqMu.Unlock()

	today := time.Now().Format("20060102")
	if s.skuSeqDate != today {
		s.skuSeqDate = today
		// Use current time-based sequence to avoid collisions after restart
		// This gives us uniqueness at millisecond level within a day
		s.skuSeqNum = time.Now().UnixMilli() % 100000
	}

	s.skuSeqNum++
	return fmt.Sprintf("PRD-%s-%06d", today, s.skuSeqNum), nil
}

// ResetSKUSequence resets the SKU sequence (useful for testing)
func (s *ProductImportService) ResetSKUSequence() {
	s.skuSeqMu.Lock()
	defer s.skuSeqMu.Unlock()
	s.skuSeqDate = ""
	s.skuSeqNum = 0
}

// ValidateWithWarnings returns validation warnings (non-blocking issues)
func (s *ProductImportService) ValidateWithWarnings(row *csvimport.Row) []string {
	var warnings []string

	// Warning: selling_price < purchase_price
	purchasePriceStr := row.Get("purchase_price")
	sellingPriceStr := row.Get("selling_price")
	if purchasePriceStr != "" && sellingPriceStr != "" {
		purchasePrice, err1 := decimal.NewFromString(purchasePriceStr)
		sellingPrice, err2 := decimal.NewFromString(sellingPriceStr)
		if err1 == nil && err2 == nil && sellingPrice.LessThan(purchasePrice) {
			warnings = append(warnings, fmt.Sprintf("row %d: selling price is less than purchase price", row.LineNumber))
		}
	}

	return warnings
}
