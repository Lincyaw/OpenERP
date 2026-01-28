// Package parser provides OpenAPI specification parsing for the load generator.
// It extracts endpoint definitions, parameters, and response schemas from Swagger 2.0/OpenAPI 3.0 specs.
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"gopkg.in/yaml.v3"
)

// Errors returned by the parser package.
var (
	// ErrInvalidSpec is returned when the OpenAPI spec is invalid.
	ErrInvalidSpec = errors.New("parser: invalid OpenAPI specification")
	// ErrSpecNotFound is returned when the spec file is not found.
	ErrSpecNotFound = errors.New("parser: specification file not found")
	// ErrUnsupportedVersion is returned when the spec version is not supported.
	ErrUnsupportedVersion = errors.New("parser: unsupported OpenAPI version")
)

// ParameterLocation defines where a parameter is located.
type ParameterLocation string

const (
	// ParameterLocationPath indicates a path parameter (e.g., {id} in /users/{id}).
	ParameterLocationPath ParameterLocation = "path"
	// ParameterLocationQuery indicates a query parameter (e.g., ?page=1).
	ParameterLocationQuery ParameterLocation = "query"
	// ParameterLocationHeader indicates a header parameter.
	ParameterLocationHeader ParameterLocation = "header"
	// ParameterLocationBody indicates a request body parameter.
	ParameterLocationBody ParameterLocation = "body"
	// ParameterLocationFormData indicates a form data parameter.
	ParameterLocationFormData ParameterLocation = "formData"
)

// ParameterType defines the data type of a parameter.
type ParameterType string

const (
	ParameterTypeString  ParameterType = "string"
	ParameterTypeInteger ParameterType = "integer"
	ParameterTypeNumber  ParameterType = "number"
	ParameterTypeBoolean ParameterType = "boolean"
	ParameterTypeArray   ParameterType = "array"
	ParameterTypeObject  ParameterType = "object"
	ParameterTypeFile    ParameterType = "file"
)

// InputPin represents an input parameter for an endpoint.
type InputPin struct {
	// Name is the parameter name.
	Name string `json:"name" yaml:"name"`

	// Location is where the parameter is found (path, query, header, body).
	Location ParameterLocation `json:"location" yaml:"location"`

	// Type is the data type of the parameter.
	Type ParameterType `json:"type" yaml:"type"`

	// Format is the format hint (e.g., "uuid", "date-time", "email").
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Required indicates whether the parameter is required.
	Required bool `json:"required" yaml:"required"`

	// Description is the parameter description from the spec.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// SemanticType is the inferred or configured semantic type.
	SemanticType circuit.SemanticType `json:"semanticType,omitempty" yaml:"semanticType,omitempty"`

	// Default is the default value if specified.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`

	// Enum lists allowed values if specified.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Items describes array item type for array parameters.
	Items *SchemaInfo `json:"items,omitempty" yaml:"items,omitempty"`

	// Schema describes the full schema for body parameters.
	Schema *SchemaInfo `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Example is an example value if provided.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`

	// Minimum is the minimum value for numeric types.
	Minimum *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`

	// Maximum is the maximum value for numeric types.
	Maximum *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`

	// MinLength is the minimum length for string types.
	MinLength *int `json:"minLength,omitempty" yaml:"minLength,omitempty"`

	// MaxLength is the maximum length for string types.
	MaxLength *int `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`

	// Pattern is a regex pattern for string validation.
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// OutputPin represents an output field from an endpoint response.
type OutputPin struct {
	// Name is the field name (dot notation for nested fields, e.g., "data.id").
	Name string `json:"name" yaml:"name"`

	// JSONPath is the JSONPath to extract this value (e.g., "$.data.id").
	JSONPath string `json:"jsonPath" yaml:"jsonPath"`

	// Type is the data type of the field.
	Type ParameterType `json:"type" yaml:"type"`

	// Format is the format hint (e.g., "uuid", "date-time").
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Description is the field description from the spec.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// SemanticType is the inferred or configured semantic type.
	SemanticType circuit.SemanticType `json:"semanticType,omitempty" yaml:"semanticType,omitempty"`

	// IsArray indicates whether this field is an array.
	IsArray bool `json:"isArray,omitempty" yaml:"isArray,omitempty"`

	// Example is an example value if provided.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
}

// SchemaInfo holds simplified schema information.
type SchemaInfo struct {
	// Type is the schema type.
	Type ParameterType `json:"type" yaml:"type"`

	// Format is the format hint.
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Ref is the original $ref if this was resolved from a reference.
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// Properties lists properties for object types.
	Properties map[string]*SchemaInfo `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Items describes array item schema.
	Items *SchemaInfo `json:"items,omitempty" yaml:"items,omitempty"`

	// Required lists required property names for objects.
	Required []string `json:"required,omitempty" yaml:"required,omitempty"`

	// Description is the schema description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Enum lists allowed values.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Example is an example value.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`

	// Default is the default value.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`

	// AdditionalProperties indicates whether additional properties are allowed.
	AdditionalProperties *SchemaInfo `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	// AllOf combines multiple schemas.
	AllOf []*SchemaInfo `json:"allOf,omitempty" yaml:"allOf,omitempty"`
}

// EndpointUnit represents a parsed API endpoint with its input and output pins.
type EndpointUnit struct {
	// OperationID is the unique identifier from the spec.
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`

	// Name is a generated name if operationID is not available.
	Name string `json:"name" yaml:"name"`

	// Path is the URL path (e.g., "/users/{id}").
	Path string `json:"path" yaml:"path"`

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH).
	Method string `json:"method" yaml:"method"`

	// Summary is a brief description of the endpoint.
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Description is a detailed description of the endpoint.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Tags categorize the endpoint.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Consumes lists accepted content types.
	Consumes []string `json:"consumes,omitempty" yaml:"consumes,omitempty"`

	// Produces lists produced content types.
	Produces []string `json:"produces,omitempty" yaml:"produces,omitempty"`

	// InputPins are the input parameters for this endpoint.
	InputPins []InputPin `json:"inputPins" yaml:"inputPins"`

	// OutputPins are the output fields from successful responses.
	OutputPins []OutputPin `json:"outputPins" yaml:"outputPins"`

	// RequiresAuth indicates whether authentication is required.
	RequiresAuth bool `json:"requiresAuth" yaml:"requiresAuth"`

	// SecuritySchemes lists required security schemes.
	SecuritySchemes []string `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`

	// SuccessStatusCodes lists successful response status codes.
	SuccessStatusCodes []int `json:"successStatusCodes,omitempty" yaml:"successStatusCodes,omitempty"`

	// Deprecated indicates whether the endpoint is deprecated.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// OpenAPISpec represents the parsed OpenAPI/Swagger specification.
type OpenAPISpec struct {
	// Version is the OpenAPI/Swagger version ("2.0" or "3.0.x").
	Version string `json:"version" yaml:"version"`

	// Title is the API title.
	Title string `json:"title" yaml:"title"`

	// Description is the API description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// BasePath is the base path for all endpoints.
	BasePath string `json:"basePath,omitempty" yaml:"basePath,omitempty"`

	// Host is the API host.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`

	// Endpoints is the list of parsed endpoints.
	Endpoints []EndpointUnit `json:"endpoints" yaml:"endpoints"`

	// Tags lists all available tags with descriptions.
	Tags map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// SecurityDefinitions describes available security schemes.
	SecurityDefinitions map[string]SecurityScheme `json:"securityDefinitions,omitempty" yaml:"securityDefinitions,omitempty"`
}

// SecurityScheme describes a security scheme.
type SecurityScheme struct {
	// Type is the security scheme type (apiKey, basic, oauth2).
	Type string `json:"type" yaml:"type"`

	// Name is the header/query parameter name for apiKey.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// In is the location (header, query) for apiKey.
	In string `json:"in,omitempty" yaml:"in,omitempty"`

	// Description is the scheme description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Parser parses OpenAPI/Swagger specifications.
// DefaultMaxRefDepth is the maximum $ref resolution depth.
// Set to 20 to handle reasonably deep schema hierarchies while
// preventing stack overflow on circular references.
const DefaultMaxRefDepth = 20

// Parser parses OpenAPI/Swagger specifications.
//
// Thread Safety: Parser is NOT safe for concurrent use. Each goroutine
// should use its own Parser instance or serialize access externally.
// The ParseFile and ParseBytes methods modify internal state.
type Parser struct {
	// rawSpec holds the raw parsed spec for reference resolution.
	rawSpec map[string]any

	// resolvedRefs caches resolved $ref values to prevent infinite loops.
	resolvedRefs map[string]*SchemaInfo

	// maxRefDepth limits reference resolution depth.
	maxRefDepth int
}

// NewParser creates a new OpenAPI parser.
func NewParser() *Parser {
	return &Parser{
		resolvedRefs: make(map[string]*SchemaInfo),
		maxRefDepth:  DefaultMaxRefDepth,
	}
}

// ParseFile parses an OpenAPI/Swagger spec from a file.
func (p *Parser) ParseFile(path string) (*OpenAPISpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrSpecNotFound, path)
		}
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	return p.ParseBytes(data)
}

// ParseBytes parses an OpenAPI/Swagger spec from bytes.
func (p *Parser) ParseBytes(data []byte) (*OpenAPISpec, error) {
	// Reset resolved refs for fresh parse
	p.resolvedRefs = make(map[string]*SchemaInfo)

	// Try to parse as YAML first (also works for JSON)
	var rawSpec map[string]any
	if err := yaml.Unmarshal(data, &rawSpec); err != nil {
		// Try JSON as fallback
		if jsonErr := json.Unmarshal(data, &rawSpec); jsonErr != nil {
			return nil, fmt.Errorf("%w: failed to parse as YAML or JSON", ErrInvalidSpec)
		}
	}

	p.rawSpec = rawSpec

	// Determine version
	version := p.detectVersion(rawSpec)
	if version == "" {
		return nil, fmt.Errorf("%w: cannot determine spec version", ErrInvalidSpec)
	}

	// Parse based on version
	if strings.HasPrefix(version, "2") {
		return p.parseSwagger2(rawSpec, version)
	} else if strings.HasPrefix(version, "3") {
		return p.parseOpenAPI3(rawSpec, version)
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedVersion, version)
}

// detectVersion determines the OpenAPI/Swagger version from the raw spec.
func (p *Parser) detectVersion(spec map[string]any) string {
	// Check for Swagger 2.0
	if swagger, ok := spec["swagger"].(string); ok {
		return swagger
	}

	// Check for OpenAPI 3.x
	if openapi, ok := spec["openapi"].(string); ok {
		return openapi
	}

	return ""
}

// parseSwagger2 parses a Swagger 2.0 specification.
func (p *Parser) parseSwagger2(spec map[string]any, version string) (*OpenAPISpec, error) {
	result := &OpenAPISpec{
		Version:             version,
		Tags:                make(map[string]string),
		SecurityDefinitions: make(map[string]SecurityScheme),
	}

	// Parse info section
	if info, ok := spec["info"].(map[string]any); ok {
		result.Title = getString(info, "title")
		result.Description = getString(info, "description")
	}

	// Parse host and basePath
	result.Host = getString(spec, "host")
	result.BasePath = getString(spec, "basePath")

	// Parse tags
	if tags, ok := spec["tags"].([]any); ok {
		for _, t := range tags {
			if tag, ok := t.(map[string]any); ok {
				name := getString(tag, "name")
				desc := getString(tag, "description")
				if name != "" {
					result.Tags[name] = desc
				}
			}
		}
	}

	// Parse security definitions
	if secDefs, ok := spec["securityDefinitions"].(map[string]any); ok {
		for name, def := range secDefs {
			if defMap, ok := def.(map[string]any); ok {
				result.SecurityDefinitions[name] = SecurityScheme{
					Type:        getString(defMap, "type"),
					Name:        getString(defMap, "name"),
					In:          getString(defMap, "in"),
					Description: getString(defMap, "description"),
				}
			}
		}
	}

	// Parse global consumes/produces
	globalConsumes := getStringSlice(spec, "consumes")
	globalProduces := getStringSlice(spec, "produces")

	// Parse paths
	if paths, ok := spec["paths"].(map[string]any); ok {
		for path, pathItem := range paths {
			if pathItemMap, ok := pathItem.(map[string]any); ok {
				endpoints := p.parsePathItemSwagger2(path, pathItemMap, globalConsumes, globalProduces, spec)
				result.Endpoints = append(result.Endpoints, endpoints...)
			}
		}
	}

	// Sort endpoints by path and method for consistent output
	sort.Slice(result.Endpoints, func(i, j int) bool {
		if result.Endpoints[i].Path != result.Endpoints[j].Path {
			return result.Endpoints[i].Path < result.Endpoints[j].Path
		}
		return result.Endpoints[i].Method < result.Endpoints[j].Method
	})

	return result, nil
}

// parsePathItemSwagger2 parses operations for a path item in Swagger 2.0.
func (p *Parser) parsePathItemSwagger2(path string, pathItem map[string]any, globalConsumes, globalProduces []string, spec map[string]any) []EndpointUnit {
	var endpoints []EndpointUnit

	// Path-level parameters
	pathParams := p.parseParametersSwagger2(pathItem["parameters"])

	methods := []string{"get", "post", "put", "delete", "patch", "head", "options"}
	for _, method := range methods {
		if op, ok := pathItem[method].(map[string]any); ok {
			endpoint := p.parseOperationSwagger2(path, strings.ToUpper(method), op, pathParams, globalConsumes, globalProduces, spec)
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// parseOperationSwagger2 parses a single operation in Swagger 2.0.
func (p *Parser) parseOperationSwagger2(path, method string, op map[string]any, pathParams []InputPin, globalConsumes, globalProduces []string, spec map[string]any) EndpointUnit {
	endpoint := EndpointUnit{
		Path:   path,
		Method: method,
	}

	// Operation ID and name
	endpoint.OperationID = getString(op, "operationId")
	if endpoint.OperationID != "" {
		endpoint.Name = endpoint.OperationID
	} else {
		// Generate name from method and path
		endpoint.Name = generateEndpointName(method, path)
	}

	// Summary and description
	endpoint.Summary = getString(op, "summary")
	endpoint.Description = getString(op, "description")

	// Tags
	endpoint.Tags = getStringSlice(op, "tags")

	// Consumes/Produces (operation-level overrides global)
	endpoint.Consumes = getStringSlice(op, "consumes")
	if len(endpoint.Consumes) == 0 {
		endpoint.Consumes = globalConsumes
	}
	endpoint.Produces = getStringSlice(op, "produces")
	if len(endpoint.Produces) == 0 {
		endpoint.Produces = globalProduces
	}

	// Deprecated
	if deprecated, ok := op["deprecated"].(bool); ok {
		endpoint.Deprecated = deprecated
	}

	// Parse operation parameters and merge with path parameters
	opParams := p.parseParametersSwagger2(op["parameters"])
	endpoint.InputPins = mergeParameters(pathParams, opParams)

	// Parse security
	endpoint.RequiresAuth, endpoint.SecuritySchemes = p.parseSecuritySwagger2(op, spec)

	// Parse responses to extract output pins
	if responses, ok := op["responses"].(map[string]any); ok {
		endpoint.OutputPins, endpoint.SuccessStatusCodes = p.parseResponsesSwagger2(responses)
	}

	return endpoint
}

// parseParametersSwagger2 parses Swagger 2.0 parameters.
func (p *Parser) parseParametersSwagger2(params any) []InputPin {
	var inputPins []InputPin

	paramList, ok := params.([]any)
	if !ok {
		return inputPins
	}

	for _, param := range paramList {
		paramMap, ok := param.(map[string]any)
		if !ok {
			continue
		}

		// Handle $ref
		if ref, ok := paramMap["$ref"].(string); ok {
			resolved := p.resolveParameterRef(ref)
			if resolved != nil {
				inputPins = append(inputPins, *resolved)
			}
			continue
		}

		pin := InputPin{
			Name:        getString(paramMap, "name"),
			Location:    ParameterLocation(getString(paramMap, "in")),
			Required:    getBool(paramMap, "required"),
			Description: getString(paramMap, "description"),
			Default:     paramMap["default"],
			Enum:        getAnySlice(paramMap, "enum"),
			Pattern:     getString(paramMap, "pattern"),
		}

		// Type info depends on location
		if pin.Location == ParameterLocationBody {
			// Body parameters have schema
			if schema, ok := paramMap["schema"].(map[string]any); ok {
				pin.Schema = p.parseSchemaSwagger2(schema, 0)
				pin.Type = pin.Schema.Type
			}
		} else {
			// Other parameters have type/format directly
			pin.Type = ParameterType(getString(paramMap, "type"))
			pin.Format = getString(paramMap, "format")

			// Array items
			if items, ok := paramMap["items"].(map[string]any); ok {
				pin.Items = p.parseSchemaSwagger2(items, 0)
			}

			// Numeric constraints
			if min, ok := paramMap["minimum"].(float64); ok {
				pin.Minimum = &min
			}
			if max, ok := paramMap["maximum"].(float64); ok {
				pin.Maximum = &max
			}

			// String constraints (validate non-negative)
			if minLen, ok := paramMap["minLength"].(int); ok && minLen >= 0 {
				pin.MinLength = &minLen
			} else if minLen, ok := paramMap["minLength"].(float64); ok && minLen >= 0 {
				intVal := int(minLen)
				pin.MinLength = &intVal
			}
			if maxLen, ok := paramMap["maxLength"].(int); ok && maxLen >= 0 {
				pin.MaxLength = &maxLen
			} else if maxLen, ok := paramMap["maxLength"].(float64); ok && maxLen >= 0 {
				intVal := int(maxLen)
				pin.MaxLength = &intVal
			}
		}

		// Example
		pin.Example = paramMap["example"]

		inputPins = append(inputPins, pin)
	}

	return inputPins
}

// parseSchemaSwagger2 parses a Swagger 2.0 schema.
func (p *Parser) parseSchemaSwagger2(schema map[string]any, depth int) *SchemaInfo {
	if depth > p.maxRefDepth {
		return &SchemaInfo{Type: ParameterTypeObject}
	}

	// Handle $ref
	if ref, ok := schema["$ref"].(string); ok {
		// Check cache first
		if cached, ok := p.resolvedRefs[ref]; ok {
			return cached
		}

		// Reserve cache slot to prevent infinite recursion
		placeholder := &SchemaInfo{Ref: ref}
		p.resolvedRefs[ref] = placeholder

		resolved := p.resolveSchemaRef(ref, depth+1)
		if resolved != nil {
			resolved.Ref = ref
			p.resolvedRefs[ref] = resolved
			return resolved
		}

		return placeholder
	}

	info := &SchemaInfo{
		Type:        ParameterType(getString(schema, "type")),
		Format:      getString(schema, "format"),
		Description: getString(schema, "description"),
		Enum:        getAnySlice(schema, "enum"),
		Example:     schema["example"],
		Default:     schema["default"],
		Required:    getStringSlice(schema, "required"),
	}

	// Handle allOf
	if allOf, ok := schema["allOf"].([]any); ok {
		for _, subSchema := range allOf {
			if subMap, ok := subSchema.(map[string]any); ok {
				info.AllOf = append(info.AllOf, p.parseSchemaSwagger2(subMap, depth+1))
			}
		}
		// Merge allOf schemas
		info = p.mergeAllOf(info)
	}

	// Handle properties
	if props, ok := schema["properties"].(map[string]any); ok {
		info.Properties = make(map[string]*SchemaInfo)
		for name, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				info.Properties[name] = p.parseSchemaSwagger2(propMap, depth+1)
			}
		}
	}

	// Handle items for arrays
	if items, ok := schema["items"].(map[string]any); ok {
		info.Items = p.parseSchemaSwagger2(items, depth+1)
	}

	// Handle additionalProperties
	if addProps, ok := schema["additionalProperties"].(map[string]any); ok {
		info.AdditionalProperties = p.parseSchemaSwagger2(addProps, depth+1)
	} else if addProps, ok := schema["additionalProperties"].(bool); ok && addProps {
		info.AdditionalProperties = &SchemaInfo{Type: ParameterTypeObject}
	}

	return info
}

// mergeAllOf merges allOf schemas into a single schema.
func (p *Parser) mergeAllOf(info *SchemaInfo) *SchemaInfo {
	if len(info.AllOf) == 0 {
		return info
	}

	merged := &SchemaInfo{
		Type:       ParameterTypeObject,
		Properties: make(map[string]*SchemaInfo),
	}

	// Merge all schemas
	for _, sub := range info.AllOf {
		// Inherit type if not set
		if sub.Type != "" && merged.Type == "" {
			merged.Type = sub.Type
		}

		// Merge properties
		if sub.Properties != nil {
			for name, prop := range sub.Properties {
				merged.Properties[name] = prop
			}
		}

		// Merge required fields
		merged.Required = append(merged.Required, sub.Required...)

		// Inherit description
		if sub.Description != "" && merged.Description == "" {
			merged.Description = sub.Description
		}
	}

	return merged
}

// resolveSchemaRef resolves a $ref to a schema definition.
func (p *Parser) resolveSchemaRef(ref string, depth int) *SchemaInfo {
	// Parse ref path (e.g., "#/definitions/User")
	parts := strings.Split(ref, "/")
	if len(parts) < 3 || parts[0] != "#" {
		return nil
	}

	// Navigate to the referenced schema
	current := p.rawSpec
	for i := 1; i < len(parts); i++ {
		key := parts[i]
		if next, ok := current[key].(map[string]any); ok {
			current = next
		} else {
			return nil
		}
	}

	return p.parseSchemaSwagger2(current, depth)
}

// resolveParameterRef resolves a $ref to a parameter definition.
func (p *Parser) resolveParameterRef(ref string) *InputPin {
	// Parse ref path (e.g., "#/parameters/userId")
	parts := strings.Split(ref, "/")
	if len(parts) < 3 || parts[0] != "#" {
		return nil
	}

	// Navigate to the referenced parameter
	current := p.rawSpec
	for i := 1; i < len(parts); i++ {
		key := parts[i]
		if next, ok := current[key].(map[string]any); ok {
			current = next
		} else {
			return nil
		}
	}

	pins := p.parseParametersSwagger2([]any{current})
	if len(pins) > 0 {
		return &pins[0]
	}
	return nil
}

// parseSecuritySwagger2 parses security requirements for an operation.
func (p *Parser) parseSecuritySwagger2(op map[string]any, spec map[string]any) (bool, []string) {
	var schemes []string
	requiresAuth := false

	// Check operation-level security
	if security, ok := op["security"].([]any); ok {
		for _, sec := range security {
			if secMap, ok := sec.(map[string]any); ok {
				for name := range secMap {
					schemes = append(schemes, name)
					requiresAuth = true
				}
			}
		}
	} else {
		// Fall back to global security
		if security, ok := spec["security"].([]any); ok {
			for _, sec := range security {
				if secMap, ok := sec.(map[string]any); ok {
					for name := range secMap {
						schemes = append(schemes, name)
						requiresAuth = true
					}
				}
			}
		}
	}

	// Check for explicit empty security (no auth required)
	if security, ok := op["security"].([]any); ok && len(security) == 0 {
		requiresAuth = false
		schemes = nil
	}

	return requiresAuth, schemes
}

// parseResponsesSwagger2 parses responses to extract output pins.
func (p *Parser) parseResponsesSwagger2(responses map[string]any) ([]OutputPin, []int) {
	var outputPins []OutputPin
	var successCodes []int

	// Success response codes to check
	successKeys := []string{"200", "201", "202", "204"}

	for _, key := range successKeys {
		if resp, ok := responses[key].(map[string]any); ok {
			code := 200
			switch key {
			case "201":
				code = 201
			case "202":
				code = 202
			case "204":
				code = 204
			}
			successCodes = append(successCodes, code)

			// Parse schema
			if schema, ok := resp["schema"].(map[string]any); ok {
				pins := p.extractOutputPins(schema, "$", "", 0)
				outputPins = append(outputPins, pins...)
			}
		}
	}

	// Also check "default" response if no success codes found
	if len(successCodes) == 0 {
		if resp, ok := responses["default"].(map[string]any); ok {
			if schema, ok := resp["schema"].(map[string]any); ok {
				pins := p.extractOutputPins(schema, "$", "", 0)
				outputPins = append(outputPins, pins...)
			}
		}
	}

	// Deduplicate output pins
	outputPins = deduplicateOutputPins(outputPins)

	return outputPins, successCodes
}

// extractOutputPins recursively extracts output pins from a schema.
func (p *Parser) extractOutputPins(schema map[string]any, jsonPath, name string, depth int) []OutputPin {
	if depth > p.maxRefDepth {
		return nil
	}

	var pins []OutputPin

	// Handle $ref
	if ref, ok := schema["$ref"].(string); ok {
		resolved := p.resolveSchemaRef(ref, depth+1)
		if resolved != nil && resolved.Properties != nil {
			for propName, prop := range resolved.Properties {
				propPath := fmt.Sprintf("%s.%s", jsonPath, propName)
				pins = append(pins, p.outputPinFromSchemaInfo(propName, propPath, prop)...)
			}
		}
		return pins
	}

	// Handle allOf
	if allOf, ok := schema["allOf"].([]any); ok {
		for _, sub := range allOf {
			if subMap, ok := sub.(map[string]any); ok {
				pins = append(pins, p.extractOutputPins(subMap, jsonPath, name, depth+1)...)
			}
		}
		return deduplicateOutputPins(pins)
	}

	schemaType := getString(schema, "type")
	format := getString(schema, "format")

	// Handle object with properties
	if props, ok := schema["properties"].(map[string]any); ok {
		for propName, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				propPath := fmt.Sprintf("%s.%s", jsonPath, propName)
				pins = append(pins, p.extractOutputPins(propMap, propPath, propName, depth+1)...)
			}
		}
		return pins
	}

	// Handle array
	if schemaType == "array" {
		if items, ok := schema["items"].(map[string]any); ok {
			// For arrays, we add [*] to the path
			arrayPath := fmt.Sprintf("%s[*]", jsonPath)
			pins = append(pins, p.extractOutputPins(items, arrayPath, name, depth+1)...)
		}
		return pins
	}

	// Leaf node - create output pin
	if name != "" && schemaType != "" {
		pin := OutputPin{
			Name:        name,
			JSONPath:    jsonPath,
			Type:        ParameterType(schemaType),
			Format:      format,
			Description: getString(schema, "description"),
			Example:     schema["example"],
		}
		pins = append(pins, pin)
	}

	return pins
}

// outputPinFromSchemaInfo creates output pins from a SchemaInfo.
func (p *Parser) outputPinFromSchemaInfo(name, jsonPath string, info *SchemaInfo) []OutputPin {
	var pins []OutputPin

	// Handle nested properties
	if info.Properties != nil {
		for propName, prop := range info.Properties {
			propPath := fmt.Sprintf("%s.%s", jsonPath, propName)
			pins = append(pins, p.outputPinFromSchemaInfo(propName, propPath, prop)...)
		}
		return pins
	}

	// Handle arrays
	if info.Type == ParameterTypeArray && info.Items != nil {
		arrayPath := fmt.Sprintf("%s[*]", jsonPath)
		pins = append(pins, p.outputPinFromSchemaInfo(name, arrayPath, info.Items)...)
		// Also add the array itself as an output
		pin := OutputPin{
			Name:        name,
			JSONPath:    jsonPath,
			Type:        ParameterTypeArray,
			Format:      info.Format,
			Description: info.Description,
			IsArray:     true,
			Example:     info.Example,
		}
		pins = append(pins, pin)
		return pins
	}

	// Leaf node
	pin := OutputPin{
		Name:        name,
		JSONPath:    jsonPath,
		Type:        info.Type,
		Format:      info.Format,
		Description: info.Description,
		Example:     info.Example,
	}
	pins = append(pins, pin)

	return pins
}

// parseOpenAPI3 parses an OpenAPI 3.0 specification.
func (p *Parser) parseOpenAPI3(spec map[string]any, version string) (*OpenAPISpec, error) {
	result := &OpenAPISpec{
		Version:             version,
		Tags:                make(map[string]string),
		SecurityDefinitions: make(map[string]SecurityScheme),
	}

	// Parse info section
	if info, ok := spec["info"].(map[string]any); ok {
		result.Title = getString(info, "title")
		result.Description = getString(info, "description")
	}

	// Parse servers for basePath
	if servers, ok := spec["servers"].([]any); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]any); ok {
			url := getString(server, "url")
			result.Host = url
		}
	}

	// Parse tags
	if tags, ok := spec["tags"].([]any); ok {
		for _, t := range tags {
			if tag, ok := t.(map[string]any); ok {
				name := getString(tag, "name")
				desc := getString(tag, "description")
				if name != "" {
					result.Tags[name] = desc
				}
			}
		}
	}

	// Parse security schemes from components
	if components, ok := spec["components"].(map[string]any); ok {
		if secSchemes, ok := components["securitySchemes"].(map[string]any); ok {
			for name, def := range secSchemes {
				if defMap, ok := def.(map[string]any); ok {
					result.SecurityDefinitions[name] = SecurityScheme{
						Type:        getString(defMap, "type"),
						Name:        getString(defMap, "name"),
						In:          getString(defMap, "in"),
						Description: getString(defMap, "description"),
					}
				}
			}
		}
	}

	// Parse paths
	if paths, ok := spec["paths"].(map[string]any); ok {
		for path, pathItem := range paths {
			if pathItemMap, ok := pathItem.(map[string]any); ok {
				endpoints := p.parsePathItemOpenAPI3(path, pathItemMap, spec)
				result.Endpoints = append(result.Endpoints, endpoints...)
			}
		}
	}

	// Sort endpoints
	sort.Slice(result.Endpoints, func(i, j int) bool {
		if result.Endpoints[i].Path != result.Endpoints[j].Path {
			return result.Endpoints[i].Path < result.Endpoints[j].Path
		}
		return result.Endpoints[i].Method < result.Endpoints[j].Method
	})

	return result, nil
}

// parsePathItemOpenAPI3 parses operations for a path item in OpenAPI 3.0.
func (p *Parser) parsePathItemOpenAPI3(path string, pathItem map[string]any, spec map[string]any) []EndpointUnit {
	var endpoints []EndpointUnit

	// Path-level parameters
	pathParams := p.parseParametersOpenAPI3(pathItem["parameters"])

	methods := []string{"get", "post", "put", "delete", "patch", "head", "options"}
	for _, method := range methods {
		if op, ok := pathItem[method].(map[string]any); ok {
			endpoint := p.parseOperationOpenAPI3(path, strings.ToUpper(method), op, pathParams, spec)
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// parseOperationOpenAPI3 parses a single operation in OpenAPI 3.0.
func (p *Parser) parseOperationOpenAPI3(path, method string, op map[string]any, pathParams []InputPin, spec map[string]any) EndpointUnit {
	endpoint := EndpointUnit{
		Path:   path,
		Method: method,
	}

	// Operation ID and name
	endpoint.OperationID = getString(op, "operationId")
	if endpoint.OperationID != "" {
		endpoint.Name = endpoint.OperationID
	} else {
		endpoint.Name = generateEndpointName(method, path)
	}

	// Summary and description
	endpoint.Summary = getString(op, "summary")
	endpoint.Description = getString(op, "description")

	// Tags
	endpoint.Tags = getStringSlice(op, "tags")

	// Deprecated
	if deprecated, ok := op["deprecated"].(bool); ok {
		endpoint.Deprecated = deprecated
	}

	// Parse operation parameters
	opParams := p.parseParametersOpenAPI3(op["parameters"])
	endpoint.InputPins = mergeParameters(pathParams, opParams)

	// Parse requestBody
	if requestBody, ok := op["requestBody"].(map[string]any); ok {
		bodyPins := p.parseRequestBodyOpenAPI3(requestBody)
		endpoint.InputPins = append(endpoint.InputPins, bodyPins...)
	}

	// Parse security
	endpoint.RequiresAuth, endpoint.SecuritySchemes = p.parseSecuritySwagger2(op, spec)

	// Parse responses
	if responses, ok := op["responses"].(map[string]any); ok {
		endpoint.OutputPins, endpoint.SuccessStatusCodes = p.parseResponsesOpenAPI3(responses)
	}

	return endpoint
}

// parseParametersOpenAPI3 parses OpenAPI 3.0 parameters.
func (p *Parser) parseParametersOpenAPI3(params any) []InputPin {
	var inputPins []InputPin

	paramList, ok := params.([]any)
	if !ok {
		return inputPins
	}

	for _, param := range paramList {
		paramMap, ok := param.(map[string]any)
		if !ok {
			continue
		}

		// Handle $ref
		if ref, ok := paramMap["$ref"].(string); ok {
			resolved := p.resolveParameterRef(ref)
			if resolved != nil {
				inputPins = append(inputPins, *resolved)
			}
			continue
		}

		pin := InputPin{
			Name:        getString(paramMap, "name"),
			Location:    ParameterLocation(getString(paramMap, "in")),
			Required:    getBool(paramMap, "required"),
			Description: getString(paramMap, "description"),
		}

		// Schema contains type info in OpenAPI 3
		if schema, ok := paramMap["schema"].(map[string]any); ok {
			pin.Type = ParameterType(getString(schema, "type"))
			pin.Format = getString(schema, "format")
			pin.Default = schema["default"]
			pin.Enum = getAnySlice(schema, "enum")
			pin.Pattern = getString(schema, "pattern")
			pin.Example = schema["example"]

			if items, ok := schema["items"].(map[string]any); ok {
				pin.Items = p.parseSchemaSwagger2(items, 0)
			}
		}

		inputPins = append(inputPins, pin)
	}

	return inputPins
}

// parseRequestBodyOpenAPI3 parses OpenAPI 3.0 request body.
func (p *Parser) parseRequestBodyOpenAPI3(body map[string]any) []InputPin {
	var pins []InputPin

	required := getBool(body, "required")

	if content, ok := body["content"].(map[string]any); ok {
		// Prefer application/json
		for _, mediaType := range []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"} {
			if media, ok := content[mediaType].(map[string]any); ok {
				if schema, ok := media["schema"].(map[string]any); ok {
					pin := InputPin{
						Name:        "body",
						Location:    ParameterLocationBody,
						Required:    required,
						Description: getString(body, "description"),
						Schema:      p.parseSchemaSwagger2(schema, 0),
					}
					pin.Type = pin.Schema.Type
					pins = append(pins, pin)
					break
				}
			}
		}
	}

	return pins
}

// parseResponsesOpenAPI3 parses OpenAPI 3.0 responses.
func (p *Parser) parseResponsesOpenAPI3(responses map[string]any) ([]OutputPin, []int) {
	var outputPins []OutputPin
	var successCodes []int

	successKeys := []string{"200", "201", "202", "204"}

	for _, key := range successKeys {
		if resp, ok := responses[key].(map[string]any); ok {
			code := 200
			switch key {
			case "201":
				code = 201
			case "202":
				code = 202
			case "204":
				code = 204
			}
			successCodes = append(successCodes, code)

			if content, ok := resp["content"].(map[string]any); ok {
				if jsonContent, ok := content["application/json"].(map[string]any); ok {
					if schema, ok := jsonContent["schema"].(map[string]any); ok {
						pins := p.extractOutputPins(schema, "$", "", 0)
						outputPins = append(outputPins, pins...)
					}
				}
			}
		}
	}

	outputPins = deduplicateOutputPins(outputPins)
	return outputPins, successCodes
}

// Helper functions

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func getAnySlice(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// generateEndpointName generates a name from method and path.
func generateEndpointName(method, path string) string {
	// Remove path parameters and clean up
	cleaned := pathParamRegex.ReplaceAllString(path, "")
	cleaned = strings.ReplaceAll(cleaned, "//", "/")
	cleaned = strings.Trim(cleaned, "/")
	cleaned = strings.ReplaceAll(cleaned, "/", ".")

	if cleaned == "" {
		cleaned = "root"
	}

	return fmt.Sprintf("%s.%s", strings.ToLower(method), cleaned)
}

// mergeParameters merges path-level and operation-level parameters.
// Operation parameters override path parameters with the same name and location.
func mergeParameters(pathParams, opParams []InputPin) []InputPin {
	if len(pathParams) == 0 {
		return opParams
	}
	if len(opParams) == 0 {
		return pathParams
	}

	// Build map of operation params
	opMap := make(map[string]InputPin)
	for _, p := range opParams {
		key := fmt.Sprintf("%s:%s", p.Location, p.Name)
		opMap[key] = p
	}

	// Start with operation params
	result := make([]InputPin, 0, len(pathParams)+len(opParams))

	// Add path params that aren't overridden
	for _, p := range pathParams {
		key := fmt.Sprintf("%s:%s", p.Location, p.Name)
		if _, exists := opMap[key]; !exists {
			result = append(result, p)
		}
	}

	// Add all operation params
	result = append(result, opParams...)

	return result
}

// deduplicateOutputPins removes duplicate output pins by JSONPath.
func deduplicateOutputPins(pins []OutputPin) []OutputPin {
	seen := make(map[string]bool)
	result := make([]OutputPin, 0, len(pins))

	for _, pin := range pins {
		if !seen[pin.JSONPath] {
			seen[pin.JSONPath] = true
			result = append(result, pin)
		}
	}

	return result
}

// GetEndpointsByTag returns all endpoints with the given tag.
func (s *OpenAPISpec) GetEndpointsByTag(tag string) []EndpointUnit {
	var result []EndpointUnit
	for _, ep := range s.Endpoints {
		for _, t := range ep.Tags {
			if t == tag {
				result = append(result, ep)
				break
			}
		}
	}
	return result
}

// GetEndpointsByMethod returns all endpoints with the given HTTP method.
func (s *OpenAPISpec) GetEndpointsByMethod(method string) []EndpointUnit {
	var result []EndpointUnit
	upper := strings.ToUpper(method)
	for _, ep := range s.Endpoints {
		if ep.Method == upper {
			result = append(result, ep)
		}
	}
	return result
}

// GetAuthenticatedEndpoints returns all endpoints that require authentication.
func (s *OpenAPISpec) GetAuthenticatedEndpoints() []EndpointUnit {
	var result []EndpointUnit
	for _, ep := range s.Endpoints {
		if ep.RequiresAuth {
			result = append(result, ep)
		}
	}
	return result
}

// GetPublicEndpoints returns all endpoints that don't require authentication.
func (s *OpenAPISpec) GetPublicEndpoints() []EndpointUnit {
	var result []EndpointUnit
	for _, ep := range s.Endpoints {
		if !ep.RequiresAuth {
			result = append(result, ep)
		}
	}
	return result
}

// Summary returns a summary of the parsed specification.
func (s *OpenAPISpec) Summary() string {
	tagCounts := make(map[string]int)
	methodCounts := make(map[string]int)
	authCount := 0

	for _, ep := range s.Endpoints {
		methodCounts[ep.Method]++
		if ep.RequiresAuth {
			authCount++
		}
		for _, t := range ep.Tags {
			tagCounts[t]++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OpenAPI Specification: %s (v%s)\n", s.Title, s.Version))
	sb.WriteString(fmt.Sprintf("Total Endpoints: %d\n", len(s.Endpoints)))
	sb.WriteString(fmt.Sprintf("Authenticated: %d, Public: %d\n", authCount, len(s.Endpoints)-authCount))
	sb.WriteString("\nBy Method:\n")

	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if count, ok := methodCounts[method]; ok {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", method, count))
		}
	}

	sb.WriteString("\nBy Tag:\n")
	// Sort tags for consistent output
	var tags []string
	for tag := range tagCounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", tag, tagCounts[tag]))
	}

	return sb.String()
}
