package providers

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/printing"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockTakingProvider implements DataProvider for STOCK_TAKING document type.
// It loads stock taking data from the repository for use in print templates.
type StockTakingProvider struct {
	stockTakingRepo inventory.StockTakingRepository
	warehouseRepo   partner.WarehouseRepository
}

// NewStockTakingProvider creates a new StockTakingProvider.
func NewStockTakingProvider(
	stockTakingRepo inventory.StockTakingRepository,
	warehouseRepo partner.WarehouseRepository,
) *StockTakingProvider {
	return &StockTakingProvider{
		stockTakingRepo: stockTakingRepo,
		warehouseRepo:   warehouseRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *StockTakingProvider) GetDocType() printing.DocType {
	return printing.DocTypeStockTaking
}

// GetData retrieves stock taking data for rendering.
func (p *StockTakingProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the stock taking
	st, err := p.stockTakingRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load stock taking: %w", err)
	}

	// Load warehouse details
	warehouse, err := p.warehouseRepo.FindByIDForTenant(ctx, tenantID, st.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load warehouse: %w", err)
	}

	// Build document data
	docData := infra.NewDocumentData(printing.DocTypeStockTaking, st.TakingNumber)
	docData.Meta.Status = string(st.Status)
	docData.Meta.StatusText = statusToText(string(st.Status))
	docData.Meta.CreatedAt = st.CreatedAt
	docData.Meta.UpdatedAt = st.UpdatedAt
	docData.Meta.Remark = st.Remark
	docData.Meta.CreatedAtFormatted = st.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = st.UpdatedAt.Format("2006-01-02")
	docData.Meta.CreatedBy = st.CreatedByName

	// Build warehouse info
	warehouseInfo := infra.WarehouseInfo{
		ID:      warehouse.ID,
		Code:    warehouse.Code,
		Name:    warehouse.Name,
		Address: warehouse.Address,
		Phone:   warehouse.Phone,
		Manager: warehouse.ContactName,
	}

	// Build items and calculate summary
	items := make([]infra.StockTakingItemData, len(st.Items))
	matchedItems := 0
	surplusItems := 0
	shortageItems := 0
	totalSurplusQty := decimal.Zero
	totalShortageQty := decimal.Zero

	for i, item := range st.Items {
		variance := item.DifferenceQty
		varianceType := "MATCH"
		if variance.GreaterThan(decimal.Zero) {
			varianceType = "SURPLUS"
			surplusItems++
			totalSurplusQty = totalSurplusQty.Add(variance)
		} else if variance.LessThan(decimal.Zero) {
			varianceType = "SHORTAGE"
			shortageItems++
			totalShortageQty = totalShortageQty.Add(variance.Abs())
		} else {
			matchedItems++
		}

		items[i] = infra.StockTakingItemData{
			Index:                   i + 1,
			ProductID:               item.ProductID,
			ProductCode:             item.ProductCode,
			ProductName:             item.ProductName,
			Location:                "", // Stock taking items don't track location
			Unit:                    item.Unit,
			SystemQuantity:          item.SystemQuantity,
			ActualQuantity:          item.ActualQuantity,
			Variance:                variance,
			VarianceType:            varianceType,
			Remark:                  item.Remark,
			SystemQuantityFormatted: formatQuantity(item.SystemQuantity),
			ActualQuantityFormatted: formatQuantity(item.ActualQuantity),
			VarianceFormatted:       formatSignedQuantity(variance),
		}
	}

	// Build stock taking data
	stockTakingData := infra.StockTakingData{
		ID:               st.ID,
		TakingNo:         st.TakingNumber,
		Warehouse:        warehouseInfo,
		Items:            items,
		Status:           string(st.Status),
		StartedAt:        st.TakingDate,
		CompletedAt:      st.CompletedAt,
		CountedBy:        st.CreatedByName,
		VerifiedBy:       st.ApprovedByName,
		Remark:           st.Remark,
		TotalItems:       st.TotalItems,
		MatchedItems:     matchedItems,
		SurplusItems:     surplusItems,
		ShortageItems:    shortageItems,
		TotalSurplusQty:  totalSurplusQty,
		TotalShortageQty: totalShortageQty,
	}

	// Format dates
	stockTakingData.StartedAtFormatted = st.TakingDate.Format("2006-01-02")
	if st.CompletedAt != nil {
		stockTakingData.CompletedAtFormatted = st.CompletedAt.Format("2006-01-02")
	}

	docData.Document = stockTakingData

	return docData, nil
}

// formatSignedQuantity formats a quantity with sign indicator
func formatSignedQuantity(q decimal.Decimal) string {
	if q.GreaterThan(decimal.Zero) {
		return "+" + q.String()
	}
	return q.String()
}
