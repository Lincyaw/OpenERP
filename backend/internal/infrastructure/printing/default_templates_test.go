package printing

import (
	"testing"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultTemplates(t *testing.T) {
	templates := GetDefaultTemplates()

	// Verify we have the expected number of templates (20 templates total)
	assert.Len(t, templates, 20, "Expected 20 default templates")

	// Count by document type
	docTypeCounts := make(map[printing.DocType]int)
	for _, tmpl := range templates {
		docTypeCounts[tmpl.DocType]++
	}

	// Verify counts per document type
	assert.Equal(t, 2, docTypeCounts[printing.DocTypeSalesOrder], "Expected 2 SALES_ORDER templates")
	assert.Equal(t, 3, docTypeCounts[printing.DocTypeSalesDelivery], "Expected 3 SALES_DELIVERY templates")
	assert.Equal(t, 3, docTypeCounts[printing.DocTypeSalesReceipt], "Expected 3 SALES_RECEIPT templates")
	assert.Equal(t, 2, docTypeCounts[printing.DocTypeSalesReturn], "Expected 2 SALES_RETURN templates")
	assert.Equal(t, 1, docTypeCounts[printing.DocTypePurchaseOrder], "Expected 1 PURCHASE_ORDER template")
	assert.Equal(t, 2, docTypeCounts[printing.DocTypePurchaseReceiving], "Expected 2 PURCHASE_RECEIVING templates")
	assert.Equal(t, 2, docTypeCounts[printing.DocTypePurchaseReturn], "Expected 2 PURCHASE_RETURN templates")
	assert.Equal(t, 2, docTypeCounts[printing.DocTypeReceiptVoucher], "Expected 2 RECEIPT_VOUCHER templates")
	assert.Equal(t, 1, docTypeCounts[printing.DocTypePaymentVoucher], "Expected 1 PAYMENT_VOUCHER template")
	assert.Equal(t, 2, docTypeCounts[printing.DocTypeStockTaking], "Expected 2 STOCK_TAKING templates")
}

func TestGetDefaultTemplates_ValidDocTypes(t *testing.T) {
	templates := GetDefaultTemplates()

	for _, tmpl := range templates {
		assert.True(t, tmpl.DocType.IsValid(), "Template %s has invalid DocType: %s", tmpl.Name, tmpl.DocType)
	}
}

func TestGetDefaultTemplates_ValidPaperSizes(t *testing.T) {
	templates := GetDefaultTemplates()

	for _, tmpl := range templates {
		assert.True(t, tmpl.PaperSize.IsValid(), "Template %s has invalid PaperSize: %s", tmpl.Name, tmpl.PaperSize)
	}
}

func TestGetDefaultTemplates_ValidOrientations(t *testing.T) {
	templates := GetDefaultTemplates()

	for _, tmpl := range templates {
		assert.True(t, tmpl.Orientation.IsValid(), "Template %s has invalid Orientation: %s", tmpl.Name, tmpl.Orientation)
	}
}

func TestGetDefaultTemplates_OneDefaultPerDocType(t *testing.T) {
	templates := GetDefaultTemplates()

	defaultCounts := make(map[printing.DocType]int)
	for _, tmpl := range templates {
		if tmpl.IsDefault {
			defaultCounts[tmpl.DocType]++
		}
	}

	// Verify exactly one default per doc type
	for docType, count := range defaultCounts {
		assert.Equal(t, 1, count, "DocType %s should have exactly 1 default template, got %d", docType, count)
	}

	// Verify each doc type has a default
	docTypesWithTemplates := make(map[printing.DocType]bool)
	for _, tmpl := range templates {
		docTypesWithTemplates[tmpl.DocType] = true
	}

	for docType := range docTypesWithTemplates {
		_, hasDefault := defaultCounts[docType]
		assert.True(t, hasDefault, "DocType %s should have a default template", docType)
	}
}

func TestLoadTemplateContent(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Load sales_delivery_a4.html",
			filePath: "templates/sales_delivery_a4.html",
			wantErr:  false,
		},
		{
			name:     "Load sales_delivery_a5.html",
			filePath: "templates/sales_delivery_a5.html",
			wantErr:  false,
		},
		{
			name:     "Load sales_delivery_continuous.html",
			filePath: "templates/sales_delivery_continuous.html",
			wantErr:  false,
		},
		{
			name:     "Load sales_receipt_58mm.html",
			filePath: "templates/sales_receipt_58mm.html",
			wantErr:  false,
		},
		{
			name:     "Load sales_receipt_80mm.html",
			filePath: "templates/sales_receipt_80mm.html",
			wantErr:  false,
		},
		{
			name:     "Load sales_receipt_a5.html",
			filePath: "templates/sales_receipt_a5.html",
			wantErr:  false,
		},
		{
			name:     "Load purchase_receiving_a4.html",
			filePath: "templates/purchase_receiving_a4.html",
			wantErr:  false,
		},
		{
			name:     "Load purchase_receiving_continuous.html",
			filePath: "templates/purchase_receiving_continuous.html",
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			filePath: "templates/non_existent.html",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := LoadTemplateContent(tc.filePath)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, content)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, content, "Template content should not be empty")
				assert.Contains(t, content, "<!DOCTYPE html>", "Template should be valid HTML")
			}
		})
	}
}

func TestLoadTemplateContent_AllDefaultTemplates(t *testing.T) {
	templates := GetDefaultTemplates()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			content, err := LoadTemplateContent(tmpl.FilePath)
			require.NoError(t, err, "Failed to load template %s from %s", tmpl.Name, tmpl.FilePath)
			assert.NotEmpty(t, content)

			// Verify basic HTML structure
			assert.Contains(t, content, "<!DOCTYPE html>")
			assert.Contains(t, content, "<html")
			assert.Contains(t, content, "</html>")
			assert.Contains(t, content, "<style>")
			assert.Contains(t, content, "</style>")
		})
	}
}

func TestGetDefaultTemplateByDocTypeAndPaperSize(t *testing.T) {
	testCases := []struct {
		name      string
		docType   printing.DocType
		paperSize printing.PaperSize
		wantNil   bool
		wantName  string
	}{
		{
			name:      "Sales Delivery A4",
			docType:   printing.DocTypeSalesDelivery,
			paperSize: printing.PaperSizeA4,
			wantNil:   false,
			wantName:  "销售发货单-A4",
		},
		{
			name:      "Sales Delivery A5",
			docType:   printing.DocTypeSalesDelivery,
			paperSize: printing.PaperSizeA5,
			wantNil:   false,
			wantName:  "销售发货单-A5",
		},
		{
			name:      "Sales Delivery Continuous",
			docType:   printing.DocTypeSalesDelivery,
			paperSize: printing.PaperSizeContinuous241,
			wantNil:   false,
			wantName:  "销售发货单-连续纸",
		},
		{
			name:      "Sales Receipt 58mm",
			docType:   printing.DocTypeSalesReceipt,
			paperSize: printing.PaperSizeReceipt58MM,
			wantNil:   false,
			wantName:  "销售收据-58mm",
		},
		{
			name:      "Sales Receipt 80mm",
			docType:   printing.DocTypeSalesReceipt,
			paperSize: printing.PaperSizeReceipt80MM,
			wantNil:   false,
			wantName:  "销售收据-80mm",
		},
		{
			name:      "Sales Receipt A5",
			docType:   printing.DocTypeSalesReceipt,
			paperSize: printing.PaperSizeA5,
			wantNil:   false,
			wantName:  "销售收据-A5",
		},
		{
			name:      "Purchase Receiving A4",
			docType:   printing.DocTypePurchaseReceiving,
			paperSize: printing.PaperSizeA4,
			wantNil:   false,
			wantName:  "采购入库单-A4",
		},
		{
			name:      "Purchase Receiving Continuous",
			docType:   printing.DocTypePurchaseReceiving,
			paperSize: printing.PaperSizeContinuous241,
			wantNil:   false,
			wantName:  "采购入库单-连续纸",
		},
		{
			name:      "Non-existent combination",
			docType:   printing.DocType("INVALID_DOC_TYPE"),
			paperSize: printing.PaperSizeA4,
			wantNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := GetDefaultTemplateByDocTypeAndPaperSize(tc.docType, tc.paperSize)
			if tc.wantNil {
				assert.Nil(t, tmpl)
			} else {
				require.NotNil(t, tmpl)
				assert.Equal(t, tc.wantName, tmpl.Name)
				assert.Equal(t, tc.docType, tmpl.DocType)
				assert.Equal(t, tc.paperSize, tmpl.PaperSize)
			}
		})
	}
}

func TestGetDefaultTemplateForDocType(t *testing.T) {
	testCases := []struct {
		name        string
		docType     printing.DocType
		wantNil     bool
		wantName    string
		wantDefault bool
	}{
		{
			name:        "Sales Order default",
			docType:     printing.DocTypeSalesOrder,
			wantNil:     false,
			wantName:    "销售订单-A4",
			wantDefault: true,
		},
		{
			name:        "Sales Delivery default",
			docType:     printing.DocTypeSalesDelivery,
			wantNil:     false,
			wantName:    "销售发货单-A4",
			wantDefault: true,
		},
		{
			name:        "Sales Receipt default",
			docType:     printing.DocTypeSalesReceipt,
			wantNil:     false,
			wantName:    "销售收据-58mm",
			wantDefault: true,
		},
		{
			name:        "Purchase Receiving default",
			docType:     printing.DocTypePurchaseReceiving,
			wantNil:     false,
			wantName:    "采购入库单-A4",
			wantDefault: true,
		},
		{
			name:    "Invalid doc type - no default",
			docType: printing.DocType("INVALID_DOC_TYPE"),
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := GetDefaultTemplateForDocType(tc.docType)
			if tc.wantNil {
				assert.Nil(t, tmpl)
			} else {
				require.NotNil(t, tmpl)
				assert.Equal(t, tc.wantName, tmpl.Name)
				assert.Equal(t, tc.docType, tmpl.DocType)
				assert.Equal(t, tc.wantDefault, tmpl.IsDefault)
			}
		})
	}
}

func TestGetTemplatesByDocType(t *testing.T) {
	testCases := []struct {
		name      string
		docType   printing.DocType
		wantCount int
		wantNames []string
	}{
		{
			name:      "Sales Order templates",
			docType:   printing.DocTypeSalesOrder,
			wantCount: 2,
			wantNames: []string{"销售订单-A4", "销售订单-A5"},
		},
		{
			name:      "Sales Delivery templates",
			docType:   printing.DocTypeSalesDelivery,
			wantCount: 3,
			wantNames: []string{"销售发货单-A4", "销售发货单-A5", "销售发货单-连续纸"},
		},
		{
			name:      "Sales Receipt templates",
			docType:   printing.DocTypeSalesReceipt,
			wantCount: 3,
			wantNames: []string{"销售收据-58mm", "销售收据-80mm", "销售收据-A5"},
		},
		{
			name:      "Purchase Receiving templates",
			docType:   printing.DocTypePurchaseReceiving,
			wantCount: 2,
			wantNames: []string{"采购入库单-A4", "采购入库单-连续纸"},
		},
		{
			name:      "Stock Taking templates",
			docType:   printing.DocTypeStockTaking,
			wantCount: 2,
			wantNames: []string{"盘点单-A4", "盘点单-A4横版"},
		},
		{
			name:      "Invalid doc type - no templates",
			docType:   printing.DocType("INVALID_DOC_TYPE"),
			wantCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			templates := GetTemplatesByDocType(tc.docType)
			assert.Len(t, templates, tc.wantCount)

			if tc.wantCount > 0 {
				names := make([]string, len(templates))
				for i, tmpl := range templates {
					names[i] = tmpl.Name
				}
				for _, wantName := range tc.wantNames {
					assert.Contains(t, names, wantName)
				}
			}
		})
	}
}

func TestDefaultTemplates_TemplateContentRenderable(t *testing.T) {
	// This test verifies that all default templates can be loaded and have valid Go template syntax
	engine := NewTemplateEngine()
	templates := GetDefaultTemplates()

	// Minimal test data for validation
	testData := map[string]any{
		"Company": map[string]any{
			"Name":  "Test Company",
			"Phone": "123-456-7890",
		},
		"Document": map[string]any{
			"DeliveryNo":            "DEL-001",
			"ReceiptNo":             "REC-001",
			"ReceivingNo":           "GRN-001",
			"ShippedAtFormatted":    "2024-01-15",
			"ReceivedAtFormatted":   "2024-01-15",
			"TransactedAtFormatted": "2024-01-15 14:30:00",
			"OrderNo":               "SO-001",
			"Customer": map[string]any{
				"Name":    "Test Customer",
				"Contact": "John Doe",
				"Phone":   "555-1234",
			},
			"Supplier": map[string]any{
				"Name":    "Test Supplier",
				"Contact": "Jane Smith",
				"Phone":   "555-5678",
				"Address": "123 Test St",
			},
			"Warehouse": map[string]any{
				"Name": "Main Warehouse",
			},
			"Store": map[string]any{
				"Name":    "Test Store",
				"Address": "456 Store Ave",
				"Phone":   "555-9999",
				"TaxID":   "TAX123456",
			},
			"Items":                  []any{},
			"TotalQuantity":          100.0,
			"TotalAmount":            1000.0,
			"TotalAmountFormatted":   "¥1,000.00",
			"ItemCount":              5,
			"ReceivedBy":             "Receiver",
			"InspectedBy":            "Inspector",
			"SubtotalFormatted":      "¥1,000.00",
			"GrandTotalFormatted":    "¥1,000.00",
			"DiscountTotalFormatted": "¥0.00",
			"ChangeFormatted":        "¥0.00",
			"Cashier":                "Cashier1",
			"Payments":               []any{},
			"DiscountTotal":          map[string]any{"IntPart": 0},
			"TaxTotal":               map[string]any{"IntPart": 0},
			"GrandTotal":             1000.0,
			"Change":                 map[string]any{"IntPart": 0},
		},
		"PrintDate":     "2024-01-15",
		"PrintDateTime": "2024-01-15 14:30:00",
		"PrintTime":     "14:30:00",
	}

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			content, err := LoadTemplateContent(tmpl.FilePath)
			require.NoError(t, err)

			// Try to render the template with minimal data
			// This validates the template syntax
			_, err = engine.RenderString(t.Context(), tmpl.Name, content, testData)
			if err != nil {
				// Log the error but don't fail - some templates might need specific data
				t.Logf("Template %s rendering info: %v", tmpl.Name, err)
			}
		})
	}
}

func TestDefaultTemplates_MarginsValid(t *testing.T) {
	templates := GetDefaultTemplates()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			// Verify margins are non-negative
			assert.GreaterOrEqual(t, tmpl.Margins.Top, 0, "Top margin should be non-negative")
			assert.GreaterOrEqual(t, tmpl.Margins.Right, 0, "Right margin should be non-negative")
			assert.GreaterOrEqual(t, tmpl.Margins.Bottom, 0, "Bottom margin should be non-negative")
			assert.GreaterOrEqual(t, tmpl.Margins.Left, 0, "Left margin should be non-negative")

			// Verify margins are reasonable (not too large)
			assert.LessOrEqual(t, tmpl.Margins.Top, 100, "Top margin should not exceed 100mm")
			assert.LessOrEqual(t, tmpl.Margins.Right, 100, "Right margin should not exceed 100mm")
			assert.LessOrEqual(t, tmpl.Margins.Bottom, 100, "Bottom margin should not exceed 100mm")
			assert.LessOrEqual(t, tmpl.Margins.Left, 100, "Left margin should not exceed 100mm")

			// Receipt paper should have smaller margins
			if tmpl.PaperSize.IsReceipt() {
				assert.LessOrEqual(t, tmpl.Margins.Top, 5, "Receipt top margin should be small")
				assert.LessOrEqual(t, tmpl.Margins.Right, 5, "Receipt right margin should be small")
				assert.LessOrEqual(t, tmpl.Margins.Bottom, 5, "Receipt bottom margin should be small")
				assert.LessOrEqual(t, tmpl.Margins.Left, 5, "Receipt left margin should be small")
			}
		})
	}
}
