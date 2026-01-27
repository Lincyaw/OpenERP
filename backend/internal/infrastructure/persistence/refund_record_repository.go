package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormRefundRecordRepository implements RefundRecordRepository using GORM
type GormRefundRecordRepository struct {
	db *gorm.DB
}

// NewGormRefundRecordRepository creates a new GormRefundRecordRepository
func NewGormRefundRecordRepository(db *gorm.DB) *GormRefundRecordRepository {
	return &GormRefundRecordRepository{db: db}
}

// FindByID finds a refund record by ID
func (r *GormRefundRecordRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.RefundRecord, error) {
	var model models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a refund record by ID for a specific tenant
func (r *GormRefundRecordRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.RefundRecord, error) {
	var model models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		First(&model, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByRefundNumber finds by refund number for a tenant
func (r *GormRefundRecordRepository) FindByRefundNumber(ctx context.Context, tenantID uuid.UUID, refundNumber string) (*finance.RefundRecord, error) {
	var model models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		First(&model, "refund_number = ? AND tenant_id = ?", refundNumber, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByGatewayRefundID finds by gateway refund ID
func (r *GormRefundRecordRepository) FindByGatewayRefundID(ctx context.Context, gatewayType finance.PaymentGatewayType, gatewayRefundID string) (*finance.RefundRecord, error) {
	var model models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		First(&model, "gateway_type = ? AND gateway_refund_id = ?", gatewayType, gatewayRefundID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindBySource finds by source document (e.g., sales return, credit memo)
func (r *GormRefundRecordRepository) FindBySource(ctx context.Context, tenantID uuid.UUID, sourceType finance.RefundSourceType, sourceID uuid.UUID) ([]finance.RefundRecord, error) {
	var recordModels []models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND source_type = ? AND source_id = ?", tenantID, sourceType, sourceID).
		Find(&recordModels).Error; err != nil {
		return nil, err
	}
	records := make([]finance.RefundRecord, len(recordModels))
	for i, model := range recordModels {
		records[i] = *model.ToDomain()
	}
	return records, nil
}

// FindAllForTenant finds all refund records for a tenant with filtering
func (r *GormRefundRecordRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.RefundRecordFilter) ([]finance.RefundRecord, error) {
	var recordModels []models.RefundRecordModel
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Apply filters
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.SourceType != nil {
		query = query.Where("source_type = ?", *filter.SourceType)
	}
	if filter.SourceID != nil {
		query = query.Where("source_id = ?", *filter.SourceID)
	}
	if filter.GatewayType != nil {
		query = query.Where("gateway_type = ?", *filter.GatewayType)
	}
	if filter.FromDate != nil {
		query = query.Where("requested_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("requested_at <= ?", *filter.ToDate)
	}
	if filter.MinAmount != nil {
		query = query.Where("refund_amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("refund_amount <= ?", *filter.MaxAmount)
	}

	// Order by requested_at descending (newest first)
	query = query.Order("requested_at DESC")

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		if filter.Page > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	if err := query.Find(&recordModels).Error; err != nil {
		return nil, err
	}
	records := make([]finance.RefundRecord, len(recordModels))
	for i, model := range recordModels {
		records[i] = *model.ToDomain()
	}
	return records, nil
}

// FindByCustomer finds refund records for a customer
func (r *GormRefundRecordRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter finance.RefundRecordFilter) ([]finance.RefundRecord, error) {
	filter.CustomerID = &customerID
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindByStatus finds refund records by status for a tenant
func (r *GormRefundRecordRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.RefundRecordStatus, filter finance.RefundRecordFilter) ([]finance.RefundRecord, error) {
	filter.Status = &status
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindPending finds all pending refund records for a tenant
func (r *GormRefundRecordRepository) FindPending(ctx context.Context, tenantID uuid.UUID) ([]finance.RefundRecord, error) {
	var recordModels []models.RefundRecordModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status IN ?", tenantID, []finance.RefundRecordStatus{
			finance.RefundRecordStatusPending,
			finance.RefundRecordStatusProcessing,
		}).
		Order("requested_at ASC").
		Find(&recordModels).Error; err != nil {
		return nil, err
	}
	records := make([]finance.RefundRecord, len(recordModels))
	for i, model := range recordModels {
		records[i] = *model.ToDomain()
	}
	return records, nil
}

// Save creates or updates a refund record
func (r *GormRefundRecordRepository) Save(ctx context.Context, record *finance.RefundRecord) error {
	model := models.RefundRecordModelFromDomain(record)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormRefundRecordRepository) SaveWithLock(ctx context.Context, record *finance.RefundRecord) error {
	model := models.RefundRecordModelFromDomain(record)
	result := r.db.WithContext(ctx).
		Model(model).
		Where("id = ? AND version = ?", record.ID, record.Version-1).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock error: version mismatch")
	}
	return nil
}

// Delete soft deletes a refund record
func (r *GormRefundRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.RefundRecordModel{}, "id = ?", id).Error
}

// DeleteForTenant soft deletes a refund record for a tenant
func (r *GormRefundRecordRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.RefundRecordModel{}, "id = ? AND tenant_id = ?", id, tenantID).Error
}

// CountForTenant counts refund records for a tenant with optional filters
func (r *GormRefundRecordRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.RefundRecordFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.RefundRecordModel{}).Where("tenant_id = ?", tenantID)

	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.SourceType != nil {
		query = query.Where("source_type = ?", *filter.SourceType)
	}
	if filter.GatewayType != nil {
		query = query.Where("gateway_type = ?", *filter.GatewayType)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts refund records by status for a tenant
func (r *GormRefundRecordRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.RefundRecordStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCustomer counts refund records for a customer
func (r *GormRefundRecordRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumByCustomer calculates total refund amount for a customer
func (r *GormRefundRecordRepository) SumByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.NullDecimal
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Select("COALESCE(SUM(refund_amount), 0)").
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	if sum.Valid {
		return sum.Decimal, nil
	}
	return decimal.Zero, nil
}

// SumForTenant calculates total refund amount for a tenant
func (r *GormRefundRecordRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.NullDecimal
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ?", tenantID).
		Select("COALESCE(SUM(refund_amount), 0)").
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	if sum.Valid {
		return sum.Decimal, nil
	}
	return decimal.Zero, nil
}

// SumSuccessfulByCustomer calculates total successful refund amount for a customer
func (r *GormRefundRecordRepository) SumSuccessfulByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.NullDecimal
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND customer_id = ? AND status = ?", tenantID, customerID, finance.RefundRecordStatusSuccess).
		Select("COALESCE(SUM(actual_refund_amount), 0)").
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	if sum.Valid {
		return sum.Decimal, nil
	}
	return decimal.Zero, nil
}

// ExistsByRefundNumber checks if a refund number exists for a tenant
func (r *GormRefundRecordRepository) ExistsByRefundNumber(ctx context.Context, tenantID uuid.UUID, refundNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND refund_number = ?", tenantID, refundNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByGatewayRefundID checks if a refund exists for the given gateway refund ID
func (r *GormRefundRecordRepository) ExistsByGatewayRefundID(ctx context.Context, gatewayType finance.PaymentGatewayType, gatewayRefundID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("gateway_type = ? AND gateway_refund_id = ?", gatewayType, gatewayRefundID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateRefundNumber generates a unique refund number for a tenant
func (r *GormRefundRecordRepository) GenerateRefundNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Get the current date prefix
	now := time.Now()
	prefix := fmt.Sprintf("RF-%d-%02d-", now.Year(), now.Month())

	// Find the maximum sequence number for this tenant and prefix
	var maxNumber string
	if err := r.db.WithContext(ctx).
		Model(&models.RefundRecordModel{}).
		Where("tenant_id = ? AND refund_number LIKE ?", tenantID, prefix+"%").
		Order("refund_number DESC").
		Limit(1).
		Pluck("refund_number", &maxNumber).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// Calculate the next sequence number
	nextSeq := 1
	if maxNumber != "" {
		// Extract the sequence number from the max number
		var seq int
		if _, err := fmt.Sscanf(maxNumber, prefix+"%d", &seq); err == nil {
			nextSeq = seq + 1
		}
	}

	return fmt.Sprintf("%s%04d", prefix, nextSeq), nil
}

// Ensure GormRefundRecordRepository implements finance.RefundRecordRepository
var _ finance.RefundRecordRepository = (*GormRefundRecordRepository)(nil)
