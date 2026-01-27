package printing

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DataProvider is the interface for providing document data for template rendering.
// Each document type should have its own implementation.
type DataProvider interface {
	// GetDocType returns the document type this provider handles
	GetDocType() printing.DocType
	// GetData retrieves the document data for rendering
	// documentID is the ID of the document to render
	GetData(ctx context.Context, tenantID, documentID uuid.UUID) (*DocumentData, error)
}

// DocumentData is the common structure for all document data used in templates.
// It contains both common metadata and document-specific data.
type DocumentData struct {
	// Common metadata
	Meta DocumentMeta `json:"meta"`

	// Company/Tenant information
	Company CompanyInfo `json:"company"`

	// Document-specific data (varies by document type)
	// This will be one of: SalesOrderData, SalesDeliveryData, etc.
	Document any `json:"document"`

	// Formatted/computed fields for convenience
	PrintDate     string `json:"printDate"`
	PrintDateTime string `json:"printDateTime"`
	PrintTime     string `json:"printTime"`
}

// DocumentMeta contains common metadata for all documents
type DocumentMeta struct {
	DocType     printing.DocType `json:"docType"`
	DocTypeName string           `json:"docTypeName"` // Chinese name
	DocNo       string           `json:"docNo"`       // Document number
	Status      string           `json:"status"`
	StatusText  string           `json:"statusText"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	CreatedBy   string           `json:"createdBy"`
	Remark      string           `json:"remark"`
}

// CompanyInfo contains tenant/company information for printing
type CompanyInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Phone   string    `json:"phone"`
	Fax     string    `json:"fax"`
	Email   string    `json:"email"`
	Website string    `json:"website"`
	Logo    string    `json:"logo"` // URL or base64
	TaxID   string    `json:"taxId"`
}

// =============================================================================
// Sales Order Data
// =============================================================================

// SalesOrderData represents sales order data for template rendering
type SalesOrderData struct {
	ID             uuid.UUID            `json:"id"`
	OrderNumber    string               `json:"orderNumber"`
	Customer       CustomerInfo         `json:"customer"`
	Warehouse      *WarehouseInfo       `json:"warehouse"`
	Items          []SalesOrderItemData `json:"items"`
	TotalAmount    decimal.Decimal      `json:"totalAmount"`
	DiscountAmount decimal.Decimal      `json:"discountAmount"`
	PayableAmount  decimal.Decimal      `json:"payableAmount"`
	TotalQuantity  decimal.Decimal      `json:"totalQuantity"`
	ItemCount      int                  `json:"itemCount"`
	Status         string               `json:"status"`
	ConfirmedAt    *time.Time           `json:"confirmedAt"`
	ShippedAt      *time.Time           `json:"shippedAt"`
	CompletedAt    *time.Time           `json:"completedAt"`
	Remark         string               `json:"remark"`

	// Formatted fields
	TotalAmountFormatted    string `json:"totalAmountFormatted"`
	DiscountAmountFormatted string `json:"discountAmountFormatted"`
	PayableAmountFormatted  string `json:"payableAmountFormatted"`
	PayableAmountChinese    string `json:"payableAmountChinese"`
}

// SalesOrderItemData represents a line item in sales order
type SalesOrderItemData struct {
	Index       int             `json:"index"` // 1-based index
	ProductID   uuid.UUID       `json:"productId"`
	ProductCode string          `json:"productCode"`
	ProductName string          `json:"productName"`
	Unit        string          `json:"unit"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Amount      decimal.Decimal `json:"amount"`
	Remark      string          `json:"remark"`

	// Formatted fields
	QuantityFormatted  string `json:"quantityFormatted"`
	UnitPriceFormatted string `json:"unitPriceFormatted"`
	AmountFormatted    string `json:"amountFormatted"`
}

// =============================================================================
// Sales Delivery Data
// =============================================================================

// SalesDeliveryData represents sales delivery/shipping data for template rendering
type SalesDeliveryData struct {
	ID            uuid.UUID               `json:"id"`
	DeliveryNo    string                  `json:"deliveryNo"`
	OrderNo       string                  `json:"orderNo"` // Related sales order
	OrderID       *uuid.UUID              `json:"orderId"`
	Customer      CustomerInfo            `json:"customer"`
	Warehouse     WarehouseInfo           `json:"warehouse"`
	ShippingAddr  *AddressInfo            `json:"shippingAddr"`
	Items         []SalesDeliveryItemData `json:"items"`
	TotalQuantity decimal.Decimal         `json:"totalQuantity"`
	TotalAmount   decimal.Decimal         `json:"totalAmount"`
	ItemCount     int                     `json:"itemCount"`
	ShippedAt     time.Time               `json:"shippedAt"`
	Carrier       string                  `json:"carrier"`
	TrackingNo    string                  `json:"trackingNo"`
	DriverName    string                  `json:"driverName"`
	DriverPhone   string                  `json:"driverPhone"`
	VehiclePlate  string                  `json:"vehiclePlate"`
	Remark        string                  `json:"remark"`

	// Formatted fields
	TotalAmountFormatted string `json:"totalAmountFormatted"`
	ShippedAtFormatted   string `json:"shippedAtFormatted"`

	// Signature areas (for multi-copy delivery notes)
	SignatureAreas []SignatureArea `json:"signatureAreas"`
}

// SalesDeliveryItemData represents a line item in delivery
type SalesDeliveryItemData struct {
	Index       int             `json:"index"`
	ProductID   uuid.UUID       `json:"productId"`
	ProductCode string          `json:"productCode"`
	ProductName string          `json:"productName"`
	Unit        string          `json:"unit"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Amount      decimal.Decimal `json:"amount"`
	BatchNo     string          `json:"batchNo"`

	// Formatted fields
	QuantityFormatted  string `json:"quantityFormatted"`
	UnitPriceFormatted string `json:"unitPriceFormatted"`
	AmountFormatted    string `json:"amountFormatted"`
}

// =============================================================================
// Sales Receipt Data (POS/Retail Receipt)
// =============================================================================

// SalesReceiptData represents a POS/retail receipt for template rendering
type SalesReceiptData struct {
	ID            uuid.UUID              `json:"id"`
	ReceiptNo     string                 `json:"receiptNo"`
	Store         StoreInfo              `json:"store"`
	Cashier       string                 `json:"cashier"`
	Items         []SalesReceiptItemData `json:"items"`
	Subtotal      decimal.Decimal        `json:"subtotal"`
	DiscountTotal decimal.Decimal        `json:"discountTotal"`
	TaxTotal      decimal.Decimal        `json:"taxTotal"`
	GrandTotal    decimal.Decimal        `json:"grandTotal"`
	TotalQuantity decimal.Decimal        `json:"totalQuantity"`
	ItemCount     int                    `json:"itemCount"`
	Payments      []PaymentInfo          `json:"payments"`
	Change        decimal.Decimal        `json:"change"`
	TransactedAt  time.Time              `json:"transactedAt"`
	Remark        string                 `json:"remark"`

	// Formatted fields
	SubtotalFormatted      string `json:"subtotalFormatted"`
	DiscountTotalFormatted string `json:"discountTotalFormatted"`
	TaxTotalFormatted      string `json:"taxTotalFormatted"`
	GrandTotalFormatted    string `json:"grandTotalFormatted"`
	ChangeFormatted        string `json:"changeFormatted"`
	TransactedAtFormatted  string `json:"transactedAtFormatted"`

	// QR code for receipt verification
	QRCode string `json:"qrCode"`
}

// SalesReceiptItemData represents a line item in receipt
type SalesReceiptItemData struct {
	Index       int             `json:"index"`
	ProductCode string          `json:"productCode"`
	ProductName string          `json:"productName"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Discount    decimal.Decimal `json:"discount"`
	Amount      decimal.Decimal `json:"amount"`

	// Formatted (compact for receipt)
	Line string `json:"line"` // e.g., "Apple x2 ¥10.00"
}

// =============================================================================
// Purchase Order Data
// =============================================================================

// PurchaseOrderData represents purchase order data for template rendering
type PurchaseOrderData struct {
	ID               uuid.UUID               `json:"id"`
	OrderNumber      string                  `json:"orderNumber"`
	Supplier         SupplierInfo            `json:"supplier"`
	Warehouse        *WarehouseInfo          `json:"warehouse"`
	Items            []PurchaseOrderItemData `json:"items"`
	TotalAmount      decimal.Decimal         `json:"totalAmount"`
	TotalQuantity    decimal.Decimal         `json:"totalQuantity"`
	ItemCount        int                     `json:"itemCount"`
	Status           string                  `json:"status"`
	ExpectedDelivery *time.Time              `json:"expectedDelivery"`
	ConfirmedAt      *time.Time              `json:"confirmedAt"`
	Remark           string                  `json:"remark"`

	// Formatted fields
	TotalAmountFormatted      string `json:"totalAmountFormatted"`
	TotalAmountChinese        string `json:"totalAmountChinese"`
	ExpectedDeliveryFormatted string `json:"expectedDeliveryFormatted"`
}

// PurchaseOrderItemData represents a line item in purchase order
type PurchaseOrderItemData struct {
	Index       int             `json:"index"`
	ProductID   uuid.UUID       `json:"productId"`
	ProductCode string          `json:"productCode"`
	ProductName string          `json:"productName"`
	Unit        string          `json:"unit"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unitPrice"`
	Amount      decimal.Decimal `json:"amount"`
	Remark      string          `json:"remark"`

	// Formatted fields
	QuantityFormatted  string `json:"quantityFormatted"`
	UnitPriceFormatted string `json:"unitPriceFormatted"`
	AmountFormatted    string `json:"amountFormatted"`
}

// =============================================================================
// Purchase Receiving Data
// =============================================================================

// PurchaseReceivingData represents purchase receiving/GRN data for template rendering
type PurchaseReceivingData struct {
	ID              uuid.UUID                   `json:"id"`
	ReceivingNo     string                      `json:"receivingNo"`
	PurchaseOrderNo string                      `json:"purchaseOrderNo"`
	PurchaseOrderID *uuid.UUID                  `json:"purchaseOrderId"`
	Supplier        SupplierInfo                `json:"supplier"`
	Warehouse       WarehouseInfo               `json:"warehouse"`
	Items           []PurchaseReceivingItemData `json:"items"`
	TotalAmount     decimal.Decimal             `json:"totalAmount"`
	TotalQuantity   decimal.Decimal             `json:"totalQuantity"`
	ItemCount       int                         `json:"itemCount"`
	ReceivedAt      time.Time                   `json:"receivedAt"`
	ReceivedBy      string                      `json:"receivedBy"`
	InspectedBy     string                      `json:"inspectedBy"`
	Remark          string                      `json:"remark"`

	// Formatted fields
	TotalAmountFormatted string `json:"totalAmountFormatted"`
	ReceivedAtFormatted  string `json:"receivedAtFormatted"`

	// Signature areas
	SignatureAreas []SignatureArea `json:"signatureAreas"`
}

// PurchaseReceivingItemData represents a line item in receiving
type PurchaseReceivingItemData struct {
	Index            int             `json:"index"`
	ProductID        uuid.UUID       `json:"productId"`
	ProductCode      string          `json:"productCode"`
	ProductName      string          `json:"productName"`
	Unit             string          `json:"unit"`
	OrderedQuantity  decimal.Decimal `json:"orderedQuantity"`
	ReceivedQuantity decimal.Decimal `json:"receivedQuantity"`
	UnitPrice        decimal.Decimal `json:"unitPrice"`
	Amount           decimal.Decimal `json:"amount"`
	BatchNo          string          `json:"batchNo"`
	ExpiryDate       *time.Time      `json:"expiryDate"`
	QualityStatus    string          `json:"qualityStatus"` // PASS, REJECT, PENDING

	// Formatted fields
	OrderedQuantityFormatted  string `json:"orderedQuantityFormatted"`
	ReceivedQuantityFormatted string `json:"receivedQuantityFormatted"`
	UnitPriceFormatted        string `json:"unitPriceFormatted"`
	AmountFormatted           string `json:"amountFormatted"`
	ExpiryDateFormatted       string `json:"expiryDateFormatted"`
}

// =============================================================================
// Receipt Voucher Data (Finance)
// =============================================================================

// ReceiptVoucherData represents receipt voucher data for template rendering
type ReceiptVoucherData struct {
	ID                uuid.UUID        `json:"id"`
	VoucherNo         string           `json:"voucherNo"`
	Customer          CustomerInfo     `json:"customer"`
	PaymentMethod     string           `json:"paymentMethod"`
	PaymentMethodText string           `json:"paymentMethodText"`
	Amount            decimal.Decimal  `json:"amount"`
	BankAccount       string           `json:"bankAccount"`
	ReferenceNo       string           `json:"referenceNo"` // External payment reference
	Allocations       []AllocationInfo `json:"allocations"` // How this receipt is allocated
	Status            string           `json:"status"`
	ConfirmedAt       *time.Time       `json:"confirmedAt"`
	ReceivedBy        string           `json:"receivedBy"`
	Remark            string           `json:"remark"`

	// Formatted fields
	AmountFormatted string `json:"amountFormatted"`
	AmountChinese   string `json:"amountChinese"`
}

// =============================================================================
// Payment Voucher Data (Finance)
// =============================================================================

// PaymentVoucherData represents payment voucher data for template rendering
type PaymentVoucherData struct {
	ID                uuid.UUID        `json:"id"`
	VoucherNo         string           `json:"voucherNo"`
	Supplier          SupplierInfo     `json:"supplier"`
	PaymentMethod     string           `json:"paymentMethod"`
	PaymentMethodText string           `json:"paymentMethodText"`
	Amount            decimal.Decimal  `json:"amount"`
	BankAccount       string           `json:"bankAccount"`
	ReferenceNo       string           `json:"referenceNo"`
	Allocations       []AllocationInfo `json:"allocations"`
	Status            string           `json:"status"`
	ConfirmedAt       *time.Time       `json:"confirmedAt"`
	PaidBy            string           `json:"paidBy"`
	ApprovedBy        string           `json:"approvedBy"`
	Remark            string           `json:"remark"`

	// Formatted fields
	AmountFormatted string `json:"amountFormatted"`
	AmountChinese   string `json:"amountChinese"`
}

// =============================================================================
// Stock Taking Data
// =============================================================================

// StockTakingData represents stock taking/inventory count data for template rendering
type StockTakingData struct {
	ID          uuid.UUID             `json:"id"`
	TakingNo    string                `json:"takingNo"`
	Warehouse   WarehouseInfo         `json:"warehouse"`
	Items       []StockTakingItemData `json:"items"`
	Status      string                `json:"status"`
	StartedAt   time.Time             `json:"startedAt"`
	CompletedAt *time.Time            `json:"completedAt"`
	CountedBy   string                `json:"countedBy"`
	VerifiedBy  string                `json:"verifiedBy"`
	Remark      string                `json:"remark"`

	// Summary
	TotalItems       int             `json:"totalItems"`
	MatchedItems     int             `json:"matchedItems"`
	SurplusItems     int             `json:"surplusItems"`
	ShortageItems    int             `json:"shortageItems"`
	TotalSurplusQty  decimal.Decimal `json:"totalSurplusQty"`
	TotalShortageQty decimal.Decimal `json:"totalShortageQty"`

	// Formatted fields
	StartedAtFormatted   string `json:"startedAtFormatted"`
	CompletedAtFormatted string `json:"completedAtFormatted"`
}

// StockTakingItemData represents a line item in stock taking
type StockTakingItemData struct {
	Index          int             `json:"index"`
	ProductID      uuid.UUID       `json:"productId"`
	ProductCode    string          `json:"productCode"`
	ProductName    string          `json:"productName"`
	Location       string          `json:"location"` // Shelf/bin location
	Unit           string          `json:"unit"`
	SystemQuantity decimal.Decimal `json:"systemQuantity"`
	ActualQuantity decimal.Decimal `json:"actualQuantity"`
	Variance       decimal.Decimal `json:"variance"`     // Actual - System
	VarianceType   string          `json:"varianceType"` // MATCH, SURPLUS, SHORTAGE
	Remark         string          `json:"remark"`

	// Formatted fields
	SystemQuantityFormatted string `json:"systemQuantityFormatted"`
	ActualQuantityFormatted string `json:"actualQuantityFormatted"`
	VarianceFormatted       string `json:"varianceFormatted"`
}

// =============================================================================
// Common Info Types
// =============================================================================

// CustomerInfo contains customer information for printing
type CustomerInfo struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Contact string    `json:"contact"`
	Phone   string    `json:"phone"`
	Email   string    `json:"email"`
	Address string    `json:"address"`
	TaxID   string    `json:"taxId"`
}

// SupplierInfo contains supplier information for printing
type SupplierInfo struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Contact     string    `json:"contact"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Address     string    `json:"address"`
	BankName    string    `json:"bankName"`
	BankAccount string    `json:"bankAccount"`
	TaxID       string    `json:"taxId"`
}

// WarehouseInfo contains warehouse information for printing
type WarehouseInfo struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Phone   string    `json:"phone"`
	Manager string    `json:"manager"`
}

// StoreInfo contains store/shop information for receipts
type StoreInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Phone   string    `json:"phone"`
	Email   string    `json:"email"`
	TaxID   string    `json:"taxId"`
}

// AddressInfo contains address information
type AddressInfo struct {
	Province string `json:"province"`
	City     string `json:"city"`
	District string `json:"district"`
	Street   string `json:"street"`
	PostCode string `json:"postCode"`
	Full     string `json:"full"` // Full formatted address
}

// PaymentInfo represents a payment in a transaction
type PaymentInfo struct {
	Method      string          `json:"method"`
	MethodText  string          `json:"methodText"`
	Amount      decimal.Decimal `json:"amount"`
	ReferenceNo string          `json:"referenceNo"`

	// Formatted
	AmountFormatted string `json:"amountFormatted"`
}

// AllocationInfo represents allocation of a voucher to a receivable/payable
type AllocationInfo struct {
	DocumentNo   string          `json:"documentNo"`
	DocumentDate time.Time       `json:"documentDate"`
	Amount       decimal.Decimal `json:"amount"`

	// Formatted
	AmountFormatted       string `json:"amountFormatted"`
	DocumentDateFormatted string `json:"documentDateFormatted"`
}

// SignatureArea represents a signature area on a document
type SignatureArea struct {
	Label  string `json:"label"`  // e.g., "送货人", "收货人", "验收人"
	Name   string `json:"name"`   // Pre-filled name if known
	Date   string `json:"date"`   // Pre-filled date if known
	Signed bool   `json:"signed"` // Whether this has been signed
}

// =============================================================================
// Helper Functions for Data Providers
// =============================================================================

// NewDocumentData creates a new DocumentData with common fields initialized
func NewDocumentData(docType printing.DocType, docNo string) *DocumentData {
	now := time.Now()
	return &DocumentData{
		Meta: DocumentMeta{
			DocType:     docType,
			DocTypeName: docType.DisplayName(),
			DocNo:       docNo,
		},
		PrintDate:     now.Format("2006-01-02"),
		PrintDateTime: now.Format("2006-01-02 15:04:05"),
		PrintTime:     now.Format("15:04:05"),
	}
}

// FormatMoneyValue formats a decimal as money string for data providers
func FormatMoneyValue(d decimal.Decimal) string {
	return "¥" + formatDecimalWithCommas(d, 2)
}

// MoneyToChinese converts decimal to Chinese uppercase for data providers
func MoneyToChinese(d decimal.Decimal) string {
	return moneyToChinese(d)
}

// formatDecimalWithCommas formats a decimal with thousand separators
func formatDecimalWithCommas(d decimal.Decimal, precision int) string {
	sign := ""
	if d.IsNegative() {
		sign = "-"
		d = d.Abs()
	}

	parts := splitDecimal(d.StringFixed(int32(precision)))
	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = "." + parts[1]
	}

	// Add thousand separators
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}

	return sign + string(result) + decPart
}

func splitDecimal(s string) []string {
	for i, c := range s {
		if c == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
