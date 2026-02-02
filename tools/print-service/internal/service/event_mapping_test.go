package service

import (
	"testing"
)

func TestNewEventMapper(t *testing.T) {
	mapper := NewEventMapper()
	if mapper == nil {
		t.Fatal("Expected non-nil EventMapper")
	}
	if mapper.mappings == nil {
		t.Fatal("Expected non-nil mappings map")
	}
}

func TestEventMapper_GetMapping(t *testing.T) {
	mapper := NewEventMapper()

	tests := []struct {
		name        string
		eventType   string
		wantDocType DocumentType
		wantTrigger TriggerEvent
		wantNil     bool
	}{
		{
			name:        "SalesOrderConfirmed maps to SALES_ORDER CONFIRMED",
			eventType:   "SalesOrderConfirmed",
			wantDocType: DocTypeSalesOrder,
			wantTrigger: TriggerConfirmed,
			wantNil:     false,
		},
		{
			name:        "SalesOrderShipped maps to SALES_ORDER SHIPPED",
			eventType:   "SalesOrderShipped",
			wantDocType: DocTypeSalesOrder,
			wantTrigger: TriggerShipped,
			wantNil:     false,
		},
		{
			name:        "PurchaseOrderReceived maps to PURCHASE_ORDER RECEIVED",
			eventType:   "PurchaseOrderReceived",
			wantDocType: DocTypePurchaseOrder,
			wantTrigger: TriggerReceived,
			wantNil:     false,
		},
		{
			name:        "ReceiptVoucherPaid maps to RECEIPT_VOUCHER PAID",
			eventType:   "ReceiptVoucherPaid",
			wantDocType: DocTypeReceiptVoucher,
			wantTrigger: TriggerPaid,
			wantNil:     false,
		},
		{
			name:        "PaymentVoucherPaid maps to PAYMENT_VOUCHER PAID",
			eventType:   "PaymentVoucherPaid",
			wantDocType: DocTypePaymentVoucher,
			wantTrigger: TriggerPaid,
			wantNil:     false,
		},
		{
			name:      "Unknown event returns nil",
			eventType: "UnknownEvent",
			wantNil:   true,
		},
		{
			name:      "Empty event returns nil",
			eventType: "",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := mapper.GetMapping(tt.eventType)

			if tt.wantNil {
				if mapping != nil {
					t.Errorf("Expected nil mapping for event '%s', got %+v", tt.eventType, mapping)
				}
				return
			}

			if mapping == nil {
				t.Fatalf("Expected non-nil mapping for event '%s'", tt.eventType)
			}

			if mapping.DocumentType != tt.wantDocType {
				t.Errorf("Expected DocumentType '%s', got '%s'", tt.wantDocType, mapping.DocumentType)
			}

			if mapping.TriggerEvent != tt.wantTrigger {
				t.Errorf("Expected TriggerEvent '%s', got '%s'", tt.wantTrigger, mapping.TriggerEvent)
			}
		})
	}
}

func TestEventMapper_IsSupported(t *testing.T) {
	mapper := NewEventMapper()

	tests := []struct {
		eventType string
		want      bool
	}{
		{"SalesOrderConfirmed", true},
		{"SalesOrderShipped", true},
		{"PurchaseOrderReceived", true},
		{"ReceiptVoucherPaid", true},
		{"PaymentVoucherPaid", true},
		{"UnknownEvent", false},
		{"", false},
		{"PrintJobCreated", false}, // Internal event, not a domain event
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := mapper.IsSupported(tt.eventType)
			if got != tt.want {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestEventMapper_SupportedEvents(t *testing.T) {
	mapper := NewEventMapper()
	events := mapper.SupportedEvents()

	// Should have 5 supported events
	if len(events) != 5 {
		t.Errorf("Expected 5 supported events, got %d", len(events))
	}

	// Check that all expected events are present
	expectedEvents := map[string]bool{
		"SalesOrderConfirmed":   false,
		"SalesOrderShipped":     false,
		"PurchaseOrderReceived": false,
		"ReceiptVoucherPaid":    false,
		"PaymentVoucherPaid":    false,
	}

	for _, event := range events {
		if _, ok := expectedEvents[event]; ok {
			expectedEvents[event] = true
		} else {
			t.Errorf("Unexpected event in list: %s", event)
		}
	}

	for event, found := range expectedEvents {
		if !found {
			t.Errorf("Expected event not found: %s", event)
		}
	}
}

func TestDocumentType_Values(t *testing.T) {
	tests := []struct {
		docType  DocumentType
		expected string
	}{
		{DocTypeSalesOrder, "SALES_ORDER"},
		{DocTypePurchaseOrder, "PURCHASE_ORDER"},
		{DocTypeReceiptVoucher, "RECEIPT_VOUCHER"},
		{DocTypePaymentVoucher, "PAYMENT_VOUCHER"},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			if string(tt.docType) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.docType))
			}
		})
	}
}

func TestTriggerEvent_Values(t *testing.T) {
	tests := []struct {
		trigger  TriggerEvent
		expected string
	}{
		{TriggerConfirmed, "CONFIRMED"},
		{TriggerShipped, "SHIPPED"},
		{TriggerReceived, "RECEIVED"},
		{TriggerPaid, "PAID"},
	}

	for _, tt := range tests {
		t.Run(string(tt.trigger), func(t *testing.T) {
			if string(tt.trigger) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.trigger))
			}
		})
	}
}
