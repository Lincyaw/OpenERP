package providers

import (
	"context"
	"fmt"

	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/printing"
	infra "github.com/erp/backend/internal/infrastructure/printing"
	"github.com/google/uuid"
)

// PaymentVoucherProvider implements DataProvider for PAYMENT_VOUCHER document type.
// It loads payment voucher data from the repository for use in print templates.
type PaymentVoucherProvider struct {
	paymentVoucherRepo finance.PaymentVoucherRepository
	supplierRepo       partner.SupplierRepository
}

// NewPaymentVoucherProvider creates a new PaymentVoucherProvider.
func NewPaymentVoucherProvider(
	paymentVoucherRepo finance.PaymentVoucherRepository,
	supplierRepo partner.SupplierRepository,
) *PaymentVoucherProvider {
	return &PaymentVoucherProvider{
		paymentVoucherRepo: paymentVoucherRepo,
		supplierRepo:       supplierRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *PaymentVoucherProvider) GetDocType() printing.DocType {
	return printing.DocTypePaymentVoucher
}

// GetData retrieves payment voucher data for rendering.
func (p *PaymentVoucherProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the payment voucher
	voucher, err := p.paymentVoucherRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load payment voucher: %w", err)
	}

	// Load supplier details
	supplier, err := p.supplierRepo.FindByIDForTenant(ctx, tenantID, voucher.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("failed to load supplier: %w", err)
	}

	// Build document data
	docData := infra.NewDocumentData(printing.DocTypePaymentVoucher, voucher.VoucherNumber)
	docData.Meta.Status = string(voucher.Status)
	docData.Meta.StatusText = statusToText(string(voucher.Status))
	docData.Meta.CreatedAt = voucher.CreatedAt
	docData.Meta.UpdatedAt = voucher.UpdatedAt
	docData.Meta.Remark = voucher.Remark
	docData.Meta.CreatedAtFormatted = voucher.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = voucher.UpdatedAt.Format("2006-01-02")

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

	// Build allocations
	allocations := make([]infra.AllocationInfo, len(voucher.Allocations))
	for i, alloc := range voucher.Allocations {
		allocations[i] = infra.AllocationInfo{
			DocumentNo:            alloc.PayableNumber,
			DocumentDate:          alloc.AllocatedAt,
			Amount:                alloc.Amount,
			AmountFormatted:       infra.FormatMoneyValue(alloc.Amount),
			DocumentDateFormatted: alloc.AllocatedAt.Format("2006-01-02"),
		}
	}

	// Build payment voucher data
	paymentVoucherData := infra.PaymentVoucherData{
		ID:                voucher.ID,
		VoucherNo:         voucher.VoucherNumber,
		Supplier:          supplierInfo,
		PaymentMethod:     string(voucher.PaymentMethod),
		PaymentMethodText: paymentMethodToText(string(voucher.PaymentMethod)),
		Amount:            voucher.Amount,
		BankAccount:       voucher.PaymentReference,
		ReferenceNo:       voucher.PaymentReference,
		Allocations:       allocations,
		Status:            string(voucher.Status),
		ConfirmedAt:       voucher.ConfirmedAt,
		PaidBy:            "", // Would need to load from user repo if needed
		ApprovedBy:        "", // Would need to load from user repo if needed
		Remark:            voucher.Remark,
		AmountFormatted:   infra.FormatMoneyValue(voucher.Amount),
		AmountChinese:     infra.MoneyToChinese(voucher.Amount),
	}

	docData.Document = paymentVoucherData

	return docData, nil
}
