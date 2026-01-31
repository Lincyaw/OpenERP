package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormReceiptVoucherRepository implements ReceiptVoucherRepository using GORM
type GormReceiptVoucherRepository struct {
	db *gorm.DB
}

// NewGormReceiptVoucherRepository creates a new GormReceiptVoucherRepository
func NewGormReceiptVoucherRepository(db *gorm.DB) *GormReceiptVoucherRepository {
	return &GormReceiptVoucherRepository{db: db}
}

// FindByID finds a receipt voucher by ID
func (r *GormReceiptVoucherRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.ReceiptVoucher, error) {
	var model models.ReceiptVoucherModel
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByIDForTenant finds a receipt voucher by ID for a specific tenant
func (r *GormReceiptVoucherRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.ReceiptVoucher, error) {
	var model models.ReceiptVoucherModel
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&model, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindByVoucherNumber finds by voucher number for a tenant
func (r *GormReceiptVoucherRepository) FindByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (*finance.ReceiptVoucher, error) {
	var model models.ReceiptVoucherModel
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&model, "voucher_number = ? AND tenant_id = ?", voucherNumber, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// FindAllForTenant finds all receipt vouchers for a tenant with filtering
func (r *GormReceiptVoucherRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	var voucherModels []models.ReceiptVoucherModel
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Apply filters
	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	} else if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.PaymentMethod != nil {
		query = query.Where("payment_method = ?", *filter.PaymentMethod)
	}
	if filter.FromDate != nil {
		query = query.Where("receipt_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("receipt_date <= ?", *filter.ToDate)
	}
	if filter.MinAmount != nil {
		query = query.Where("amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("amount <= ?", *filter.MaxAmount)
	}
	if filter.HasUnallocated != nil && *filter.HasUnallocated {
		query = query.Where("unallocated_amount > 0")
	}

	// Apply pagination
	if filter.PageSize > 0 {
		query = query.Limit(filter.PageSize)
		if filter.Page > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	if err := query.Preload("Allocations").Find(&voucherModels).Error; err != nil {
		return nil, err
	}
	vouchers := make([]finance.ReceiptVoucher, len(voucherModels))
	for i, model := range voucherModels {
		vouchers[i] = *model.ToDomain()
	}
	return vouchers, nil
}

// FindByCustomer finds receipt vouchers for a customer
func (r *GormReceiptVoucherRepository) FindByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	filter.CustomerID = &customerID
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindByStatus finds receipt vouchers by status for a tenant
func (r *GormReceiptVoucherRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus, filter finance.ReceiptVoucherFilter) ([]finance.ReceiptVoucher, error) {
	filter.Status = &status
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindWithUnallocatedAmount finds vouchers that have unallocated amount
func (r *GormReceiptVoucherRepository) FindWithUnallocatedAmount(ctx context.Context, tenantID, customerID uuid.UUID) ([]finance.ReceiptVoucher, error) {
	var voucherModels []models.ReceiptVoucherModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND unallocated_amount > 0", tenantID, customerID).
		Preload("Allocations").
		Find(&voucherModels).Error; err != nil {
		return nil, err
	}
	vouchers := make([]finance.ReceiptVoucher, len(voucherModels))
	for i, model := range voucherModels {
		vouchers[i] = *model.ToDomain()
	}
	return vouchers, nil
}

// Save creates or updates a receipt voucher
func (r *GormReceiptVoucherRepository) Save(ctx context.Context, voucher *finance.ReceiptVoucher) error {
	model := models.ReceiptVoucherModelFromDomain(voucher)
	return r.db.WithContext(ctx).Save(model).Error
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormReceiptVoucherRepository) SaveWithLock(ctx context.Context, voucher *finance.ReceiptVoucher) error {
	// Get current version from database
	var currentVersion int
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Where("id = ?", voucher.ID).
		Select("version").
		Scan(&currentVersion).Error; err != nil {
		return err
	}

	// Check version matches
	if currentVersion != voucher.Version {
		return shared.NewDomainError("CONCURRENT_MODIFICATION", "The voucher has been modified by another user")
	}

	// Increment version
	voucher.Version++

	model := models.ReceiptVoucherModelFromDomain(voucher)
	result := r.db.WithContext(ctx).
		Model(model).
		Where("id = ? AND version = ?", voucher.ID, currentVersion).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.NewDomainError("CONCURRENT_MODIFICATION", "The voucher has been modified by another user")
	}
	return nil
}

// Delete soft deletes a receipt voucher
func (r *GormReceiptVoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.ReceiptVoucherModel{}, "id = ?", id).Error
}

// DeleteForTenant soft deletes a receipt voucher for a tenant
func (r *GormReceiptVoucherRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.ReceiptVoucherModel{}, "id = ? AND tenant_id = ?", id, tenantID).Error
}

// CountForTenant counts receipt vouchers for a tenant with optional filters
func (r *GormReceiptVoucherRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.ReceiptVoucherFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ReceiptVoucherModel{}).Where("tenant_id = ?", tenantID)

	if filter.CustomerID != nil {
		query = query.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts receipt vouchers by status for a tenant
func (r *GormReceiptVoucherRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCustomer counts receipt vouchers for a customer
func (r *GormReceiptVoucherRepository) CountByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumByCustomer calculates total receipt amount for a customer
func (r *GormReceiptVoucherRepository) SumByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// SumForTenant calculates total receipt amount for a tenant
func (r *GormReceiptVoucherRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", tenantID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// SumUnallocatedByCustomer calculates total unallocated amount for a customer
func (r *GormReceiptVoucherRepository) SumUnallocatedByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Select("COALESCE(SUM(unallocated_amount), 0)").
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// ExistsByVoucherNumber checks if a voucher number exists for a tenant
func (r *GormReceiptVoucherRepository) ExistsByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Where("tenant_id = ? AND voucher_number = ?", tenantID, voucherNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateVoucherNumber generates a unique voucher number for a tenant
func (r *GormReceiptVoucherRepository) GenerateVoucherNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var maxNumber string
	if err := r.db.WithContext(ctx).
		Model(&models.ReceiptVoucherModel{}).
		Select("voucher_number").
		Where("tenant_id = ?", tenantID).
		Order("voucher_number DESC").
		Limit(1).
		Scan(&maxNumber).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// Generate next number (simple increment for now)
	// Format: RV-YYYYMMDD-NNNN
	nextSeq := 1
	if maxNumber != "" {
		// Extract sequence from existing number
		var seq int
		if _, err := fmt.Sscanf(maxNumber[len(maxNumber)-4:], "%04d", &seq); err == nil {
			nextSeq = seq + 1
		}
	}

	return fmt.Sprintf("RV-%s-%04d", uuid.New().String()[:8], nextSeq), nil
}

// FindByPaymentReference finds a receipt voucher by payment reference (e.g., gateway order number)
func (r *GormReceiptVoucherRepository) FindByPaymentReference(ctx context.Context, paymentReference string) (*finance.ReceiptVoucher, error) {
	var model models.ReceiptVoucherModel
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&model, "payment_reference = ?", paymentReference).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

// Ensure GormReceiptVoucherRepository implements the interface
var _ finance.ReceiptVoucherRepository = (*GormReceiptVoucherRepository)(nil)
