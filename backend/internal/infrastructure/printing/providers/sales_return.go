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

// SalesReturnProvider implements DataProvider for SALES_RETURN document type.
// It loads sales return data from the repository for use in print templates.
type SalesReturnProvider struct {
	salesReturnRepo trade.SalesReturnRepository
	customerRepo    partner.CustomerRepository
	warehouseRepo   partner.WarehouseRepository
}

// NewSalesReturnProvider creates a new SalesReturnProvider.
func NewSalesReturnProvider(
	salesReturnRepo trade.SalesReturnRepository,
	customerRepo partner.CustomerRepository,
	warehouseRepo partner.WarehouseRepository,
) *SalesReturnProvider {
	return &SalesReturnProvider{
		salesReturnRepo: salesReturnRepo,
		customerRepo:    customerRepo,
		warehouseRepo:   warehouseRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *SalesReturnProvider) GetDocType() printing.DocType {
	return printing.DocTypeSalesReturn
}

// GetData retrieves sales return data for rendering.
func (p *SalesReturnProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the sales return
	ret, err := p.salesReturnRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sales return: %w", err)
	}

	// Load customer details
	customer, err := p.customerRepo.FindByIDForTenant(ctx, tenantID, ret.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load customer: %w", err)
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
	docData := infra.NewDocumentData(printing.DocTypeSalesReturn, ret.ReturnNumber)
	docData.Meta.Status = string(ret.Status)
	docData.Meta.StatusText = statusToText(string(ret.Status))
	docData.Meta.CreatedAt = ret.CreatedAt
	docData.Meta.UpdatedAt = ret.UpdatedAt
	docData.Meta.Remark = ret.Remark
	docData.Meta.CreatedAtFormatted = ret.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = ret.UpdatedAt.Format("2006-01-02")

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

	// Build return items
	items := make([]infra.SalesReturnItemData, len(ret.Items))
	totalQuantity := decimal.Zero
	for i, item := range ret.Items {
		totalQuantity = totalQuantity.Add(item.ReturnQuantity)
		items[i] = infra.SalesReturnItemData{
			Index:                 i + 1,
			ProductID:             item.ProductID,
			ProductCode:           item.ProductCode,
			ProductName:           item.ProductName,
			Unit:                  item.Unit,
			Quantity:              item.ReturnQuantity,
			UnitPrice:             item.UnitPrice,
			Amount:                item.RefundAmount,
			Reason:                item.Reason,
			QuantityFormatted:     formatQuantity(item.ReturnQuantity),
			UnitPriceFormatted:    infra.FormatMoneyValue(item.UnitPrice),
			RefundAmountFormatted: infra.FormatMoneyValue(item.RefundAmount),
		}
	}

	// Build sales return data
	salesReturnData := infra.SalesReturnData{
		ID:                   ret.ID,
		ReturnNumber:         ret.ReturnNumber,
		SalesOrderNumber:     ret.SalesOrderNumber,
		Customer:             customerInfo,
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

	docData.Document = salesReturnData

	return docData, nil
}
