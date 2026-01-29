package printing

import (
	"embed"
	"fmt"

	"github.com/erp/backend/internal/domain/printing"
)

//go:embed templates/*.html
var templateFS embed.FS

// DefaultTemplate represents a default print template configuration
type DefaultTemplate struct {
	DocType     printing.DocType
	Name        string
	Description string
	PaperSize   printing.PaperSize
	Orientation printing.Orientation
	Margins     printing.Margins
	FilePath    string // Path within embed.FS
	IsDefault   bool   // Whether this is the default for its doc type
}

// GetDefaultTemplates returns all default template configurations
func GetDefaultTemplates() []DefaultTemplate {
	return []DefaultTemplate{
		// =============================================================================
		// SALES_ORDER templates
		// =============================================================================
		{
			DocType:     printing.DocTypeSalesOrder,
			Name:        "销售订单-A4",
			Description: "标准A4尺寸销售订单模板，包含客户信息、商品明细、金额汇总",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_order_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeSalesOrder,
			Name:        "销售订单-A5横版",
			Description: "紧凑A5横版销售订单模板，适合小批量订单",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_order_a5.html",
			IsDefault:   false,
		},

		// =============================================================================
		// SALES_DELIVERY templates
		// =============================================================================
		{
			DocType:     printing.DocTypeSalesDelivery,
			Name:        "销售发货单-A4",
			Description: "标准A4尺寸销售发货单/送货单模板，包含客户信息、商品明细、签收栏",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_delivery_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeSalesDelivery,
			Name:        "销售发货单-A5横版",
			Description: "紧凑A5横版销售发货单，适合小批量发货",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_delivery_a5.html",
			IsDefault:   false,
		},
		{
			DocType:     printing.DocTypeSalesDelivery,
			Name:        "销售发货单-连续纸",
			Description: "241mm连续纸格式，适用于针式打印机多联打印",
			PaperSize:   printing.PaperSizeContinuous241,
			Orientation: printing.OrientationPortrait,
			Margins: printing.Margins{
				Top:    5,
				Right:  5,
				Bottom: 5,
				Left:   5,
			},
			FilePath:  "templates/sales_delivery_continuous.html",
			IsDefault: false,
		},

		// =============================================================================
		// SALES_RECEIPT templates
		// =============================================================================
		{
			DocType:     printing.DocTypeSalesReceipt,
			Name:        "销售收据-58mm",
			Description: "58mm热敏小票，适用于收银台热敏打印机",
			PaperSize:   printing.PaperSizeReceipt58MM,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.ReceiptMargins(),
			FilePath:    "templates/sales_receipt_58mm.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeSalesReceipt,
			Name:        "销售收据-80mm",
			Description: "80mm热敏小票，内容更详细，适用于大型收银台",
			PaperSize:   printing.PaperSizeReceipt80MM,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.ReceiptMargins(),
			FilePath:    "templates/sales_receipt_80mm.html",
			IsDefault:   false,
		},
		{
			DocType:     printing.DocTypeSalesReceipt,
			Name:        "销售收据-A5横版",
			Description: "A5横版销售收据，正式发票替代品",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_receipt_a5.html",
			IsDefault:   false,
		},

		// =============================================================================
		// SALES_RETURN templates
		// =============================================================================
		{
			DocType:     printing.DocTypeSalesReturn,
			Name:        "销售退货单-A4",
			Description: "标准A4尺寸销售退货单，包含客户信息、退货明细、退款金额",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_return_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeSalesReturn,
			Name:        "销售退货单-A5横版",
			Description: "紧凑A5横版销售退货单，适合小批量退货",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_return_a5.html",
			IsDefault:   false,
		},

		// =============================================================================
		// PURCHASE_ORDER templates
		// =============================================================================
		{
			DocType:     printing.DocTypePurchaseOrder,
			Name:        "采购订单-A4",
			Description: "标准A4尺寸采购订单，包含供应商信息、商品明细、金额汇总",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/purchase_order_a4.html",
			IsDefault:   true,
		},

		// =============================================================================
		// PURCHASE_RECEIVING templates
		// =============================================================================
		{
			DocType:     printing.DocTypePurchaseReceiving,
			Name:        "采购入库单-A4",
			Description: "标准A4尺寸采购入库单，包含供应商信息、商品明细、批次、质检状态",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/purchase_receiving_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypePurchaseReceiving,
			Name:        "采购入库单-连续纸",
			Description: "241mm连续纸格式，适用于仓库针式打印机",
			PaperSize:   printing.PaperSizeContinuous241,
			Orientation: printing.OrientationPortrait,
			Margins: printing.Margins{
				Top:    5,
				Right:  5,
				Bottom: 5,
				Left:   5,
			},
			FilePath:  "templates/purchase_receiving_continuous.html",
			IsDefault: false,
		},

		// =============================================================================
		// PURCHASE_RETURN templates
		// =============================================================================
		{
			DocType:     printing.DocTypePurchaseReturn,
			Name:        "采购退货单-A4",
			Description: "标准A4尺寸采购退货单，包含供应商信息、退货明细、退款金额",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/purchase_return_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypePurchaseReturn,
			Name:        "采购退货单-A5横版",
			Description: "紧凑A5横版采购退货单，适合小批量退货",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/purchase_return_a5.html",
			IsDefault:   false,
		},

		// =============================================================================
		// RECEIPT_VOUCHER templates
		// =============================================================================
		{
			DocType:     printing.DocTypeReceiptVoucher,
			Name:        "收款单-A4",
			Description: "标准A4尺寸收款单，包含客户信息、收款金额、核销明细",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/receipt_voucher_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeReceiptVoucher,
			Name:        "收款单-A5横版",
			Description: "紧凑A5横版收款单，适合快速打印收款凭证",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/receipt_voucher_a5.html",
			IsDefault:   false,
		},

		// =============================================================================
		// PAYMENT_VOUCHER templates
		// =============================================================================
		{
			DocType:     printing.DocTypePaymentVoucher,
			Name:        "付款单-A4",
			Description: "标准A4尺寸付款单，包含供应商信息、付款金额、核销明细",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/payment_voucher_a4.html",
			IsDefault:   true,
		},

		// =============================================================================
		// STOCK_TAKING templates
		// =============================================================================
		{
			DocType:     printing.DocTypeStockTaking,
			Name:        "盘点单-A4",
			Description: "标准A4尺寸盘点单，包含仓库信息、商品明细、盘盈盘亏汇总",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/stock_taking_a4.html",
			IsDefault:   true,
		},
		{
			DocType:     printing.DocTypeStockTaking,
			Name:        "盘点单-A4横版",
			Description: "A4横向盘点单，适合商品较多的盘点场景，信息展示更全面",
			PaperSize:   printing.PaperSizeA4,
			Orientation: printing.OrientationLandscape,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/stock_taking_a4_landscape.html",
			IsDefault:   false,
		},
	}
}

// LoadTemplateContent loads the HTML content for a default template
func LoadTemplateContent(filePath string) (string, error) {
	content, err := templateFS.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}
	return string(content), nil
}

// GetDefaultTemplateByDocTypeAndPaperSize finds a default template configuration
func GetDefaultTemplateByDocTypeAndPaperSize(docType printing.DocType, paperSize printing.PaperSize) *DefaultTemplate {
	templates := GetDefaultTemplates()
	for _, t := range templates {
		if t.DocType == docType && t.PaperSize == paperSize {
			return &t
		}
	}
	return nil
}

// GetDefaultTemplateForDocType finds the default template for a document type
func GetDefaultTemplateForDocType(docType printing.DocType) *DefaultTemplate {
	templates := GetDefaultTemplates()
	for _, t := range templates {
		if t.DocType == docType && t.IsDefault {
			return &t
		}
	}
	return nil
}

// GetTemplatesByDocType returns all templates for a document type
func GetTemplatesByDocType(docType printing.DocType) []DefaultTemplate {
	templates := GetDefaultTemplates()
	var result []DefaultTemplate
	for _, t := range templates {
		if t.DocType == docType {
			result = append(result, t)
		}
	}
	return result
}
