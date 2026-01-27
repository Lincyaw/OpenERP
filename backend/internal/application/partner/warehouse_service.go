package partner

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// WarehouseService handles warehouse-related business operations
type WarehouseService struct {
	warehouseRepo     partner.WarehouseRepository
	inventoryItemRepo inventory.InventoryItemRepository
}

// NewWarehouseService creates a new WarehouseService
func NewWarehouseService(warehouseRepo partner.WarehouseRepository, inventoryItemRepo inventory.InventoryItemRepository) *WarehouseService {
	return &WarehouseService{
		warehouseRepo:     warehouseRepo,
		inventoryItemRepo: inventoryItemRepo,
	}
}

// DeleteOptions contains options for warehouse deletion
type DeleteOptions struct {
	// Force allows deletion even if the warehouse has inventory (requires admin)
	Force bool
}

// Create creates a new warehouse
func (s *WarehouseService) Create(ctx context.Context, tenantID uuid.UUID, req CreateWarehouseRequest) (*WarehouseResponse, error) {
	// Check if code already exists
	exists, err := s.warehouseRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Warehouse with this code already exists")
	}

	// Create the warehouse
	warehouseType := partner.WarehouseType(req.Type)
	warehouse, err := partner.NewWarehouse(tenantID, req.Code, req.Name, warehouseType)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.ShortName != "" {
		if err := warehouse.Update(req.Name, req.ShortName); err != nil {
			return nil, err
		}
	}

	// Set contact
	if req.ContactName != "" || req.Phone != "" || req.Email != "" {
		if err := warehouse.SetContact(req.ContactName, req.Phone, req.Email); err != nil {
			return nil, err
		}
	}

	// Set address
	if req.Address != "" || req.City != "" || req.Province != "" || req.PostalCode != "" || req.Country != "" {
		if err := warehouse.SetAddress(req.Address, req.City, req.Province, req.PostalCode, req.Country); err != nil {
			return nil, err
		}
	}

	// Set capacity
	if req.Capacity != nil {
		if err := warehouse.SetCapacity(*req.Capacity); err != nil {
			return nil, err
		}
	}

	// Set notes
	if req.Notes != "" {
		warehouse.SetNotes(req.Notes)
	}

	// Set sort order
	if req.SortOrder != nil {
		warehouse.SetSortOrder(*req.SortOrder)
	}

	// Set attributes
	if req.Attributes != "" {
		if err := warehouse.SetAttributes(req.Attributes); err != nil {
			return nil, err
		}
	}

	// Handle default warehouse setting
	if req.IsDefault != nil && *req.IsDefault {
		// Clear other defaults first
		if err := s.warehouseRepo.ClearDefault(ctx, tenantID); err != nil {
			return nil, err
		}
		warehouse.SetDefault(true)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		warehouse.SetCreatedBy(*req.CreatedBy)
	}

	// Save the warehouse
	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// GetByID retrieves a warehouse by ID
func (s *WarehouseService) GetByID(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// GetByCode retrieves a warehouse by code
func (s *WarehouseService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// GetDefault retrieves the default warehouse for a tenant
func (s *WarehouseService) GetDefault(ctx context.Context, tenantID uuid.UUID) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindDefault(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// List retrieves a list of warehouses with filtering and pagination
func (s *WarehouseService) List(ctx context.Context, tenantID uuid.UUID, filter WarehouseListFilter) ([]WarehouseListResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "is_default"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "desc"
	}

	// Build domain filter
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
		Filters:  make(map[string]any),
	}

	// Add specific filters
	if filter.Status != "" {
		domainFilter.Filters["status"] = mapAPIStatusToDomain(filter.Status)
	}
	if filter.Type != "" {
		domainFilter.Filters["type"] = filter.Type
	}
	if filter.City != "" {
		domainFilter.Filters["city"] = filter.City
	}
	if filter.Province != "" {
		domainFilter.Filters["province"] = filter.Province
	}
	if filter.IsDefault != nil {
		domainFilter.Filters["is_default"] = *filter.IsDefault
	}

	// Get warehouses
	warehouses, err := s.warehouseRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.warehouseRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToWarehouseListResponses(warehouses), total, nil
}

// Update updates a warehouse
func (s *WarehouseService) Update(ctx context.Context, tenantID, warehouseID uuid.UUID, req UpdateWarehouseRequest) (*WarehouseResponse, error) {
	// Get existing warehouse
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	// Update name and short name
	if req.Name != nil {
		shortName := warehouse.ShortName
		if req.ShortName != nil {
			shortName = *req.ShortName
		}
		if err := warehouse.Update(*req.Name, shortName); err != nil {
			return nil, err
		}
	} else if req.ShortName != nil {
		if err := warehouse.Update(warehouse.Name, *req.ShortName); err != nil {
			return nil, err
		}
	}

	// Update contact
	if req.ContactName != nil || req.Phone != nil || req.Email != nil {
		contactName := warehouse.ContactName
		phone := warehouse.Phone
		email := warehouse.Email

		if req.ContactName != nil {
			contactName = *req.ContactName
		}
		if req.Phone != nil {
			phone = *req.Phone
		}
		if req.Email != nil {
			email = *req.Email
		}

		if err := warehouse.SetContact(contactName, phone, email); err != nil {
			return nil, err
		}
	}

	// Update address
	if req.Address != nil || req.City != nil || req.Province != nil || req.PostalCode != nil || req.Country != nil {
		address := warehouse.Address
		city := warehouse.City
		province := warehouse.Province
		postalCode := warehouse.PostalCode
		country := warehouse.Country

		if req.Address != nil {
			address = *req.Address
		}
		if req.City != nil {
			city = *req.City
		}
		if req.Province != nil {
			province = *req.Province
		}
		if req.PostalCode != nil {
			postalCode = *req.PostalCode
		}
		if req.Country != nil {
			country = *req.Country
		}

		if err := warehouse.SetAddress(address, city, province, postalCode, country); err != nil {
			return nil, err
		}
	}

	// Update capacity
	if req.Capacity != nil {
		if err := warehouse.SetCapacity(*req.Capacity); err != nil {
			return nil, err
		}
	}

	// Handle default warehouse setting
	if req.IsDefault != nil {
		if *req.IsDefault && !warehouse.IsDefault {
			// Clear other defaults first
			if err := s.warehouseRepo.ClearDefault(ctx, tenantID); err != nil {
				return nil, err
			}
			warehouse.SetDefault(true)
		} else if !*req.IsDefault && warehouse.IsDefault {
			warehouse.SetDefault(false)
		}
	}

	// Update notes
	if req.Notes != nil {
		warehouse.SetNotes(*req.Notes)
	}

	// Update sort order
	if req.SortOrder != nil {
		warehouse.SetSortOrder(*req.SortOrder)
	}

	// Update attributes
	if req.Attributes != nil {
		if err := warehouse.SetAttributes(*req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the warehouse
	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// UpdateCode updates a warehouse's code
func (s *WarehouseService) UpdateCode(ctx context.Context, tenantID, warehouseID uuid.UUID, newCode string) (*WarehouseResponse, error) {
	// Get existing warehouse
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	// Check if new code already exists (if different from current)
	if newCode != warehouse.Code {
		exists, err := s.warehouseRepo.ExistsByCode(ctx, tenantID, newCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Warehouse with this code already exists")
		}
	}

	// Update the code
	if err := warehouse.UpdateCode(newCode); err != nil {
		return nil, err
	}

	// Save the warehouse
	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// Delete deletes a warehouse (without force option)
func (s *WarehouseService) Delete(ctx context.Context, tenantID, warehouseID uuid.UUID) error {
	return s.DeleteWithOptions(ctx, tenantID, warehouseID, DeleteOptions{Force: false})
}

// DeleteWithOptions deletes a warehouse with optional force deletion
// When force is false, deletion is blocked if the warehouse has inventory items
// When force is true (requires admin permission checked by handler), inventory items are orphaned
func (s *WarehouseService) DeleteWithOptions(ctx context.Context, tenantID, warehouseID uuid.UUID, opts DeleteOptions) error {
	// Verify warehouse exists
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return err
	}

	// Cannot delete default warehouse
	if warehouse.IsDefault {
		return shared.NewDomainError("CANNOT_DELETE", "Cannot delete the default warehouse")
	}

	// Check if warehouse has inventory items (unless force delete)
	if !opts.Force && s.inventoryItemRepo != nil {
		inventoryCount, err := s.inventoryItemRepo.CountByWarehouse(ctx, tenantID, warehouseID)
		if err != nil {
			return shared.NewDomainError("INVENTORY_CHECK_FAILED", "Failed to check warehouse inventory. Please try again or contact support.")
		}

		if inventoryCount > 0 {
			return shared.NewDomainError("HAS_INVENTORY",
				fmt.Sprintf("Cannot delete warehouse: %d inventory item(s) exist in this warehouse. Transfer or clear inventory first, or use force delete with admin permission.", inventoryCount))
		}
	}

	return s.warehouseRepo.DeleteForTenant(ctx, tenantID, warehouseID)
}

// Enable enables a warehouse
func (s *WarehouseService) Enable(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	if err := warehouse.Enable(); err != nil {
		return nil, err
	}

	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// Disable disables a warehouse
func (s *WarehouseService) Disable(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	if err := warehouse.Disable(); err != nil {
		return nil, err
	}

	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// SetDefault sets a warehouse as the default warehouse
func (s *WarehouseService) SetDefault(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseResponse, error) {
	warehouse, err := s.warehouseRepo.FindByIDForTenant(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}

	// Cannot set inactive warehouse as default
	if !warehouse.IsActive() {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot set an inactive warehouse as default")
	}

	// If already default, nothing to do
	if warehouse.IsDefault {
		response := ToWarehouseResponse(warehouse)
		return &response, nil
	}

	// Clear other defaults first
	if err := s.warehouseRepo.ClearDefault(ctx, tenantID); err != nil {
		return nil, err
	}

	warehouse.SetDefault(true)

	if err := s.warehouseRepo.Save(ctx, warehouse); err != nil {
		return nil, err
	}

	response := ToWarehouseResponse(warehouse)
	return &response, nil
}

// CountByStatus returns warehouse counts by status for a tenant
func (s *WarehouseService) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	activeCount, err := s.warehouseRepo.CountByStatus(ctx, tenantID, partner.WarehouseStatusActive)
	if err != nil {
		return nil, err
	}
	counts["active"] = activeCount

	inactiveCount, err := s.warehouseRepo.CountByStatus(ctx, tenantID, partner.WarehouseStatusInactive)
	if err != nil {
		return nil, err
	}
	counts["inactive"] = inactiveCount

	counts["total"] = activeCount + inactiveCount

	return counts, nil
}

// CountByType returns warehouse counts by type for a tenant
func (s *WarehouseService) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	types := []partner.WarehouseType{
		partner.WarehouseTypePhysical,
		partner.WarehouseTypeVirtual,
		partner.WarehouseTypeConsign,
		partner.WarehouseTypeTransit,
	}

	var total int64
	for _, t := range types {
		count, err := s.warehouseRepo.CountByType(ctx, tenantID, t)
		if err != nil {
			return nil, err
		}
		counts[string(t)] = count
		total += count
	}
	counts["total"] = total

	return counts, nil
}
