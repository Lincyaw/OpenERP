package partner

import (
	"context"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerService handles customer-related business operations
type CustomerService struct {
	customerRepo partner.CustomerRepository
}

// NewCustomerService creates a new CustomerService
func NewCustomerService(customerRepo partner.CustomerRepository) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
	}
}

// Create creates a new customer
func (s *CustomerService) Create(ctx context.Context, tenantID uuid.UUID, req CreateCustomerRequest) (*CustomerResponse, error) {
	// Check if code already exists
	exists, err := s.customerRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this code already exists")
	}

	// Check if phone already exists (if provided)
	if req.Phone != "" {
		exists, err = s.customerRepo.ExistsByPhone(ctx, tenantID, req.Phone)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this phone already exists")
		}
	}

	// Check if email already exists (if provided)
	if req.Email != "" {
		exists, err = s.customerRepo.ExistsByEmail(ctx, tenantID, req.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this email already exists")
		}
	}

	// Create the customer
	customerType := partner.CustomerType(req.Type)
	customer, err := partner.NewCustomer(tenantID, req.Code, req.Name, customerType)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.ShortName != "" {
		if err := customer.Update(req.Name, req.ShortName); err != nil {
			return nil, err
		}
	}

	// Set contact
	if req.ContactName != "" || req.Phone != "" || req.Email != "" {
		if err := customer.SetContact(req.ContactName, req.Phone, req.Email); err != nil {
			return nil, err
		}
	}

	// Set address
	if req.Address != "" || req.City != "" || req.Province != "" || req.PostalCode != "" || req.Country != "" {
		if err := customer.SetAddress(req.Address, req.City, req.Province, req.PostalCode, req.Country); err != nil {
			return nil, err
		}
	}

	// Set tax ID
	if req.TaxID != "" {
		if err := customer.SetTaxID(req.TaxID); err != nil {
			return nil, err
		}
	}

	// Set credit limit
	if req.CreditLimit != nil && !req.CreditLimit.IsZero() {
		if err := customer.SetCreditLimit(*req.CreditLimit); err != nil {
			return nil, err
		}
	}

	// Set notes
	if req.Notes != "" {
		customer.SetNotes(req.Notes)
	}

	// Set sort order
	if req.SortOrder != nil {
		customer.SetSortOrder(*req.SortOrder)
	}

	// Set attributes
	if req.Attributes != "" {
		if err := customer.SetAttributes(req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// GetByID retrieves a customer by ID
func (s *CustomerService) GetByID(ctx context.Context, tenantID, customerID uuid.UUID) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// GetByCode retrieves a customer by code
func (s *CustomerService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// List retrieves a list of customers with filtering and pagination
func (s *CustomerService) List(ctx context.Context, tenantID uuid.UUID, filter CustomerListFilter) ([]CustomerListResponse, int64, error) {
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
	if filter.Level != "" {
		domainFilter.Filters["level"] = filter.Level
	}
	if filter.City != "" {
		domainFilter.Filters["city"] = filter.City
	}
	if filter.Province != "" {
		domainFilter.Filters["province"] = filter.Province
	}

	// Get customers
	customers, err := s.customerRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.customerRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToCustomerListResponses(customers), total, nil
}

// Update updates a customer
func (s *CustomerService) Update(ctx context.Context, tenantID, customerID uuid.UUID, req UpdateCustomerRequest) (*CustomerResponse, error) {
	// Get existing customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	// Update name and short name
	if req.Name != nil {
		shortName := customer.ShortName
		if req.ShortName != nil {
			shortName = *req.ShortName
		}
		if err := customer.Update(*req.Name, shortName); err != nil {
			return nil, err
		}
	} else if req.ShortName != nil {
		if err := customer.Update(customer.Name, *req.ShortName); err != nil {
			return nil, err
		}
	}

	// Update contact
	if req.ContactName != nil || req.Phone != nil || req.Email != nil {
		contactName := customer.ContactName
		phone := customer.Phone
		email := customer.Email

		if req.ContactName != nil {
			contactName = *req.ContactName
		}
		if req.Phone != nil {
			// Check for duplicate phone
			if *req.Phone != "" && *req.Phone != customer.Phone {
				exists, err := s.customerRepo.ExistsByPhone(ctx, tenantID, *req.Phone)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this phone already exists")
				}
			}
			phone = *req.Phone
		}
		if req.Email != nil {
			// Check for duplicate email
			if *req.Email != "" && *req.Email != customer.Email {
				exists, err := s.customerRepo.ExistsByEmail(ctx, tenantID, *req.Email)
				if err != nil {
					return nil, err
				}
				if exists {
					return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this email already exists")
				}
			}
			email = *req.Email
		}

		if err := customer.SetContact(contactName, phone, email); err != nil {
			return nil, err
		}
	}

	// Update address
	if req.Address != nil || req.City != nil || req.Province != nil || req.PostalCode != nil || req.Country != nil {
		address := customer.Address
		city := customer.City
		province := customer.Province
		postalCode := customer.PostalCode
		country := customer.Country

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

		if err := customer.SetAddress(address, city, province, postalCode, country); err != nil {
			return nil, err
		}
	}

	// Update tax ID
	if req.TaxID != nil {
		if err := customer.SetTaxID(*req.TaxID); err != nil {
			return nil, err
		}
	}

	// Update credit limit
	if req.CreditLimit != nil {
		if err := customer.SetCreditLimit(*req.CreditLimit); err != nil {
			return nil, err
		}
	}

	// Update level
	if req.Level != nil {
		level := partner.CustomerLevel(*req.Level)
		if err := customer.SetLevel(level); err != nil {
			return nil, err
		}
	}

	// Update notes
	if req.Notes != nil {
		customer.SetNotes(*req.Notes)
	}

	// Update sort order
	if req.SortOrder != nil {
		customer.SetSortOrder(*req.SortOrder)
	}

	// Update attributes
	if req.Attributes != nil {
		if err := customer.SetAttributes(*req.Attributes); err != nil {
			return nil, err
		}
	}

	// Save the customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// UpdateCode updates a customer's code
func (s *CustomerService) UpdateCode(ctx context.Context, tenantID, customerID uuid.UUID, newCode string) (*CustomerResponse, error) {
	// Get existing customer
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	// Check if new code already exists (if different from current)
	if newCode != customer.Code {
		exists, err := s.customerRepo.ExistsByCode(ctx, tenantID, newCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer with this code already exists")
		}
	}

	// Update the code
	if err := customer.UpdateCode(newCode); err != nil {
		return nil, err
	}

	// Save the customer
	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// Delete deletes a customer
func (s *CustomerService) Delete(ctx context.Context, tenantID, customerID uuid.UUID) error {
	// Verify customer exists
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return err
	}

	// Check if customer has balance
	if customer.HasBalance() {
		return shared.NewDomainError("CANNOT_DELETE", "Cannot delete customer with positive balance")
	}

	// TODO: Check if customer has outstanding receivables or orders
	// This should be implemented when those modules are available

	return s.customerRepo.DeleteForTenant(ctx, tenantID, customerID)
}

// Activate activates a customer
func (s *CustomerService) Activate(ctx context.Context, tenantID, customerID uuid.UUID) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.Activate(); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// Deactivate deactivates a customer
func (s *CustomerService) Deactivate(ctx context.Context, tenantID, customerID uuid.UUID) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// Suspend suspends a customer
func (s *CustomerService) Suspend(ctx context.Context, tenantID, customerID uuid.UUID) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.Suspend(); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// AddBalance adds to a customer's prepaid balance
func (s *CustomerService) AddBalance(ctx context.Context, tenantID, customerID uuid.UUID, amount decimal.Decimal) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.AddBalance(amount); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// DeductBalance deducts from a customer's prepaid balance
func (s *CustomerService) DeductBalance(ctx context.Context, tenantID, customerID uuid.UUID, amount decimal.Decimal) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.DeductBalance(amount); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// RefundBalance refunds to a customer's prepaid balance
func (s *CustomerService) RefundBalance(ctx context.Context, tenantID, customerID uuid.UUID, amount decimal.Decimal) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if err := customer.RefundBalance(amount); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// SetLevel sets a customer's tier level
func (s *CustomerService) SetLevel(ctx context.Context, tenantID, customerID uuid.UUID, level string) (*CustomerResponse, error) {
	customer, err := s.customerRepo.FindByIDForTenant(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	customerLevel := partner.CustomerLevel(level)
	if err := customer.SetLevel(customerLevel); err != nil {
		return nil, err
	}

	if err := s.customerRepo.Save(ctx, customer); err != nil {
		return nil, err
	}

	response := ToCustomerResponse(customer)
	return &response, nil
}

// CountByStatus returns customer counts by status for a tenant
func (s *CustomerService) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	activeCount, err := s.customerRepo.CountByStatus(ctx, tenantID, partner.CustomerStatusActive)
	if err != nil {
		return nil, err
	}
	counts["active"] = activeCount

	inactiveCount, err := s.customerRepo.CountByStatus(ctx, tenantID, partner.CustomerStatusInactive)
	if err != nil {
		return nil, err
	}
	counts["inactive"] = inactiveCount

	suspendedCount, err := s.customerRepo.CountByStatus(ctx, tenantID, partner.CustomerStatusSuspended)
	if err != nil {
		return nil, err
	}
	counts["suspended"] = suspendedCount

	counts["total"] = activeCount + inactiveCount + suspendedCount

	return counts, nil
}

// CountByLevel returns customer counts by level for a tenant
func (s *CustomerService) CountByLevel(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	levels := []partner.CustomerLevel{
		partner.CustomerLevelNormal,
		partner.CustomerLevelSilver,
		partner.CustomerLevelGold,
		partner.CustomerLevelPlatinum,
		partner.CustomerLevelVIP,
	}

	var total int64
	for _, level := range levels {
		count, err := s.customerRepo.CountByLevel(ctx, tenantID, level)
		if err != nil {
			return nil, err
		}
		counts[string(level)] = count
		total += count
	}
	counts["total"] = total

	return counts, nil
}

// CountByType returns customer counts by type for a tenant
func (s *CustomerService) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int64, error) {
	counts := make(map[string]int64)

	individualCount, err := s.customerRepo.CountByType(ctx, tenantID, partner.CustomerTypeIndividual)
	if err != nil {
		return nil, err
	}
	counts["individual"] = individualCount

	organizationCount, err := s.customerRepo.CountByType(ctx, tenantID, partner.CustomerTypeOrganization)
	if err != nil {
		return nil, err
	}
	counts["organization"] = organizationCount

	counts["total"] = individualCount + organizationCount

	return counts, nil
}
