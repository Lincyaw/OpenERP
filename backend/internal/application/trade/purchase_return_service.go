package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
)

// PurchaseReturnService handles purchase return business operations
type PurchaseReturnService struct {
	returnRepo     trade.PurchaseReturnRepository
	orderRepo      trade.PurchaseOrderRepository
	eventPublisher shared.EventPublisher
}

// NewPurchaseReturnService creates a new PurchaseReturnService
func NewPurchaseReturnService(
	returnRepo trade.PurchaseReturnRepository,
	orderRepo trade.PurchaseOrderRepository,
) *PurchaseReturnService {
	return &PurchaseReturnService{
		returnRepo: returnRepo,
		orderRepo:  orderRepo,
	}
}

// SetEventPublisher sets the event publisher for cross-context integration
func (s *PurchaseReturnService) SetEventPublisher(publisher shared.EventPublisher) {
	s.eventPublisher = publisher
}

// Create creates a new purchase return from an existing purchase order
func (s *PurchaseReturnService) Create(ctx context.Context, tenantID uuid.UUID, req CreatePurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	// Get the purchase order
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, req.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	// Generate return number
	returnNumber, err := s.returnRepo.GenerateReturnNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create the return
	pr, err := trade.NewPurchaseReturn(tenantID, returnNumber, order)
	if err != nil {
		return nil, err
	}

	// Add items
	for _, item := range req.Items {
		// Find the order item
		orderItem := order.GetItem(item.PurchaseOrderItemID)
		if orderItem == nil {
			return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Purchase order item not found: "+item.PurchaseOrderItemID.String())
		}

		returnItem, err := pr.AddItem(orderItem, item.ReturnQuantity)
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
		if item.BatchNumber != "" {
			returnItem.SetBatchNumber(item.BatchNumber)
		}
	}

	// Set warehouse if provided, otherwise use order's warehouse
	if req.WarehouseID != nil {
		if err := pr.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Set optional fields
	if req.Reason != "" {
		pr.SetReason(req.Reason)
	}
	if req.Remark != "" {
		pr.SetRemark(req.Remark)
	}

	// Save the return
	if err := s.returnRepo.Save(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// GetByID retrieves a purchase return by ID
func (s *PurchaseReturnService) GetByID(ctx context.Context, tenantID, returnID uuid.UUID) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}
	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// GetByReturnNumber retrieves a purchase return by return number
func (s *PurchaseReturnService) GetByReturnNumber(ctx context.Context, tenantID uuid.UUID, returnNumber string) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByReturnNumber(ctx, tenantID, returnNumber)
	if err != nil {
		return nil, err
	}
	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// List retrieves a list of purchase returns with filtering and pagination
func (s *PurchaseReturnService) List(ctx context.Context, tenantID uuid.UUID, filter PurchaseReturnListFilter) ([]PurchaseReturnListItemResponse, int64, error) {
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
	if filter.SupplierID != nil {
		domainFilter.Filters["supplier_id"] = *filter.SupplierID
	}
	if filter.PurchaseOrderID != nil {
		domainFilter.Filters["purchase_order_id"] = *filter.PurchaseOrderID
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

	return ToPurchaseReturnListItemResponses(returns), total, nil
}

// ListByPurchaseOrder retrieves purchase returns for a specific purchase order
func (s *PurchaseReturnService) ListByPurchaseOrder(ctx context.Context, tenantID, purchaseOrderID uuid.UUID) ([]PurchaseReturnListItemResponse, error) {
	returns, err := s.returnRepo.FindByPurchaseOrder(ctx, tenantID, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	return ToPurchaseReturnListItemResponses(returns), nil
}

// ListPendingApproval retrieves returns pending approval
func (s *PurchaseReturnService) ListPendingApproval(ctx context.Context, tenantID uuid.UUID, filter PurchaseReturnListFilter) ([]PurchaseReturnListItemResponse, int64, error) {
	status := trade.PurchaseReturnStatusPending
	filter.Status = &status
	return s.List(ctx, tenantID, filter)
}

// Update updates a purchase return (only allowed in DRAFT status)
func (s *PurchaseReturnService) Update(ctx context.Context, tenantID, returnID uuid.UUID, req UpdatePurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	if !pr.CanModify() {
		return nil, shared.NewDomainError("INVALID_STATE", "Return can only be modified in draft status")
	}

	// Update warehouse if provided
	if req.WarehouseID != nil {
		if *req.WarehouseID == uuid.Nil {
			pr.WarehouseID = nil
		} else {
			if err := pr.SetWarehouse(*req.WarehouseID); err != nil {
				return nil, err
			}
		}
	}

	// Update reason if provided
	if req.Reason != nil {
		pr.SetReason(*req.Reason)
	}

	// Update remark if provided
	if req.Remark != nil {
		pr.SetRemark(*req.Remark)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Delete deletes a purchase return (only allowed in DRAFT status)
func (s *PurchaseReturnService) Delete(ctx context.Context, tenantID, returnID uuid.UUID) error {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return err
	}

	if !pr.IsDraft() {
		return shared.NewDomainError("INVALID_STATE", "Only draft returns can be deleted")
	}

	return s.returnRepo.DeleteForTenant(ctx, tenantID, returnID)
}

// AddItem adds an item to a return (only allowed in DRAFT status)
func (s *PurchaseReturnService) AddItem(ctx context.Context, tenantID, returnID uuid.UUID, req AddPurchaseReturnItemRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Get the purchase order
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, pr.PurchaseOrderID)
	if err != nil {
		return nil, err
	}

	// Find the order item
	orderItem := order.GetItem(req.PurchaseOrderItemID)
	if orderItem == nil {
		return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Purchase order item not found")
	}

	// Add the item
	returnItem, err := pr.AddItem(orderItem, req.ReturnQuantity)
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
	if req.BatchNumber != "" {
		returnItem.SetBatchNumber(req.BatchNumber)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// UpdateItem updates an item in a return (only allowed in DRAFT status)
func (s *PurchaseReturnService) UpdateItem(ctx context.Context, tenantID, returnID, itemID uuid.UUID, req UpdatePurchaseReturnItemRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Find the item
	item := pr.GetItem(itemID)
	if item == nil {
		return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Return item not found")
	}

	// Update quantity if provided
	if req.ReturnQuantity != nil {
		if err := pr.UpdateItemQuantity(itemID, *req.ReturnQuantity); err != nil {
			return nil, err
		}
		// Re-fetch the item after update
		item = pr.GetItem(itemID)
	}

	// Update reason if provided
	if req.Reason != nil {
		item.SetReason(*req.Reason)
	}

	// Update condition if provided
	if req.ConditionOnReturn != nil {
		item.SetCondition(*req.ConditionOnReturn)
	}

	// Update batch number if provided
	if req.BatchNumber != nil {
		item.SetBatchNumber(*req.BatchNumber)
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// RemoveItem removes an item from a return (only allowed in DRAFT status)
func (s *PurchaseReturnService) RemoveItem(ctx context.Context, tenantID, returnID, itemID uuid.UUID) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Remove the item
	if err := pr.RemoveItem(itemID); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Submit submits a return for approval
func (s *PurchaseReturnService) Submit(ctx context.Context, tenantID, returnID uuid.UUID) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Submit the return
	if err := pr.Submit(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Approve approves a return
func (s *PurchaseReturnService) Approve(ctx context.Context, tenantID, returnID, approverID uuid.UUID, req ApprovePurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Approve the return
	if err := pr.Approve(approverID, req.Note); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Reject rejects a return
func (s *PurchaseReturnService) Reject(ctx context.Context, tenantID, returnID, rejecterID uuid.UUID, req RejectPurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Reject the return
	if err := pr.Reject(rejecterID, req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Ship marks a return as shipped back to supplier
// This triggers inventory deduction via event handler
func (s *PurchaseReturnService) Ship(ctx context.Context, tenantID, returnID, shipperID uuid.UUID, req ShipPurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Ship the return
	if err := pr.Ship(shipperID, req.Note, req.TrackingNumber); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events - this will trigger inventory deduction
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Complete marks a return as completed (after supplier confirms receipt)
func (s *PurchaseReturnService) Complete(ctx context.Context, tenantID, returnID uuid.UUID, req CompletePurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Complete the return
	if err := pr.Complete(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// Cancel cancels a return
func (s *PurchaseReturnService) Cancel(ctx context.Context, tenantID, returnID uuid.UUID, req CancelPurchaseReturnRequest) (*PurchaseReturnResponse, error) {
	pr, err := s.returnRepo.FindByIDForTenant(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}

	// Cancel the return
	if err := pr.Cancel(req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.returnRepo.SaveWithLock(ctx, pr); err != nil {
		return nil, err
	}

	// Publish domain events
	if s.eventPublisher != nil {
		for _, event := range pr.GetDomainEvents() {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log but don't fail the operation
			}
		}
	}

	response := ToPurchaseReturnResponse(pr)
	return &response, nil
}

// GetStatusSummary returns a summary of returns by status for dashboard
func (s *PurchaseReturnService) GetStatusSummary(ctx context.Context, tenantID uuid.UUID) (*PurchaseReturnStatusSummary, error) {
	summary := &PurchaseReturnStatusSummary{}

	// Count by each status
	var err error
	summary.Draft, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusDraft)
	if err != nil {
		return nil, err
	}

	summary.Pending, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusPending)
	if err != nil {
		return nil, err
	}

	summary.Approved, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusApproved)
	if err != nil {
		return nil, err
	}

	summary.Rejected, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusRejected)
	if err != nil {
		return nil, err
	}

	summary.Shipped, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusShipped)
	if err != nil {
		return nil, err
	}

	summary.Completed, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusCompleted)
	if err != nil {
		return nil, err
	}

	summary.Cancelled, err = s.returnRepo.CountByStatus(ctx, tenantID, trade.PurchaseReturnStatusCancelled)
	if err != nil {
		return nil, err
	}

	summary.Total = summary.Draft + summary.Pending + summary.Approved + summary.Rejected + summary.Shipped + summary.Completed + summary.Cancelled
	summary.PendingApproval = summary.Pending
	summary.PendingShipment = summary.Approved

	return summary, nil
}
