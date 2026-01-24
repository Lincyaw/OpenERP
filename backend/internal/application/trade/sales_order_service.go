package trade

import (
	"context"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/google/uuid"
)

// SalesOrderService handles sales order business operations
type SalesOrderService struct {
	orderRepo      trade.SalesOrderRepository
	eventPublisher shared.EventPublisher
}

// NewSalesOrderService creates a new SalesOrderService
func NewSalesOrderService(orderRepo trade.SalesOrderRepository) *SalesOrderService {
	return &SalesOrderService{
		orderRepo: orderRepo,
	}
}

// SetEventPublisher sets the event publisher for cross-context integration
func (s *SalesOrderService) SetEventPublisher(publisher shared.EventPublisher) {
	s.eventPublisher = publisher
}

// Create creates a new sales order
func (s *SalesOrderService) Create(ctx context.Context, tenantID uuid.UUID, req CreateSalesOrderRequest) (*SalesOrderResponse, error) {
	// Generate order number
	orderNumber, err := s.orderRepo.GenerateOrderNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Create order
	order, err := trade.NewSalesOrder(tenantID, orderNumber, req.CustomerID, req.CustomerName)
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
		unitPrice := valueobject.NewMoneyCNY(item.UnitPrice)
		orderItem, err := order.AddItem(
			item.ProductID,
			item.ProductName,
			item.ProductCode,
			item.Unit,
			item.Quantity,
			unitPrice,
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

	// Save order
	if err := s.orderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// GetByID retrieves a sales order by ID
func (s *SalesOrderService) GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*SalesOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	response := ToSalesOrderResponse(order)
	return &response, nil
}

// GetByOrderNumber retrieves a sales order by order number
func (s *SalesOrderService) GetByOrderNumber(ctx context.Context, tenantID uuid.UUID, orderNumber string) (*SalesOrderResponse, error) {
	order, err := s.orderRepo.FindByOrderNumber(ctx, tenantID, orderNumber)
	if err != nil {
		return nil, err
	}
	response := ToSalesOrderResponse(order)
	return &response, nil
}

// List retrieves a list of sales orders with filtering and pagination
func (s *SalesOrderService) List(ctx context.Context, tenantID uuid.UUID, filter SalesOrderListFilter) ([]SalesOrderListItemResponse, int64, error) {
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
	if filter.CustomerID != nil {
		domainFilter.Filters["customer_id"] = *filter.CustomerID
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

	return ToSalesOrderListItemResponses(orders), total, nil
}

// ListByCustomer retrieves sales orders for a specific customer
func (s *SalesOrderService) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter SalesOrderListFilter) ([]SalesOrderListItemResponse, int64, error) {
	filter.CustomerID = &customerID
	return s.List(ctx, tenantID, filter)
}

// ListByStatus retrieves sales orders by status
func (s *SalesOrderService) ListByStatus(ctx context.Context, tenantID uuid.UUID, status trade.OrderStatus, filter SalesOrderListFilter) ([]SalesOrderListItemResponse, int64, error) {
	filter.Status = &status
	return s.List(ctx, tenantID, filter)
}

// Update updates a sales order (only allowed in DRAFT status)
func (s *SalesOrderService) Update(ctx context.Context, tenantID, orderID uuid.UUID, req UpdateSalesOrderRequest) (*SalesOrderResponse, error) {
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

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// AddItem adds an item to a sales order
func (s *SalesOrderService) AddItem(ctx context.Context, tenantID, orderID uuid.UUID, req AddOrderItemRequest) (*SalesOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	unitPrice := valueobject.NewMoneyCNY(req.UnitPrice)
	item, err := order.AddItem(
		req.ProductID,
		req.ProductName,
		req.ProductCode,
		req.Unit,
		req.Quantity,
		unitPrice,
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

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// UpdateItem updates an item in a sales order
func (s *SalesOrderService) UpdateItem(ctx context.Context, tenantID, orderID, itemID uuid.UUID, req UpdateOrderItemRequest) (*SalesOrderResponse, error) {
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

	// Update price
	if req.UnitPrice != nil {
		unitPrice := valueobject.NewMoneyCNY(*req.UnitPrice)
		if err := order.UpdateItemPrice(itemID, unitPrice); err != nil {
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

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// RemoveItem removes an item from a sales order
func (s *SalesOrderService) RemoveItem(ctx context.Context, tenantID, orderID, itemID uuid.UUID) (*SalesOrderResponse, error) {
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

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Confirm confirms a sales order
// This triggers stock locking via domain events (P3-BE-006)
func (s *SalesOrderService) Confirm(ctx context.Context, tenantID, orderID uuid.UUID, req ConfirmOrderRequest) (*SalesOrderResponse, error) {
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

	// Publish domain events for cross-context integration (stock locking)
	if s.eventPublisher != nil {
		events := order.GetDomainEvents()
		for _, event := range events {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log error but don't fail the operation - event handling is async
				// TODO: Consider outbox pattern for guaranteed delivery
			}
		}
		order.ClearDomainEvents()
	}

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Ship marks an order as shipped
// This triggers stock deduction via domain events (P3-BE-006)
func (s *SalesOrderService) Ship(ctx context.Context, tenantID, orderID uuid.UUID, req ShipOrderRequest) (*SalesOrderResponse, error) {
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

	// Ship order
	if err := order.Ship(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	// Publish domain events for cross-context integration (stock deduction)
	if s.eventPublisher != nil {
		events := order.GetDomainEvents()
		for _, event := range events {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log error but don't fail the operation
			}
		}
		order.ClearDomainEvents()
	}

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Complete marks an order as completed
func (s *SalesOrderService) Complete(ctx context.Context, tenantID, orderID uuid.UUID) (*SalesOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}

	if err := order.Complete(); err != nil {
		return nil, err
	}

	// Save with optimistic locking
	if err := s.orderRepo.SaveWithLock(ctx, order); err != nil {
		return nil, err
	}

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Cancel cancels a sales order
// This triggers stock unlock via domain events (P3-BE-006) if order was confirmed
func (s *SalesOrderService) Cancel(ctx context.Context, tenantID, orderID uuid.UUID, req CancelOrderRequest) (*SalesOrderResponse, error) {
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

	// Publish domain events for cross-context integration (stock unlocking)
	// The CancelledEvent includes WasConfirmed flag to indicate if locks need release
	if s.eventPublisher != nil {
		events := order.GetDomainEvents()
		for _, event := range events {
			if err := s.eventPublisher.Publish(ctx, event); err != nil {
				// Log error but don't fail the operation
			}
		}
		order.ClearDomainEvents()
	}

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Delete deletes a sales order (only allowed in DRAFT status)
func (s *SalesOrderService) Delete(ctx context.Context, tenantID, orderID uuid.UUID) error {
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
func (s *SalesOrderService) GetStatusSummary(ctx context.Context, tenantID uuid.UUID) (*OrderStatusSummary, error) {
	draft, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.OrderStatusDraft)
	if err != nil {
		return nil, err
	}

	confirmed, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.OrderStatusConfirmed)
	if err != nil {
		return nil, err
	}

	shipped, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.OrderStatusShipped)
	if err != nil {
		return nil, err
	}

	completed, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.OrderStatusCompleted)
	if err != nil {
		return nil, err
	}

	cancelled, err := s.orderRepo.CountByStatus(ctx, tenantID, trade.OrderStatusCancelled)
	if err != nil {
		return nil, err
	}

	return &OrderStatusSummary{
		Draft:     draft,
		Confirmed: confirmed,
		Shipped:   shipped,
		Completed: completed,
		Cancelled: cancelled,
		Total:     draft + confirmed + shipped + completed + cancelled,
	}, nil
}
