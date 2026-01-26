package partner

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =============================================================================
// Customer Level DTOs
// =============================================================================

// CreateCustomerLevelRequest represents a request to create a new customer level
type CreateCustomerLevelRequest struct {
	Code         string  `json:"code" binding:"required,min=1,max=50"`
	Name         string  `json:"name" binding:"required,min=1,max=100"`
	DiscountRate float64 `json:"discount_rate" binding:"gte=0,lte=1"`
	SortOrder    *int    `json:"sort_order"`
	IsDefault    bool    `json:"is_default"`
	IsActive     bool    `json:"is_active"`
	Description  string  `json:"description" binding:"max=500"`
}

// UpdateCustomerLevelRequest represents a request to update a customer level
type UpdateCustomerLevelRequest struct {
	Name         *string  `json:"name" binding:"omitempty,min=1,max=100"`
	DiscountRate *float64 `json:"discount_rate" binding:"omitempty,gte=0,lte=1"`
	SortOrder    *int     `json:"sort_order"`
	IsDefault    *bool    `json:"is_default"`
	IsActive     *bool    `json:"is_active"`
	Description  *string  `json:"description" binding:"omitempty,max=500"`
}

// CustomerLevelResponse represents a customer level in API responses
type CustomerLevelResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	DiscountRate    decimal.Decimal `json:"discount_rate"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`
	SortOrder       int             `json:"sort_order"`
	IsDefault       bool            `json:"is_default"`
	IsActive        bool            `json:"is_active"`
	Description     string          `json:"description"`
	CustomerCount   int64           `json:"customer_count,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CustomerLevelListResponse represents a list item for customer levels
type CustomerLevelListResponse struct {
	ID              uuid.UUID       `json:"id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	DiscountRate    decimal.Decimal `json:"discount_rate"`
	DiscountPercent decimal.Decimal `json:"discount_percent"`
	SortOrder       int             `json:"sort_order"`
	IsDefault       bool            `json:"is_default"`
	IsActive        bool            `json:"is_active"`
	CustomerCount   int64           `json:"customer_count"`
}

// ToCustomerLevelResponse converts a domain CustomerLevelRecord to CustomerLevelResponse
func ToCustomerLevelResponse(r *partner.CustomerLevelRecord) CustomerLevelResponse {
	return CustomerLevelResponse{
		ID:              r.ID,
		TenantID:        r.TenantID,
		Code:            r.Code,
		Name:            r.Name,
		DiscountRate:    r.DiscountRate,
		DiscountPercent: r.DiscountRate.Mul(decimal.NewFromInt(100)),
		SortOrder:       r.SortOrder,
		IsDefault:       r.IsDefault,
		IsActive:        r.IsActive,
		Description:     r.Description,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// ToCustomerLevelListResponse converts a domain CustomerLevelRecord to CustomerLevelListResponse
func ToCustomerLevelListResponse(r *partner.CustomerLevelRecord, customerCount int64) CustomerLevelListResponse {
	return CustomerLevelListResponse{
		ID:              r.ID,
		Code:            r.Code,
		Name:            r.Name,
		DiscountRate:    r.DiscountRate,
		DiscountPercent: r.DiscountRate.Mul(decimal.NewFromInt(100)),
		SortOrder:       r.SortOrder,
		IsDefault:       r.IsDefault,
		IsActive:        r.IsActive,
		CustomerCount:   customerCount,
	}
}

// =============================================================================
// Customer Level Service
// =============================================================================

// CustomerLevelService handles customer level-related business operations
type CustomerLevelService struct {
	levelRepo partner.CustomerLevelRepository
}

// NewCustomerLevelService creates a new CustomerLevelService
func NewCustomerLevelService(levelRepo partner.CustomerLevelRepository) *CustomerLevelService {
	return &CustomerLevelService{
		levelRepo: levelRepo,
	}
}

// Create creates a new customer level
func (s *CustomerLevelService) Create(ctx context.Context, tenantID uuid.UUID, req CreateCustomerLevelRequest) (*CustomerLevelResponse, error) {
	// Check if code already exists
	exists, err := s.levelRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Customer level with this code already exists")
	}

	// Create the CustomerLevel value object for validation
	discountRate := decimal.NewFromFloat(req.DiscountRate)
	level, err := partner.NewCustomerLevel(req.Code, req.Name, discountRate)
	if err != nil {
		return nil, err
	}

	// Set default sort order if not provided
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	// Create the record
	record := partner.NewCustomerLevelRecord(tenantID, level, sortOrder, req.IsDefault)
	record.IsActive = req.IsActive
	record.Description = req.Description

	// If this is set as default, we need to handle it (trigger in DB handles this)
	if err := s.levelRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// GetByID retrieves a customer level by ID
func (s *CustomerLevelService) GetByID(ctx context.Context, tenantID, levelID uuid.UUID) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return nil, err
	}

	// Get customer count for this level
	customerCount, err := s.levelRepo.CountCustomersWithLevel(ctx, tenantID, record.Code)
	if err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	response.CustomerCount = customerCount
	return &response, nil
}

// GetByCode retrieves a customer level by code
func (s *CustomerLevelService) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindByCode(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}

	// Get customer count for this level
	customerCount, err := s.levelRepo.CountCustomersWithLevel(ctx, tenantID, record.Code)
	if err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	response.CustomerCount = customerCount
	return &response, nil
}

// GetDefault retrieves the default customer level for a tenant
func (s *CustomerLevelService) GetDefault(ctx context.Context, tenantID uuid.UUID) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindDefaultForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// List retrieves all customer levels for a tenant
func (s *CustomerLevelService) List(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]CustomerLevelListResponse, error) {
	var records []*partner.CustomerLevelRecord
	var err error

	if activeOnly {
		records, err = s.levelRepo.FindActiveForTenant(ctx, tenantID)
	} else {
		records, err = s.levelRepo.FindAllForTenant(ctx, tenantID)
	}
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return []CustomerLevelListResponse{}, nil
	}

	// Collect all level codes for batch query
	codes := make([]string, len(records))
	for i, record := range records {
		codes[i] = record.Code
	}

	// Get customer counts for all levels in a single query (prevents N+1)
	customerCounts, err := s.levelRepo.CountCustomersByLevelCodes(ctx, tenantID, codes)
	if err != nil {
		return nil, err
	}

	responses := make([]CustomerLevelListResponse, len(records))
	for i, record := range records {
		customerCount := customerCounts[record.Code] // Will be 0 if not in map
		responses[i] = ToCustomerLevelListResponse(record, customerCount)
	}

	return responses, nil
}

// Update updates a customer level
func (s *CustomerLevelService) Update(ctx context.Context, tenantID, levelID uuid.UUID, req UpdateCustomerLevelRequest) (*CustomerLevelResponse, error) {
	// Get existing record
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		record.Name = *req.Name
	}
	if req.DiscountRate != nil {
		record.DiscountRate = decimal.NewFromFloat(*req.DiscountRate)
	}
	if req.SortOrder != nil {
		record.SortOrder = *req.SortOrder
	}
	if req.IsDefault != nil {
		record.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		record.IsActive = *req.IsActive
	}
	if req.Description != nil {
		record.Description = *req.Description
	}

	// Validate the updated values
	_, err = partner.NewCustomerLevel(record.Code, record.Name, record.DiscountRate)
	if err != nil {
		return nil, err
	}

	// Save the updated record
	if err := s.levelRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// Delete deletes a customer level
func (s *CustomerLevelService) Delete(ctx context.Context, tenantID, levelID uuid.UUID) error {
	// Get the record to check constraints
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return err
	}

	// Check if this level is being used by any customers
	customerCount, err := s.levelRepo.CountCustomersWithLevel(ctx, tenantID, record.Code)
	if err != nil {
		return err
	}
	if customerCount > 0 {
		return shared.NewDomainError("CANNOT_DELETE", "Cannot delete customer level that is in use by customers")
	}

	// Check if this is the default level
	if record.IsDefault {
		return shared.NewDomainError("CANNOT_DELETE", "Cannot delete the default customer level")
	}

	return s.levelRepo.DeleteForTenant(ctx, tenantID, levelID)
}

// SetDefault sets a customer level as the default for a tenant
func (s *CustomerLevelService) SetDefault(ctx context.Context, tenantID, levelID uuid.UUID) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return nil, err
	}

	// Check if level is active
	if !record.IsActive {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot set inactive level as default")
	}

	record.IsDefault = true

	// The database trigger will handle unsetting other defaults
	if err := s.levelRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// Activate activates a customer level
func (s *CustomerLevelService) Activate(ctx context.Context, tenantID, levelID uuid.UUID) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return nil, err
	}

	record.IsActive = true

	if err := s.levelRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// Deactivate deactivates a customer level
func (s *CustomerLevelService) Deactivate(ctx context.Context, tenantID, levelID uuid.UUID) (*CustomerLevelResponse, error) {
	record, err := s.levelRepo.FindByIDForTenant(ctx, tenantID, levelID)
	if err != nil {
		return nil, err
	}

	// Check if this is the default level
	if record.IsDefault {
		return nil, shared.NewDomainError("INVALID_STATE", "Cannot deactivate the default customer level")
	}

	// Check if this level is being used by any customers
	customerCount, err := s.levelRepo.CountCustomersWithLevel(ctx, tenantID, record.Code)
	if err != nil {
		return nil, err
	}
	if customerCount > 0 {
		return nil, shared.NewDomainError("CANNOT_DEACTIVATE", "Cannot deactivate customer level that is in use by customers")
	}

	record.IsActive = false

	if err := s.levelRepo.Save(ctx, record); err != nil {
		return nil, err
	}

	response := ToCustomerLevelResponse(record)
	return &response, nil
}

// InitializeDefaultLevels creates the default customer levels for a new tenant
func (s *CustomerLevelService) InitializeDefaultLevels(ctx context.Context, tenantID uuid.UUID) error {
	return s.levelRepo.InitializeDefaultLevels(ctx, tenantID)
}

// EnrichCustomerLevel enriches a partial CustomerLevel with full details from the database
func (s *CustomerLevelService) EnrichCustomerLevel(ctx context.Context, tenantID uuid.UUID, level partner.CustomerLevel) (partner.CustomerLevel, error) {
	if level.IsEmpty() {
		return level, nil
	}

	record, err := s.levelRepo.FindByCode(ctx, tenantID, level.Code())
	if err != nil {
		// If not found, return the original level without enrichment
		if err == shared.ErrNotFound {
			return level, nil
		}
		return level, err
	}

	return level.WithDetails(record.Name, record.DiscountRate), nil
}
