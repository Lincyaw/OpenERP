package printing

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAutoPrintRule_Success(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	rule, err := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventConfirmed,
		&templateID,
		true,
		2,
		"Office Printer",
	)

	require.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, tenantID, rule.TenantID)
	assert.Equal(t, DocTypeSalesOrder, rule.DocumentType)
	assert.Equal(t, TriggerEventConfirmed, rule.TriggerEvent)
	assert.Equal(t, &templateID, rule.TemplateID)
	assert.True(t, rule.AutoPrint)
	assert.Equal(t, 2, rule.Copies)
	assert.Equal(t, "Office Printer", rule.PrinterName)
	assert.True(t, rule.Enabled) // New rules are enabled by default

	// Should have created event
	events := rule.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*AutoPrintRuleCreatedEvent)
	assert.True(t, ok)
}

func TestNewAutoPrintRule_WithNilTemplateID(t *testing.T) {
	tenantID := uuid.New()

	rule, err := NewAutoPrintRule(
		tenantID,
		DocTypeSalesDelivery,
		TriggerEventCreated,
		nil, // No specific template
		false,
		1,
		"",
	)

	require.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Nil(t, rule.TemplateID)
}

func TestNewAutoPrintRule_InvalidDocumentType(t *testing.T) {
	tenantID := uuid.New()

	rule, err := NewAutoPrintRule(
		tenantID,
		DocType("INVALID"),
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)

	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "Invalid document type")
}

func TestNewAutoPrintRule_InvalidTriggerEvent(t *testing.T) {
	tenantID := uuid.New()

	rule, err := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEvent("INVALID"),
		nil,
		false,
		1,
		"",
	)

	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "Invalid trigger event")
}

func TestNewAutoPrintRule_InvalidCopies(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name   string
		copies int
	}{
		{"zero copies", 0},
		{"negative copies", -1},
		{"too many copies", 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := NewAutoPrintRule(
				tenantID,
				DocTypeSalesOrder,
				TriggerEventCreated,
				nil,
				false,
				tt.copies,
				"",
			)

			assert.Error(t, err)
			assert.Nil(t, rule)
			assert.Contains(t, err.Error(), "Copies must be between 1 and 100")
		})
	}
}

func TestNewAutoPrintRule_InvalidEventDocCombination(t *testing.T) {
	tenantID := uuid.New()

	// SHIPPED event is not applicable to PURCHASE_ORDER
	rule, err := NewAutoPrintRule(
		tenantID,
		DocTypePurchaseOrder,
		TriggerEventShipped, // Not applicable to purchase orders
		nil,
		false,
		1,
		"",
	)

	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "Trigger event is not applicable to this document type")
}

func TestAutoPrintRule_Update(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	rule.ClearDomainEvents()
	initialVersion := rule.Version

	newTemplateID := uuid.New()
	err := rule.Update(&newTemplateID, true, 5, "New Printer")

	require.NoError(t, err)
	assert.Equal(t, &newTemplateID, rule.TemplateID)
	assert.True(t, rule.AutoPrint)
	assert.Equal(t, 5, rule.Copies)
	assert.Equal(t, "New Printer", rule.PrinterName)
	assert.Equal(t, initialVersion+1, rule.Version)

	// Should have updated event
	events := rule.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*AutoPrintRuleUpdatedEvent)
	assert.True(t, ok)
}

func TestAutoPrintRule_Update_InvalidCopies(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)

	err := rule.Update(nil, false, 0, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Copies must be between 1 and 100")

	err = rule.Update(nil, false, 101, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Copies must be between 1 and 100")
}

func TestAutoPrintRule_Enable(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	rule.Disable()
	rule.ClearDomainEvents()
	initialVersion := rule.Version

	rule.Enable()

	assert.True(t, rule.Enabled)
	assert.Equal(t, initialVersion+1, rule.Version)

	events := rule.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*AutoPrintRuleEnabledEvent)
	assert.True(t, ok)
}

func TestAutoPrintRule_Enable_AlreadyEnabled(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	rule.ClearDomainEvents()
	initialVersion := rule.Version

	// Already enabled, should not change
	rule.Enable()

	assert.True(t, rule.Enabled)
	assert.Equal(t, initialVersion, rule.Version) // Version unchanged
	assert.Empty(t, rule.GetDomainEvents())       // No event
}

func TestAutoPrintRule_Disable(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	rule.ClearDomainEvents()
	initialVersion := rule.Version

	rule.Disable()

	assert.False(t, rule.Enabled)
	assert.Equal(t, initialVersion+1, rule.Version)

	events := rule.GetDomainEvents()
	assert.Len(t, events, 1)
	_, ok := events[0].(*AutoPrintRuleDisabledEvent)
	assert.True(t, ok)
}

func TestAutoPrintRule_Disable_AlreadyDisabled(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	rule.Disable()
	rule.ClearDomainEvents()
	initialVersion := rule.Version

	// Already disabled, should not change
	rule.Disable()

	assert.False(t, rule.Enabled)
	assert.Equal(t, initialVersion, rule.Version) // Version unchanged
	assert.Empty(t, rule.GetDomainEvents())       // No event
}

func TestAutoPrintRule_ShouldTrigger(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventConfirmed,
		nil,
		false,
		1,
		"",
	)

	// Should trigger for matching doc type and event
	assert.True(t, rule.ShouldTrigger(DocTypeSalesOrder, TriggerEventConfirmed))

	// Should not trigger for different doc type
	assert.False(t, rule.ShouldTrigger(DocTypePurchaseOrder, TriggerEventConfirmed))

	// Should not trigger for different event
	assert.False(t, rule.ShouldTrigger(DocTypeSalesOrder, TriggerEventCreated))

	// Should not trigger when disabled
	rule.Disable()
	assert.False(t, rule.ShouldTrigger(DocTypeSalesOrder, TriggerEventConfirmed))
}

func TestAutoPrintRule_SetCopies(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)

	// Valid copies
	err := rule.SetCopies(50)
	assert.NoError(t, err)
	assert.Equal(t, 50, rule.Copies)

	// Boundary values
	err = rule.SetCopies(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, rule.Copies)

	err = rule.SetCopies(100)
	assert.NoError(t, err)
	assert.Equal(t, 100, rule.Copies)

	// Invalid copies
	err = rule.SetCopies(0)
	assert.Error(t, err)

	err = rule.SetCopies(101)
	assert.Error(t, err)
}

func TestAutoPrintRule_Validate(t *testing.T) {
	tenantID := uuid.New()
	rule, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventConfirmed,
		nil,
		false,
		1,
		"",
	)

	// Valid rule should pass validation
	err := rule.Validate()
	assert.NoError(t, err)
}

func TestAutoPrintRule_GetEffectiveTemplateID(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	// With template ID
	rule1, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		&templateID,
		false,
		1,
		"",
	)
	assert.Equal(t, &templateID, rule1.GetEffectiveTemplateID())

	// Without template ID
	rule2, _ := NewAutoPrintRule(
		tenantID,
		DocTypeSalesOrder,
		TriggerEventCreated,
		nil,
		false,
		1,
		"",
	)
	assert.Nil(t, rule2.GetEffectiveTemplateID())
}

func TestReconstructAutoPrintRule(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	templateID := uuid.New()

	rule := ReconstructAutoPrintRule(
		id,
		tenantID,
		DocTypeSalesOrder,
		TriggerEventConfirmed,
		&templateID,
		true,
		3,
		"Test Printer",
		true,
		5,
		shared.BaseEntity{ID: id},
	)

	assert.Equal(t, id, rule.ID)
	assert.Equal(t, tenantID, rule.TenantID)
	assert.Equal(t, DocTypeSalesOrder, rule.DocumentType)
	assert.Equal(t, TriggerEventConfirmed, rule.TriggerEvent)
	assert.Equal(t, &templateID, rule.TemplateID)
	assert.True(t, rule.AutoPrint)
	assert.Equal(t, 3, rule.Copies)
	assert.Equal(t, "Test Printer", rule.PrinterName)
	assert.True(t, rule.Enabled)
	assert.Equal(t, 5, rule.Version)
}

func TestIsEventApplicableToDocType(t *testing.T) {
	tests := []struct {
		name     string
		event    TriggerEvent
		docType  DocType
		expected bool
	}{
		// CREATED applies to all
		{"CREATED + SALES_ORDER", TriggerEventCreated, DocTypeSalesOrder, true},
		{"CREATED + PURCHASE_ORDER", TriggerEventCreated, DocTypePurchaseOrder, true},
		{"CREATED + STOCK_TAKING", TriggerEventCreated, DocTypeStockTaking, true},

		// SHIPPED only for sales
		{"SHIPPED + SALES_ORDER", TriggerEventShipped, DocTypeSalesOrder, true},
		{"SHIPPED + SALES_DELIVERY", TriggerEventShipped, DocTypeSalesDelivery, true},
		{"SHIPPED + PURCHASE_ORDER", TriggerEventShipped, DocTypePurchaseOrder, false},

		// RECEIVED only for purchase
		{"RECEIVED + PURCHASE_ORDER", TriggerEventReceived, DocTypePurchaseOrder, true},
		{"RECEIVED + PURCHASE_RECEIVING", TriggerEventReceived, DocTypePurchaseReceiving, true},
		{"RECEIVED + SALES_ORDER", TriggerEventReceived, DocTypeSalesOrder, false},

		// CONFIRMED for orders and vouchers
		{"CONFIRMED + SALES_ORDER", TriggerEventConfirmed, DocTypeSalesOrder, true},
		{"CONFIRMED + PURCHASE_ORDER", TriggerEventConfirmed, DocTypePurchaseOrder, true},
		{"CONFIRMED + RECEIPT_VOUCHER", TriggerEventConfirmed, DocTypeReceiptVoucher, true},
		{"CONFIRMED + STOCK_TAKING", TriggerEventConfirmed, DocTypeStockTaking, false},

		// COMPLETED for orders and stock taking
		{"COMPLETED + SALES_ORDER", TriggerEventCompleted, DocTypeSalesOrder, true},
		{"COMPLETED + PURCHASE_ORDER", TriggerEventCompleted, DocTypePurchaseOrder, true},
		{"COMPLETED + STOCK_TAKING", TriggerEventCompleted, DocTypeStockTaking, true},
		{"COMPLETED + SALES_RECEIPT", TriggerEventCompleted, DocTypeSalesReceipt, false},

		// PAID for orders and vouchers
		{"PAID + SALES_ORDER", TriggerEventPaid, DocTypeSalesOrder, true},
		{"PAID + RECEIPT_VOUCHER", TriggerEventPaid, DocTypeReceiptVoucher, true},
		{"PAID + STOCK_TAKING", TriggerEventPaid, DocTypeStockTaking, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEventApplicableToDocType(tt.event, tt.docType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
