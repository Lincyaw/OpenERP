package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
)

// PurchaseOrderService handles purchase order business operations
type PurchaseOrderService struct {
	orderRepo       trade.PurchaseOrderRepository
	eventPublisher  shared.EventPublisher
	businessMetrics *telemetry.BusinessMetrics
}

// NewPurchaseOrderService creates a new PurchaseOrderService
func NewPurchaseOrderService(orderRepo trade.PurchaseOrderRepository) *PurchaseOrderService {
	return &PurchaseOrderService{
		orderRepo: orderRepo,
	}
}

// SetEventPublisher sets the event publisher for publishing domain events
func (s *PurchaseOrderService) SetEventPublisher(publisher shared.EventPublisher) {
	s.eventPublisher = publisher
}

// SetBusinessMetrics sets the business metrics collector
func (s *PurchaseOrderService) SetBusinessMetrics(bm *telemetry.BusinessMetrics) {
	s.businessMetrics = bm
}

// Create creates a new purchase order
func (s *PurchaseOrderService) Create(ctx context.Context, tenantID uuid.UUID, req CreatePurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	// Generate order number
	orderNumber, err := s.orderRepo.GenerateOrderNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create order
	order, err := trade.NewPurchaseOrder(tenantID, orderNumber, req.SupplierID, req.SupplierName)
	if err != nil {
		return nil, err
	}

	// Set warehouse if provided
	if req.WarehouseID != nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Add items
	for _, item := range req.Items {
		unitCost := valueobject.NewMoneyCNY(item.UnitCost)
		orderItem, err := order.AddItem(
			item.ProductID,
			item.ProductName,
			item.ProductCode,
			item.Unit,
			item.BaseUnit,
			item.Quantity,
			item.ConversionRate,
			unitCost,
		)
		if err != nil {
			return nil, err
		}
		if item.Remark != "" {
			orderItem.SetRemark(item.Remark)
		}
	}

	// Apply discount if provided
	if req.Discount != nil {
		discountMoney := valueobject.NewMoneyCNY(*req.Discount)
		if err := order.ApplyDiscount(discountMoney); err != nil {
			return nil, err
		}
	}

	// Set remark
	if req.Remark != "" {
		order.SetRemark(req.Remark)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		order.SetCreatedBy(*req.CreatedBy)
	}

	// Save order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	// Record business metrics
	if s.businessMetrics != nil {
		s.businessMetrics.RecordOrderWithAmount(ctx, tenantID, telemetry.OrderTypePurchase, order.TotalAmount)
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// GetByID retrieves a purchase order by ID
func (s *PurchaseOrderService) GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// GetByOrderNumber retrieves a purchase order by order number
func (s *PurchaseOrderService) GetByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByOrderNumber(ctx, tenantID, orderNumber)
	if err != nil {
		return nil, err
	}
	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// List retrieves a list of purchase orders with filtering and pagination
func (s *PurchaseOrderService) List(ctx context.Context, tenantID uuid.UUID, filter PurchaseOrderListFilter) ([]PurchaseOrderListItemResponse, int64, error) {
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
		Filters:  make(map[string]interface{}),
	}

	// Add specific filters
	if filter.SupplierID != nil {
		domainFilter.Filters["supplier_id"] = *filter.SupplierID
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

	// Get orders
	orders, err := s.orderRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.orderRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	return ToPurchaseOrderListItemResponses(orders), total, nil
}

// ListBySupplier retrieves purchase orders for a specific supplier
func (s *PurchaseOrderService) ListBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter PurchaseOrderListFilter) ([]PurchaseOrderListItemResponse, int64, error) {
	filter.SupplierID = &supplierID
	return s.List(ctx, tenantID, filter)
}

// ListByStatus retrieves purchase orders by status
func (s *PurchaseOrderService) ListByStatus(ctx context.Context, tenantID uuid.UUID, status trade.PurchaseOrderStatus, filter PurchaseOrderListFilter) ([]PurchaseOrderListItemResponse, int64, error) {
	filter.Status = &status
	return s.List(ctx, tenantID, filter)
}

// ListPendingReceipt retrieves purchase orders that are pending receipt
func (s *PurchaseOrderService) ListPendingReceipt(ctx context.Context, tenantID uuid.UUID, filter PurchaseOrderListFilter) ([]PurchaseOrderListItemResponse, int64, error) {
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
		Filters:  make(map[string]interface{}),
	}

	// Get orders pending receipt
	orders, err := s.orderRepo.FindPendingReceipt(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.orderRepo.CountPendingReceipt(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	return ToPurchaseOrderListItemResponses(orders), total, nil
}

// Update updates a purchase order (only allowed in DRAFT status)
func (s *PurchaseOrderService) Update(ctx context.Context, tenantID, orderID uuid.UUID, req UpdatePurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if !order.CanModify() {
		return nil, shared.NewDomainError("INVALID_STATE", "Order can only be modified in draft status")
	}

	// Update warehouse
	if req.WarehouseID != nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Update discount
	if req.Discount != nil {
		discountMoney := valueobject.NewMoneyCNY(*req.Discount)
		if err := order.ApplyDiscount(discountMoney); err != nil {
			return nil, err
		}
	}

	// Update remark
	if req.Remark != nil {
		order.SetRemark(*req.Remark)
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// AddItem adds an item to a purchase order
func (s *PurchaseOrderService) AddItem(ctx context.Context, tenantID, orderID uuid.UUID, req AddPurchaseOrderItemRequest) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	unitCost := valueobject.NewMoneyCNY(req.UnitCost)
	item, err := order.AddItem(
		req.ProductID,
		req.ProductName,
		req.ProductCode,
		req.Unit,
		req.BaseUnit,
		req.Quantity,
		req.ConversionRate,
		unitCost,
	)
	if err != nil {
		return nil, err
	}

	if req.Remark != "" {
		item.SetRemark(req.Remark)
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// UpdateItem updates an item in a purchase order
func (s *PurchaseOrderService) UpdateItem(ctx context.Context, tenantID, orderID, itemID uuid.UUID, req UpdatePurchaseOrderItemRequest) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	// Update quantity
	if req.Quantity != nil {
		if err := order.UpdateItemQuantity(itemID, *req.Quantity); err != nil {
			return nil, err
		}
	}

	// Update cost
	if req.UnitCost != nil {
		unitCost := valueobject.NewMoneyCNY(*req.UnitCost)
		if err := order.UpdateItemCost(itemID, unitCost); err != nil {
			return nil, err
		}
	}

	// Update remark
	if req.Remark != nil {
		item := order.GetItem(itemID)
		if item == nil {
			return nil, shared.NewDomainError("ITEM_NOT_FOUND", "Order item not found")
		}
		item.SetRemark(*req.Remark)
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// RemoveItem removes an item from a purchase order
func (s *PurchaseOrderService) RemoveItem(ctx context.Context, tenantID, orderID, itemID uuid.UUID) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.RemoveItem(itemID); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// Confirm confirms a purchase order
func (s *PurchaseOrderService) Confirm(ctx context.Context, tenantID, orderID uuid.UUID, req ConfirmPurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	// Set warehouse if provided and not already set
	if req.WarehouseID != nil && order.WarehouseID == nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Confirm order
	if err := order.Confirm(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// Receive processes receipt of goods for a purchase order
// Returns the order and details of what was received
func (s *PurchaseOrderService) Receive(ctx context.Context, tenantID, orderID uuid.UUID, req ReceivePurchaseOrderRequest) (*ReceiveResultResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	// Set warehouse if provided and not already set
	if req.WarehouseID != nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			return nil, err
		}
	}

	// Convert request items to domain items
	receiveItems := make([]trade.ReceiveItem, len(req.Items))
	for i, item := range req.Items {
		receiveItems[i] = trade.ReceiveItem{
			ProductID:   item.ProductID,
			Quantity:    item.Quantity,
			BatchNumber: item.BatchNumber,
			ExpiryDate:  item.ExpiryDate,
		}
		if item.UnitCost != nil {
			receiveItems[i].UnitCost = *item.UnitCost
		}
	}

	// Process receive
	receivedInfos, err := order.Receive(receiveItems)
	if err != nil {
		return nil, err
	}

	// Collect domain events before save
	events := order.GetDomainEvents()
	order.ClearDomainEvents()

	// Save with optimistic locking and events atomically (transactional outbox pattern)
	if err := s.orderRepo.SaveWithLockAndEvents(ctx, order, events); err != nil {
		return nil, err
	}

	orderResponse := ToPurchaseOrderResponse(order)
	return &ReceiveResultResponse{
		Order:           orderResponse,
		ReceivedItems:   ToReceivedItemResponses(receivedInfos),
		IsFullyReceived: order.IsCompleted(),
	}, nil
}

// Cancel cancels a purchase order
func (s *PurchaseOrderService) Cancel(ctx context.Context, tenantID, orderID uuid.UUID, req CancelPurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.Cancel(req.Reason); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToPurchaseOrderResponse(order)
	return &response, nil
}

// Delete deletes a purchase order (only allowed in DRAFT status)
func (s *PurchaseOrderService) Delete(ctx context.Context, tenantID, orderID uuid.UUID) error {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return err
	}

	if !order.IsDraft() {
		return shared.NewDomainError("INVALID_STATE", "Only draft orders can be deleted")
	}

	return s.orderRepo.DeleteForTenant(ctx, tenantID, orderID)
}

// GetStatusSummary retrieves order count summary by status for a tenant
func (s *PurchaseOrderService) GetStatusSummary(ctx context.Context, tenantID uuid.UUID) (*PurchaseOrderStatusSummary, error) {
	draft, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.PurchaseOrderStatusDraft)
	if err != nil {
		return nil, err
	}

	confirmed, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.PurchaseOrderStatusConfirmed)
	if err != nil {
		return nil, err
	}

	partialReceived, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.PurchaseOrderStatusPartialReceived)
	if err != nil {
		return nil, err
	}

	completed, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.PurchaseOrderStatusCompleted)
	if err != nil {
		return nil, err
	}

	cancelled, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.PurchaseOrderStatusCancelled)
	if err != nil {
		return nil, err
	}

	pendingReceipt, err := s.orderRepo.CountPendingReceipt(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &PurchaseOrderStatusSummary{
		Draft:           draft,
		Confirmed:       confirmed,
		PartialReceived: partialReceived,
		Completed:       completed,
		Cancelled:       cancelled,
		Total:           draft + confirmed + partialReceived + completed + cancelled,
		PendingReceipt:  pendingReceipt,
	}, nil
}

// GetReceivableItems retrieves items that can still receive goods for an order
func (s *PurchaseOrderService) GetReceivableItems(ctx context.Context, tenantID, orderID uuid.UUID) ([]PurchaseOrderItemResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if !order.CanReceiveGoods() {
		return nil, shared.NewDomainError("INVALID_STATE", "Order cannot receive goods in current status")
	}

	receivableItems := order.GetReceivableItems()
	responses := make([]PurchaseOrderItemResponse, len(receivableItems))
	for i := range receivableItems {
		responses[i] = ToPurchaseOrderItemResponse(&receivableItems[i])
	}

	return responses, nil
}
