package trade

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/erp/backend/internal/domain/shared/valueobject"
	"github.com/erp/backend/internal/domain/trade"
	"github.com/erp/backend/internal/infrastructure/telemetry"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PricingStrategyProvider provides access to pricing strategies
type PricingStrategyProvider interface {
	GetPricingStrategy(name string) (strategy.PricingStrategy, error)
	GetPricingStrategyOrDefault(name string) strategy.PricingStrategy
}

// ProductSaleValidator validates whether a product can be sold
// This interface allows the trade context to validate products without
// directly depending on the catalog domain
type ProductSaleValidator interface {
	// CanBeSold checks if a product can be sold (is active)
	// Returns true if the product can be sold, false otherwise
	// Returns an error if the product is not found
	CanBeSold(ctx context.Context, tenantID, productID uuid.UUID) (bool, error)
}

// SalesOrderService handles sales order business operations
type SalesOrderService struct {
	orderRepo        trade.SalesOrderRepository
	eventPublisher   shared.EventPublisher
	pricingProvider  PricingStrategyProvider
	productValidator ProductSaleValidator
	businessMetrics  *telemetry.BusinessMetrics
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

// SetPricingProvider sets the pricing strategy provider
func (s *SalesOrderService) SetPricingProvider(provider PricingStrategyProvider) {
	s.pricingProvider = provider
}

// SetProductValidator sets the product validator for sale eligibility checks
func (s *SalesOrderService) SetProductValidator(validator ProductSaleValidator) {
	s.productValidator = validator
}

// SetBusinessMetrics sets the business metrics collector
func (s *SalesOrderService) SetBusinessMetrics(bm *telemetry.BusinessMetrics) {
	s.businessMetrics = bm
}

// validateProductForSale validates that a product can be sold
// Returns an error if the product is disabled or not found
func (s *SalesOrderService) validateProductForSale(ctx context.Context, tenantID, productID uuid.UUID, productCode string) error {
	if s.productValidator == nil {
		// No validator configured, skip validation
		return nil
	}

	canBeSold, err := s.productValidator.CanBeSold(ctx, tenantID, productID)
	if err != nil {
		return fmt.Errorf("failed to validate product %s: %w", productCode, err)
	}

	if !canBeSold {
		return shared.NewDomainError("PRODUCT_DISABLED",
			fmt.Sprintf("Product %s is disabled and cannot be sold", productCode))
	}

	return nil
}

// calculateItemPrice calculates the unit price for an item using the pricing strategy
// If no strategy is configured or if UseProvidedPrice is true, it uses the provided price
func (s *SalesOrderService) calculateItemPrice(
	ctx context.Context,
	tenantID uuid.UUID,
	item CreateSalesOrderItemInput,
	customerType string,
	pricingStrategyName string,
) decimal.Decimal {
	// If no pricing provider or strategy specified, use provided price
	if s.pricingProvider == nil || pricingStrategyName == "" {
		return item.UnitPrice
	}

	// If UnitPrice is explicitly provided and > 0, respect it (manual override)
	if item.UnitPrice.GreaterThan(decimal.Zero) && item.BasePrice.IsZero() {
		return item.UnitPrice
	}

	// Get the pricing strategy
	pricingStrategy := s.pricingProvider.GetPricingStrategyOrDefault(pricingStrategyName)
	if pricingStrategy == nil {
		return item.UnitPrice
	}

	// Determine base price: use BasePrice from item if provided, otherwise use UnitPrice
	basePrice := item.UnitPrice
	if item.BasePrice.GreaterThan(decimal.Zero) {
		basePrice = item.BasePrice
	}

	// Build pricing context
	pricingCtx := strategy.PricingContext{
		TenantID:     tenantID.String(),
		ProductID:    item.ProductID.String(),
		CustomerType: customerType,
		Quantity:     item.Quantity,
		BasePrice:    basePrice,
		Currency:     "CNY",
	}

	// Calculate price using strategy
	result, err := pricingStrategy.CalculatePrice(ctx, pricingCtx)
	if err != nil {
		// Fallback to provided price on error
		return item.UnitPrice
	}

	return result.UnitPrice
}

// Create creates a new sales order
func (s *SalesOrderService) Create(ctx context.Context, tenantID uuid.UUID, req CreateSalesOrderRequest) (*SalesOrderResponse, error) {
	// Start tracing span for order creation flow
	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "create")
	defer span.End()

	telemetry.SetAttributes(span,
		telemetry.SpanAttrCustomerID, req.CustomerID.String(),
		telemetry.SpanAttrCustomerName, req.CustomerName,
		"items_count", len(req.Items),
	)

	// Generate order number
	orderNumber, err := s.orderRepo.GenerateOrderNumber(ctx, tenantID)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}
	telemetry.SetAttribute(span, telemetry.SpanAttrOrderNumber, orderNumber)

	// Create order
	order, err := trade.NewSalesOrder(tenantID, orderNumber, req.CustomerID, req.CustomerName)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	// Set warehouse if provided
	if req.WarehouseID != nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			telemetry.RecordError(span, err)
			return nil, err
		}
	}

	// Add items
	for _, item := range req.Items {
		// Validate product can be sold (not disabled/discontinued)
		if err := s.validateProductForSale(ctx, tenantID, item.ProductID, item.ProductCode); err != nil {
			telemetry.RecordError(span, err)
			return nil, err
		}

		// Calculate unit price using pricing strategy if configured
		calculatedUnitPrice := s.calculateItemPrice(ctx, tenantID, item, req.CustomerLevel, req.PricingStrategyName)
		unitPrice := valueobject.NewMoneyCNY(calculatedUnitPrice)
		orderItem, err := order.AddItem(
			item.ProductID,
			item.ProductName,
			item.ProductCode,
			item.Unit,
			item.BaseUnit,
			item.Quantity,
			item.ConversionRate,
			unitPrice,
		)
		if err != nil {
			telemetry.RecordError(span, err)
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
			telemetry.RecordError(span, err)
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
		telemetry.RecordError(span, err)
		return nil, err
	}

	// Record business metrics
	if s.businessMetrics != nil {
		s.businessMetrics.RecordOrderWithAmount(ctx, tenantID, telemetry.OrderTypeSales, order.TotalAmount)
	}

	// Add final attributes to span
	telemetry.SetAttribute(span, telemetry.SpanAttrOrderID, order.ID.String())
	telemetry.AddEvent(span, "order_created",
		telemetry.SpanAttrOrderID, order.ID.String(),
		telemetry.SpanAttrOrderNumber, orderNumber,
	)

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

	// Validate product can be sold (not disabled/discontinued)
	if err := s.validateProductForSale(ctx, tenantID, req.ProductID, req.ProductCode); err != nil {
		return nil, err
	}

	unitPrice := valueobject.NewMoneyCNY(req.UnitPrice)
	item, err := order.AddItem(
		req.ProductID,
		req.ProductName,
		req.ProductCode,
		req.Unit,
		req.BaseUnit,
		req.Quantity,
		req.ConversionRate,
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
	// Start tracing span for order confirmation flow
	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "confirm")
	defer span.End()

	telemetry.SetAttribute(span, telemetry.SpanAttrOrderID, orderID.String())

	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.SetAttributes(span,
		telemetry.SpanAttrOrderNumber, order.OrderNumber,
		telemetry.SpanAttrCustomerID, order.CustomerID.String(),
		"items_count", len(order.Items),
	)

	// Set warehouse if provided and not already set
	if req.WarehouseID != nil && order.WarehouseID == nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			telemetry.RecordError(span, err)
			return nil, err
		}
	}

	// Confirm order
	if err := order.Confirm(); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	// Collect domain events before save
	events := order.GetDomainEvents()
	order.ClearDomainEvents()

	// Save with optimistic locking and events atomically (transactional outbox pattern)
	if err := s.orderRepo.SaveWithLockAndEvents(ctx, order, events); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.AddEvent(span, "order_confirmed",
		telemetry.SpanAttrOrderID, orderID.String(),
		telemetry.SpanAttrOrderStatus, string(order.Status),
		"events_published", len(events),
	)

	response := ToSalesOrderResponse(order)
	return &response, nil
}

// Ship marks an order as shipped
// This triggers stock deduction via domain events (P3-BE-006)
func (s *SalesOrderService) Ship(ctx context.Context, tenantID, orderID uuid.UUID, req ShipOrderRequest) (*SalesOrderResponse, error) {
	// Start tracing span for order shipping flow
	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "ship")
	defer span.End()

	telemetry.SetAttribute(span, telemetry.SpanAttrOrderID, orderID.String())

	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.SetAttributes(span,
		telemetry.SpanAttrOrderNumber, order.OrderNumber,
		telemetry.SpanAttrCustomerID, order.CustomerID.String(),
	)

	// Set warehouse if provided and not already set
	if req.WarehouseID != nil {
		if err := order.SetWarehouse(*req.WarehouseID); err != nil {
			telemetry.RecordError(span, err)
			return nil, err
		}
	}

	// Ship order
	if err := order.Ship(); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	// Collect domain events before save
	events := order.GetDomainEvents()
	order.ClearDomainEvents()

	// Save with optimistic locking and events atomically (transactional outbox pattern)
	if err := s.orderRepo.SaveWithLockAndEvents(ctx, order, events); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.AddEvent(span, "order_shipped",
		telemetry.SpanAttrOrderID, orderID.String(),
		telemetry.SpanAttrOrderStatus, string(order.Status),
		"events_published", len(events),
	)

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
	// Start tracing span for order cancellation flow
	ctx, span := telemetry.StartServiceSpan(ctx, "sales_order", "cancel")
	defer span.End()

	telemetry.SetAttribute(span, telemetry.SpanAttrOrderID, orderID.String())

	order, err := s.orderRepo.FindByIDForTenant(ctx, tenantID, orderID)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.SetAttributes(span,
		telemetry.SpanAttrOrderNumber, order.OrderNumber,
		telemetry.SpanAttrOrderStatus, string(order.Status),
		"cancel_reason", req.Reason,
	)

	if err := order.Cancel(req.Reason); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	// Collect domain events before save
	// The CancelledEvent includes WasConfirmed flag to indicate if locks need release
	events := order.GetDomainEvents()
	order.ClearDomainEvents()

	// Save with optimistic locking and events atomically (transactional outbox pattern)
	if err := s.orderRepo.SaveWithLockAndEvents(ctx, order, events); err != nil {
		telemetry.RecordError(span, err)
		return nil, err
	}

	telemetry.AddEvent(span, "order_cancelled",
		telemetry.SpanAttrOrderID, orderID.String(),
		"events_published", len(events),
	)

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
