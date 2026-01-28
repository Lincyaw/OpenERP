// Package parser provides semantic type inference for API parameters and responses.
package parser

import (
	"regexp"
	"strings"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// InferenceResult holds the result of semantic type inference.
type InferenceResult struct {
	// SemanticType is the inferred semantic type.
	SemanticType circuit.SemanticType

	// Confidence is the confidence level (0.0-1.0).
	// 1.0 = explicit configuration
	// 0.9 = high confidence from field name + context
	// 0.8 = good confidence from field name pattern
	// 0.7 = moderate confidence from format/type
	// 0.5 = low confidence guess
	Confidence float64

	// Source describes how the type was inferred.
	Source string

	// Reason provides human-readable explanation.
	Reason string
}

// InferenceContext provides context for semantic type inference.
type InferenceContext struct {
	// EndpointPath is the API endpoint path (e.g., "/customers/{id}").
	EndpointPath string

	// EndpointMethod is the HTTP method (GET, POST, PUT, DELETE).
	EndpointMethod string

	// OperationID is the OpenAPI operation ID if available.
	OperationID string

	// Tags are the endpoint tags (e.g., ["customers", "crud"]).
	Tags []string

	// IsInput indicates if this is an input parameter (vs output field).
	IsInput bool

	// ParentField is the parent field name for nested fields.
	ParentField string

	// FieldPath is the full path to the field (e.g., "data.customer.id").
	FieldPath string
}

// SemanticInferenceEngine infers semantic types from field names and context.
type SemanticInferenceEngine struct {
	// rules is the ordered list of inference rules.
	rules []InferenceRule

	// overrides maps field patterns to explicit semantic types.
	overrides map[string]circuit.SemanticType

	// minConfidence is the minimum confidence to accept an inference.
	minConfidence float64
}

// InferenceRule defines a rule for inferring semantic types.
type InferenceRule interface {
	// Name returns the rule name for debugging.
	Name() string

	// Match attempts to infer a semantic type.
	// Returns nil if the rule doesn't match.
	Match(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult
}

// NewSemanticInferenceEngine creates a new inference engine with default rules.
func NewSemanticInferenceEngine() *SemanticInferenceEngine {
	engine := &SemanticInferenceEngine{
		overrides:     make(map[string]circuit.SemanticType),
		minConfidence: 0.7, // Default: only accept high confidence inferences
	}

	// Add default rules in priority order
	engine.rules = DefaultInferenceRules()

	return engine
}

// SetMinConfidence sets the minimum confidence threshold for accepting inferences.
func (e *SemanticInferenceEngine) SetMinConfidence(confidence float64) {
	e.minConfidence = confidence
}

// AddOverride adds an explicit semantic type override for a field pattern.
// Pattern can be exact match or use wildcards:
// - "customer_id" matches exactly "customer_id"
// - "*.customer_id" matches any field ending with "customer_id"
// - "customers.*" matches any field in customers context
func (e *SemanticInferenceEngine) AddOverride(pattern string, semanticType circuit.SemanticType) {
	e.overrides[pattern] = semanticType
}

// AddOverrides adds multiple overrides at once.
func (e *SemanticInferenceEngine) AddOverrides(overrides map[string]circuit.SemanticType) {
	for pattern, semanticType := range overrides {
		e.overrides[pattern] = semanticType
	}
}

// Infer attempts to infer the semantic type for a field.
func (e *SemanticInferenceEngine) Infer(fieldName string, dataType string, format string, ctx *InferenceContext) *InferenceResult {
	// Check explicit overrides first
	if result := e.checkOverrides(fieldName, ctx); result != nil {
		return result
	}

	// Try each rule in order
	for _, rule := range e.rules {
		if result := rule.Match(fieldName, dataType, format, ctx); result != nil {
			if result.Confidence >= e.minConfidence {
				return result
			}
		}
	}

	// Return unknown type with low confidence
	return &InferenceResult{
		SemanticType: circuit.UnknownSemanticType,
		Confidence:   0.0,
		Source:       "none",
		Reason:       "no matching inference rule",
	}
}

// InferWithAllResults returns all possible inferences, not just the best one.
// Useful for debugging and dry-run mode.
func (e *SemanticInferenceEngine) InferWithAllResults(fieldName string, dataType string, format string, ctx *InferenceContext) []*InferenceResult {
	var results []*InferenceResult

	// Check explicit overrides first
	if result := e.checkOverrides(fieldName, ctx); result != nil {
		results = append(results, result)
	}

	// Try each rule
	for _, rule := range e.rules {
		if result := rule.Match(fieldName, dataType, format, ctx); result != nil {
			results = append(results, result)
		}
	}

	return results
}

// checkOverrides checks if there's an explicit override for this field.
func (e *SemanticInferenceEngine) checkOverrides(fieldName string, ctx *InferenceContext) *InferenceResult {
	// Build possible patterns to check
	patterns := []string{
		fieldName, // Exact match
	}

	// Add context-based patterns
	if ctx != nil {
		if ctx.FieldPath != "" {
			patterns = append(patterns, ctx.FieldPath)
		}
		if ctx.EndpointPath != "" {
			// Pattern like "/customers/*:id"
			patterns = append(patterns, ctx.EndpointPath+":"+fieldName)
		}
		if len(ctx.Tags) > 0 {
			// Pattern like "customers:id"
			for _, tag := range ctx.Tags {
				patterns = append(patterns, tag+":"+fieldName)
			}
		}
	}

	// Check each pattern
	for _, pattern := range patterns {
		if semanticType, ok := e.overrides[pattern]; ok {
			return &InferenceResult{
				SemanticType: semanticType,
				Confidence:   1.0,
				Source:       "override",
				Reason:       "explicit override for pattern: " + pattern,
			}
		}
	}

	// Check wildcard patterns
	for pattern, semanticType := range e.overrides {
		if matchWildcard(pattern, fieldName) {
			return &InferenceResult{
				SemanticType: semanticType,
				Confidence:   1.0,
				Source:       "override",
				Reason:       "wildcard override: " + pattern,
			}
		}
	}

	return nil
}

// matchWildcard checks if a pattern with wildcards matches a field name.
func matchWildcard(pattern, fieldName string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == fieldName
	}

	// Convert wildcard pattern to regex
	regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
	matched, _ := regexp.MatchString(regexPattern, fieldName)
	return matched
}

// InferEndpointPins infers semantic types for all pins in an endpoint.
func (e *SemanticInferenceEngine) InferEndpointPins(endpoint *EndpointUnit) *circuit.PinRegistry {
	registry := circuit.NewPinRegistry()

	ctx := &InferenceContext{
		EndpointPath:   endpoint.Path,
		EndpointMethod: endpoint.Method,
		OperationID:    endpoint.OperationID,
		Tags:           endpoint.Tags,
	}

	// Infer input pins
	ctx.IsInput = true
	for i := range endpoint.InputPins {
		pin := &endpoint.InputPins[i]
		fieldCtx := *ctx
		fieldCtx.FieldPath = pin.Name

		result := e.Infer(pin.Name, string(pin.Type), pin.Format, &fieldCtx)
		pin.SemanticType = result.SemanticType

		// Create circuit pin
		circuitPin := circuit.NewInputPin(endpoint.Path, endpoint.Method, pin.Name, result.SemanticType)
		circuitPin.DataType = string(pin.Type)
		circuitPin.Format = pin.Format
		circuitPin.Required = pin.Required
		circuitPin.Description = pin.Description
		circuitPin.InferenceConfidence = result.Confidence
		circuitPin.InferenceSource = result.Source
		circuitPin.Location.ParameterLocation = string(pin.Location)

		registry.RegisterPin(circuitPin)
	}

	// Infer output pins
	ctx.IsInput = false
	for i := range endpoint.OutputPins {
		pin := &endpoint.OutputPins[i]
		fieldCtx := *ctx
		fieldCtx.FieldPath = pin.JSONPath

		result := e.Infer(pin.Name, string(pin.Type), pin.Format, &fieldCtx)
		pin.SemanticType = result.SemanticType

		// Create circuit pin
		circuitPin := circuit.NewOutputPin(endpoint.Path, endpoint.Method, pin.Name, pin.JSONPath, result.SemanticType)
		circuitPin.DataType = string(pin.Type)
		circuitPin.Format = pin.Format
		circuitPin.Description = pin.Description
		circuitPin.InferenceConfidence = result.Confidence
		circuitPin.InferenceSource = result.Source

		registry.RegisterPin(circuitPin)
	}

	return registry
}

// InferSpec infers semantic types for all endpoints in an OpenAPI spec.
func (e *SemanticInferenceEngine) InferSpec(spec *OpenAPISpec) *circuit.PinRegistry {
	registry := circuit.NewPinRegistry()

	for i := range spec.Endpoints {
		endpointRegistry := e.InferEndpointPins(&spec.Endpoints[i])
		for _, pin := range endpointRegistry.AllPins {
			registry.RegisterPin(pin)
		}
	}

	return registry
}

// InferenceStats holds statistics about inference results.
type InferenceStats struct {
	TotalFields      int
	InferredFields   int
	HighConfidence   int
	MediumConfidence int
	LowConfidence    int
	UnknownFields    int
	BySource         map[string]int
	ByCategory       map[string]int
	AccuracyEstimate float64
}

// CalculateStats calculates statistics for a pin registry.
func CalculateStats(registry *circuit.PinRegistry) *InferenceStats {
	stats := &InferenceStats{
		BySource:   make(map[string]int),
		ByCategory: make(map[string]int),
	}

	for _, pin := range registry.AllPins {
		stats.TotalFields++

		if pin.SemanticType == circuit.UnknownSemanticType || pin.SemanticType == "" {
			stats.UnknownFields++
			continue
		}

		stats.InferredFields++
		stats.BySource[pin.InferenceSource]++
		stats.ByCategory[pin.SemanticType.Category()]++

		switch {
		case pin.InferenceConfidence >= 0.9:
			stats.HighConfidence++
		case pin.InferenceConfidence >= 0.7:
			stats.MediumConfidence++
		default:
			stats.LowConfidence++
		}
	}

	// Estimate accuracy based on confidence distribution
	if stats.TotalFields > 0 {
		// High confidence fields are likely correct
		// Medium confidence has ~80% accuracy
		// Low confidence has ~50% accuracy
		accurateCount := float64(stats.HighConfidence) +
			float64(stats.MediumConfidence)*0.8 +
			float64(stats.LowConfidence)*0.5
		stats.AccuracyEstimate = accurateCount / float64(stats.InferredFields) * 100
	}

	return stats
}
