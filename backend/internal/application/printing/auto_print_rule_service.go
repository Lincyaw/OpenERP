package printing

import (
	"context"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// AutoPrintRuleService handles auto print rule operations
type AutoPrintRuleService struct {
	ruleRepo printing.AutoPrintRuleRepository
}

// NewAutoPrintRuleService creates a new AutoPrintRuleService
func NewAutoPrintRuleService(ruleRepo printing.AutoPrintRuleRepository) *AutoPrintRuleService {
	return &AutoPrintRuleService{
		ruleRepo: ruleRepo,
	}
}

// CreateRule creates a new auto print rule
func (s *AutoPrintRuleService) CreateRule(ctx context.Context, tenantID uuid.UUID, req CreateAutoPrintRuleRequest) (*AutoPrintRuleResponse, error) {
	// Parse document type
	docType := printing.DocType(req.DocumentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_DOCUMENT_TYPE", "Invalid document type: "+req.DocumentType)
	}

	// Parse trigger event
	triggerEvent := printing.TriggerEvent(req.TriggerEvent)
	if !triggerEvent.IsValid() {
		return nil, shared.NewDomainError("INVALID_TRIGGER_EVENT", "Invalid trigger event: "+req.TriggerEvent)
	}

	// Check for duplicate rule (same tenant, document type, trigger event)
	exists, err := s.ruleRepo.ExistsByDocTypeAndEvent(ctx, tenantID, docType, triggerEvent)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("DUPLICATE_RULE", "A rule already exists for this document type and trigger event combination")
	}

	// Parse template ID if provided
	var templateID *uuid.UUID
	if req.TemplateID != nil && *req.TemplateID != "" {
		id, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			return nil, shared.NewDomainError("INVALID_TEMPLATE_ID", "Invalid template ID format")
		}
		templateID = &id
	}

	// Set default copies if not provided
	copies := 1
	if req.Copies != nil {
		copies = *req.Copies
	}

	// Create domain entity
	rule, err := printing.NewAutoPrintRule(
		tenantID,
		docType,
		triggerEvent,
		templateID,
		req.AutoPrint,
		copies,
		req.PrinterName,
	)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}

	return toAutoPrintRuleResponse(rule), nil
}

// GetRule retrieves a rule by ID
func (s *AutoPrintRuleService) GetRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*AutoPrintRuleResponse, error) {
	rule, err := s.ruleRepo.FindByIDForTenant(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	return toAutoPrintRuleResponse(rule), nil
}

// ListRules retrieves all rules for a tenant
func (s *AutoPrintRuleService) ListRules(ctx context.Context, tenantID uuid.UUID, req ListAutoPrintRulesRequest) (*ListAutoPrintRulesResponse, error) {
	// Build filter
	filter := shared.Filter{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.OrderBy,
		OrderDir: req.OrderDir,
		Filters:  make(map[string]any),
	}

	if req.DocumentType != "" {
		filter.Filters["document_type"] = req.DocumentType
	}
	if req.TriggerEvent != "" {
		filter.Filters["trigger_event"] = req.TriggerEvent
	}
	if req.Enabled != nil {
		filter.Filters["enabled"] = *req.Enabled
	}

	// Get rules
	rules, err := s.ruleRepo.FindAllForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Get total count
	total, err := s.ruleRepo.CountForTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	// Convert to response
	items := make([]AutoPrintRuleResponse, len(rules))
	for i, rule := range rules {
		items[i] = *toAutoPrintRuleResponse(&rule)
	}

	return &ListAutoPrintRulesResponse{
		Items: items,
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
	}, nil
}

// UpdateRule updates an existing rule
func (s *AutoPrintRuleService) UpdateRule(ctx context.Context, tenantID, ruleID uuid.UUID, req UpdateAutoPrintRuleRequest) (*AutoPrintRuleResponse, error) {
	// Get existing rule
	rule, err := s.ruleRepo.FindByIDForTenant(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}

	// Parse template ID if provided
	var templateID *uuid.UUID
	if req.TemplateID != nil && *req.TemplateID != "" {
		id, err := uuid.Parse(*req.TemplateID)
		if err != nil {
			return nil, shared.NewDomainError("INVALID_TEMPLATE_ID", "Invalid template ID format")
		}
		templateID = &id
	}

	// Set default copies if not provided
	copies := rule.Copies
	if req.Copies != nil {
		copies = *req.Copies
	}

	// Update rule
	if err := rule.Update(templateID, req.AutoPrint, copies, req.PrinterName); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}

	return toAutoPrintRuleResponse(rule), nil
}

// DeleteRule deletes a rule
func (s *AutoPrintRuleService) DeleteRule(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	return s.ruleRepo.DeleteForTenant(ctx, tenantID, ruleID)
}

// EnableRule enables a rule
func (s *AutoPrintRuleService) EnableRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*AutoPrintRuleResponse, error) {
	rule, err := s.ruleRepo.FindByIDForTenant(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}

	rule.Enable()

	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}

	return toAutoPrintRuleResponse(rule), nil
}

// DisableRule disables a rule
func (s *AutoPrintRuleService) DisableRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*AutoPrintRuleResponse, error) {
	rule, err := s.ruleRepo.FindByIDForTenant(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}

	rule.Disable()

	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}

	return toAutoPrintRuleResponse(rule), nil
}

// LookupRule finds a rule by document type and trigger event
func (s *AutoPrintRuleService) LookupRule(ctx context.Context, tenantID uuid.UUID, documentType, triggerEvent string) (*AutoPrintRuleResponse, error) {
	docType := printing.DocType(documentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_DOCUMENT_TYPE", "Invalid document type: "+documentType)
	}

	event := printing.TriggerEvent(triggerEvent)
	if !event.IsValid() {
		return nil, shared.NewDomainError("INVALID_TRIGGER_EVENT", "Invalid trigger event: "+triggerEvent)
	}

	rule, err := s.ruleRepo.FindByDocTypeAndEvent(ctx, tenantID, docType, event)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, shared.ErrNotFound
	}

	return toAutoPrintRuleResponse(rule), nil
}

// GetTriggerEvents returns all trigger events applicable to a document type
func (s *AutoPrintRuleService) GetTriggerEvents(documentType string) ([]TriggerEventResponse, error) {
	docType := printing.DocType(documentType)
	if !docType.IsValid() {
		return nil, shared.NewDomainError("INVALID_DOCUMENT_TYPE", "Invalid document type: "+documentType)
	}

	allEvents := printing.AllTriggerEvents()
	var applicable []TriggerEventResponse

	for _, event := range allEvents {
		applicableTypes := event.ApplicableDocTypes()
		for _, t := range applicableTypes {
			if t == docType {
				applicable = append(applicable, TriggerEventResponse{
					Code:        string(event),
					DisplayName: event.DisplayName(),
				})
				break
			}
		}
	}

	return applicable, nil
}

// toAutoPrintRuleResponse converts domain entity to response DTO
func toAutoPrintRuleResponse(rule *printing.AutoPrintRule) *AutoPrintRuleResponse {
	var templateID *string
	if rule.TemplateID != nil {
		id := rule.TemplateID.String()
		templateID = &id
	}

	return &AutoPrintRuleResponse{
		ID:           rule.ID.String(),
		TenantID:     rule.TenantID.String(),
		DocumentType: string(rule.DocumentType),
		TriggerEvent: string(rule.TriggerEvent),
		TemplateID:   templateID,
		AutoPrint:    rule.AutoPrint,
		Copies:       rule.Copies,
		PrinterName:  rule.PrinterName,
		Enabled:      rule.Enabled,
		CreatedAt:    rule.CreatedAt,
		UpdatedAt:    rule.UpdatedAt,
	}
}
