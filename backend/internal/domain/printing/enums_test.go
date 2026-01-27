package printing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		docType  DocType
		expected bool
	}{
		{"valid SALES_ORDER", DocTypeSalesOrder, true},
		{"valid SALES_DELIVERY", DocTypeSalesDelivery, true},
		{"valid SALES_RECEIPT", DocTypeSalesReceipt, true},
		{"valid SALES_RETURN", DocTypeSalesReturn, true},
		{"valid PURCHASE_ORDER", DocTypePurchaseOrder, true},
		{"valid PURCHASE_RECEIVING", DocTypePurchaseReceiving, true},
		{"valid PURCHASE_RETURN", DocTypePurchaseReturn, true},
		{"valid RECEIPT_VOUCHER", DocTypeReceiptVoucher, true},
		{"valid PAYMENT_VOUCHER", DocTypePaymentVoucher, true},
		{"valid STOCK_TAKING", DocTypeStockTaking, true},
		{"invalid empty", DocType(""), false},
		{"invalid unknown", DocType("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.docType.IsValid())
		})
	}
}

func TestDocType_DisplayName(t *testing.T) {
	tests := []struct {
		docType  DocType
		expected string
	}{
		{DocTypeSalesOrder, "销售订单"},
		{DocTypeSalesDelivery, "销售发货单"},
		{DocTypeSalesReceipt, "销售收据"},
		{DocTypePurchaseOrder, "采购订单"},
		{DocTypeReceiptVoucher, "收款单"},
		{DocTypeStockTaking, "盘点单"},
	}

	for _, tt := range tests {
		t.Run(tt.docType.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.docType.DisplayName())
		})
	}
}

func TestAllDocTypes(t *testing.T) {
	docTypes := AllDocTypes()
	assert.Len(t, docTypes, 10)
	for _, dt := range docTypes {
		assert.True(t, dt.IsValid())
	}
}

func TestPaperSize_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		paperSize PaperSize
		expected  bool
	}{
		{"valid A4", PaperSizeA4, true},
		{"valid A5", PaperSizeA5, true},
		{"valid RECEIPT_58MM", PaperSizeReceipt58MM, true},
		{"valid RECEIPT_80MM", PaperSizeReceipt80MM, true},
		{"valid CONTINUOUS_241", PaperSizeContinuous241, true},
		{"invalid empty", PaperSize(""), false},
		{"invalid unknown", PaperSize("LETTER"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.paperSize.IsValid())
		})
	}
}

func TestPaperSize_Dimensions(t *testing.T) {
	tests := []struct {
		paperSize      PaperSize
		expectedWidth  int
		expectedHeight int
	}{
		{PaperSizeA4, 210, 297},
		{PaperSizeA5, 148, 210},
		{PaperSizeReceipt58MM, 58, 0},
		{PaperSizeReceipt80MM, 80, 0},
		{PaperSizeContinuous241, 241, 0},
	}

	for _, tt := range tests {
		t.Run(tt.paperSize.String(), func(t *testing.T) {
			w, h := tt.paperSize.Dimensions()
			assert.Equal(t, tt.expectedWidth, w)
			assert.Equal(t, tt.expectedHeight, h)
		})
	}
}

func TestPaperSize_IsReceipt(t *testing.T) {
	assert.True(t, PaperSizeReceipt58MM.IsReceipt())
	assert.True(t, PaperSizeReceipt80MM.IsReceipt())
	assert.False(t, PaperSizeA4.IsReceipt())
	assert.False(t, PaperSizeA5.IsReceipt())
	assert.False(t, PaperSizeContinuous241.IsReceipt())
}

func TestPaperSize_IsContinuous(t *testing.T) {
	assert.True(t, PaperSizeContinuous241.IsContinuous())
	assert.False(t, PaperSizeA4.IsContinuous())
	assert.False(t, PaperSizeReceipt58MM.IsContinuous())
}

func TestAllPaperSizes(t *testing.T) {
	paperSizes := AllPaperSizes()
	assert.Len(t, paperSizes, 5)
	for _, ps := range paperSizes {
		assert.True(t, ps.IsValid())
	}
}

func TestOrientation_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		orientation Orientation
		expected    bool
	}{
		{"valid PORTRAIT", OrientationPortrait, true},
		{"valid LANDSCAPE", OrientationLandscape, true},
		{"invalid empty", Orientation(""), false},
		{"invalid unknown", Orientation("ROTATED"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.orientation.IsValid())
		})
	}
}

func TestTemplateStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   TemplateStatus
		expected bool
	}{
		{"valid ACTIVE", TemplateStatusActive, true},
		{"valid INACTIVE", TemplateStatusInactive, true},
		{"invalid empty", TemplateStatus(""), false},
		{"invalid unknown", TemplateStatus("DRAFT"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected bool
	}{
		{"valid PENDING", JobStatusPending, true},
		{"valid RENDERING", JobStatusRendering, true},
		{"valid COMPLETED", JobStatusCompleted, true},
		{"valid FAILED", JobStatusFailed, true},
		{"invalid empty", JobStatus(""), false},
		{"invalid unknown", JobStatus("QUEUED"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestJobStatus_IsTerminal(t *testing.T) {
	assert.False(t, JobStatusPending.IsTerminal())
	assert.False(t, JobStatusRendering.IsTerminal())
	assert.True(t, JobStatusCompleted.IsTerminal())
	assert.True(t, JobStatusFailed.IsTerminal())
}

func TestJobStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     JobStatus
		to       JobStatus
		expected bool
	}{
		// From PENDING
		{"PENDING -> RENDERING", JobStatusPending, JobStatusRendering, true},
		{"PENDING -> FAILED", JobStatusPending, JobStatusFailed, true},
		{"PENDING -> COMPLETED", JobStatusPending, JobStatusCompleted, false},
		{"PENDING -> PENDING", JobStatusPending, JobStatusPending, false},

		// From RENDERING
		{"RENDERING -> COMPLETED", JobStatusRendering, JobStatusCompleted, true},
		{"RENDERING -> FAILED", JobStatusRendering, JobStatusFailed, true},
		{"RENDERING -> PENDING", JobStatusRendering, JobStatusPending, false},
		{"RENDERING -> RENDERING", JobStatusRendering, JobStatusRendering, false},

		// From COMPLETED (terminal)
		{"COMPLETED -> PENDING", JobStatusCompleted, JobStatusPending, false},
		{"COMPLETED -> RENDERING", JobStatusCompleted, JobStatusRendering, false},
		{"COMPLETED -> FAILED", JobStatusCompleted, JobStatusFailed, false},

		// From FAILED (terminal)
		{"FAILED -> PENDING", JobStatusFailed, JobStatusPending, false},
		{"FAILED -> RENDERING", JobStatusFailed, JobStatusRendering, false},
		{"FAILED -> COMPLETED", JobStatusFailed, JobStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.from.CanTransitionTo(tt.to))
		})
	}
}
