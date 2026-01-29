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

// PurchaseReturnProvider implements DataProvider for PURCHASE_RETURN document type.
// It loads purchase return data from the repository for use in print templates.
type PurchaseReturnProvider struct {
	purchaseReturnRepo trade.PurchaseReturnRepository
	supplierRepo       partner.SupplierRepository
	warehouseRepo      partner.WarehouseRepository
}

// NewPurchaseReturnProvider creates a new PurchaseReturnProvider.
func NewPurchaseReturnProvider(
	purchaseReturnRepo trade.PurchaseReturnRepository,
	supplierRepo partner.SupplierRepository,
	warehouseRepo partner.WarehouseRepository,
) *PurchaseReturnProvider {
	return &PurchaseReturnProvider{
		purchaseReturnRepo: purchaseReturnRepo,
		supplierRepo:       supplierRepo,
		warehouseRepo:      warehouseRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *PurchaseReturnProvider) GetDocType() printing.DocType {
	return printing.DocTypePurchaseReturn
}

// GetData retrieves purchase return data for rendering.
func (p *PurchaseReturnProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the purchase return
	ret, err := p.purchaseReturnRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load purchase return: %w", err)
	}

	// Load supplier details
	supplier, err := p.supplierRepo.FindByIDForTenant(ctx, tenantID, ret.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("failed to load supplier: %w", err)
	}

	// Load warehouse details if set
	var warehouseInfo *infra.WarehouseInfo
	if ret.WarehouseID != nil {
		warehouse, err := p.warehouseRepo.FindByIDForTenant(ctx, tenantID, *ret.WarehouseID)
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
	docData := infra.NewDocumentData(printing.DocTypePurchaseReturn, ret.ReturnNumber)
	docData.Meta.Status = string(ret.Status)
	docData.Meta.StatusText = statusToText(string(ret.Status))
	docData.Meta.CreatedAt = ret.CreatedAt
	docData.Meta.UpdatedAt = ret.UpdatedAt
	docData.Meta.Remark = ret.Remark
	docData.Meta.CreatedAtFormatted = ret.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = ret.UpdatedAt.Format("2006-01-02")

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

	// Build return items
	items := make([]infra.PurchaseReturnItemData, len(ret.Items))
	totalQuantity := decimal.Zero
	for i, item := range ret.Items {
		totalQuantity = totalQuantity.Add(item.ReturnQuantity)
		items[i] = infra.PurchaseReturnItemData{
			Index:                 i + 1,
			ProductID:             item.ProductID,
			ProductCode:           item.ProductCode,
			ProductName:           item.ProductName,
			Unit:                  item.Unit,
			Quantity:              item.ReturnQuantity,
			UnitPrice:             item.UnitCost,
			Amount:                item.RefundAmount,
			Reason:                item.Reason,
			QuantityFormatted:     formatQuantity(item.ReturnQuantity),
			UnitPriceFormatted:    infra.FormatMoneyValue(item.UnitCost),
			RefundAmountFormatted: infra.FormatMoneyValue(item.RefundAmount),
		}
	}

	// Build purchase return data
	purchaseReturnData := infra.PurchaseReturnData{
		ID:                   ret.ID,
		ReturnNumber:         ret.ReturnNumber,
		PurchaseOrderNumber:  ret.PurchaseOrderNumber,
		Supplier:             supplierInfo,
		Warehouse:            warehouseInfo,
		Items:                items,
		TotalRefund:          ret.TotalRefund,
		TotalQuantity:        totalQuantity,
		ItemCount:            len(ret.Items),
		Status:               string(ret.Status),
		Reason:               ret.Reason,
		ApprovedAt:           ret.ApprovedAt,
		CompletedAt:          ret.CompletedAt,
		Remark:               ret.Remark,
		TotalRefundFormatted: infra.FormatMoneyValue(ret.TotalRefund),
		TotalRefundChinese:   infra.MoneyToChinese(ret.TotalRefund),
	}

	docData.Document = purchaseReturnData

	return docData, nil
}
