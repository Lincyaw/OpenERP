package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
)

// SalesReturnService handles sales return business operations
type SalesReturnService struct {
	returnRepo     trade.SalesReturnRepository
	orderRepo      trade.SalesOrderRepository
	eventPublisher shared.EventPublisher
}

// NewSalesReturnService creates a new SalesReturnService
func NewSalesReturnService(
	returnRepo trade.SalesReturnRepository,
	orderRepo trade.SalesOrderRepository,
) *SalesReturnService {
	return &SalesReturnService{
		returnRepo: returnRepo,
		orderRepo:  orderRepo,
	}
}

// SetEventPublisher sets the event publisher for cross-context integration
func (s *SalesReturnService) SetEventPublisher(publisher shared.EventPublisher) {
	s.eventPublisher = publisher
}

// Create creates a new sales return from an existing sales order
func (s *SalesReturnService) Create(ctx context.Context, tenantID uuid.UUID, req CreateSalesReturnRequest) (*SalesReturnResponse, error) {
	// Get the sales order
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, req.SalesOrderID)
	if err != nil {
		return nil, err
	}

	// Generate return number
	returnNumber, err := s.returnRepo.GenerateReturnNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create the return
	sr, err := trade.NewSalesReturn(tenantID, returnNumber, order)
	if err != nil {
		return nil, err
	}

	// Add items
	for _, item := range req.Items {
		// Find the order item
		orderItem := order.GetItem(item.SalesOrderItemID)
		if orderItem == nil {
			return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Sales order item not found: "+item.SalesOrderItemID.String())
		}

		returnItem, err := sr.AddItem(orderItem, item.ReturnQuantity)
		if err != nil {
			return nil, err
		}

		// Set optional fields
		if item.Reason != "" {
			returnItem.SetReason(item.Reason)
		}
		if item.ConditionOnReturn != "" {
			returnItem.SetCondition(item.ConditionOnReturn)
		}
	}

	// Set warehouse if provided, otherwise use order's warehouse
	if req.WarehouseID != nil {
		if err := sr.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Set optional fields
	if req.Reason != "" {
		sr.SetReason(req.Reason)
	}
	if req.Remark != "" {
		sr.SetRemark(req.Remark)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		sr.SetCreatedBy(*req.CreatedBy)
	}

	// Save the return
	if err := s.returnRepo.Save(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// GetByID retrieves a sales return by ID
func (s *SalesReturnService) GetByID(ctx context.Context, tenantID, returnID uuid.UUID) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}
	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// GetByReturnNumber retrieves a sales return by return number
func (s *SalesReturnService) GetByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByReturnNumber(ctx, tenantID, returnNumber)
	if err != nil {
		return nil, err
	}
	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// List retrieves a list of sales returns with filtering and pagination
func (s *SalesReturnService) List(ctx context.Context, tenantID uuid.UUID, filter SalesReturnListFilter) ([]SalesReturnListItemResponse, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "created_at"
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
	if filter.CustomerID != nil {
		domainFilter.Filters["customer_id"] = *filter.CustomerID
	}
	if filter.SalesOrderID != nil {
		domainFilter.Filters["sales_order_id"] = *filter.SalesOrderID
	}
	if filter.WarehouseID != nil {
		domainFilter.Filters["warehouse_id"] = *filter.WarehouseID
	}
	if filter.Status != nil {
		domainFilter.Filters["status"] = string(*filter.Status)
	}
	if len(filter.Statuses) > 0 {
		domainFilter.Filters["statuses"] = filter.Statuses
	}
	if filter.StartDate != nil {
		domainFilter.Filters["start_date"] = *filter.StartDate
	}
	if filter.EndDate != nil {
		domainFilter.Filters["end_date"] = *filter.EndDate
	}
	if filter.MinAmount != nil {
		domainFilter.Filters["min_amount"] = *filter.MinAmount
	}
	if filter.MaxAmount != nil {
		domainFilter.Filters["max_amount"] = *filter.MaxAmount
	}

	// Get returns
	returns, err := s.returnRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.returnRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToSalesReturnListItemResponses(returns), total, nil
}

// ListBySalesOrder retrieves sales returns for a specific sales order
func (s *SalesReturnService) ListBySalesOrder(ctx context.Context, tenantID, salesOrderID uuid.UUID) ([]SalesReturnListItemResponse, error) {
	returns, err := s.returnRepo.FindBySalesOrder(ctx, tenantID, salesOrderID)
	if err != nil {
		return nil, err
	}
	return ToSalesReturnListItemResponses(returns), nil
}

// ListPendingApproval retrieves returns pending approval
func (s *SalesReturnService) ListPendingApproval(ctx context.Context, tenantID uuid.UUID, filter SalesReturnListFilter) ([]SalesReturnListItemResponse, int64, error) {
	status := trade.ReturnStatusPending
	filter.Status = &status
	return s.List(ctx, tenantID, filter)
}

// Update updates a sales return (only allowed in DRAFT status)
func (s *SalesReturnService) Update(ctx context.Context, tenantID, returnID uuid.UUID, req UpdateSalesReturnRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	if !sr.CanModify() {
		return nil, shared.NewDomainError("INVALID_STATE", "Return can only be modified in draft status")
	}

	// Update warehouse if provided
	if req.WarehouseID != nil {
		if *req.WarehouseID == uuid.Nil {
			sr.WarehouseID = nil
		} else {
			if err := sr.SetWarehouse(*req.WarehouseID); err != nil {
				return nil, err
			}
		}
	}

	// Update reason if provided
	if req.Reason != nil {
		sr.SetReason(*req.Reason)
	}

	// Update remark if provided
	if req.Remark != nil {
		sr.SetRemark(*req.Remark)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Delete deletes a sales return (only allowed in DRAFT status)
func (s *SalesReturnService) Delete(ctx context.Context, tenantID, returnID uuid.UUID) error {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return err
	}

	if !sr.IsDraft() {
		return shared.NewDomainError("INVALID_STATE", "Only draft returns can be deleted")
	}

	return s.returnRepo.DeleteForTenant(ctx, tenantID, returnID)
}

// AddItem adds an item to a return (only allowed in DRAFT status)
func (s *SalesReturnService) AddItem(ctx context.Context, tenantID, returnID uuid.UUID, req AddReturnItemRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Get the sales order
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, sr.SalesOrderID)
	if err != nil {
		return nil, err
	}

	// Find the order item
	orderItem := order.GetItem(req.SalesOrderItemID)
	if orderItem == nil {
		return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Sales order item not found")
	}

	// Add the item
	returnItem, err := sr.AddItem(orderItem, req.ReturnQuantity)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if req.Reason != "" {
		returnItem.SetReason(req.Reason)
	}
	if req.ConditionOnReturn != "" {
		returnItem.SetCondition(req.ConditionOnReturn)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// UpdateItem updates an item in a return (only allowed in DRAFT status)
func (s *SalesReturnService) UpdateItem(ctx context.Context, tenantID, returnID, itemID uuid.UUID, req UpdateReturnItemRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Find the item
	item := sr.GetItem(itemID)
	if item == nil {
		return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
	}

	// Update quantity if provided
	if req.ReturnQuantity != nil {
		if err := sr.UpdateItemQuantity(itemID, *req.ReturnQuantity); err != nil {
			return nil, err
		}
		// Re-fetch the item after update
		item = sr.GetItem(itemID)
	}

	// Update reason if provided
	if req.Reason != nil {
		item.SetReason(*req.Reason)
	}

	// Update condition if provided
	if req.ConditionOnReturn != nil {
		item.SetCondition(*req.ConditionOnReturn)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// RemoveItem removes an item from a return (only allowed in DRAFT status)
func (s *SalesReturnService) RemoveItem(ctx context.Context, tenantID, returnID, itemID uuid.UUID) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Remove the item
	if err := sr.RemoveItem(itemID); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Submit submits a return for approval
func (s *SalesReturnService) Submit(ctx context.Context, tenantID, returnID uuid.UUID) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Submit the return
	if err := sr.Submit(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Approve approves a return
func (s *SalesReturnService) Approve(ctx context.Context, tenantID, returnID, approverID uuid.UUID, req ApproveReturnRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Approve the return
	if err := sr.Approve(approverID, req.Note); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Reject rejects a return
func (s *SalesReturnService) Reject(ctx context.Context, tenantID, returnID, rejecterID uuid.UUID, req RejectReturnRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Reject the return
	if err := sr.Reject(rejecterID, req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Complete marks a return as completed (after stock restoration)
func (s *SalesReturnService) Complete(ctx context.Context, tenantID, returnID uuid.UUID, req CompleteReturnRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Set warehouse if provided
	if req.WarehouseID != nil {
		if err := sr.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Complete the return
	if err := sr.Complete(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// Cancel cancels a return
func (s *SalesReturnService) Cancel(ctx context.Context, tenantID, returnID uuid.UUID, req CancelReturnRequest) (*SalesReturnResponse, error) {
	sr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Cancel the return
	if err := sr.Cancel(req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, sr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range sr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToSalesReturnResponse(sr)
	return &response, nil
}

// GetStatusSummary returns a summary of returns by status for dashboard
func (s *SalesReturnService) GetStatusSummary(ctx context.Context, tenantID uuid.UUID) (*ReturnStatusSummary, error) {
	summary := &ReturnStatusSummary{}

	// Count by each status
	var err error
	summary.Draft, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusDraft)
	if err != nil {
		return nil, err
	}

	summary.Pending, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusPending)
	if err != nil {
		return nil, err
	}

	summary.Approved, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusApproved)
	if err != nil {
		return nil, err
	}

	summary.Rejected, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusRejected)
	if err != nil {
		return nil, err
	}

	summary.Completed, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusCompleted)
	if err != nil {
		return nil, err
	}

	summary.Cancelled, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.ReturnStatusCancelled)
	if err != nil {
		return nil, err
	}

	summary.Total = summary.Draft + summary.Pending + summary.Approved + summary.Rejected + summary.Completed + summary.Cancelled
	summary.PendingApproval = summary.Pending

	return summary, nil
}
