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

// ReceiptVoucherProvider implements DataProvider for RECEIPT_VOUCHER document type.
// It loads receipt voucher data from the repository for use in print templates.
type ReceiptVoucherProvider struct {
	receiptVoucherRepo finance.ReceiptVoucherRepository
	customerRepo       partner.CustomerRepository
}

// NewReceiptVoucherProvider creates a new ReceiptVoucherProvider.
func NewReceiptVoucherProvider(
	receiptVoucherRepo finance.ReceiptVoucherRepository,
	customerRepo partner.CustomerRepository,
) *ReceiptVoucherProvider {
	return &ReceiptVoucherProvider{
		receiptVoucherRepo: receiptVoucherRepo,
		customerRepo:       customerRepo,
	}
}

// GetDocType returns the document type this provider handles.
func (p *ReceiptVoucherProvider) GetDocType() printing.DocType {
	return printing.DocTypeReceiptVoucher
}

// GetData retrieves receipt voucher data for rendering.
func (p *ReceiptVoucherProvider) GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*infra.DocumentData, error) {
	// Load the receipt voucher
	voucher, err := p.receiptVoucherRepo.FindByIDForTenant(ctx, tenantID, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load receipt voucher: %w", err)
	}

	// Load customer details
	customer, err := p.customerRepo.FindByIDForTenant(ctx, tenantID, voucher.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load customer: %w", err)
	}

	// Build document data
	docData := infra.NewDocumentData(printing.DocTypeReceiptVoucher, voucher.VoucherNumber)
	docData.Meta.Status = string(voucher.Status)
	docData.Meta.StatusText = statusToText(string(voucher.Status))
	docData.Meta.CreatedAt = voucher.CreatedAt
	docData.Meta.UpdatedAt = voucher.UpdatedAt
	docData.Meta.Remark = voucher.Remark
	docData.Meta.CreatedAtFormatted = voucher.CreatedAt.Format("2006-01-02")
	docData.Meta.UpdatedAtFormatted = voucher.UpdatedAt.Format("2006-01-02")

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

	// Build allocations
	allocations := make([]infra.AllocationInfo, len(voucher.Allocations))
	for i, alloc := range voucher.Allocations {
		allocations[i] = infra.AllocationInfo{
			DocumentNo:            alloc.ReceivableNumber,
			DocumentDate:          alloc.AllocatedAt,
			Amount:                alloc.Amount,
			AmountFormatted:       infra.FormatMoneyValue(alloc.Amount),
			DocumentDateFormatted: alloc.AllocatedAt.Format("2006-01-02"),
		}
	}

	// Build receipt voucher data
	receiptVoucherData := infra.ReceiptVoucherData{
		ID:                voucher.ID,
		VoucherNo:         voucher.VoucherNumber,
		Customer:          customerInfo,
		PaymentMethod:     string(voucher.PaymentMethod),
		PaymentMethodText: paymentMethodToText(string(voucher.PaymentMethod)),
		Amount:            voucher.Amount,
		BankAccount:       voucher.PaymentReference,
		ReferenceNo:       voucher.PaymentReference,
		Allocations:       allocations,
		Status:            string(voucher.Status),
		ConfirmedAt:       voucher.ConfirmedAt,
		ReceivedBy:        "", // Would need to load from user repo if needed
		Remark:            voucher.Remark,
		AmountFormatted:   infra.FormatMoneyValue(voucher.Amount),
		AmountChinese:     infra.MoneyToChinese(voucher.Amount),
	}

	docData.Document = receiptVoucherData

	return docData, nil
}

// paymentMethodToText converts payment method code to display text
func paymentMethodToText(method string) string {
	methodMap := map[string]string{
		"CASH":          "现金",
		"BANK_TRANSFER": "银行转账",
		"WECHAT":        "微信支付",
		"ALIPAY":        "支付宝",
		"CHECK":         "支票",
		"BALANCE":       "余额抵扣",
		"OTHER":         "其他",
	}
	if text, ok := methodMap[method]; ok {
		return text
	}
	return method
}
