// Package parser provides OpenAPI specification parsing for the load generator.
// It extracts endpoint definitions, parameters, and response schemas from OpenAPI 3.x specs.
package parser

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/getkin/kin-openapi/openapi3"
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
	// ParameterLocationCookie indicates a cookie parameter.
	ParameterLocationCookie ParameterLocation = "cookie"
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
	// Type is the security scheme type (apiKey, http, oauth2, openIdConnect).
	Type string `json:"type" yaml:"type"`

	// Name is the header/query parameter name for apiKey.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// In is the location (header, query, cookie) for apiKey.
	In string `json:"in,omitempty" yaml:"in,omitempty"`

	// Description is the scheme description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Scheme is the HTTP auth scheme (e.g., "bearer") for type "http".
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// DefaultMaxRefDepth is the maximum $ref resolution depth.
const DefaultMaxRefDepth = 20

// Parser parses OpenAPI specifications using kin-openapi.
//
// Thread Safety: Parser is safe for concurrent use after initialization.
type Parser struct {
	// maxRefDepth limits reference resolution depth.
	maxRefDepth int
}

// NewParser creates a new OpenAPI parser.
func NewParser() *Parser {
	return &Parser{
		maxRefDepth: DefaultMaxRefDepth,
	}
}

// ParseFile parses an OpenAPI spec from a file.
func (p *Parser) ParseFile(path string) (*OpenAPISpec, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrSpecNotFound, path)
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}

	return p.convertDoc(doc)
}

// ParseBytes parses an OpenAPI spec from bytes.
func (p *Parser) ParseBytes(data []byte) (*OpenAPISpec, error) {
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}

	return p.convertDoc(doc)
}

// convertDoc converts a kin-openapi document to our OpenAPISpec.
func (p *Parser) convertDoc(doc *openapi3.T) (*OpenAPISpec, error) {
	if doc.OpenAPI == "" {
		return nil, fmt.Errorf("%w: cannot determine spec version", ErrInvalidSpec)
	}

	result := &OpenAPISpec{
		Version:             doc.OpenAPI,
		Tags:                make(map[string]string),
		SecurityDefinitions: make(map[string]SecurityScheme),
	}

	// Parse info
	if doc.Info != nil {
		result.Title = doc.Info.Title
		result.Description = doc.Info.Description
	}

	// Parse servers for host/basePath
	if len(doc.Servers) > 0 {
		result.Host = doc.Servers[0].URL
	}

	// Parse tags
	for _, tag := range doc.Tags {
		if tag != nil {
			result.Tags[tag.Name] = tag.Description
		}
	}

	// Parse security schemes from components
	if doc.Components != nil && doc.Components.SecuritySchemes != nil {
		for name, schemeRef := range doc.Components.SecuritySchemes {
			if schemeRef != nil && schemeRef.Value != nil {
				scheme := schemeRef.Value
				result.SecurityDefinitions[name] = SecurityScheme{
					Type:        scheme.Type,
					Name:        scheme.Name,
					In:          scheme.In,
					Description: scheme.Description,
					Scheme:      scheme.Scheme,
				}
			}
		}
	}

	// Parse paths
	if doc.Paths != nil {
		for path, pathItem := range doc.Paths.Map() {
			if pathItem == nil {
				continue
			}
			endpoints := p.parsePathItem(path, pathItem, doc)
			result.Endpoints = append(result.Endpoints, endpoints...)
		}
	}

	// Sort endpoints for consistent output
	sort.Slice(result.Endpoints, func(i, j int) bool {
		if result.Endpoints[i].Path != result.Endpoints[j].Path {
			return result.Endpoints[i].Path < result.Endpoints[j].Path
		}
		return result.Endpoints[i].Method < result.Endpoints[j].Method
	})

	return result, nil
}

// parsePathItem parses all operations for a path.
func (p *Parser) parsePathItem(path string, item *openapi3.PathItem, doc *openapi3.T) []EndpointUnit {
	var endpoints []EndpointUnit

	// Path-level parameters
	pathParams := p.parseParameters(item.Parameters)

	operations := map[string]*openapi3.Operation{
		"GET":     item.Get,
		"POST":    item.Post,
		"PUT":     item.Put,
		"DELETE":  item.Delete,
		"PATCH":   item.Patch,
		"HEAD":    item.Head,
		"OPTIONS": item.Options,
	}

	for method, op := range operations {
		if op != nil {
			endpoint := p.parseOperation(path, method, op, pathParams, doc)
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// parseOperation parses a single operation.
func (p *Parser) parseOperation(path, method string, op *openapi3.Operation, pathParams []InputPin, doc *openapi3.T) EndpointUnit {
	endpoint := EndpointUnit{
		Path:       path,
		Method:     method,
		Deprecated: op.Deprecated,
	}

	// Operation ID and name
	endpoint.OperationID = op.OperationID
	if endpoint.OperationID != "" {
		endpoint.Name = endpoint.OperationID
	} else {
		endpoint.Name = generateEndpointName(method, path)
	}

	// Summary and description
	endpoint.Summary = op.Summary
	endpoint.Description = op.Description

	// Tags
	endpoint.Tags = op.Tags

	// Parse operation parameters and merge with path parameters
	opParams := p.parseParameters(op.Parameters)
	endpoint.InputPins = mergeParameters(pathParams, opParams)

	// Parse request body
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		bodyPins := p.parseRequestBody(op.RequestBody.Value)
		endpoint.InputPins = append(endpoint.InputPins, bodyPins...)
	}

	// Parse security
	endpoint.RequiresAuth, endpoint.SecuritySchemes = p.parseSecurity(op, doc)

	// Parse responses
	if op.Responses != nil {
		endpoint.OutputPins, endpoint.SuccessStatusCodes = p.parseResponses(op.Responses)
	}

	return endpoint
}

// parseParameters parses OpenAPI parameters.
func (p *Parser) parseParameters(params openapi3.Parameters) []InputPin {
	var pins []InputPin

	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		param := paramRef.Value

		pin := InputPin{
			Name:        param.Name,
			Location:    ParameterLocation(param.In),
			Required:    param.Required,
			Description: param.Description,
		}

		// Schema contains type info
		if param.Schema != nil && param.Schema.Value != nil {
			schema := param.Schema.Value
			pin.Type = getSchemaType(schema)
			pin.Format = schema.Format
			pin.Default = schema.Default
			pin.Pattern = schema.Pattern
			pin.Example = schema.Example

			// Enum values
			if len(schema.Enum) > 0 {
				pin.Enum = schema.Enum
			}

			// Items for array
			if schema.Items != nil && schema.Items.Value != nil {
				pin.Items = p.convertSchema(schema.Items.Value, 0)
			}

			// Numeric constraints
			if schema.Min != nil {
				pin.Minimum = schema.Min
			}
			if schema.Max != nil {
				pin.Maximum = schema.Max
			}

			// String constraints
			if schema.MinLength > 0 {
				minLen := int(schema.MinLength)
				pin.MinLength = &minLen
			}
			if schema.MaxLength != nil {
				maxLen := int(*schema.MaxLength)
				pin.MaxLength = &maxLen
			}
		}

		pins = append(pins, pin)
	}

	return pins
}

// parseRequestBody parses OpenAPI 3.0 request body.
func (p *Parser) parseRequestBody(body *openapi3.RequestBody) []InputPin {
	var pins []InputPin

	if body.Content == nil {
		return pins
	}

	// Prefer application/json
	mediaTypes := []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}
	for _, mediaType := range mediaTypes {
		if media, ok := body.Content[mediaType]; ok && media.Schema != nil && media.Schema.Value != nil {
			pin := InputPin{
				Name:        "body",
				Location:    ParameterLocationBody,
				Required:    body.Required,
				Description: body.Description,
				Schema:      p.convertSchema(media.Schema.Value, 0),
			}
			if pin.Schema != nil {
				pin.Type = pin.Schema.Type
			}
			pins = append(pins, pin)
			break
		}
	}

	return pins
}

// convertSchema converts kin-openapi schema to our SchemaInfo.
func (p *Parser) convertSchema(schema *openapi3.Schema, depth int) *SchemaInfo {
	if schema == nil || depth > p.maxRefDepth {
		return nil
	}

	info := &SchemaInfo{
		Type:        getSchemaType(schema),
		Format:      schema.Format,
		Description: schema.Description,
		Example:     schema.Example,
		Default:     schema.Default,
		Required:    schema.Required,
	}

	// Enum values
	if len(schema.Enum) > 0 {
		info.Enum = schema.Enum
	}

	// AllOf
	if len(schema.AllOf) > 0 {
		for _, ref := range schema.AllOf {
			if ref != nil && ref.Value != nil {
				info.AllOf = append(info.AllOf, p.convertSchema(ref.Value, depth+1))
			}
		}
		// Merge allOf into single schema
		info = p.mergeAllOf(info)
	}

	// Properties
	if len(schema.Properties) > 0 {
		info.Properties = make(map[string]*SchemaInfo)
		for name, propRef := range schema.Properties {
			if propRef != nil && propRef.Value != nil {
				info.Properties[name] = p.convertSchema(propRef.Value, depth+1)
			}
		}
	}

	// Items for arrays
	if schema.Items != nil && schema.Items.Value != nil {
		info.Items = p.convertSchema(schema.Items.Value, depth+1)
	}

	// AdditionalProperties
	if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		if schema.AdditionalProperties.Schema != nil && schema.AdditionalProperties.Schema.Value != nil {
			info.AdditionalProperties = p.convertSchema(schema.AdditionalProperties.Schema.Value, depth+1)
		} else {
			info.AdditionalProperties = &SchemaInfo{Type: ParameterTypeObject}
		}
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

	for _, sub := range info.AllOf {
		if sub == nil {
			continue
		}
		if sub.Type != "" && merged.Type == "" {
			merged.Type = sub.Type
		}
		if sub.Properties != nil {
			for name, prop := range sub.Properties {
				merged.Properties[name] = prop
			}
		}
		merged.Required = append(merged.Required, sub.Required...)
		if sub.Description != "" && merged.Description == "" {
			merged.Description = sub.Description
		}
	}

	return merged
}

// parseSecurity parses security requirements for an operation.
func (p *Parser) parseSecurity(op *openapi3.Operation, doc *openapi3.T) (bool, []string) {
	var schemes []string
	requiresAuth := false

	// Check operation-level security
	security := op.Security
	if security == nil {
		// Fall back to global security
		security = &doc.Security
	}

	if security != nil {
		for _, req := range *security {
			for name := range req {
				schemes = append(schemes, name)
				requiresAuth = true
			}
		}
	}

	// Empty security array means no auth required
	if op.Security != nil && len(*op.Security) == 0 {
		requiresAuth = false
		schemes = nil
	}

	return requiresAuth, schemes
}

// parseResponses parses responses to extract output pins.
func (p *Parser) parseResponses(responses *openapi3.Responses) ([]OutputPin, []int) {
	var outputPins []OutputPin
	var successCodes []int

	successKeys := []string{"200", "201", "202", "204"}

	for _, key := range successKeys {
		if respRef := responses.Value(key); respRef != nil && respRef.Value != nil {
			resp := respRef.Value
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

			// Parse JSON response content
			if resp.Content != nil {
				if jsonContent, ok := resp.Content["application/json"]; ok && jsonContent.Schema != nil && jsonContent.Schema.Value != nil {
					pins := p.extractOutputPins(jsonContent.Schema.Value, "$", "", 0)
					outputPins = append(outputPins, pins...)
				}
			}
		}
	}

	outputPins = deduplicateOutputPins(outputPins)
	return outputPins, successCodes
}

// extractOutputPins recursively extracts output pins from a schema.
func (p *Parser) extractOutputPins(schema *openapi3.Schema, jsonPath, name string, depth int) []OutputPin {
	if schema == nil || depth > p.maxRefDepth {
		return nil
	}

	var pins []OutputPin

	// Handle allOf
	if len(schema.AllOf) > 0 {
		for _, ref := range schema.AllOf {
			if ref != nil && ref.Value != nil {
				pins = append(pins, p.extractOutputPins(ref.Value, jsonPath, name, depth+1)...)
			}
		}
		return deduplicateOutputPins(pins)
	}

	schemaType := getSchemaType(schema)
	format := schema.Format

	// Handle object with properties
	if len(schema.Properties) > 0 {
		for propName, propRef := range schema.Properties {
			if propRef != nil && propRef.Value != nil {
				propPath := fmt.Sprintf("%s.%s", jsonPath, propName)
				pins = append(pins, p.extractOutputPins(propRef.Value, propPath, propName, depth+1)...)
			}
		}
		return pins
	}

	// Handle array
	if schemaType == ParameterTypeArray && schema.Items != nil && schema.Items.Value != nil {
		arrayPath := fmt.Sprintf("%s[*]", jsonPath)
		pins = append(pins, p.extractOutputPins(schema.Items.Value, arrayPath, name, depth+1)...)
		return pins
	}

	// Leaf node - create output pin
	if name != "" && schemaType != "" {
		pin := OutputPin{
			Name:        name,
			JSONPath:    jsonPath,
			Type:        schemaType,
			Format:      format,
			Description: schema.Description,
			Example:     schema.Example,
		}
		pins = append(pins, pin)
	}

	return pins
}

// Helper functions

// getSchemaType extracts the type from an OpenAPI schema.
// In OpenAPI 3.1, Type can be a slice of types; we take the first one.
func getSchemaType(schema *openapi3.Schema) ParameterType {
	if schema == nil || schema.Type == nil {
		return ""
	}
	types := schema.Type.Slice()
	if len(types) == 0 {
		return ""
	}
	return ParameterType(types[0])
}

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// generateEndpointName generates a name from method and path.
func generateEndpointName(method, path string) string {
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
func mergeParameters(pathParams, opParams []InputPin) []InputPin {
	if len(pathParams) == 0 {
		return opParams
	}
	if len(opParams) == 0 {
		return pathParams
	}

	opMap := make(map[string]InputPin)
	for _, p := range opParams {
		key := fmt.Sprintf("%s:%s", p.Location, p.Name)
		opMap[key] = p
	}

	result := make([]InputPin, 0, len(pathParams)+len(opParams))

	for _, p := range pathParams {
		key := fmt.Sprintf("%s:%s", p.Location, p.Name)
		if _, exists := opMap[key]; !exists {
			result = append(result, p)
		}
	}

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
