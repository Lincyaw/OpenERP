package importapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	csvimport "github.com/erp/backend/internal/infrastructure/import"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InventoryImportRow represents a row from the inventory CSV import
type InventoryImportRow struct {
	ProductSKU     string `csv:"product_sku"`
	WarehouseCode  string `csv:"warehouse_code"`
	Quantity       string `csv:"quantity"`
	UnitCost       string `csv:"unit_cost"`
	BatchNumber    string `csv:"batch_number"`
	ProductionDate string `csv:"production_date"`
	ExpiryDate     string `csv:"expiry_date"`
	Notes          string `csv:"notes"`
}

// InventoryImportResult represents the result of an inventory import operation
type InventoryImportResult struct {
	TotalRows    int                  `json:"total_rows"`
	ImportedRows int                  `json:"imported_rows"`
	UpdatedRows  int                  `json:"updated_rows"`
	SkippedRows  int                  `json:"skipped_rows"`
	ErrorRows    int                  `json:"error_rows"`
	Errors       []csvimport.RowError `json:"errors,omitempty"`
	IsTruncated  bool                 `json:"is_truncated,omitempty"`
	TotalErrors  int                  `json:"total_errors,omitempty"`
}

// InventoryImportService handles inventory bulk import operations for initial data migration
type InventoryImportService struct {
	inventoryRepo   inventory.InventoryItemRepository
	transactionRepo inventory.InventoryTransactionRepository
	productRepo     catalog.ProductRepository
	warehouseRepo   partner.WarehouseRepository
	eventBus        shared.EventPublisher
}

// NewInventoryImportService creates a new InventoryImportService
func NewInventoryImportService(
	inventoryRepo inventory.InventoryItemRepository,
	transactionRepo inventory.InventoryTransactionRepository,
	productRepo catalog.ProductRepository,
	warehouseRepo partner.WarehouseRepository,
	eventBus shared.EventPublisher,
) *InventoryImportService {
	return &InventoryImportService{
		inventoryRepo:   inventoryRepo,
		transactionRepo: transactionRepo,
		productRepo:     productRepo,
		warehouseRepo:   warehouseRepo,
		eventBus:        eventBus,
	}
}

// GetValidationRules returns the validation rules for inventory import
func (s *InventoryImportService) GetValidationRules() []csvimport.FieldRule {
	zero := decimal.Zero
	return []csvimport.FieldRule{
		csvimport.Field("product_sku").Required().String().MinLength(1).MaxLength(50).Build(),
		csvimport.Field("warehouse_code").Required().String().MinLength(1).MaxLength(50).Build(),
		csvimport.Field("quantity").Required().Decimal().MinValue(zero).Build(),
		csvimport.Field("unit_cost").Required().Decimal().MinValue(zero).Build(),
		csvimport.Field("batch_number").String().MaxLength(50).Build(),
		csvimport.Field("production_date").Date().Build(),
		csvimport.Field("expiry_date").Date().Build(),
		csvimport.Field("notes").String().MaxLength(500).Build(),
	}
}

// LookupReference checks if a reference (product or warehouse) exists
func (s *InventoryImportService) LookupReference(ctx context.Context, tenantID uuid.UUID, refType, value string) (bool, error) {
	if value == "" {
		return false, nil // empty is not found (but also not an error)
	}

	switch refType {
	case "products":
		product, err := s.productRepo.FindByCode(ctx, tenantID, value)
		if err == shared.ErrNotFound {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return product != nil, nil
	case "warehouses":
		warehouse, err := s.warehouseRepo.FindByCode(ctx, tenantID, value)
		if err == shared.ErrNotFound {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return warehouse != nil, nil
	default:
		return false, nil
	}
}

// LookupUnique checks if a combination of product+warehouse+batch is unique
// For inventory import, uniqueness is handled differently - we check if inventory already exists
func (s *InventoryImportService) LookupUnique(ctx context.Context, tenantID uuid.UUID, field, value string) (bool, error) {
	// Inventory import doesn't use simple field-based uniqueness
	// The uniqueness check for product+warehouse combination is done during import
	return false, nil
}

// Import imports inventory from validated rows
func (s *InventoryImportService) Import(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	session *csvimport.ImportSession,
	validRows []*csvimport.Row,
	conflictMode ConflictMode,
) (*InventoryImportResult, error) {
	if session.State != csvimport.StateValidated {
		return nil, shared.NewDomainError("INVALID_STATE", "Import session must be in validated state")
	}

	if !session.IsValid() {
		return nil, shared.NewDomainError("VALIDATION_ERRORS", "Cannot import session with validation errors")
	}

	// Update session state
	session.UpdateState(csvimport.StateImporting)

	result := &InventoryImportResult{
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

		err := s.importRow(ctx, tenantID, userID, session, row, conflictMode, result, errors)
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

// importRow imports a single inventory row
func (s *InventoryImportService) importRow(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	session *csvimport.ImportSession,
	row *csvimport.Row,
	conflictMode ConflictMode,
	result *InventoryImportResult,
	errors *csvimport.ErrorCollection,
) error {
	// Parse row data with trimming for sanitization
	productSKU := strings.TrimSpace(row.Get("product_sku"))
	warehouseCode := strings.TrimSpace(row.Get("warehouse_code"))
	quantityStr := strings.TrimSpace(row.Get("quantity"))
	unitCostStr := strings.TrimSpace(row.Get("unit_cost"))
	batchNumber := strings.TrimSpace(row.Get("batch_number"))
	productionDateStr := strings.TrimSpace(row.Get("production_date"))
	expiryDateStr := strings.TrimSpace(row.Get("expiry_date"))
	notes := strings.TrimSpace(row.Get("notes"))

	// Parse quantity
	quantity, err := decimal.NewFromString(quantityStr)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "quantity", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
		result.ErrorRows++
		return nil
	}

	// Parse unit cost
	unitCost, err := decimal.NewFromString(unitCostStr)
	if err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "unit_cost", csvimport.ErrCodeImportInvalidType, "invalid decimal value"))
		result.ErrorRows++
		return nil
	}

	// Lookup product by SKU/code
	product, err := s.productRepo.FindByCode(ctx, tenantID, productSKU)
	if err != nil {
		if err == shared.ErrNotFound {
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "product_sku", csvimport.ErrCodeImportReferenceNotFound,
				fmt.Sprintf("product with SKU '%s' not found", productSKU), productSKU))
			result.ErrorRows++
			return nil
		}
		return fmt.Errorf("failed to lookup product: %w", err)
	}

	// Lookup warehouse by code
	warehouse, err := s.warehouseRepo.FindByCode(ctx, tenantID, warehouseCode)
	if err != nil {
		if err == shared.ErrNotFound {
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "warehouse_code", csvimport.ErrCodeImportReferenceNotFound,
				fmt.Sprintf("warehouse with code '%s' not found", warehouseCode), warehouseCode))
			result.ErrorRows++
			return nil
		}
		return fmt.Errorf("failed to lookup warehouse: %w", err)
	}

	// Parse production date if provided
	var productionDate *time.Time
	if productionDateStr != "" {
		parsedDate, err := parseDate(productionDateStr)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "production_date", csvimport.ErrCodeImportInvalidType, "invalid date format"))
			result.ErrorRows++
			return nil
		}
		productionDate = &parsedDate
	}

	// Parse expiry date if provided
	var expiryDate *time.Time
	if expiryDateStr != "" {
		parsedDate, err := parseDate(expiryDateStr)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "expiry_date", csvimport.ErrCodeImportInvalidType, "invalid date format"))
			result.ErrorRows++
			return nil
		}
		expiryDate = &parsedDate
	}

	// Validate expiry_date > production_date if both provided
	if productionDate != nil && expiryDate != nil {
		if !expiryDate.After(*productionDate) {
			errors.Add(csvimport.NewRowError(row.LineNumber, "expiry_date", csvimport.ErrCodeImportValidation,
				"expiry_date must be after production_date"))
			result.ErrorRows++
			return nil
		}
	}

	// Check if inventory already exists for this product+warehouse combination
	existingItem, err := s.inventoryRepo.FindByWarehouseAndProduct(ctx, tenantID, warehouse.ID, product.ID)
	if err != nil && err != shared.ErrNotFound {
		return fmt.Errorf("failed to check existing inventory: %w", err)
	}

	// If batch is provided, check for duplicate batch within the same product+warehouse
	if batchNumber != "" && existingItem != nil {
		for _, batch := range existingItem.Batches {
			if batch.BatchNumber == batchNumber {
				switch conflictMode {
				case ConflictModeSkip:
					result.SkippedRows++
					return nil
				case ConflictModeFail:
					errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "batch_number", csvimport.ErrCodeImportDuplicateInDB,
						fmt.Sprintf("batch '%s' already exists for product '%s' in warehouse '%s'", batchNumber, productSKU, warehouseCode), batchNumber))
					result.ErrorRows++
					return nil
				case ConflictModeUpdate:
					// Update mode: update the batch quantity
					return s.updateExistingBatch(ctx, tenantID, userID, existingItem, &batch, quantity, unitCost, notes, result, errors, row)
				}
			}
		}
	}

	// Prepare batch info if batch number is provided
	var batchInfo *inventory.BatchInfo
	if batchNumber != "" {
		batchInfo = inventory.NewBatchInfo(batchNumber, productionDate, expiryDate)
	}

	// Handle existing inventory item
	if existingItem != nil && batchNumber == "" {
		// No batch specified, we're dealing with existing inventory without batch
		switch conflictMode {
		case ConflictModeSkip:
			result.SkippedRows++
			return nil
		case ConflictModeFail:
			errors.Add(csvimport.NewRowErrorWithValue(row.LineNumber, "product_sku", csvimport.ErrCodeImportDuplicateInDB,
				fmt.Sprintf("inventory already exists for product '%s' in warehouse '%s'", productSKU, warehouseCode), productSKU))
			result.ErrorRows++
			return nil
		case ConflictModeUpdate:
			return s.updateExistingInventory(ctx, tenantID, userID, existingItem, quantity, unitCost, batchInfo, notes, result, errors, row)
		}
	}

	// Create or update inventory item
	var item *inventory.InventoryItem
	if existingItem != nil {
		item = existingItem
	} else {
		item, err = inventory.NewInventoryItem(tenantID, warehouse.ID, product.ID)
		if err != nil {
			errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
			result.ErrorRows++
			return nil
		}
	}

	// Record balance before import
	balanceBefore := item.AvailableQuantity.Amount()

	// Create Money value for unit cost
	unitCostMoney := valueobject.NewMoneyCNY(unitCost)

	// Increase stock (this will calculate weighted average cost and create batch if needed)
	if err := item.IncreaseStock(quantity, unitCostMoney, batchInfo); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Save inventory item
	if err := s.inventoryRepo.Save(ctx, item); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save inventory: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Create inventory transaction for audit
	balanceAfter := item.AvailableQuantity.Amount()
	txn, err := inventory.NewInventoryTransaction(
		tenantID,
		item.ID,
		warehouse.ID,
		product.ID,
		inventory.TransactionTypeInbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		inventory.SourceTypeInitialStock,
		session.ID.String(),
	)
	if err != nil {
		log.Printf("WARNING: failed to create transaction for inventory import: %v", err)
	} else {
		txn.WithOperatorID(userID)
		txn.WithReason("Initial inventory import" + noteSuffix(notes))
		txn.WithCostMethod("moving_average")
		if err := s.transactionRepo.Create(ctx, txn); err != nil {
			log.Printf("WARNING: failed to save inventory transaction: %v", err)
		}
	}

	// Publish domain events
	if s.eventBus != nil {
		events := item.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for inventory import: %v", err)
			}
		}
		item.ClearDomainEvents()
	}

	result.ImportedRows++
	return nil
}

// updateExistingInventory updates an existing inventory item (for update mode)
func (s *InventoryImportService) updateExistingInventory(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	item *inventory.InventoryItem,
	quantity decimal.Decimal,
	unitCost decimal.Decimal,
	batchInfo *inventory.BatchInfo,
	notes string,
	result *InventoryImportResult,
	errors *csvimport.ErrorCollection,
	row *csvimport.Row,
) error {
	// Record balance before update
	balanceBefore := item.AvailableQuantity.Amount()

	// Create Money value for unit cost
	unitCostMoney := valueobject.NewMoneyCNY(unitCost)

	// Increase stock (this will recalculate weighted average cost)
	if err := item.IncreaseStock(quantity, unitCostMoney, batchInfo); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Save inventory item
	if err := s.inventoryRepo.Save(ctx, item); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save inventory: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Create inventory transaction for audit
	balanceAfter := item.AvailableQuantity.Amount()
	txn, err := inventory.NewInventoryTransaction(
		tenantID,
		item.ID,
		item.WarehouseID,
		item.ProductID,
		inventory.TransactionTypeInbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		inventory.SourceTypeInitialStock,
		"import-update",
	)
	if err != nil {
		log.Printf("WARNING: failed to create transaction for inventory update: %v", err)
	} else {
		txn.WithOperatorID(userID)
		txn.WithReason("Inventory import update" + noteSuffix(notes))
		txn.WithCostMethod("moving_average")
		if err := s.transactionRepo.Create(ctx, txn); err != nil {
			log.Printf("WARNING: failed to save inventory transaction: %v", err)
		}
	}

	// Publish domain events
	if s.eventBus != nil {
		events := item.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for inventory update: %v", err)
			}
		}
		item.ClearDomainEvents()
	}

	result.UpdatedRows++
	return nil
}

// updateExistingBatch updates an existing batch (for update mode when batch exists)
func (s *InventoryImportService) updateExistingBatch(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	item *inventory.InventoryItem,
	existingBatch *inventory.StockBatch,
	quantity decimal.Decimal,
	unitCost decimal.Decimal,
	notes string,
	result *InventoryImportResult,
	errors *csvimport.ErrorCollection,
	row *csvimport.Row,
) error {
	// For batch update, we add to the existing batch quantity
	// Record balance before update
	balanceBefore := item.AvailableQuantity.Amount()

	// Create batch info with the same batch number
	batchInfo := inventory.NewBatchInfo(existingBatch.BatchNumber, existingBatch.ProductionDate, existingBatch.ExpiryDate)

	// Create Money value for unit cost
	unitCostMoney := valueobject.NewMoneyCNY(unitCost)

	// Note: This creates a new batch entry. If we want to truly "update" the batch,
	// we would need to modify the domain model. For now, adding stock creates a new batch.
	if err := item.IncreaseStock(quantity, unitCostMoney, batchInfo); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, err.Error()))
		result.ErrorRows++
		return nil
	}

	// Save inventory item
	if err := s.inventoryRepo.Save(ctx, item); err != nil {
		errors.Add(csvimport.NewRowError(row.LineNumber, "", csvimport.ErrCodeImportValidation, "failed to save inventory: "+err.Error()))
		result.ErrorRows++
		return nil
	}

	// Create inventory transaction for audit
	balanceAfter := item.AvailableQuantity.Amount()
	txn, err := inventory.NewInventoryTransaction(
		tenantID,
		item.ID,
		item.WarehouseID,
		item.ProductID,
		inventory.TransactionTypeInbound,
		quantity,
		unitCost,
		balanceBefore,
		balanceAfter,
		inventory.SourceTypeInitialStock,
		"import-batch-update",
	)
	if err != nil {
		log.Printf("WARNING: failed to create transaction for batch update: %v", err)
	} else {
		txn.WithOperatorID(userID)
		txn.WithReason("Inventory import batch update" + noteSuffix(notes))
		txn.WithCostMethod("moving_average")
		txn.WithBatchID(existingBatch.ID)
		if err := s.transactionRepo.Create(ctx, txn); err != nil {
			log.Printf("WARNING: failed to save inventory transaction: %v", err)
		}
	}

	// Publish domain events
	if s.eventBus != nil {
		events := item.GetDomainEvents()
		if len(events) > 0 {
			if err := s.eventBus.Publish(ctx, events...); err != nil {
				log.Printf("WARNING: failed to publish domain events for batch update: %v", err)
			}
		}
		item.ClearDomainEvents()
	}

	result.UpdatedRows++
	return nil
}

// parseDate parses a date string in various formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02-01-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
}

// noteSuffix returns a formatted note suffix for transaction reasons
func noteSuffix(notes string) string {
	if notes == "" {
		return ""
	}
	return " - " + notes
}

// ValidateWithWarnings returns validation warnings (non-blocking issues)
func (s *InventoryImportService) ValidateWithWarnings(row *csvimport.Row) []string {
	var warnings []string

	// Warning: quantity is very high
	quantityStr := row.Get("quantity")
	if quantityStr != "" {
		quantity, err := decimal.NewFromString(quantityStr)
		if err == nil && quantity.GreaterThan(decimal.NewFromInt(1000000)) {
			warnings = append(warnings, fmt.Sprintf("row %d: quantity is unusually high (>1,000,000)", row.LineNumber))
		}
	}

	// Warning: unit_cost is very high
	unitCostStr := row.Get("unit_cost")
	if unitCostStr != "" {
		unitCost, err := decimal.NewFromString(unitCostStr)
		if err == nil && unitCost.GreaterThan(decimal.NewFromInt(1000000)) {
			warnings = append(warnings, fmt.Sprintf("row %d: unit cost is unusually high (>1,000,000)", row.LineNumber))
		}
	}

	// Warning: expiry date is in the past
	expiryDateStr := row.Get("expiry_date")
	if expiryDateStr != "" {
		if expiryDate, err := parseDate(expiryDateStr); err == nil {
			if expiryDate.Before(time.Now()) {
				warnings = append(warnings, fmt.Sprintf("row %d: expiry date is in the past", row.LineNumber))
			}
		}
	}

	// Warning: batch_number without expiry_date (may be intentional for non-perishables)
	batchNumber := row.Get("batch_number")
	if batchNumber != "" && expiryDateStr == "" {
		warnings = append(warnings, fmt.Sprintf("row %d: batch number specified without expiry date", row.LineNumber))
	}

	return warnings
}
