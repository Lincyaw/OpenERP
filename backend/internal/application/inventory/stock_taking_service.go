package inventory

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// StockTakingService provides application services for stock taking operations
type StockTakingService struct {
	stockTakingRepo inventory.StockTakingRepository
	eventBus        shared.EventBus
}

// NewStockTakingService creates a new StockTakingService
func NewStockTakingService(
	stockTakingRepo inventory.StockTakingRepository,
	eventBus shared.EventBus,
) *StockTakingService {
	return &StockTakingService{
		stockTakingRepo: stockTakingRepo,
		eventBus:        eventBus,
	}
}

// ===================== Query Methods =====================

// GetByID retrieves a stock taking by ID
func (s *StockTakingService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// GetByTakingNumber retrieves a stock taking by its number
func (s *StockTakingService) GetByTakingNumber(ctx context.Context, tenantID uuid.UUID, takingNumber string) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByTakingNumber(ctx, tenantID, takingNumber)
	if err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// List retrieves a paginated list of stock takings
func (s *StockTakingService) List(ctx context.Context, tenantID uuid.UUID, filter StockTakingListFilter) ([]StockTakingListResponse, int64, error) {
	// Build domain filter
	domainFilter := inventory.StockTakingFilter{
		Filter: shared.Filter{
			Page:     filter.Page,
			PageSize: filter.PageSize,
			OrderBy:  filter.OrderBy,
			OrderDir: filter.OrderDir,
			Search:   filter.Search,
		},
		WarehouseID: filter.WarehouseID,
		Status:      filter.Status,
		StartDate:   filter.StartDate,
		EndDate:     filter.EndDate,
		CreatedByID: filter.CreatedByID,
	}

	// Get total count
	total, err := s.stockTakingRepo.CountForTenant(ctx, tenantID, domainFilter.Filter)
	if err != nil {
		return nil, 0, err
	}

	// Get stock takings
	sts, err := s.stockTakingRepo.FindAllForTenant(ctx, tenantID, domainFilter.Filter)
	if err != nil {
		return nil, 0, err
	}

	return ToStockTakingListResponses(sts), total, nil
}

// ListByWarehouse retrieves stock takings for a specific warehouse
func (s *StockTakingService) ListByWarehouse(ctx context.Context, tenantID, warehouseID uuid.UUID, filter StockTakingListFilter) ([]StockTakingListResponse, int64, error) {
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
	}

	sts, err := s.stockTakingRepo.FindByWarehouse(ctx, tenantID, warehouseID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.stockTakingRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToStockTakingListResponses(sts), total, nil
}

// ListByStatus retrieves stock takings with a specific status
func (s *StockTakingService) ListByStatus(ctx context.Context, tenantID uuid.UUID, status inventory.StockTakingStatus, filter StockTakingListFilter) ([]StockTakingListResponse, int64, error) {
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
	}

	sts, err := s.stockTakingRepo.FindByStatus(ctx, tenantID, status, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.stockTakingRepo.CountByStatus(ctx, tenantID, status)
	if err != nil {
		return nil, 0, err
	}

	return ToStockTakingListResponses(sts), total, nil
}

// ListPendingApproval retrieves stock takings pending approval
func (s *StockTakingService) ListPendingApproval(ctx context.Context, tenantID uuid.UUID, filter StockTakingListFilter) ([]StockTakingListResponse, int64, error) {
	domainFilter := shared.Filter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		OrderBy:  filter.OrderBy,
		OrderDir: filter.OrderDir,
		Search:   filter.Search,
	}

	sts, err := s.stockTakingRepo.FindPendingApproval(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.stockTakingRepo.CountByStatus(ctx, tenantID, inventory.StockTakingStatusPendingApproval)
	if err != nil {
		return nil, 0, err
	}

	return ToStockTakingListResponses(sts), total, nil
}

// GetProgress retrieves the progress of a stock taking
func (s *StockTakingService) GetProgress(ctx context.Context, tenantID, id uuid.UUID) (*StockTakingProgressResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	response := ToStockTakingProgressResponse(st)
	return &response, nil
}

// ===================== Command Methods =====================

// Create creates a new stock taking
func (s *StockTakingService) Create(ctx context.Context, tenantID uuid.UUID, req CreateStockTakingRequest) (*StockTakingResponse, error) {
	// Generate taking number
	takingNumber, err := s.stockTakingRepo.GenerateTakingNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Determine taking date
	takingDate := time.Now()
	if req.TakingDate != nil {
		takingDate = *req.TakingDate
	}

	// Create stock taking aggregate
	st, err := inventory.NewStockTaking(
		tenantID,
		req.WarehouseID,
		req.WarehouseName,
		takingNumber,
		takingDate,
		req.CreatedByID,
		req.CreatedByName,
	)
	if err != nil {
		return nil, err
	}

	if req.Remark != "" {
		st.SetRemark(req.Remark)
	}

	// Save to repository
	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// Update updates a stock taking (only in DRAFT status)
func (s *StockTakingService) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateStockTakingRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if st.Status != inventory.StockTakingStatusDraft {
		return nil, shared.NewDomainError("INVALID_STATUS", "Can only update stock taking in DRAFT status")
	}

	st.SetRemark(req.Remark)

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// Delete deletes a stock taking (only in DRAFT status)
func (s *StockTakingService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if st.Status != inventory.StockTakingStatusDraft {
		return shared.NewDomainError("INVALID_STATUS", "Can only delete stock taking in DRAFT status")
	}

	return s.stockTakingRepo.DeleteForTenant(ctx, tenantID, id)
}

// AddItem adds an item to a stock taking
func (s *StockTakingService) AddItem(ctx context.Context, tenantID, stockTakingID uuid.UUID, req AddStockTakingItemRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, stockTakingID)
	if err != nil {
		return nil, err
	}

	if err := st.AddItem(req.ProductID, req.ProductName, req.ProductCode, req.Unit, req.SystemQuantity, req.UnitCost); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// AddItems adds multiple items to a stock taking
func (s *StockTakingService) AddItems(ctx context.Context, tenantID, stockTakingID uuid.UUID, req AddStockTakingItemsRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, stockTakingID)
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		if err := st.AddItem(item.ProductID, item.ProductName, item.ProductCode, item.Unit, item.SystemQuantity, item.UnitCost); err != nil {
			return nil, err
		}
	}

	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// RemoveItem removes an item from a stock taking
func (s *StockTakingService) RemoveItem(ctx context.Context, tenantID, stockTakingID, productID uuid.UUID) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, stockTakingID)
	if err != nil {
		return nil, err
	}

	if err := st.RemoveItem(productID); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// StartCounting starts the counting process
func (s *StockTakingService) StartCounting(ctx context.Context, tenantID, id uuid.UUID) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := st.StartCounting(); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// RecordCount records the actual count for an item
func (s *StockTakingService) RecordCount(ctx context.Context, tenantID, stockTakingID uuid.UUID, req RecordCountRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, stockTakingID)
	if err != nil {
		return nil, err
	}

	if err := st.RecordItemCount(req.ProductID, req.ActualQuantity, req.Remark); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// RecordCounts records multiple counts at once
func (s *StockTakingService) RecordCounts(ctx context.Context, tenantID, stockTakingID uuid.UUID, req RecordCountsRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, stockTakingID)
	if err != nil {
		return nil, err
	}

	for _, count := range req.Counts {
		if err := st.RecordItemCount(count.ProductID, count.ActualQuantity, count.Remark); err != nil {
			return nil, err
		}
	}

	if err := s.stockTakingRepo.SaveWithItems(ctx, st); err != nil {
		return nil, err
	}

	response := ToStockTakingResponse(st)
	return &response, nil
}

// SubmitForApproval submits the stock taking for approval
func (s *StockTakingService) SubmitForApproval(ctx context.Context, tenantID, id uuid.UUID) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := st.SubmitForApproval(); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// Approve approves the stock taking
func (s *StockTakingService) Approve(ctx context.Context, tenantID, id uuid.UUID, req ApproveStockTakingRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := st.Approve(req.ApproverID, req.ApproverName, req.Note); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events (StockTakingApprovedEvent triggers inventory adjustments)
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// Reject rejects the stock taking
func (s *StockTakingService) Reject(ctx context.Context, tenantID, id uuid.UUID, req RejectStockTakingRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := st.Reject(req.ApproverID, req.ApproverName, req.Reason); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// Cancel cancels the stock taking
func (s *StockTakingService) Cancel(ctx context.Context, tenantID, id uuid.UUID, req CancelStockTakingRequest) (*StockTakingResponse, error) {
	st, err := s.stockTakingRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := st.Cancel(req.Reason); err != nil {
		return nil, err
	}

	if err := s.stockTakingRepo.Save(ctx, st); err != nil {
		return nil, err
	}

	// Publish domain events
	s.publishEvents(ctx, st)

	response := ToStockTakingResponse(st)
	return &response, nil
}

// publishEvents publishes domain events from the aggregate
func (s *StockTakingService) publishEvents(ctx context.Context, st *inventory.StockTaking) {
	if s.eventBus == nil {
		return
	}

	for _, event := range st.GetDomainEvents() {
		_ = s.eventBus.Publish(ctx, event)
	}
	st.ClearDomainEvents()
}
