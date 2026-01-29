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

// PurchaseOrderProvider implements DataProvider for PURCHASE_ORDER document type.
// It loads purchase order data from the repository for use in print templates.
type PurchaseOrderProvider struct {
	purchaseOrderRepo trade.PurchaseOrderRepository
	supplierRepo      partner.SupplierRepository
	warehouseRepo     partner.WarehouseRepository
}

// NewPurchaseOrderProvider creates a new PurchaseOrderProvider.
func NewPurchaseOrderProvider(
	purchaseOrderRepo trade.PurchaseOrderRepository,
	supplierRepo partner.SupplierRepository,
	warehouseRepo partner.WarehouseRepository,
) *PurchaseOrderProvider {
	return &PurchaseOrderProvider{
		purchaseOrderRepo: purchaseOrderRepo,
		supplierRepo:      supplierRepo,
		warehouseRepo:     warehouseRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *PurchaseOrderProvider) GetDocType() printing.DocType {
	return printing.DocTypePurchaseOrder
}

// GetData retrieves purchase order data for rendering.
func (p *PurchaseOrderProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the purchase order
	order, err := p.purchaseOrderRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purchase order: %w", err)
	}

	// Load supplier details
	supplier, err := p.supplierRepo.FindByIDForTenant(ctx, tenantID, order.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("failed to load supplier: %w", err)
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
	docData := infra.NewDocumentData(printing.DocTypePurchaseOrder, order.OrderNumber)
	docData.Meta.Status = string(order.Status)
	docData.Meta.StatusText = statusToText(string(order.Status))
	docData.Meta.CreatedAt = order.CreatedAt
	docData.Meta.UpdatedAt = order.UpdatedAt
	docData.Meta.Remark = order.Remark
	docData.Meta.CreatedAtFormatted = order.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = order.UpdatedAt.Format("2006-01-02")

	// Build supplier info
	supplierInfo := infra.SupplierInfo{
		ID:          supplier.ID,
		Code:        supplier.Code,
		Name:        supplier.Name,
		Contact:     supplier.ContactName,
		Phone:       supplier.Phone,
		Email:       supplier.Email,
		Address:     supplier.Address,
		BankName:    supplier.BankName,
		BankAccount: supplier.BankAccount,
		TaxID:       supplier.TaxID,
	}

	// Build order items
	items := make([]infra.PurchaseOrderItemData, len(order.Items))
	totalQuantity := decimal.Zero
	for i, item := range order.Items {
		totalQuantity = totalQuantity.Add(item.OrderedQuantity)
		items[i] = infra.PurchaseOrderItemData{
			Index:              i + 1,
			ProductID:          item.ProductID,
			ProductCode:        item.ProductCode,
			ProductName:        item.ProductName,
			Unit:               item.Unit,
			Quantity:           item.OrderedQuantity,
			UnitPrice:          item.UnitCost,
			Amount:             item.Amount,
			Remark:             item.Remark,
			QuantityFormatted:  formatQuantity(item.OrderedQuantity),
			UnitPriceFormatted: infra.FormatMoneyValue(item.UnitCost),
			AmountFormatted:    infra.FormatMoneyValue(item.Amount),
		}
	}

	// Build purchase order data
	purchaseOrderData := infra.PurchaseOrderData{
		ID:                   order.ID,
		OrderNumber:          order.OrderNumber,
		Supplier:             supplierInfo,
		Warehouse:            warehouseInfo,
		Items:                items,
		TotalAmount:          order.TotalAmount,
		TotalQuantity:        totalQuantity,
		ItemCount:            len(order.Items),
		Status:               string(order.Status),
		ConfirmedAt:          order.ConfirmedAt,
		Remark:               order.Remark,
		TotalAmountFormatted: infra.FormatMoneyValue(order.TotalAmount),
		TotalAmountChinese:   infra.MoneyToChinese(order.TotalAmount),
	}

	docData.Document = purchaseOrderData

	return docData, nil
}
