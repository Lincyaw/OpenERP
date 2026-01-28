package persistence

import (
	"github.com/erp/backend/internal/domain/report"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormInventoryReportRepository implements InventoryReportRepository using GORM
type GormInventoryReportRepository struct {
	db *gorm.DB
}

// NewGormInventoryReportRepository creates a new GormInventoryReportRepository
func NewGormInventoryReportRepository(db *gorm.DB) *GormInventoryReportRepository {
	return &GormInventoryReportRepository{db: db}
}

// GetInventorySummary returns aggregated inventory summary
func (r *GormInventoryReportRepository) GetInventorySummary(filter report.InventoryTurnoverFilter) (*report.InventorySummary, error) {
	type summaryResult struct {
		TotalProducts   int64
		TotalQuantity   decimal.Decimal
		TotalValue      decimal.Decimal
		LowStockCount   int64
		OutOfStockCount int64
	}

	var result summaryResult

	query := r.db.Table("inventory_items ii").
		Select(`
			COUNT(DISTINCT ii.product_id) as total_products,
			COALESCE(SUM(ii.available_quantity), 0) as total_quantity,
			COALESCE(SUM(ii.available_quantity * ii.average_cost), 0) as total_value,
			SUM(CASE WHEN ii.available_quantity > 0 AND ii.available_quantity <= 10 THEN 1 ELSE 0 END) as low_stock_count,
			SUM(CASE WHEN ii.available_quantity = 0 THEN 1 ELSE 0 END) as out_of_stock_count
		`).
		Where("ii.tenant_id = ?", filter.TenantID)

	if filter.WarehouseID != nil {
		query = query.Where("ii.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.ProductID != nil {
		query = query.Where("ii.product_id = ?", *filter.ProductID)
	}
	if filter.CategoryID != nil {
		query = query.Joins("JOIN products p ON p.id = ii.product_id").
			Where("p.category_id = ?", *filter.CategoryID)
	}

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}

	return &report.InventorySummary{
		TotalProducts:   result.TotalProducts,
		TotalQuantity:   result.TotalQuantity,
		TotalValue:      result.TotalValue,
		AvgTurnoverRate: decimal.Zero, // Would need historical data calculation
		LowStockCount:   result.LowStockCount,
		OutOfStockCount: result.OutOfStockCount,
		OverstockCount:  0, // Would need threshold configuration
	}, nil
}

// GetInventoryTurnover returns inventory turnover for products
func (r *GormInventoryReportRepository) GetInventoryTurnover(filter report.InventoryTurnoverFilter) ([]report.InventoryTurnover, error) {
	type turnoverResult struct {
		ProductID     uuid.UUID
		ProductSKU    string
		ProductName   string
		CategoryID    *uuid.UUID
		CategoryName  string
		WarehouseID   *uuid.UUID
		WarehouseName string
		EndingStock   decimal.Decimal
		SoldQuantity  decimal.Decimal
		StockValue    decimal.Decimal
	}

	var results []turnoverResult

	// Get current inventory with sold quantities in period
	query := r.db.Table("inventory_items ii").
		Select(`
			ii.product_id,
			p.code as product_sku,
			p.name as product_name,
			p.category_id,
			COALESCE(c.name, '') as category_name,
			ii.warehouse_id,
			w.name as warehouse_name,
			ii.available_quantity as ending_stock,
			COALESCE((
				SELECT SUM(soi.quantity)
				FROM sales_order_items soi
				JOIN sales_orders so ON so.id = soi.order_id
				WHERE soi.product_id = ii.product_id
				AND so.warehouse_id = ii.warehouse_id
				AND so.tenant_id = ii.tenant_id
				AND so.created_at BETWEEN ? AND ?
				AND so.status IN ('CONFIRMED', 'SHIPPED', 'COMPLETED')
			), 0) as sold_quantity,
			(ii.available_quantity * ii.average_cost) as stock_value
		`, filter.StartDate, filter.EndDate).
		Joins("JOIN products p ON p.id = ii.product_id").
		Joins("JOIN warehouses w ON w.id = ii.warehouse_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("ii.tenant_id = ?", filter.TenantID)

	if filter.WarehouseID != nil {
		query = query.Where("ii.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.ProductID != nil {
		query = query.Where("ii.product_id = ?", *filter.ProductID)
	}
	if filter.CategoryID != nil {
		query = query.Where("p.category_id = ?", *filter.CategoryID)
	}

	query = query.Order("sold_quantity DESC")

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	turnovers := make([]report.InventoryTurnover, len(results))
	for i, res := range results {
		// Calculate turnover metrics
		// Beginning stock is estimated as ending stock + sold quantity
		beginningStock := res.EndingStock.Add(res.SoldQuantity)
		averageStock := beginningStock.Add(res.EndingStock).Div(decimal.NewFromInt(2))

		var turnoverRate, daysOfInventory decimal.Decimal
		if !averageStock.IsZero() {
			turnoverRate = res.SoldQuantity.Div(averageStock)
			// Calculate days of inventory (period days / turnover rate)
			periodDays := filter.EndDate.Sub(filter.StartDate).Hours() / 24
			if !turnoverRate.IsZero() {
				daysOfInventory = decimal.NewFromFloat(periodDays).Div(turnoverRate)
			}
		}

		warehouseName := res.WarehouseName
		turnovers[i] = report.InventoryTurnover{
			ProductID:       res.ProductID,
			ProductSKU:      res.ProductSKU,
			ProductName:     res.ProductName,
			CategoryID:      res.CategoryID,
			CategoryName:    res.CategoryName,
			WarehouseID:     res.WarehouseID,
			WarehouseName:   warehouseName,
			BeginningStock:  beginningStock,
			EndingStock:     res.EndingStock,
			AverageStock:    averageStock,
			SoldQuantity:    res.SoldQuantity,
			TurnoverRate:    turnoverRate,
			DaysOfInventory: daysOfInventory,
			StockValue:      res.StockValue,
		}
	}

	return turnovers, nil
}

// GetInventoryValueByCategory returns inventory value grouped by category
func (r *GormInventoryReportRepository) GetInventoryValueByCategory(filter report.InventoryTurnoverFilter) ([]report.InventoryValueByCategory, error) {
	type categoryResult struct {
		CategoryID    *uuid.UUID
		CategoryName  string
		ProductCount  int64
		TotalQuantity decimal.Decimal
		TotalValue    decimal.Decimal
	}

	var results []categoryResult

	// First get total value for percentage calculation
	var totalInventoryValue decimal.Decimal
	if err := r.db.Table("inventory_items ii").
		Select("COALESCE(SUM(ii.available_quantity * ii.average_cost), 0)").
		Where("ii.tenant_id = ?", filter.TenantID).
		Scan(&totalInventoryValue).Error; err != nil {
		return nil, err
	}

	err := r.db.Table("inventory_items ii").
		Select(`
			p.category_id,
			COALESCE(c.name, 'Uncategorized') as category_name,
			COUNT(DISTINCT ii.product_id) as product_count,
			COALESCE(SUM(ii.available_quantity), 0) as total_quantity,
			COALESCE(SUM(ii.available_quantity * ii.average_cost), 0) as total_value
		`).
		Joins("JOIN products p ON p.id = ii.product_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("ii.tenant_id = ?", filter.TenantID).
		Group("p.category_id, c.name").
		Order("total_value DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	values := make([]report.InventoryValueByCategory, len(results))
	for i, r := range results {
		var percentage decimal.Decimal
		if !totalInventoryValue.IsZero() {
			percentage = r.TotalValue.Div(totalInventoryValue).Mul(decimal.NewFromInt(100))
		}

		values[i] = report.InventoryValueByCategory{
			CategoryID:    r.CategoryID,
			CategoryName:  r.CategoryName,
			ProductCount:  r.ProductCount,
			TotalQuantity: r.TotalQuantity,
			TotalValue:    r.TotalValue,
			Percentage:    percentage,
		}
	}

	return values, nil
}

// GetInventoryValueByWarehouse returns inventory value grouped by warehouse
func (r *GormInventoryReportRepository) GetInventoryValueByWarehouse(filter report.InventoryTurnoverFilter) ([]report.InventoryValueByWarehouse, error) {
	type warehouseResult struct {
		WarehouseID   uuid.UUID
		WarehouseName string
		ProductCount  int64
		TotalQuantity decimal.Decimal
		TotalValue    decimal.Decimal
	}

	var results []warehouseResult

	// First get total value for percentage calculation
	var totalInventoryValue decimal.Decimal
	if err := r.db.Table("inventory_items ii").
		Select("COALESCE(SUM(ii.available_quantity * ii.average_cost), 0)").
		Where("ii.tenant_id = ?", filter.TenantID).
		Scan(&totalInventoryValue).Error; err != nil {
		return nil, err
	}

	err := r.db.Table("inventory_items ii").
		Select(`
			ii.warehouse_id,
			w.name as warehouse_name,
			COUNT(DISTINCT ii.product_id) as product_count,
			COALESCE(SUM(ii.available_quantity), 0) as total_quantity,
			COALESCE(SUM(ii.available_quantity * ii.average_cost), 0) as total_value
		`).
		Joins("JOIN warehouses w ON w.id = ii.warehouse_id").
		Where("ii.tenant_id = ?", filter.TenantID).
		Group("ii.warehouse_id, w.name").
		Order("total_value DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	values := make([]report.InventoryValueByWarehouse, len(results))
	for i, r := range results {
		var percentage decimal.Decimal
		if !totalInventoryValue.IsZero() {
			percentage = r.TotalValue.Div(totalInventoryValue).Mul(decimal.NewFromInt(100))
		}

		values[i] = report.InventoryValueByWarehouse{
			WarehouseID:   r.WarehouseID,
			WarehouseName: r.WarehouseName,
			ProductCount:  r.ProductCount,
			TotalQuantity: r.TotalQuantity,
			TotalValue:    r.TotalValue,
			Percentage:    percentage,
		}
	}

	return values, nil
}

// GetSlowMovingProducts returns products with low turnover
func (r *GormInventoryReportRepository) GetSlowMovingProducts(filter report.InventoryTurnoverFilter) ([]report.InventoryTurnover, error) {
	type slowMovingResult struct {
		ProductID     uuid.UUID
		ProductSKU    string
		ProductName   string
		CategoryID    *uuid.UUID
		CategoryName  string
		WarehouseID   *uuid.UUID
		WarehouseName string
		EndingStock   decimal.Decimal
		SoldQuantity  decimal.Decimal
		StockValue    decimal.Decimal
	}

	var results []slowMovingResult

	topN := filter.TopN
	if topN <= 0 {
		topN = 10
	}

	// Find products with stock but low sales
	query := r.db.Table("inventory_items ii").
		Select(`
			ii.product_id,
			p.code as product_sku,
			p.name as product_name,
			p.category_id,
			COALESCE(c.name, '') as category_name,
			ii.warehouse_id,
			w.name as warehouse_name,
			ii.available_quantity as ending_stock,
			COALESCE((
				SELECT SUM(soi.quantity)
				FROM sales_order_items soi
				JOIN sales_orders so ON so.id = soi.order_id
				WHERE soi.product_id = ii.product_id
				AND so.warehouse_id = ii.warehouse_id
				AND so.tenant_id = ii.tenant_id
				AND so.created_at BETWEEN ? AND ?
				AND so.status IN ('CONFIRMED', 'SHIPPED', 'COMPLETED')
			), 0) as sold_quantity,
			(ii.available_quantity * ii.average_cost) as stock_value
		`, filter.StartDate, filter.EndDate).
		Joins("JOIN products p ON p.id = ii.product_id").
		Joins("JOIN warehouses w ON w.id = ii.warehouse_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("ii.tenant_id = ?", filter.TenantID).
		Where("ii.available_quantity > 0").           // Has stock
		Order("sold_quantity ASC, stock_value DESC"). // Lowest sales first, then highest value
		Limit(topN)

	if filter.WarehouseID != nil {
		query = query.Where("ii.warehouse_id = ?", *filter.WarehouseID)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	turnovers := make([]report.InventoryTurnover, len(results))
	for i, res := range results {
		beginningStock := res.EndingStock.Add(res.SoldQuantity)
		averageStock := beginningStock.Add(res.EndingStock).Div(decimal.NewFromInt(2))

		var turnoverRate, daysOfInventory decimal.Decimal
		if !averageStock.IsZero() {
			turnoverRate = res.SoldQuantity.Div(averageStock)
			periodDays := filter.EndDate.Sub(filter.StartDate).Hours() / 24
			if !turnoverRate.IsZero() {
				daysOfInventory = decimal.NewFromFloat(periodDays).Div(turnoverRate)
			}
		}

		warehouseName := res.WarehouseName
		turnovers[i] = report.InventoryTurnover{
			ProductID:       res.ProductID,
			ProductSKU:      res.ProductSKU,
			ProductName:     res.ProductName,
			CategoryID:      res.CategoryID,
			CategoryName:    res.CategoryName,
			WarehouseID:     res.WarehouseID,
			WarehouseName:   warehouseName,
			BeginningStock:  beginningStock,
			EndingStock:     res.EndingStock,
			AverageStock:    averageStock,
			SoldQuantity:    res.SoldQuantity,
			TurnoverRate:    turnoverRate,
			DaysOfInventory: daysOfInventory,
			StockValue:      res.StockValue,
		}
	}

	return turnovers, nil
}
