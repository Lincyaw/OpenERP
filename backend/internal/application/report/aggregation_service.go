package report

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/report"
	"github.com/erp/backend/internal/infrastructure/scheduler"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ReportCacheRepository defines the interface for storing cached report data
type ReportCacheRepository interface {
	// Sales Summary
	SaveSalesSummaryCache(ctx context.Context, cache *SalesSummaryCacheModel) error
	GetSalesSummaryCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*SalesSummaryCacheModel, error)

	// Daily Sales
	SaveSalesDailyCache(ctx context.Context, caches []*SalesDailyCacheModel) error
	GetSalesDailyCache(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*SalesDailyCacheModel, error)

	// Inventory Summary
	SaveInventorySummaryCache(ctx context.Context, cache *InventorySummaryCacheModel) error
	GetInventorySummaryCache(ctx context.Context, tenantID uuid.UUID, snapshotDate time.Time) (*InventorySummaryCacheModel, error)

	// Monthly P&L
	SavePnlMonthlyCache(ctx context.Context, cache *PnlMonthlyCacheModel) error
	GetPnlMonthlyCache(ctx context.Context, tenantID uuid.UUID, year, month int) (*PnlMonthlyCacheModel, error)

	// Product Ranking
	SaveProductRankingCache(ctx context.Context, caches []*ProductRankingCacheModel) error
	GetProductRankingCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, topN int) ([]*ProductRankingCacheModel, error)

	// Customer Ranking
	SaveCustomerRankingCache(ctx context.Context, caches []*CustomerRankingCacheModel) error
	GetCustomerRankingCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, topN int) ([]*CustomerRankingCacheModel, error)

	// Metadata
	UpdateCacheMetadata(ctx context.Context, tenantID uuid.UUID, reportType string, periodStart, periodEnd time.Time) error
	InvalidateCache(ctx context.Context, tenantID uuid.UUID, reportType string) error
}

// Cache models for storing pre-computed data
type SalesSummaryCacheModel struct {
	ID               uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID         uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	PeriodStart      time.Time       `gorm:"column:period_start;not null"`
	PeriodEnd        time.Time       `gorm:"column:period_end;not null"`
	TotalOrders      int64           `gorm:"column:total_orders;default:0"`
	TotalQuantity    decimal.Decimal `gorm:"column:total_quantity;type:decimal(20,4);default:0"`
	TotalSalesAmount decimal.Decimal `gorm:"column:total_sales_amount;type:decimal(20,4);default:0"`
	TotalCostAmount  decimal.Decimal `gorm:"column:total_cost_amount;type:decimal(20,4);default:0"`
	TotalGrossProfit decimal.Decimal `gorm:"column:total_gross_profit;type:decimal(20,4);default:0"`
	AvgOrderValue    decimal.Decimal `gorm:"column:avg_order_value;type:decimal(20,4);default:0"`
	ProfitMargin     decimal.Decimal `gorm:"column:profit_margin;type:decimal(10,4);default:0"`
	ComputedAt       time.Time       `gorm:"column:computed_at"`
	CreatedAt        time.Time       `gorm:"column:created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at"`
}

func (SalesSummaryCacheModel) TableName() string {
	return "report_sales_summary_cache"
}

type SalesDailyCacheModel struct {
	ID          uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID    uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	Date        time.Time       `gorm:"column:date;type:date;not null"`
	OrderCount  int64           `gorm:"column:order_count;default:0"`
	TotalAmount decimal.Decimal `gorm:"column:total_amount;type:decimal(20,4);default:0"`
	TotalProfit decimal.Decimal `gorm:"column:total_profit;type:decimal(20,4);default:0"`
	ItemsSold   decimal.Decimal `gorm:"column:items_sold;type:decimal(20,4);default:0"`
	ComputedAt  time.Time       `gorm:"column:computed_at"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at"`
}

func (SalesDailyCacheModel) TableName() string {
	return "report_sales_daily_cache"
}

type InventorySummaryCacheModel struct {
	ID              uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID        uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	SnapshotDate    time.Time       `gorm:"column:snapshot_date;type:date;not null"`
	TotalProducts   int64           `gorm:"column:total_products;default:0"`
	TotalQuantity   decimal.Decimal `gorm:"column:total_quantity;type:decimal(20,4);default:0"`
	TotalValue      decimal.Decimal `gorm:"column:total_value;type:decimal(20,4);default:0"`
	AvgTurnoverRate decimal.Decimal `gorm:"column:avg_turnover_rate;type:decimal(10,4);default:0"`
	LowStockCount   int64           `gorm:"column:low_stock_count;default:0"`
	OutOfStockCount int64           `gorm:"column:out_of_stock_count;default:0"`
	OverstockCount  int64           `gorm:"column:overstock_count;default:0"`
	ComputedAt      time.Time       `gorm:"column:computed_at"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

func (InventorySummaryCacheModel) TableName() string {
	return "report_inventory_summary_cache"
}

type PnlMonthlyCacheModel struct {
	ID              uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID        uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	Year            int             `gorm:"column:year;not null"`
	Month           int             `gorm:"column:month;not null"`
	SalesRevenue    decimal.Decimal `gorm:"column:sales_revenue;type:decimal(20,4);default:0"`
	SalesReturns    decimal.Decimal `gorm:"column:sales_returns;type:decimal(20,4);default:0"`
	NetSalesRevenue decimal.Decimal `gorm:"column:net_sales_revenue;type:decimal(20,4);default:0"`
	COGS            decimal.Decimal `gorm:"column:cogs;type:decimal(20,4);default:0"`
	GrossProfit     decimal.Decimal `gorm:"column:gross_profit;type:decimal(20,4);default:0"`
	GrossMargin     decimal.Decimal `gorm:"column:gross_margin;type:decimal(10,4);default:0"`
	OtherIncome     decimal.Decimal `gorm:"column:other_income;type:decimal(20,4);default:0"`
	TotalIncome     decimal.Decimal `gorm:"column:total_income;type:decimal(20,4);default:0"`
	Expenses        decimal.Decimal `gorm:"column:expenses;type:decimal(20,4);default:0"`
	NetProfit       decimal.Decimal `gorm:"column:net_profit;type:decimal(20,4);default:0"`
	NetMargin       decimal.Decimal `gorm:"column:net_margin;type:decimal(10,4);default:0"`
	ComputedAt      time.Time       `gorm:"column:computed_at"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

func (PnlMonthlyCacheModel) TableName() string {
	return "report_pnl_monthly_cache"
}

type ProductRankingCacheModel struct {
	ID            uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID      uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	PeriodStart   time.Time       `gorm:"column:period_start;type:date;not null"`
	PeriodEnd     time.Time       `gorm:"column:period_end;type:date;not null"`
	Rank          int             `gorm:"column:rank;not null"`
	ProductID     uuid.UUID       `gorm:"column:product_id;type:uuid;not null"`
	ProductSKU    string          `gorm:"column:product_sku;size:100;not null"`
	ProductName   string          `gorm:"column:product_name;size:255;not null"`
	CategoryName  string          `gorm:"column:category_name;size:255"`
	TotalQuantity decimal.Decimal `gorm:"column:total_quantity;type:decimal(20,4);default:0"`
	TotalAmount   decimal.Decimal `gorm:"column:total_amount;type:decimal(20,4);default:0"`
	TotalProfit   decimal.Decimal `gorm:"column:total_profit;type:decimal(20,4);default:0"`
	OrderCount    int64           `gorm:"column:order_count;default:0"`
	ComputedAt    time.Time       `gorm:"column:computed_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at"`
}

func (ProductRankingCacheModel) TableName() string {
	return "report_product_ranking_cache"
}

type CustomerRankingCacheModel struct {
	ID            uuid.UUID       `gorm:"column:id;type:uuid;primaryKey"`
	TenantID      uuid.UUID       `gorm:"column:tenant_id;type:uuid;not null"`
	PeriodStart   time.Time       `gorm:"column:period_start;type:date;not null"`
	PeriodEnd     time.Time       `gorm:"column:period_end;type:date;not null"`
	Rank          int             `gorm:"column:rank;not null"`
	CustomerID    uuid.UUID       `gorm:"column:customer_id;type:uuid;not null"`
	CustomerName  string          `gorm:"column:customer_name;size:255;not null"`
	TotalOrders   int64           `gorm:"column:total_orders;default:0"`
	TotalQuantity decimal.Decimal `gorm:"column:total_quantity;type:decimal(20,4);default:0"`
	TotalAmount   decimal.Decimal `gorm:"column:total_amount;type:decimal(20,4);default:0"`
	TotalProfit   decimal.Decimal `gorm:"column:total_profit;type:decimal(20,4);default:0"`
	ComputedAt    time.Time       `gorm:"column:computed_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at"`
}

func (CustomerRankingCacheModel) TableName() string {
	return "report_customer_ranking_cache"
}

// ReportAggregationService computes and caches report data
type ReportAggregationService struct {
	salesRepo     report.SalesReportRepository
	inventoryRepo report.InventoryReportRepository
	financeRepo   report.FinanceReportRepository
	cacheRepo     ReportCacheRepository
	logger        *zap.Logger
}

// NewReportAggregationService creates a new aggregation service
func NewReportAggregationService(
	salesRepo report.SalesReportRepository,
	inventoryRepo report.InventoryReportRepository,
	financeRepo report.FinanceReportRepository,
	cacheRepo ReportCacheRepository,
	logger *zap.Logger,
) *ReportAggregationService {
	return &ReportAggregationService{
		salesRepo:     salesRepo,
		inventoryRepo: inventoryRepo,
		financeRepo:   financeRepo,
		cacheRepo:     cacheRepo,
		logger:        logger,
	}
}

// Execute implements scheduler.JobExecutor
func (s *ReportAggregationService) Execute(ctx context.Context, job *scheduler.Job) error {
	if job.TenantID == nil {
		return scheduler.ErrInvalidReportType
	}

	tenantID := *job.TenantID

	switch job.ReportType {
	case scheduler.ReportTypeSalesSummary:
		return s.computeSalesSummary(ctx, tenantID, job.PeriodStart, job.PeriodEnd)
	case scheduler.ReportTypeSalesDailyTrend:
		return s.computeSalesDailyTrend(ctx, tenantID, job.PeriodStart, job.PeriodEnd)
	case scheduler.ReportTypeInventorySummary:
		return s.computeInventorySummary(ctx, tenantID, job.PeriodEnd)
	case scheduler.ReportTypeProfitLossMonthly:
		return s.computePnlMonthly(ctx, tenantID, job.PeriodStart, job.PeriodEnd)
	case scheduler.ReportTypeProductRanking:
		return s.computeProductRanking(ctx, tenantID, job.PeriodStart, job.PeriodEnd)
	case scheduler.ReportTypeCustomerRanking:
		return s.computeCustomerRanking(ctx, tenantID, job.PeriodStart, job.PeriodEnd)
	default:
		return scheduler.ErrInvalidReportType
	}
}

// computeSalesSummary computes and caches sales summary
func (s *ReportAggregationService) computeSalesSummary(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	filter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: periodStart,
		EndDate:   periodEnd,
	}

	summary, err := s.salesRepo.GetSalesSummary(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	cache := &SalesSummaryCacheModel{
		ID:               uuid.New(),
		TenantID:         tenantID,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		TotalOrders:      summary.TotalOrders,
		TotalQuantity:    summary.TotalQuantity,
		TotalSalesAmount: summary.TotalSalesAmount,
		TotalCostAmount:  summary.TotalCostAmount,
		TotalGrossProfit: summary.TotalGrossProfit,
		AvgOrderValue:    summary.AvgOrderValue,
		ProfitMargin:     summary.ProfitMargin,
		ComputedAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.cacheRepo.SaveSalesSummaryCache(ctx, cache); err != nil {
		return err
	}

	s.logger.Info("Sales summary computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Time("period_start", periodStart),
		zap.Time("period_end", periodEnd),
	)

	return nil
}

// computeSalesDailyTrend computes and caches daily sales trend
func (s *ReportAggregationService) computeSalesDailyTrend(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	filter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: periodStart,
		EndDate:   periodEnd,
	}

	trends, err := s.salesRepo.GetDailySalesTrend(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	caches := make([]*SalesDailyCacheModel, len(trends))
	for i, trend := range trends {
		caches[i] = &SalesDailyCacheModel{
			ID:          uuid.New(),
			TenantID:    tenantID,
			Date:        trend.Date,
			OrderCount:  trend.OrderCount,
			TotalAmount: trend.TotalAmount,
			TotalProfit: trend.TotalProfit,
			ItemsSold:   trend.ItemsSold,
			ComputedAt:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	if err := s.cacheRepo.SaveSalesDailyCache(ctx, caches); err != nil {
		return err
	}

	s.logger.Info("Daily sales trend computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("days", len(caches)),
	)

	return nil
}

// computeInventorySummary computes and caches inventory summary
func (s *ReportAggregationService) computeInventorySummary(ctx context.Context, tenantID uuid.UUID, snapshotDate time.Time) error {
	filter := report.InventoryTurnoverFilter{
		TenantID:  tenantID,
		StartDate: snapshotDate.AddDate(0, -1, 0), // Last month for turnover calculation
		EndDate:   snapshotDate,
	}

	summary, err := s.inventoryRepo.GetInventorySummary(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	cache := &InventorySummaryCacheModel{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SnapshotDate:    snapshotDate,
		TotalProducts:   summary.TotalProducts,
		TotalQuantity:   summary.TotalQuantity,
		TotalValue:      summary.TotalValue,
		AvgTurnoverRate: summary.AvgTurnoverRate,
		LowStockCount:   summary.LowStockCount,
		OutOfStockCount: summary.OutOfStockCount,
		OverstockCount:  summary.OverstockCount,
		ComputedAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.cacheRepo.SaveInventorySummaryCache(ctx, cache); err != nil {
		return err
	}

	s.logger.Info("Inventory summary computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Time("snapshot_date", snapshotDate),
	)

	return nil
}

// computePnlMonthly computes and caches monthly P&L
func (s *ReportAggregationService) computePnlMonthly(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	filter := report.FinanceReportFilter{
		TenantID:  tenantID,
		StartDate: periodStart,
		EndDate:   periodEnd,
	}

	pnl, err := s.financeRepo.GetProfitLossStatement(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	cache := &PnlMonthlyCacheModel{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Year:            periodStart.Year(),
		Month:           int(periodStart.Month()),
		SalesRevenue:    pnl.SalesRevenue,
		SalesReturns:    pnl.SalesReturns,
		NetSalesRevenue: pnl.NetSalesRevenue,
		COGS:            pnl.COGS,
		GrossProfit:     pnl.GrossProfit,
		GrossMargin:     pnl.GrossMargin,
		OtherIncome:     pnl.OtherIncome,
		TotalIncome:     pnl.TotalIncome,
		Expenses:        pnl.Expenses,
		NetProfit:       pnl.NetProfit,
		NetMargin:       pnl.NetMargin,
		ComputedAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.cacheRepo.SavePnlMonthlyCache(ctx, cache); err != nil {
		return err
	}

	s.logger.Info("Monthly P&L computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("year", cache.Year),
		zap.Int("month", cache.Month),
	)

	return nil
}

// computeProductRanking computes and caches product ranking
func (s *ReportAggregationService) computeProductRanking(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	filter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: periodStart,
		EndDate:   periodEnd,
		TopN:      50, // Default top 50 products
	}

	rankings, err := s.salesRepo.GetProductSalesRanking(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	caches := make([]*ProductRankingCacheModel, len(rankings))
	for i, ranking := range rankings {
		caches[i] = &ProductRankingCacheModel{
			ID:            uuid.New(),
			TenantID:      tenantID,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			Rank:          ranking.Rank,
			ProductID:     ranking.ProductID,
			ProductSKU:    ranking.ProductSKU,
			ProductName:   ranking.ProductName,
			CategoryName:  ranking.CategoryName,
			TotalQuantity: ranking.TotalQuantity,
			TotalAmount:   ranking.TotalAmount,
			TotalProfit:   ranking.TotalProfit,
			OrderCount:    ranking.OrderCount,
			ComputedAt:    now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	}

	if err := s.cacheRepo.SaveProductRankingCache(ctx, caches); err != nil {
		return err
	}

	s.logger.Info("Product ranking computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("products", len(caches)),
	)

	return nil
}

// computeCustomerRanking computes and caches customer ranking
func (s *ReportAggregationService) computeCustomerRanking(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	filter := report.SalesReportFilter{
		TenantID:  tenantID,
		StartDate: periodStart,
		EndDate:   periodEnd,
		TopN:      50, // Default top 50 customers
	}

	rankings, err := s.salesRepo.GetCustomerSalesRanking(filter)
	if err != nil {
		return err
	}

	now := time.Now()
	caches := make([]*CustomerRankingCacheModel, len(rankings))
	for i, ranking := range rankings {
		caches[i] = &CustomerRankingCacheModel{
			ID:            uuid.New(),
			TenantID:      tenantID,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			Rank:          ranking.Rank,
			CustomerID:    ranking.CustomerID,
			CustomerName:  ranking.CustomerName,
			TotalOrders:   ranking.TotalOrders,
			TotalQuantity: ranking.TotalQuantity,
			TotalAmount:   ranking.TotalAmount,
			TotalProfit:   ranking.TotalProfit,
			ComputedAt:    now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	}

	if err := s.cacheRepo.SaveCustomerRankingCache(ctx, caches); err != nil {
		return err
	}

	s.logger.Info("Customer ranking computed and cached",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("customers", len(caches)),
	)

	return nil
}

// RefreshReport manually refreshes a specific report type
func (s *ReportAggregationService) RefreshReport(ctx context.Context, tenantID uuid.UUID, reportType scheduler.ReportType, periodStart, periodEnd time.Time) error {
	job := &scheduler.Job{
		ID:          uuid.New(),
		TenantID:    &tenantID,
		ReportType:  reportType,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}
	return s.Execute(ctx, job)
}

// RefreshAllReports refreshes all report types for a tenant
func (s *ReportAggregationService) RefreshAllReports(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	for _, reportType := range scheduler.AllReportTypes() {
		if err := s.RefreshReport(ctx, tenantID, reportType, periodStart, periodEnd); err != nil {
			s.logger.Error("Failed to refresh report",
				zap.String("tenant_id", tenantID.String()),
				zap.String("report_type", string(reportType)),
				zap.Error(err),
			)
			// Continue with other reports
		}
	}
	return nil
}

// GormReportCacheRepository is the GORM implementation of ReportCacheRepository
type GormReportCacheRepository struct {
	db *gorm.DB
}

// NewGormReportCacheRepository creates a new GORM report cache repository
func NewGormReportCacheRepository(db *gorm.DB) *GormReportCacheRepository {
	return &GormReportCacheRepository{db: db}
}

// SaveSalesSummaryCache saves sales summary cache (upsert)
func (r *GormReportCacheRepository) SaveSalesSummaryCache(ctx context.Context, cache *SalesSummaryCacheModel) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ? AND period_end = ?", cache.TenantID, cache.PeriodStart, cache.PeriodEnd).
		Assign(cache).
		FirstOrCreate(cache).Error
}

// GetSalesSummaryCache retrieves sales summary cache
func (r *GormReportCacheRepository) GetSalesSummaryCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*SalesSummaryCacheModel, error) {
	var cache SalesSummaryCacheModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, periodStart, periodEnd).
		First(&cache).Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

// SaveSalesDailyCache saves daily sales cache
func (r *GormReportCacheRepository) SaveSalesDailyCache(ctx context.Context, caches []*SalesDailyCacheModel) error {
	if len(caches) == 0 {
		return nil
	}

	// Delete existing records for the date range and tenant
	tenantID := caches[0].TenantID
	dates := make([]time.Time, len(caches))
	for i, c := range caches {
		dates[i] = c.Date
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing
		if err := tx.Where("tenant_id = ? AND date IN ?", tenantID, dates).Delete(&SalesDailyCacheModel{}).Error; err != nil {
			return err
		}
		// Insert new
		return tx.Create(caches).Error
	})
}

// GetSalesDailyCache retrieves daily sales cache
func (r *GormReportCacheRepository) GetSalesDailyCache(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*SalesDailyCacheModel, error) {
	var caches []*SalesDailyCacheModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND date >= ? AND date <= ?", tenantID, startDate, endDate).
		Order("date ASC").
		Find(&caches).Error
	return caches, err
}

// SaveInventorySummaryCache saves inventory summary cache (upsert)
func (r *GormReportCacheRepository) SaveInventorySummaryCache(ctx context.Context, cache *InventorySummaryCacheModel) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND snapshot_date = ?", cache.TenantID, cache.SnapshotDate).
		Assign(cache).
		FirstOrCreate(cache).Error
}

// GetInventorySummaryCache retrieves inventory summary cache
func (r *GormReportCacheRepository) GetInventorySummaryCache(ctx context.Context, tenantID uuid.UUID, snapshotDate time.Time) (*InventorySummaryCacheModel, error) {
	var cache InventorySummaryCacheModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND snapshot_date = ?", tenantID, snapshotDate).
		First(&cache).Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

// SavePnlMonthlyCache saves monthly P&L cache (upsert)
func (r *GormReportCacheRepository) SavePnlMonthlyCache(ctx context.Context, cache *PnlMonthlyCacheModel) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND year = ? AND month = ?", cache.TenantID, cache.Year, cache.Month).
		Assign(cache).
		FirstOrCreate(cache).Error
}

// GetPnlMonthlyCache retrieves monthly P&L cache
func (r *GormReportCacheRepository) GetPnlMonthlyCache(ctx context.Context, tenantID uuid.UUID, year, month int) (*PnlMonthlyCacheModel, error) {
	var cache PnlMonthlyCacheModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND year = ? AND month = ?", tenantID, year, month).
		First(&cache).Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

// SaveProductRankingCache saves product ranking cache
func (r *GormReportCacheRepository) SaveProductRankingCache(ctx context.Context, caches []*ProductRankingCacheModel) error {
	if len(caches) == 0 {
		return nil
	}

	tenantID := caches[0].TenantID
	periodStart := caches[0].PeriodStart
	periodEnd := caches[0].PeriodEnd

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing
		if err := tx.Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, periodStart, periodEnd).
			Delete(&ProductRankingCacheModel{}).Error; err != nil {
			return err
		}
		// Insert new
		return tx.Create(caches).Error
	})
}

// GetProductRankingCache retrieves product ranking cache
func (r *GormReportCacheRepository) GetProductRankingCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, topN int) ([]*ProductRankingCacheModel, error) {
	var caches []*ProductRankingCacheModel
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, periodStart, periodEnd).
		Order("rank ASC")

	if topN > 0 {
		query = query.Limit(topN)
	}

	err := query.Find(&caches).Error
	return caches, err
}

// SaveCustomerRankingCache saves customer ranking cache
func (r *GormReportCacheRepository) SaveCustomerRankingCache(ctx context.Context, caches []*CustomerRankingCacheModel) error {
	if len(caches) == 0 {
		return nil
	}

	tenantID := caches[0].TenantID
	periodStart := caches[0].PeriodStart
	periodEnd := caches[0].PeriodEnd

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing
		if err := tx.Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, periodStart, periodEnd).
			Delete(&CustomerRankingCacheModel{}).Error; err != nil {
			return err
		}
		// Insert new
		return tx.Create(caches).Error
	})
}

// GetCustomerRankingCache retrieves customer ranking cache
func (r *GormReportCacheRepository) GetCustomerRankingCache(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, topN int) ([]*CustomerRankingCacheModel, error) {
	var caches []*CustomerRankingCacheModel
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, periodStart, periodEnd).
		Order("rank ASC")

	if topN > 0 {
		query = query.Limit(topN)
	}

	err := query.Find(&caches).Error
	return caches, err
}

// UpdateCacheMetadata updates cache metadata
func (r *GormReportCacheRepository) UpdateCacheMetadata(ctx context.Context, tenantID uuid.UUID, reportType string, periodStart, periodEnd time.Time) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO report_cache_metadata (id, tenant_id, report_type, period_start, period_end, computed_at, is_valid, version, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, ?, ?, true, 1, ?, ?)
		ON CONFLICT (tenant_id, report_type, period_start, period_end)
		DO UPDATE SET computed_at = ?, is_valid = true, version = report_cache_metadata.version + 1, updated_at = ?
	`, tenantID, reportType, periodStart, periodEnd, now, now, now, now, now)
	return result.Error
}

// InvalidateCache marks cache as invalid
func (r *GormReportCacheRepository) InvalidateCache(ctx context.Context, tenantID uuid.UUID, reportType string) error {
	return r.db.WithContext(ctx).
		Model(&struct {
			TableName string `gorm:"-"`
		}{}).
		Table("report_cache_metadata").
		Where("tenant_id = ? AND report_type = ?", tenantID, reportType).
		Update("is_valid", false).Error
}
