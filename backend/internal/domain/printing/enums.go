package printing

// DocType represents the type of business document that can be printed
type DocType string

const (
	// Sales documents
	DocTypeSalesOrder    DocType = "SALES_ORDER"    // 销售订单
	DocTypeSalesDelivery DocType = "SALES_DELIVERY" // 销售发货单/送货单
	DocTypeSalesReceipt  DocType = "SALES_RECEIPT"  // 销售收据/小票
	DocTypeSalesReturn   DocType = "SALES_RETURN"   // 销售退货单

	// Purchase documents
	DocTypePurchaseOrder     DocType = "PURCHASE_ORDER"     // 采购订单
	DocTypePurchaseReceiving DocType = "PURCHASE_RECEIVING" // 采购入库单
	DocTypePurchaseReturn    DocType = "PURCHASE_RETURN"    // 采购退货单

	// Financial documents
	DocTypeReceiptVoucher DocType = "RECEIPT_VOUCHER" // 收款单
	DocTypePaymentVoucher DocType = "PAYMENT_VOUCHER" // 付款单

	// Inventory documents
	DocTypeStockTaking DocType = "STOCK_TAKING" // 盘点单
)

// IsValid checks if the DocType is a valid value
func (d DocType) IsValid() bool {
	switch d {
	case DocTypeSalesOrder, DocTypeSalesDelivery, DocTypeSalesReceipt, DocTypeSalesReturn,
		DocTypePurchaseOrder, DocTypePurchaseReceiving, DocTypePurchaseReturn,
		DocTypeReceiptVoucher, DocTypePaymentVoucher, DocTypeStockTaking:
		return true
	}
	return false
}

// String returns the string representation of DocType
func (d DocType) String() string {
	return string(d)
}

// DisplayName returns the Chinese display name for DocType
func (d DocType) DisplayName() string {
	switch d {
	case DocTypeSalesOrder:
		return "销售订单"
	case DocTypeSalesDelivery:
		return "销售发货单"
	case DocTypeSalesReceipt:
		return "销售收据"
	case DocTypeSalesReturn:
		return "销售退货单"
	case DocTypePurchaseOrder:
		return "采购订单"
	case DocTypePurchaseReceiving:
		return "采购入库单"
	case DocTypePurchaseReturn:
		return "采购退货单"
	case DocTypeReceiptVoucher:
		return "收款单"
	case DocTypePaymentVoucher:
		return "付款单"
	case DocTypeStockTaking:
		return "盘点单"
	default:
		return string(d)
	}
}

// AllDocTypes returns all valid DocType values
func AllDocTypes() []DocType {
	return []DocType{
		DocTypeSalesOrder, DocTypeSalesDelivery, DocTypeSalesReceipt, DocTypeSalesReturn,
		DocTypePurchaseOrder, DocTypePurchaseReceiving, DocTypePurchaseReturn,
		DocTypeReceiptVoucher, DocTypePaymentVoucher, DocTypeStockTaking,
	}
}

// PaperSize represents the paper size for printing
type PaperSize string

const (
	PaperSizeA4            PaperSize = "A4"             // 210mm x 297mm
	PaperSizeA5            PaperSize = "A5"             // 148mm x 210mm
	PaperSizeReceipt58MM   PaperSize = "RECEIPT_58MM"   // 58mm thermal receipt
	PaperSizeReceipt80MM   PaperSize = "RECEIPT_80MM"   // 80mm thermal receipt
	PaperSizeContinuous241 PaperSize = "CONTINUOUS_241" // 241mm continuous paper (dot matrix)
)

// IsValid checks if the PaperSize is a valid value
func (p PaperSize) IsValid() bool {
	switch p {
	case PaperSizeA4, PaperSizeA5, PaperSizeReceipt58MM, PaperSizeReceipt80MM, PaperSizeContinuous241:
		return true
	}
	return false
}

// String returns the string representation of PaperSize
func (p PaperSize) String() string {
	return string(p)
}

// Dimensions returns the paper dimensions in millimeters (width, height)
// For receipt paper, width is the paper width and height is variable
func (p PaperSize) Dimensions() (width, height int) {
	switch p {
	case PaperSizeA4:
		return 210, 297
	case PaperSizeA5:
		return 148, 210
	case PaperSizeReceipt58MM:
		return 58, 0 // Height is variable for receipt paper
	case PaperSizeReceipt80MM:
		return 80, 0 // Height is variable for receipt paper
	case PaperSizeContinuous241:
		return 241, 0 // Height is variable for continuous paper
	default:
		return 210, 297 // Default to A4
	}
}

// IsReceipt returns true if this is a receipt paper size
func (p PaperSize) IsReceipt() bool {
	return p == PaperSizeReceipt58MM || p == PaperSizeReceipt80MM
}

// IsContinuous returns true if this is continuous feed paper
func (p PaperSize) IsContinuous() bool {
	return p == PaperSizeContinuous241
}

// AllPaperSizes returns all valid PaperSize values
func AllPaperSizes() []PaperSize {
	return []PaperSize{
		PaperSizeA4, PaperSizeA5, PaperSizeReceipt58MM, PaperSizeReceipt80MM, PaperSizeContinuous241,
	}
}

// Orientation represents the page orientation for printing
type Orientation string

const (
	OrientationPortrait  Orientation = "PORTRAIT"  // 纵向
	OrientationLandscape Orientation = "LANDSCAPE" // 横向
)

// IsValid checks if the Orientation is a valid value
func (o Orientation) IsValid() bool {
	switch o {
	case OrientationPortrait, OrientationLandscape:
		return true
	}
	return false
}

// String returns the string representation of Orientation
func (o Orientation) String() string {
	return string(o)
}

// TemplateStatus represents the status of a print template
type TemplateStatus string

const (
	TemplateStatusActive   TemplateStatus = "ACTIVE"   // 可用
	TemplateStatusInactive TemplateStatus = "INACTIVE" // 禁用
)

// IsValid checks if the TemplateStatus is a valid value
func (s TemplateStatus) IsValid() bool {
	switch s {
	case TemplateStatusActive, TemplateStatusInactive:
		return true
	}
	return false
}

// String returns the string representation of TemplateStatus
func (s TemplateStatus) String() string {
	return string(s)
}

// JobStatus represents the status of a print job
type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"   // 等待处理
	JobStatusRendering JobStatus = "RENDERING" // 正在渲染
	JobStatusCompleted JobStatus = "COMPLETED" // 完成
	JobStatusFailed    JobStatus = "FAILED"    // 失败
)

// IsValid checks if the JobStatus is a valid value
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusPending, JobStatusRendering, JobStatusCompleted, JobStatusFailed:
		return true
	}
	return false
}

// String returns the string representation of JobStatus
func (s JobStatus) String() string {
	return string(s)
}

// IsTerminal returns true if this is a terminal status (no further transitions)
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed
}

// CanTransitionTo checks if the status can transition to the target status
func (s JobStatus) CanTransitionTo(target JobStatus) bool {
	switch s {
	case JobStatusPending:
		return target == JobStatusRendering || target == JobStatusFailed
	case JobStatusRendering:
		return target == JobStatusCompleted || target == JobStatusFailed
	case JobStatusCompleted, JobStatusFailed:
		return false // Terminal states
	}
	return false
}
