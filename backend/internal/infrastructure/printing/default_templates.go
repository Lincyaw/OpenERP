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
		// SALES_DELIVERY templates
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
			Name:        "销售发货单-A5",
			Description: "紧凑A5尺寸销售发货单，适合小批量发货",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationPortrait,
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

		// SALES_RECEIPT templates
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
			Name:        "销售收据-A5",
			Description: "A5尺寸销售收据，正式发票替代品",
			PaperSize:   printing.PaperSizeA5,
			Orientation: printing.OrientationPortrait,
			Margins:     printing.DefaultMargins(),
			FilePath:    "templates/sales_receipt_a5.html",
			IsDefault:   false,
		},

		// PURCHASE_RECEIVING templates
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
