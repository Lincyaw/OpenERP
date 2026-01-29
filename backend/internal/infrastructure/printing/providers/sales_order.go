package providers

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/trade"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SalesOrderProvider implements DataProvider for SALES_ORDER document type.
// It loads sales order data from the repository for use in print templates.
type SalesOrderProvider struct {
	salesOrderRepo trade.SalesOrderRepository
	customerRepo   partner.CustomerRepository
	warehouseRepo  partner.WarehouseRepository
}

// NewSalesOrderProvider creates a new SalesOrderProvider.
func NewSalesOrderProvider(
	salesOrderRepo trade.SalesOrderRepository,
	customerRepo partner.CustomerRepository,
	warehouseRepo partner.WarehouseRepository,
) *SalesOrderProvider {
	return &SalesOrderProvider{
		salesOrderRepo: salesOrderRepo,
		customerRepo:   customerRepo,
		warehouseRepo:  warehouseRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *SalesOrderProvider) GetDocType() printing.DocType {
	return printing.DocTypeSalesOrder
}

// GetData retrieves sales order data for rendering.
func (p *SalesOrderProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the sales order
	order, err := p.salesOrderRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sales order: %w", err)
	}

	// Load customer details
	customer, err := p.customerRepo.FindByIDForTenant(ctx, tenantID, order.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load customer: %w", err)
	}

	// Load warehouse details if set
	var warehouseInfo *infra.WarehouseInfo
	if order.WarehouseID != nil {
		warehouse, err := p.warehouseRepo.FindByIDForTenant(ctx, tenantID, *order.WarehouseID)
		if err != nil {
			return nil, fmt.Errorf("failed to load warehouse: %w", err)
		}
		warehouseInfo = &infra.WarehouseInfo{
			ID:      warehouse.ID,
			Code:    warehouse.Code,
			Name:    warehouse.Name,
			Address: warehouse.Address,
			Phone:   warehouse.Phone,
			Manager: warehouse.ContactName,
		}
	}

	// Build document data
	docData := infra.NewDocumentData(printing.DocTypeSalesOrder, order.OrderNumber)
	docData.Meta.Status = string(order.Status)
	docData.Meta.StatusText = statusToText(string(order.Status))
	docData.Meta.CreatedAt = order.CreatedAt
	docData.Meta.UpdatedAt = order.UpdatedAt
	docData.Meta.Remark = order.Remark
	docData.Meta.CreatedAtFormatted = order.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = order.UpdatedAt.Format("2006-01-02")

	// Build customer info
	customerInfo := infra.CustomerInfo{
		ID:      customer.ID,
		Code:    customer.Code,
		Name:    customer.Name,
		Contact: customer.ContactName,
		Phone:   customer.Phone,
		Email:   customer.Email,
		Address: customer.Address,
		TaxID:   customer.TaxID,
	}

	// Build order items
	items := make([]infra.SalesOrderItemData, len(order.Items))
	totalQuantity := decimal.Zero
	for i, item := range order.Items {
		totalQuantity = totalQuantity.Add(item.Quantity)
		items[i] = infra.SalesOrderItemData{
			Index:              i + 1,
			ProductID:          item.ProductID,
			ProductCode:        item.ProductCode,
			ProductName:        item.ProductName,
			Unit:               item.Unit,
			Quantity:           item.Quantity,
			UnitPrice:          item.UnitPrice,
			Amount:             item.Amount,
			Remark:             item.Remark,
			QuantityFormatted:  formatQuantity(item.Quantity),
			UnitPriceFormatted: infra.FormatMoneyValue(item.UnitPrice),
			AmountFormatted:    infra.FormatMoneyValue(item.Amount),
		}
	}

	// Build sales order data
	salesOrderData := infra.SalesOrderData{
		ID:                      order.ID,
		OrderNumber:             order.OrderNumber,
		Customer:                customerInfo,
		Warehouse:               warehouseInfo,
		Items:                   items,
		TotalAmount:             order.TotalAmount,
		DiscountAmount:          order.DiscountAmount,
		PayableAmount:           order.PayableAmount,
		TotalQuantity:           totalQuantity,
		ItemCount:               len(order.Items),
		Status:                  string(order.Status),
		ConfirmedAt:             order.ConfirmedAt,
		ShippedAt:               order.ShippedAt,
		CompletedAt:             order.CompletedAt,
		Remark:                  order.Remark,
		TotalAmountFormatted:    infra.FormatMoneyValue(order.TotalAmount),
		DiscountAmountFormatted: infra.FormatMoneyValue(order.DiscountAmount),
		PayableAmountFormatted:  infra.FormatMoneyValue(order.PayableAmount),
		PayableAmountChinese:    infra.MoneyToChinese(order.PayableAmount),
	}

	docData.Document = salesOrderData

	return docData, nil
}

// statusToText converts status code to display text
func statusToText(status string) string {
	statusMap := map[string]string{
		"DRAFT":            "草稿",
		"CONFIRMED":        "已确认",
		"SHIPPED":          "已发货",
		"COMPLETED":        "已完成",
		"CANCELLED":        "已取消",
		"PENDING":          "待处理",
		"APPROVED":         "已审批",
		"REJECTED":         "已拒绝",
		"RECEIVED":         "已收货",
		"PARTIAL_RECEIVED": "部分收货",
		"SUBMITTED":        "已提交",
		"COUNTING":         "盘点中",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return status
}

// formatQuantity formats a quantity value
func formatQuantity(q decimal.Decimal) string {
	// Remove trailing zeros
	return q.String()
}
