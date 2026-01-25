package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// GormPaymentVoucherRepository implements PaymentVoucherRepository using GORM
type GormPaymentVoucherRepository struct {
	db *gorm.DB
}

// NewGormPaymentVoucherRepository creates a new GormPaymentVoucherRepository
func NewGormPaymentVoucherRepository(db *gorm.DB) *GormPaymentVoucherRepository {
	return &GormPaymentVoucherRepository{db: db}
}

// FindByID finds a payment voucher by ID
func (r *GormPaymentVoucherRepository) FindByID(ctx context.Context, id uuid.UUID) (*finance.PaymentVoucher, error) {
	var voucher finance.PaymentVoucher
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&voucher, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &voucher, nil
}

// FindByIDForTenant finds a payment voucher by ID for a specific tenant
func (r *GormPaymentVoucherRepository) FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*finance.PaymentVoucher, error) {
	var voucher finance.PaymentVoucher
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&voucher, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &voucher, nil
}

// FindByVoucherNumber finds by voucher number for a tenant
func (r *GormPaymentVoucherRepository) FindByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (*finance.PaymentVoucher, error) {
	var voucher finance.PaymentVoucher
	if err := r.db.WithContext(ctx).
		Preload("Allocations").
		First(&voucher, "voucher_number = ? AND tenant_id = ?", voucherNumber, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &voucher, nil
}

// FindAllForTenant finds all payment vouchers for a tenant with filtering
func (r *GormPaymentVoucherRepository) FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.PaymentVoucherFilter) ([]finance.PaymentVoucher, error) {
	var vouchers []finance.PaymentVoucher
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Apply filters
	if filter.SupplierID != nil {
		query = query.Where("supplier_id = ?", *filter.SupplierID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.PaymentMethod != nil {
		query = query.Where("payment_method = ?", *filter.PaymentMethod)
	}
	if filter.FromDate != nil {
		query = query.Where("payment_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("payment_date <= ?", *filter.ToDate)
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

	if err := query.Preload("Allocations").Find(&vouchers).Error; err != nil {
		return nil, err
	}
	return vouchers, nil
}

// FindBySupplier finds payment vouchers for a supplier
func (r *GormPaymentVoucherRepository) FindBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter finance.PaymentVoucherFilter) ([]finance.PaymentVoucher, error) {
	filter.SupplierID = &supplierID
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindByStatus finds payment vouchers by status for a tenant
func (r *GormPaymentVoucherRepository) FindByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus, filter finance.PaymentVoucherFilter) ([]finance.PaymentVoucher, error) {
	filter.Status = &status
	return r.FindAllForTenant(ctx, tenantID, filter)
}

// FindWithUnallocatedAmount finds vouchers that have unallocated amount
func (r *GormPaymentVoucherRepository) FindWithUnallocatedAmount(ctx context.Context, tenantID, supplierID uuid.UUID) ([]finance.PaymentVoucher, error) {
	var vouchers []finance.PaymentVoucher
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND supplier_id = ? AND unallocated_amount > 0", tenantID, supplierID).
		Preload("Allocations").
		Find(&vouchers).Error; err != nil {
		return nil, err
	}
	return vouchers, nil
}

// Save creates or updates a payment voucher
func (r *GormPaymentVoucherRepository) Save(ctx context.Context, voucher *finance.PaymentVoucher) error {
	return r.db.WithContext(ctx).Save(voucher).Error
}

// SaveWithLock saves with optimistic locking (version check)
func (r *GormPaymentVoucherRepository) SaveWithLock(ctx context.Context, voucher *finance.PaymentVoucher) error {
	result := r.db.WithContext(ctx).
		Model(voucher).
		Where("id = ? AND version = ?", voucher.ID, voucher.Version-1).
		Updates(voucher)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock error: version mismatch")
	}
	return nil
}

// Delete soft deletes a payment voucher
func (r *GormPaymentVoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&finance.PaymentVoucher{}, "id = ?", id).Error
}

// DeleteForTenant soft deletes a payment voucher for a tenant
func (r *GormPaymentVoucherRepository) DeleteForTenant(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&finance.PaymentVoucher{}, "id = ? AND tenant_id = ?", id, tenantID).Error
}

// CountForTenant counts payment vouchers for a tenant with optional filters
func (r *GormPaymentVoucherRepository) CountForTenant(ctx context.Context, tenantID uuid.UUID, filter finance.PaymentVoucherFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&finance.PaymentVoucher{}).Where("tenant_id = ?", tenantID)

	if filter.SupplierID != nil {
		query = query.Where("supplier_id = ?", *filter.SupplierID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus counts payment vouchers by status for a tenant
func (r *GormPaymentVoucherRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status finance.VoucherStatus) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Where("tenant_id = ? AND status = ?", tenantID, status).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBySupplier counts payment vouchers for a supplier
func (r *GormPaymentVoucherRepository) CountBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SumBySupplier calculates total payment amount for a supplier
func (r *GormPaymentVoucherRepository) SumBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// SumForTenant calculates total payment amount for a tenant
func (r *GormPaymentVoucherRepository) SumForTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("tenant_id = ?", tenantID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// SumUnallocatedBySupplier calculates total unallocated amount for a supplier
func (r *GormPaymentVoucherRepository) SumUnallocatedBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (decimal.Decimal, error) {
	var sum decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Select("COALESCE(SUM(unallocated_amount), 0)").
		Where("tenant_id = ? AND supplier_id = ?", tenantID, supplierID).
		Scan(&sum).Error; err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// ExistsByVoucherNumber checks if a voucher number exists for a tenant
func (r *GormPaymentVoucherRepository) ExistsByVoucherNumber(ctx context.Context, tenantID uuid.UUID, voucherNumber string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Where("tenant_id = ? AND voucher_number = ?", tenantID, voucherNumber).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GenerateVoucherNumber generates a unique voucher number for a tenant
func (r *GormPaymentVoucherRepository) GenerateVoucherNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var maxNumber string
	if err := r.db.WithContext(ctx).
		Model(&finance.PaymentVoucher{}).
		Select("voucher_number").
		Where("tenant_id = ?", tenantID).
		Order("voucher_number DESC").
		Limit(1).
		Scan(&maxNumber).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// Generate next number (simple increment for now)
	// Format: PV-XXXXXXXX-NNNN
	nextSeq := 1
	if maxNumber != "" {
		// Extract sequence from existing number
		var seq int
		if _, err := fmt.Sscanf(maxNumber[len(maxNumber)-4:], "%04d", &seq); err == nil {
			nextSeq = seq + 1
		}
	}

	return fmt.Sprintf("PV-%s-%04d", uuid.New().String()[:8], nextSeq), nil
}

// Ensure GormPaymentVoucherRepository implements the interface
var _ finance.PaymentVoucherRepository = (*GormPaymentVoucherRepository)(nil)
