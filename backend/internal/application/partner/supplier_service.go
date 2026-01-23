package partner

import (
	"context"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierService handles supplier-related business operations
type SupplierService struct {
	supplierRepo partner.SupplierRepository
}

// NewSupplierService creates a new SupplierService
func NewSupplierService(supplierRepo partner.SupplierRepository) *SupplierService {
	return &SupplierService{
		supplierRepo: supplierRepo,
	}
}

// Create creates a new supplier
func (s *SupplierService) Create(ctx context.Context, tenantID uuid.UUID, req CreateSupplierRequest) (*SupplierResponse, error) {
	// Check if code already exists
	exists, err := s.supplierRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this code already exists")
	}

	// Check if phone already exists (if provided)
	if req.Phone != "" {
		exists, err = s.supplierRepo.ExistsByPhone(ctx, tenantID, req.Phone)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this phone already exists")
		}
	}

	// Check if email already exists (if provided)
	if req.Email != "" {
		exists, err = s.supplierRepo.ExistsByEmail(ctx, tenantID, req.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this email already exists")
		}
	}

	// Create the supplier
	supplierType := partner.SupplierType(req.Type)
	supplier, err := partner.NewSupplier(tenantID, req.Code, req.Name, supplierType)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.ShortName != "" {
		if err := supplier.Update(req.Name, req.ShortName); err != nil {
			return nil, err
		}
	}

	// Set contact
	if req.ContactName != "" || req.Phone != "" || req.Email != "" {
		if err := supplier.SetContact(req.ContactName, req.Phone, req.Email); err != nil {
			return nil, err
		}
	}

	// Set address
	if req.Address != "" || req.City != "" || req.Province != "" || req.PostalCode != "" || req.Country != "" {
		if err := supplier.SetAddress(req.Address, req.City, req.Province, req.PostalCode, req.Country); err != nil {
			return nil, err
		}
	}

	// Set tax ID
	if req.TaxID != "" {
		if err := supplier.SetTaxID(req.TaxID); err != nil {
			return nil, err
		}
	}

	// Set bank info
	if req.BankName != "" || req.BankAccount != "" {
		if err := supplier.SetBankInfo(req.BankName, req.BankAccount); err != nil {
			return nil, err
		}
	}

	// Set payment terms
	creditDays := 0
	creditLimit := decimal.Zero
	if req.CreditDays != nil {
		creditDays = *req.CreditDays
	}
	if req.CreditLimit != nil {
		creditLimit = *req.CreditLimit
	}
	if creditDays > 0 || !creditLimit.IsZero() {
		if err := supplier.SetPaymentTerms(creditDays, creditLimit); err != nil {
			return nil, err
		}
	}

	// Set rating
	if req.Rating != nil {
		if err := supplier.SetRating(*req.Rating); err != nil {
			return nil, err
		}
	}

	// Set notes
	if req.Notes != "" {
		supplier.SetNotes(req.Notes)
	}

	// Set sort order
	if req.SortOrder != nil {
		supplier.SetSortOrder(*req.SortOrder)
	}

	// Set attributes
	if req.Attributes != "" {
		if err := supplier.SetAttributes(req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the supplier
	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// GetByID retrieves a supplier by ID
func (s *SupplierService) GetByID(ctx context.Context, tenantID, supplierID uuid.UUID) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// GetByCode retrieves a supplier by code
func (s *SupplierService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// List retrieves a list of suppliers with filtering and pagination
func (s *SupplierService) List(ctx context.Context, tenantID uuid.UUID, filter SupplierListFilter) ([]SupplierListResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "sort_order"
	}
	if filter.OrderDir == "" {
		filter.OrderDir = "asc"
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
		domainFilter.Filters["status"] = filter.Status
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
	if filter.MinRating != nil {
		domainFilter.Filters["min_rating"] = *filter.MinRating
	}
	if filter.MaxRating != nil {
		domainFilter.Filters["max_rating"] = *filter.MaxRating
	}

	// Get suppliers
	suppliers, err := s.supplierRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.supplierRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToSupplierListResponses(suppliers), total, nil
}

// Update updates a supplier
func (s *SupplierService) Update(ctx context.Context, tenantID, supplierID uuid.UUID, req UpdateSupplierRequest) (*SupplierResponse, error) {
	// Get existing supplier
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	// Update name and short name
	if req.Name != nil {
		shortName := supplier.ShortName
		if req.ShortName != nil {
			shortName = *req.ShortName
		}
		if err := supplier.Update(*req.Name, shortName); err != nil {
			return nil, err
		}
	} else if req.ShortName != nil {
		if err := supplier.Update(supplier.Name, *req.ShortName); err != nil {
			return nil, err
		}
	}

	// Update contact
	if req.ContactName != nil || req.Phone != nil || req.Email != nil {
		contactName := supplier.ContactName
		phone := supplier.Phone
		email := supplier.Email

		if req.ContactName != nil {
			contactName = *req.ContactName
		}
		if req.Phone != nil {
			// Check for duplicate phone
			if *req.Phone != "" && *req.Phone != supplier.Phone {
				exists, err := s.supplierRepo.ExistsByPhone(ctx, tenantID, *req.Phone)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this phone already exists")
				}
			}
			phone = *req.Phone
		}
		if req.Email != nil {
			// Check for duplicate email
			if *req.Email != "" && *req.Email != supplier.Email {
				exists, err := s.supplierRepo.ExistsByEmail(ctx, tenantID, *req.Email)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this email already exists")
				}
			}
			email = *req.Email
		}

		if err := supplier.SetContact(contactName, phone, email); err != nil {
			return nil, err
		}
	}

	// Update address
	if req.Address != nil || req.City != nil || req.Province != nil || req.PostalCode != nil || req.Country != nil {
		address := supplier.Address
		city := supplier.City
		province := supplier.Province
		postalCode := supplier.PostalCode
		country := supplier.Country

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

		if err := supplier.SetAddress(address, city, province, postalCode, country); err != nil {
			return nil, err
		}
	}

	// Update tax ID
	if req.TaxID != nil {
		if err := supplier.SetTaxID(*req.TaxID); err != nil {
			return nil, err
		}
	}

	// Update bank info
	if req.BankName != nil || req.BankAccount != nil {
		bankName := supplier.BankName
		bankAccount := supplier.BankAccount
		if req.BankName != nil {
			bankName = *req.BankName
		}
		if req.BankAccount != nil {
			bankAccount = *req.BankAccount
		}
		if err := supplier.SetBankInfo(bankName, bankAccount); err != nil {
			return nil, err
		}
	}

	// Update payment terms
	if req.CreditDays != nil || req.CreditLimit != nil {
		creditDays := supplier.CreditDays
		creditLimit := supplier.CreditLimit
		if req.CreditDays != nil {
			creditDays = *req.CreditDays
		}
		if req.CreditLimit != nil {
			creditLimit = *req.CreditLimit
		}
		if err := supplier.SetPaymentTerms(creditDays, creditLimit); err != nil {
			return nil, err
		}
	}

	// Update rating
	if req.Rating != nil {
		if err := supplier.SetRating(*req.Rating); err != nil {
			return nil, err
		}
	}

	// Update notes
	if req.Notes != nil {
		supplier.SetNotes(*req.Notes)
	}

	// Update sort order
	if req.SortOrder != nil {
		supplier.SetSortOrder(*req.SortOrder)
	}

	// Update attributes
	if req.Attributes != nil {
		if err := supplier.SetAttributes(*req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the supplier
	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// UpdateCode updates a supplier's code
func (s *SupplierService) UpdateCode(ctx context.Context, tenantID, supplierID uuid.UUID, newCode string) (*SupplierResponse, error) {
	// Get existing supplier
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	// Check if new code already exists (if different from current)
	if newCode != supplier.Code {
		exists, err := s.supplierRepo.ExistsByCode(ctx, tenantID, newCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Supplier with this code already exists")
		}
	}

	// Update the code
	if err := supplier.UpdateCode(newCode); err != nil {
		return nil, err
	}

	// Save the supplier
	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// Delete deletes a supplier
func (s *SupplierService) Delete(ctx context.Context, tenantID, supplierID uuid.UUID) error {
	// Verify supplier exists
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return err
	}

	// Check if supplier has outstanding balance
	if supplier.HasBalance() {
		return shared.NewDomainError("CANNOT_DELETE", "Cannot delete supplier with outstanding balance")
	}

	// TODO: Check if supplier has outstanding payables or orders
	// This should be implemented when those modules are available

	return s.supplierRepo.DeleteForTenant(ctx, tenantID, supplierID)
}

// Activate activates a supplier
func (s *SupplierService) Activate(ctx context.Context, tenantID, supplierID uuid.UUID) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	if err := supplier.Activate(); err != nil {
		return nil, err
	}

	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// Deactivate deactivates a supplier
func (s *SupplierService) Deactivate(ctx context.Context, tenantID, supplierID uuid.UUID) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	if err := supplier.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// Block blocks a supplier
func (s *SupplierService) Block(ctx context.Context, tenantID, supplierID uuid.UUID) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	if err := supplier.Block(); err != nil {
		return nil, err
	}

	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// SetRating sets a supplier's rating
func (s *SupplierService) SetRating(ctx context.Context, tenantID, supplierID uuid.UUID, rating int) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	if err := supplier.SetRating(rating); err != nil {
		return nil, err
	}

	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// SetPaymentTerms sets a supplier's payment terms
func (s *SupplierService) SetPaymentTerms(ctx context.Context, tenantID, supplierID uuid.UUID, creditDays int, creditLimit decimal.Decimal) (*SupplierResponse, error) {
	supplier, err := s.supplierRepo.FindByIDForTenant(ctx, tenantID, supplierID)
	if err != nil {
		return nil, err
	}

	if err := supplier.SetPaymentTerms(creditDays, creditLimit); err != nil {
		return nil, err
	}

	if err := s.supplierRepo.Save(ctx, supplier); err != nil {
		return nil, err
	}

	response := ToSupplierResponse(supplier)
	return &response, nil
}

// CountByStatus returns supplier counts by status for a tenant
func (s *SupplierService) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	activeCount, err := s.supplierRepo.CountByStatus(ctx, tenantID, partner.SupplierStatusActive)
	if err != nil {
		return nil, err
	}
	counts["active"] = activeCount

	inactiveCount, err := s.supplierRepo.CountByStatus(ctx, tenantID, partner.SupplierStatusInactive)
	if err != nil {
		return nil, err
	}
	counts["inactive"] = inactiveCount

	blockedCount, err := s.supplierRepo.CountByStatus(ctx, tenantID, partner.SupplierStatusBlocked)
	if err != nil {
		return nil, err
	}
	counts["blocked"] = blockedCount

	counts["total"] = activeCount + inactiveCount + blockedCount

	return counts, nil
}

// CountByType returns supplier counts by type for a tenant
func (s *SupplierService) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	types := []partner.SupplierType{
		partner.SupplierTypeManufacturer,
		partner.SupplierTypeDistributor,
		partner.SupplierTypeRetailer,
		partner.SupplierTypeService,
	}

	var total int64
	for _, t := range types {
		count, err := s.supplierRepo.CountByType(ctx, tenantID, t)
		if err != nil {
			return nil, err
		}
		counts[string(t)] = count
		total += count
	}
	counts["total"] = total

	return counts, nil
}
