package persistence

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/billing"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UsageHistoryModel is the GORM model for usage history snapshots
type UsageHistoryModel struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID      `gorm:"type:uuid;index;not null"`
	SnapshotDate    time.Time      `gorm:"type:date;not null"`
	UsersCount      int64          `gorm:"not null;default:0"`
	ProductsCount   int64          `gorm:"not null;default:0"`
	WarehousesCount int64          `gorm:"not null;default:0"`
	CustomersCount  int64          `gorm:"not null;default:0"`
	SuppliersCount  int64          `gorm:"not null;default:0"`
	OrdersCount     int64          `gorm:"not null;default:0"`
	StorageBytes    int64          `gorm:"not null;default:0"`
	APICallsCount   int64          `gorm:"not null;default:0"`
	Metadata        map[string]any `gorm:"type:jsonb;serializer:json"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
}

// TableName returns the table name for the model
func (UsageHistoryModel) TableName() string {
	return "usage_history"
}

// ToEntity converts the model to a domain entity
func (m *UsageHistoryModel) ToEntity() *billing.UsageHistory {
	return &billing.UsageHistory{
		ID:              m.ID,
		TenantID:        m.TenantID,
		SnapshotDate:    m.SnapshotDate,
		UsersCount:      m.UsersCount,
		ProductsCount:   m.ProductsCount,
		WarehousesCount: m.WarehousesCount,
		CustomersCount:  m.CustomersCount,
		SuppliersCount:  m.SuppliersCount,
		OrdersCount:     m.OrdersCount,
		StorageBytes:    m.StorageBytes,
		APICallsCount:   m.APICallsCount,
		Metadata:        m.Metadata,
		CreatedAt:       m.CreatedAt,
	}
}

// UsageHistoryModelFromEntity creates a model from a domain entity
func UsageHistoryModelFromEntity(e *billing.UsageHistory) *UsageHistoryModel {
	return &UsageHistoryModel{
		ID:              e.ID,
		TenantID:        e.TenantID,
		SnapshotDate:    e.SnapshotDate,
		UsersCount:      e.UsersCount,
		ProductsCount:   e.ProductsCount,
		WarehousesCount: e.WarehousesCount,
		CustomersCount:  e.CustomersCount,
		SuppliersCount:  e.SuppliersCount,
		OrdersCount:     e.OrdersCount,
		StorageBytes:    e.StorageBytes,
		APICallsCount:   e.APICallsCount,
		Metadata:        e.Metadata,
		CreatedAt:       e.CreatedAt,
	}
}

// UsageHistoryRepository implements the billing.UsageHistoryRepository interface
type UsageHistoryRepository struct {
	db *gorm.DB
}

// NewUsageHistoryRepository creates a new usage history repository
func NewUsageHistoryRepository(db *gorm.DB) *UsageHistoryRepository {
	return &UsageHistoryRepository{db: db}
}

// Save persists a new usage history snapshot
func (r *UsageHistoryRepository) Save(ctx context.Context, history *billing.UsageHistory) error {
	model := UsageHistoryModelFromEntity(history)
	return r.db.WithContext(ctx).Create(model).Error
}

// SaveBatch persists multiple usage history snapshots in a single transaction
func (r *UsageHistoryRepository) SaveBatch(ctx context.Context, histories []*billing.UsageHistory) error {
	if len(histories) == 0 {
		return nil
	}

	models := make([]*UsageHistoryModel, len(histories))
	for i, h := range histories {
		models[i] = UsageHistoryModelFromEntity(h)
	}

	return r.db.WithContext(ctx).CreateInBatches(models, 100).Error
}

// Upsert creates or updates a usage history snapshot for a tenant and date
func (r *UsageHistoryRepository) Upsert(ctx context.Context, history *billing.UsageHistory) error {
	model := UsageHistoryModelFromEntity(history)

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "snapshot_date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"users_count",
			"products_count",
			"warehouses_count",
			"customers_count",
			"suppliers_count",
			"orders_count",
			"storage_bytes",
			"api_calls_count",
			"metadata",
		}),
	}).Create(model).Error
}

// FindByID retrieves a usage history snapshot by its ID
func (r *UsageHistoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*billing.UsageHistory, error) {
	var model UsageHistoryModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByTenantAndDate retrieves a specific snapshot for a tenant and date
func (r *UsageHistoryRepository) FindByTenantAndDate(ctx context.Context, tenantID uuid.UUID, date time.Time) (*billing.UsageHistory, error) {
	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	var model UsageHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND snapshot_date = ?", tenantID, normalizedDate).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByTenant retrieves all snapshots for a tenant within a date range
func (r *UsageHistoryRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageHistoryFilter) ([]*billing.UsageHistory, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	if filter.StartDate != nil {
		query = query.Where("snapshot_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("snapshot_date <= ?", *filter.EndDate)
	}

	// Apply pagination
	if filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset < 0 {
			offset = 0
		}
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Order by date descending (most recent first)
	query = query.Order("snapshot_date DESC")

	var models []UsageHistoryModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	histories := make([]*billing.UsageHistory, len(models))
	for i, model := range models {
		histories[i] = model.ToEntity()
	}
	return histories, nil
}

// FindLatestByTenant retrieves the most recent snapshot for a tenant
func (r *UsageHistoryRepository) FindLatestByTenant(ctx context.Context, tenantID uuid.UUID) (*billing.UsageHistory, error) {
	var model UsageHistoryModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("snapshot_date DESC").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// CountByTenant counts snapshots for a tenant within a date range
func (r *UsageHistoryRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID, filter billing.UsageHistoryFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&UsageHistoryModel{}).Where("tenant_id = ?", tenantID)

	if filter.StartDate != nil {
		query = query.Where("snapshot_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("snapshot_date <= ?", *filter.EndDate)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteOlderThan removes snapshots older than the specified date (for data retention)
func (r *UsageHistoryRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("snapshot_date < ?", before).
		Delete(&UsageHistoryModel{})
	return result.RowsAffected, result.Error
}

// DeleteByTenant removes all snapshots for a tenant
func (r *UsageHistoryRepository) DeleteByTenant(ctx context.Context, tenantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Delete(&UsageHistoryModel{}).Error
}

// GetAllTenantIDs retrieves all unique tenant IDs that have usage history
func (r *UsageHistoryRepository) GetAllTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	var tenantIDs []uuid.UUID
	if err := r.db.WithContext(ctx).
		Model(&UsageHistoryModel{}).
		Distinct("tenant_id").
		Pluck("tenant_id", &tenantIDs).Error; err != nil {
		return nil, err
	}
	return tenantIDs, nil
}

// Ensure UsageHistoryRepository implements the interface
var _ billing.UsageHistoryRepository = (*UsageHistoryRepository)(nil)
