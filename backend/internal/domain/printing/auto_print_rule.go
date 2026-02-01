package printing

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// AutoPrintRule represents a rule for automatic printing when business events occur.
// Each rule defines what document type and trigger event combination should
// automatically generate a print job.
type AutoPrintRule struct {
	shared.TenantAggregateRoot
	DocumentType DocType      // Type of document this rule applies to
	TriggerEvent TriggerEvent // Business event that triggers printing
	TemplateID   *uuid.UUID   // Optional: specific template to use (nil = use default)
	AutoPrint    bool         // Whether to automatically send to printer
	Copies       int          // Number of copies to print (1-100)
	PrinterName  string       // Optional: specific printer name
	Enabled      bool         // Whether this rule is active
}

// NewAutoPrintRule creates a new AutoPrintRule with validation
func NewAutoPrintRule(
	tenantID uuid.UUID,
	docType DocType,
	triggerEvent TriggerEvent,
	templateID *uuid.UUID,
	autoPrint bool,
	copies int,
	printerName string,
) (*AutoPrintRule, error) {
	// Validate document type
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_DOCUMENT_TYPE", "Invalid document type")
	}

	// Validate trigger event
	if !triggerEvent.IsValid() {
		return nil, shared.NewDomainError("INVALID_TRIGGER_EVENT", "Invalid trigger event")
	}

	// Validate copies range (1-100)
	if copies < 1 || copies > 100 {
		return nil, shared.NewDomainError("INVALID_COPIES", "Copies must be between 1 and 100")
	}

	// Validate trigger event is applicable to document type
	if !isEventApplicableToDocType(triggerEvent, docType) {
		return nil, shared.NewDomainError("INVALID_EVENT_DOC_COMBINATION",
			"Trigger event is not applicable to this document type")
	}

	rule := &AutoPrintRule{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		DocumentType:        docType,
		TriggerEvent:        triggerEvent,
		TemplateID:          templateID,
		AutoPrint:           autoPrint,
		Copies:              copies,
		PrinterName:         printerName,
		Enabled:             true, // New rules are enabled by default
	}

	rule.AddDomainEvent(NewAutoPrintRuleCreatedEvent(rule))
	return rule, nil
}

// isEventApplicableToDocType checks if a trigger event can be used with a document type
func isEventApplicableToDocType(event TriggerEvent, docType DocType) bool {
	applicableTypes := event.ApplicableDocTypes()
	for _, t := range applicableTypes {
		if t == docType {
			return true
		}
	}
	return false
}

// Update updates the rule's configuration
func (r *AutoPrintRule) Update(
	templateID *uuid.UUID,
	autoPrint bool,
	copies int,
	printerName string,
) error {
	// Validate copies range
	if copies < 1 || copies > 100 {
		return shared.NewDomainError("INVALID_COPIES", "Copies must be between 1 and 100")
	}

	r.TemplateID = templateID
	r.AutoPrint = autoPrint
	r.Copies = copies
	r.PrinterName = printerName
	r.IncrementVersion()

	r.AddDomainEvent(NewAutoPrintRuleUpdatedEvent(r))
	return nil
}

// Enable enables the rule
func (r *AutoPrintRule) Enable() {
	if !r.Enabled {
		r.Enabled = true
		r.IncrementVersion()
		r.AddDomainEvent(NewAutoPrintRuleEnabledEvent(r))
	}
}

// Disable disables the rule
func (r *AutoPrintRule) Disable() {
	if r.Enabled {
		r.Enabled = false
		r.IncrementVersion()
		r.AddDomainEvent(NewAutoPrintRuleDisabledEvent(r))
	}
}

// SetTemplateID sets the template ID for this rule
func (r *AutoPrintRule) SetTemplateID(templateID *uuid.UUID) {
	r.TemplateID = templateID
	r.IncrementVersion()
}

// SetCopies sets the number of copies with validation
func (r *AutoPrintRule) SetCopies(copies int) error {
	if copies < 1 || copies > 100 {
		return shared.NewDomainError("INVALID_COPIES", "Copies must be between 1 and 100")
	}
	r.Copies = copies
	r.IncrementVersion()
	return nil
}

// SetPrinterName sets the printer name
func (r *AutoPrintRule) SetPrinterName(printerName string) {
	r.PrinterName = printerName
	r.IncrementVersion()
}

// SetAutoPrint sets whether to automatically send to printer
func (r *AutoPrintRule) SetAutoPrint(autoPrint bool) {
	r.AutoPrint = autoPrint
	r.IncrementVersion()
}

// ShouldTrigger checks if this rule should trigger for the given document type and event
func (r *AutoPrintRule) ShouldTrigger(docType DocType, event TriggerEvent) bool {
	return r.Enabled && r.DocumentType == docType && r.TriggerEvent == event
}

// GetEffectiveTemplateID returns the template ID to use, or nil if default should be used
func (r *AutoPrintRule) GetEffectiveTemplateID() *uuid.UUID {
	return r.TemplateID
}

// Validate performs full validation of the rule
func (r *AutoPrintRule) Validate() error {
	if !r.DocumentType.IsValid() {
		return shared.NewDomainError("INVALID_DOCUMENT_TYPE", "Invalid document type")
	}

	if !r.TriggerEvent.IsValid() {
		return shared.NewDomainError("INVALID_TRIGGER_EVENT", "Invalid trigger event")
	}

	if r.Copies < 1 || r.Copies > 100 {
		return shared.NewDomainError("INVALID_COPIES", "Copies must be between 1 and 100")
	}

	if !isEventApplicableToDocType(r.TriggerEvent, r.DocumentType) {
		return shared.NewDomainError("INVALID_EVENT_DOC_COMBINATION",
			"Trigger event is not applicable to this document type")
	}

	return nil
}

// ReconstructAutoPrintRule reconstructs an AutoPrintRule from persistence
// This is used by the repository to create domain objects from database records
func ReconstructAutoPrintRule(
	id uuid.UUID,
	tenantID uuid.UUID,
	docType DocType,
	triggerEvent TriggerEvent,
	templateID *uuid.UUID,
	autoPrint bool,
	copies int,
	printerName string,
	enabled bool,
	version int,
	base shared.BaseEntity,
) *AutoPrintRule {
	return &AutoPrintRule{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        id,
					CreatedAt: base.CreatedAt,
					UpdatedAt: base.UpdatedAt,
				},
				Version: version,
			},
			TenantID: tenantID,
		},
		DocumentType: docType,
		TriggerEvent: triggerEvent,
		TemplateID:   templateID,
		AutoPrint:    autoPrint,
		Copies:       copies,
		PrinterName:  printerName,
		Enabled:      enabled,
	}
}
