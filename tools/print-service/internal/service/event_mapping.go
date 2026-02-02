// Package service provides the main print service implementation.
package service

// DocumentType represents the type of document for printing.
type DocumentType string

// Document types supported by the print service.
const (
	DocTypeSalesOrder     DocumentType = "SALES_ORDER"
	DocTypePurchaseOrder  DocumentType = "PURCHASE_ORDER"
	DocTypeReceiptVoucher DocumentType = "RECEIPT_VOUCHER"
	DocTypePaymentVoucher DocumentType = "PAYMENT_VOUCHER"
)

// TriggerEvent represents the event that triggers printing.
type TriggerEvent string

// Trigger events supported by the print service.
const (
	TriggerConfirmed TriggerEvent = "CONFIRMED"
	TriggerShipped   TriggerEvent = "SHIPPED"
	TriggerReceived  TriggerEvent = "RECEIVED"
	TriggerPaid      TriggerEvent = "PAID"
)

// EventMapping represents the mapping from a domain event to document type and trigger.
type EventMapping struct {
	DocumentType DocumentType
	TriggerEvent TriggerEvent
}

// EventMapper maps domain events to document types and trigger events.
type EventMapper struct {
	mappings map[string]EventMapping
}

// NewEventMapper creates a new event mapper with predefined mappings.
func NewEventMapper() *EventMapper {
	return &EventMapper{
		mappings: map[string]EventMapping{
			"SalesOrderConfirmed":   {DocumentType: DocTypeSalesOrder, TriggerEvent: TriggerConfirmed},
			"SalesOrderShipped":     {DocumentType: DocTypeSalesOrder, TriggerEvent: TriggerShipped},
			"PurchaseOrderReceived": {DocumentType: DocTypePurchaseOrder, TriggerEvent: TriggerReceived},
			"ReceiptVoucherPaid":    {DocumentType: DocTypeReceiptVoucher, TriggerEvent: TriggerPaid},
			"PaymentVoucherPaid":    {DocumentType: DocTypePaymentVoucher, TriggerEvent: TriggerPaid},
		},
	}
}

// GetMapping returns the event mapping for a given event type.
// Returns nil if no mapping exists.
func (m *EventMapper) GetMapping(eventType string) *EventMapping {
	if mapping, ok := m.mappings[eventType]; ok {
		return &mapping
	}
	return nil
}

// IsSupported returns true if the event type is supported.
func (m *EventMapper) IsSupported(eventType string) bool {
	_, ok := m.mappings[eventType]
	return ok
}

// SupportedEvents returns a list of all supported event types.
func (m *EventMapper) SupportedEvents() []string {
	events := make([]string, 0, len(m.mappings))
	for event := range m.mappings {
		events = append(events, event)
	}
	return events
}
