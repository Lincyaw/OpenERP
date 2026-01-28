// Package executor provides request building and execution functionality for the load generator.
package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/pool"
)

// Constants for parameter locations
const (
	paramLocationPath   = "path"
	paramLocationQuery  = "query"
	paramLocationBody   = "body"
	paramLocationHeader = "header"
)

// Common query parameter field names
var commonQueryParams = map[string]bool{
	"page":       true,
	"page_size":  true,
	"limit":      true,
	"offset":     true,
	"sort_by":    true,
	"sort_order": true,
	"keyword":    true,
	"filter":     true,
}

// Valid HTTP methods
var validHTTPMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
}

// RequestBuilder builds HTTP requests from EndpointUnit configurations using parameters from the pool.
type RequestBuilder struct {
	pool pool.ParameterPool
}

// NewRequestBuilder creates a new request builder with the given parameter pool.
func NewRequestBuilder(pool pool.ParameterPool) *RequestBuilder {
	return &RequestBuilder{
		pool: pool,
	}
}

// BuildRequest constructs an HTTP request for the given endpoint unit.
// It fills in parameters from the pool based on the endpoint's input pins.
func (rb *RequestBuilder) BuildRequest(unit *circuit.EndpointUnit) (*http.Request, error) {
	if unit == nil {
		return nil, fmt.Errorf("endpoint unit is nil")
	}

	// Validate endpoint unit
	if err := rb.validateEndpointUnit(unit); err != nil {
		return nil, fmt.Errorf("invalid endpoint unit: %w", err)
	}

	// Validate that we have all required parameters
	if err := rb.validateRequiredParameters(unit); err != nil {
		return nil, fmt.Errorf("missing required parameters: %w", err)
	}

	// Build the URL with path parameters
	path, err := rb.buildPath(unit.Path, unit.InputPins)
	if err != nil {
		return nil, fmt.Errorf("failed to build path: %w", err)
	}

	// Build query parameters
	queryParams, err := rb.buildQueryParams(unit.InputPins)
	if err != nil {
		return nil, fmt.Errorf("failed to build query params: %w", err)
	}

	// Build request body
	body, contentType, err := rb.buildBody(unit.InputPins)
	if err != nil {
		return nil, fmt.Errorf("failed to build body: %w", err)
	}

	// Create the request
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	}
	req, err := http.NewRequest(unit.Method, path+queryParams, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if err := rb.addHeaders(req, unit.InputPins); err != nil {
		return nil, fmt.Errorf("failed to add headers: %w", err)
	}

	// Set content type if body is present or for POST/PUT/PATCH methods
	if contentType != "" || (unit.Method == "POST" || unit.Method == "PUT" || unit.Method == "PATCH") {
		if contentType == "" {
			contentType = "application/json"
		}
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// buildPath replaces path parameters in the URL with values from the pool.
// Path parameters are in the format {paramName} or :paramName.
func (rb *RequestBuilder) buildPath(path string, inputPins []circuit.SemanticType) (string, error) {
	// Create a map of available parameters by field name
	paramMap := make(map[string]circuit.SemanticType)
	for _, pin := range inputPins {
		fieldName := pin.Field()
		if fieldName != "" {
			paramMap[fieldName] = pin
		}
		// Also map by full semantic type for disambiguation
		paramMap[string(pin)] = pin
	}

	// Regular expression to match path parameters: {param} or :param
	pathParamRegex := regexp.MustCompile(`[{:]\w+[}]?`)

	// Find all path parameters
	matches := pathParamRegex.FindAllString(path, -1)
	result := path

	for _, match := range matches {
		// Extract parameter name
		paramName := strings.Trim(match, "{:")
		paramName = strings.TrimRight(paramName, "}")

		// Convert camelCase to snake_case for matching
		paramNameSnake := toSnakeCase(paramName)

		// Try to find a matching semantic type
		var paramSemantic circuit.SemanticType

		// First try exact field name match
		if pin, exists := paramMap[paramName]; exists {
			paramSemantic = pin
		} else if pin, exists := paramMap[paramNameSnake]; exists {
			// Try snake_case version
			paramSemantic = pin
		} else {
			// Try to match by entity type in the semantic type
			for _, pin := range inputPins {
				pinStr := string(pin)
				// Check if parameter name contains entity info
				if strings.Contains(paramName, "warehouse") && strings.Contains(pinStr, "warehouse") {
					paramSemantic = pin
					break
				} else if strings.Contains(paramName, "product") && strings.Contains(pinStr, "product") {
					paramSemantic = pin
					break
				} else if strings.Contains(paramName, "customer") && strings.Contains(pinStr, "customer") {
					paramSemantic = pin
					break
				}
			}
		}

		if paramSemantic == "" {
			// Try common patterns
			paramSemantic = circuit.SemanticType(fmt.Sprintf("common.%s", paramNameSnake))
		}

		// Get value from pool
		value, err := rb.pool.Get(paramSemantic)
		if err != nil {
			return "", fmt.Errorf("parameter %s not found in pool: %w", paramName, err)
		}

		// Convert value to string
		strValue := fmt.Sprintf("%v", value.Data)
		// For path parameters, we need to escape special characters
		escapedValue := url.PathEscape(strValue)

		// Replace in result
		result = strings.Replace(result, match, escapedValue, 1)
	}

	return result, nil
}

// toSnakeCase converts camelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// buildQueryParams builds query parameters from input pins.
func (rb *RequestBuilder) buildQueryParams(inputPins []circuit.SemanticType) (string, error) {
	params := url.Values{}

	for _, pin := range inputPins {
		// Extract field name from semantic type
		fieldName := pin.Field()
		if fieldName == "" {
			continue
		}

		// Skip path parameters (they would have been replaced in buildPath)
		if strings.Contains(string(pin), paramLocationPath) {
			continue
		}

		// Skip body and header parameters
		if strings.Contains(string(pin), paramLocationBody) || strings.Contains(string(pin), paramLocationHeader) {
			continue
		}

		// Skip common pagination/filtering parameters that should be in query
		if commonQueryParams[fieldName] {
			// Get value from pool
			value, err := rb.pool.Get(pin)
			if err != nil {
				continue // Skip if no value available
			}

			// Add to query parameters
			strValue := fmt.Sprintf("%v", value.Data)
			params.Add(fieldName, strValue)
		}
	}

	if len(params) == 0 {
		return "", nil
	}

	return "?" + params.Encode(), nil
}

// buildBody constructs the request body from body parameters.
func (rb *RequestBuilder) buildBody(inputPins []circuit.SemanticType) (io.Reader, string, error) {
	bodyParams := make(map[string]any)

	// Track used field names to detect conflicts
	fieldUsage := make(map[string][]circuit.SemanticType)

	for _, pin := range inputPins {
		// Skip path, query, and header parameters
		fieldName := pin.Field()
		if fieldName == "" {
			continue
		}

		// Skip if it's likely a path, query, or header parameter
		if strings.Contains(string(pin), "path") ||
			strings.Contains(string(pin), "query") ||
			strings.Contains(string(pin), "header") {
			continue
		}

		// Skip common query parameter fields
		if fieldName == "page" || fieldName == "page_size" ||
			fieldName == "limit" || fieldName == "offset" ||
			fieldName == "sort_by" || fieldName == "sort_order" ||
			fieldName == "keyword" || fieldName == "filter" {
			continue
		}

		// Get value from pool
		value, err := rb.pool.Get(pin)
		if err != nil {
			continue // Skip if no value available
		}

		// Track field usage for conflict detection
		fieldUsage[fieldName] = append(fieldUsage[fieldName], pin)

		// Use full semantic type path for disambiguation when conflicts exist
		key := fieldName
		if len(fieldUsage[fieldName]) > 1 || pin.Entity() != "" {
			// Always use full semantic type for disambiguation to avoid conflicts
			key = strings.ReplaceAll(string(pin), ".", "_")
		}

		// Add to body parameters
		bodyParams[key] = value.Data
	}

	if len(bodyParams) == 0 {
		return nil, "", nil
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(bodyParams)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal body: %w", err)
	}

	return bytes.NewReader(jsonBytes), "application/json", nil
}

// validateEndpointUnit validates the endpoint unit configuration.
func (rb *RequestBuilder) validateEndpointUnit(unit *circuit.EndpointUnit) error {
	if unit.Path == "" {
		return fmt.Errorf("endpoint path is empty")
	}
	if unit.Method == "" {
		return fmt.Errorf("HTTP method is empty")
	}
	// Normalize method to uppercase
	unit.Method = strings.ToUpper(unit.Method)
	if !validHTTPMethods[unit.Method] {
		return fmt.Errorf("invalid HTTP method: %s", unit.Method)
	}
	return nil
}

// validateRequiredParameters checks if all required parameters are available in the pool.
func (rb *RequestBuilder) validateRequiredParameters(unit *circuit.EndpointUnit) error {
	missingParams := []string{}

	// Check path parameters
	pathParamRegex := regexp.MustCompile(`[{:]\w+[}]?`)
	pathParams := pathParamRegex.FindAllString(unit.Path, -1)

	for _, param := range pathParams {
		paramName := strings.Trim(param, "{:")
		paramName = strings.TrimRight(paramName, "}")
		paramNameSnake := toSnakeCase(paramName)

		found := false
		for _, pin := range unit.InputPins {
			// Check various matching patterns
			if strings.HasSuffix(string(pin), "."+paramName) ||
				strings.HasSuffix(string(pin), "."+paramNameSnake) ||
				(strings.Contains(paramName, "warehouse") && strings.Contains(string(pin), "warehouse")) ||
				(strings.Contains(paramName, "product") && strings.Contains(string(pin), "product")) ||
				(strings.Contains(paramName, "customer") && strings.Contains(string(pin), "customer")) {
				// Check if value exists in pool
				if _, err := rb.pool.Get(pin); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			// Try common pattern
			commonType := circuit.SemanticType(fmt.Sprintf("common.%s", paramNameSnake))
			if _, err := rb.pool.Get(commonType); err != nil {
				missingParams = append(missingParams, paramName)
			}
		}
	}

	if len(missingParams) > 0 {
		return fmt.Errorf("missing required path parameters: %s", strings.Join(missingParams, ", "))
	}

	return nil
}

// addHeaders adds headers to the request based on header pins.
func (rb *RequestBuilder) addHeaders(req *http.Request, inputPins []circuit.SemanticType) error {
	for _, pin := range inputPins {
		// Check if this pin should be a header
		// For now, we'll use a simple heuristic: if it contains "token" or "auth"
		fieldName := pin.Field()
		if fieldName == "" {
			continue
		}

		// Check if this looks like an auth/token header
		if !strings.Contains(strings.ToLower(fieldName), "token") &&
			!strings.Contains(strings.ToLower(fieldName), "auth") &&
			!strings.Contains(strings.ToLower(fieldName), "key") {
			continue
		}

		// Get value from pool
		value, err := rb.pool.Get(pin)
		if err != nil {
			continue // Skip if no value available
		}

		// Convert header name to canonical format
		headerName := http.CanonicalHeaderKey(fieldName)
		strValue := fmt.Sprintf("%v", value.Data)
		req.Header.Add(headerName, strValue)
	}

	return nil
}

// BuildSalesOrderRequest is a convenience method for building a sales order request.
// This demonstrates building a complex POST /trade/sales-orders request.
func (rb *RequestBuilder) BuildSalesOrderRequest() (*http.Request, error) {
	// Example sales order endpoint unit
	unit := &circuit.EndpointUnit{
		Name:   "create-sales-order",
		Path:   "/trade/sales-orders",
		Method: "POST",
		InputPins: []circuit.SemanticType{
			// Customer info
			circuit.EntityCustomerID,
			// Order details
			circuit.OrderSalesNumber,
			// Items
			circuit.OrderItemQuantity,
			circuit.EntityProductID,
			// Payment
			circuit.FinancePaymentAmount,
		},
	}

	return rb.BuildRequest(unit)
}

// ============================================================================
// Extended Request Builder - Works with rich InputPin structures
// ============================================================================

// InputPin represents a typed input parameter with explicit location.
// This bridges the gap between circuit.SemanticType and parser.InputPin.
type InputPin struct {
	// Name is the parameter name.
	Name string

	// Location specifies where this parameter goes (path, query, body, header).
	Location string

	// SemanticType is the semantic classification for pool lookup.
	SemanticType circuit.SemanticType

	// Required indicates if this parameter is required.
	Required bool

	// Type is the data type (string, integer, number, boolean, array, object).
	Type string

	// Items describes array item schema for array types.
	Items *SchemaInfo

	// Schema describes the full schema for body parameters with nested structures.
	Schema *SchemaInfo

	// JSONPath is the path to set/get the value in the body (for nested structures).
	// For example: "items[0].product_id" or "shipping.address.city"
	JSONPath string

	// Default is the default value if no value is in the pool.
	Default any
}

// SchemaInfo holds schema information for complex body structures.
type SchemaInfo struct {
	// Type is the schema type (object, array, string, etc.).
	Type string

	// Properties lists properties for object types.
	Properties map[string]*SchemaInfo

	// Items describes array item schema.
	Items *SchemaInfo

	// Required lists required property names for objects.
	Required []string
}

// ExtendedEndpointUnit represents an endpoint with rich InputPin information.
type ExtendedEndpointUnit struct {
	// Name is the endpoint identifier.
	Name string

	// Path is the URL path (e.g., "/trade/sales-orders").
	Path string

	// Method is the HTTP method.
	Method string

	// InputPins are the typed input parameters.
	InputPins []InputPin

	// BodySchema defines the expected body structure for complex requests.
	BodySchema *SchemaInfo
}

// BuildRequestFromExtended constructs an HTTP request from an ExtendedEndpointUnit.
// This method properly handles:
// - Path parameters with explicit location
// - Query parameters with explicit location
// - Header parameters with explicit location
// - Complex body structures with nested objects and arrays
func (rb *RequestBuilder) BuildRequestFromExtended(unit *ExtendedEndpointUnit) (*http.Request, error) {
	if unit == nil {
		return nil, fmt.Errorf("endpoint unit is nil")
	}

	// Validate method
	method := strings.ToUpper(unit.Method)
	if !validHTTPMethods[method] {
		return nil, fmt.Errorf("invalid HTTP method: %s", unit.Method)
	}
	if unit.Path == "" {
		return nil, fmt.Errorf("endpoint path is empty")
	}

	// Categorize pins by location
	pathPins := make([]InputPin, 0)
	queryPins := make([]InputPin, 0)
	headerPins := make([]InputPin, 0)
	bodyPins := make([]InputPin, 0)

	for _, pin := range unit.InputPins {
		switch strings.ToLower(pin.Location) {
		case paramLocationPath:
			pathPins = append(pathPins, pin)
		case paramLocationQuery:
			queryPins = append(queryPins, pin)
		case paramLocationHeader:
			headerPins = append(headerPins, pin)
		case paramLocationBody, "":
			// Default to body for POST/PUT/PATCH
			bodyPins = append(bodyPins, pin)
		}
	}

	// Build path with replacements
	path, err := rb.buildPathFromPins(unit.Path, pathPins)
	if err != nil {
		return nil, fmt.Errorf("failed to build path: %w", err)
	}

	// Build query string
	queryString, err := rb.buildQueryFromPins(queryPins)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Build body
	var bodyReader io.Reader
	var contentType string
	if len(bodyPins) > 0 || unit.BodySchema != nil {
		bodyReader, contentType, err = rb.buildBodyFromPins(bodyPins, unit.BodySchema)
		if err != nil {
			return nil, fmt.Errorf("failed to build body: %w", err)
		}
	}

	// Create request
	fullURL := path + queryString
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for _, pin := range headerPins {
		value, err := rb.getValueForPin(pin)
		if err != nil {
			if pin.Required {
				return nil, fmt.Errorf("missing required header %s: %w", pin.Name, err)
			}
			continue
		}
		headerName := http.CanonicalHeaderKey(pin.Name)
		req.Header.Set(headerName, fmt.Sprintf("%v", value))
	}

	// Set content type
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// buildPathFromPins builds a URL path by replacing path parameters.
func (rb *RequestBuilder) buildPathFromPins(pathTemplate string, pathPins []InputPin) (string, error) {
	// Create a map for quick lookup by name
	pinMap := make(map[string]InputPin)
	for _, pin := range pathPins {
		pinMap[pin.Name] = pin
		pinMap[toSnakeCase(pin.Name)] = pin
	}

	// Find all path parameters
	pathParamRegex := regexp.MustCompile(`[{:]\w+[}]?`)
	matches := pathParamRegex.FindAllString(pathTemplate, -1)
	result := pathTemplate

	for _, match := range matches {
		// Extract parameter name
		paramName := strings.Trim(match, "{:")
		paramName = strings.TrimRight(paramName, "}")
		paramNameSnake := toSnakeCase(paramName)

		// Find the pin
		var pin InputPin
		var found bool
		if p, ok := pinMap[paramName]; ok {
			pin = p
			found = true
		} else if p, ok := pinMap[paramNameSnake]; ok {
			pin = p
			found = true
		}

		if !found {
			// Try to find by matching entity patterns in path pins
			for _, p := range pathPins {
				if matchesPathParam(paramName, p) {
					pin = p
					found = true
					break
				}
			}
		}

		if !found {
			return "", fmt.Errorf("no pin found for path parameter: %s", paramName)
		}

		// Get value
		value, err := rb.getValueForPin(pin)
		if err != nil {
			return "", fmt.Errorf("path parameter %s: %w", paramName, err)
		}

		// Replace in path
		escapedValue := url.PathEscape(fmt.Sprintf("%v", value))
		result = strings.Replace(result, match, escapedValue, 1)
	}

	return result, nil
}

// matchesPathParam checks if a pin matches a path parameter name.
func matchesPathParam(paramName string, pin InputPin) bool {
	pinName := strings.ToLower(pin.Name)
	paramLower := strings.ToLower(paramName)

	// Direct match
	if pinName == paramLower || pinName == toSnakeCase(paramLower) {
		return true
	}

	// Match by suffix (e.g., "warehouse_id" matches pin named "id" with entity warehouse)
	if strings.HasSuffix(paramLower, "_id") || strings.HasSuffix(paramLower, "id") {
		entity := strings.TrimSuffix(strings.TrimSuffix(paramLower, "_id"), "id")
		if entity != "" && strings.Contains(pinName, entity) {
			return true
		}
		if pin.SemanticType != "" {
			semanticStr := strings.ToLower(string(pin.SemanticType))
			if strings.Contains(semanticStr, entity) && strings.Contains(semanticStr, "id") {
				return true
			}
		}
	}

	return false
}

// buildQueryFromPins builds a query string from query parameters.
func (rb *RequestBuilder) buildQueryFromPins(queryPins []InputPin) (string, error) {
	if len(queryPins) == 0 {
		return "", nil
	}

	params := url.Values{}
	for _, pin := range queryPins {
		value, err := rb.getValueForPin(pin)
		if err != nil {
			if pin.Required {
				return "", fmt.Errorf("missing required query param %s: %w", pin.Name, err)
			}
			continue
		}

		// Handle array values
		switch v := value.(type) {
		case []any:
			for _, item := range v {
				params.Add(pin.Name, fmt.Sprintf("%v", item))
			}
		case []string:
			for _, item := range v {
				params.Add(pin.Name, item)
			}
		default:
			params.Add(pin.Name, fmt.Sprintf("%v", value))
		}
	}

	if len(params) == 0 {
		return "", nil
	}

	return "?" + params.Encode(), nil
}

// buildBodyFromPins constructs the request body from body parameters.
// Supports complex nested structures with objects and arrays.
func (rb *RequestBuilder) buildBodyFromPins(bodyPins []InputPin, schema *SchemaInfo) (io.Reader, string, error) {
	body := make(map[string]any)

	// Process each body pin
	for _, pin := range bodyPins {
		value, err := rb.getValueForPin(pin)
		if err != nil {
			if pin.Required {
				return nil, "", fmt.Errorf("missing required body param %s: %w", pin.Name, err)
			}
			continue
		}

		// Handle JSONPath for nested structures
		if pin.JSONPath != "" {
			if err := setNestedValue(body, pin.JSONPath, value); err != nil {
				return nil, "", fmt.Errorf("failed to set nested value at %s: %w", pin.JSONPath, err)
			}
		} else {
			// Simple field
			body[pin.Name] = value
		}
	}

	// If a schema is provided, validate and fill with defaults
	if schema != nil {
		body = applySchemaDefaults(body, schema)
	}

	if len(body) == 0 {
		return nil, "", nil
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal body: %w", err)
	}

	return bytes.NewReader(jsonBytes), "application/json", nil
}

// getValueForPin retrieves a value from the pool for the given pin.
func (rb *RequestBuilder) getValueForPin(pin InputPin) (any, error) {
	// Try semantic type first
	if pin.SemanticType != "" {
		value, err := rb.pool.Get(pin.SemanticType)
		if err == nil {
			return value.Data, nil
		}
	}

	// Try by name as semantic type
	nameType := circuit.SemanticType(pin.Name)
	value, err := rb.pool.Get(nameType)
	if err == nil {
		return value.Data, nil
	}

	// Try common patterns
	commonType := circuit.SemanticType(fmt.Sprintf("common.%s", toSnakeCase(pin.Name)))
	value, err = rb.pool.Get(commonType)
	if err == nil {
		return value.Data, nil
	}

	// Use default if available
	if pin.Default != nil {
		return pin.Default, nil
	}

	return nil, fmt.Errorf("no value found for pin: %s", pin.Name)
}

// setNestedValue sets a value at a nested path in a map.
// Supports paths like "items[0].product_id", "shipping.address.city".
func setNestedValue(m map[string]any, path string, value any) error {
	parts := parseJSONPath(path)
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	current := any(m)
	for i, part := range parts[:len(parts)-1] {
		switch p := part.(type) {
		case string:
			// Object key
			obj, ok := current.(map[string]any)
			if !ok {
				return fmt.Errorf("expected object at path %v", parts[:i+1])
			}
			if obj[p] == nil {
				// Create intermediate object or array based on next part
				if i+1 < len(parts)-1 {
					if _, isInt := parts[i+1].(int); isInt {
						obj[p] = make([]any, 0)
					} else {
						obj[p] = make(map[string]any)
					}
				} else {
					obj[p] = make(map[string]any)
				}
			}
			current = obj[p]
		case int:
			// Array index
			arr, ok := current.([]any)
			if !ok {
				return fmt.Errorf("expected array at path %v", parts[:i+1])
			}
			// Ensure array is large enough
			for len(arr) <= p {
				arr = append(arr, make(map[string]any))
			}
			// Update parent reference
			if i > 0 {
				if prevKey, ok := parts[i-1].(string); ok {
					if parentObj, ok := findParent(m, parts[:i]).(map[string]any); ok {
						parentObj[prevKey] = arr
					}
				}
			}
			current = arr[p]
		}
	}

	// Set final value
	lastPart := parts[len(parts)-1]
	switch p := lastPart.(type) {
	case string:
		obj, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object for final key %s", p)
		}
		obj[p] = value
	case int:
		arr, ok := current.([]any)
		if !ok {
			return fmt.Errorf("expected array for final index %d", p)
		}
		for len(arr) <= p {
			arr = append(arr, nil)
		}
		arr[p] = value
		// Update parent reference
		if len(parts) > 1 {
			if prevKey, ok := parts[len(parts)-2].(string); ok {
				if parentObj, ok := findParent(m, parts[:len(parts)-1]).(map[string]any); ok {
					parentObj[prevKey] = arr
				}
			}
		}
	}

	return nil
}

// parseJSONPath parses a JSONPath-like string into path components.
// "items[0].product_id" -> ["items", 0, "product_id"]
func parseJSONPath(path string) []any {
	var parts []any
	current := ""

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case '[':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			// Find closing bracket
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				indexStr := path[i+1 : j]
				var index int
				if _, err := fmt.Sscanf(indexStr, "%d", &index); err == nil {
					parts = append(parts, index)
				}
				i = j
			}
		default:
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// findParent navigates to the parent node at the given path.
func findParent(m map[string]any, path []any) any {
	current := any(m)
	for _, part := range path {
		switch p := part.(type) {
		case string:
			if obj, ok := current.(map[string]any); ok {
				current = obj[p]
			} else {
				return nil
			}
		case int:
			if arr, ok := current.([]any); ok && p < len(arr) {
				current = arr[p]
			} else {
				return nil
			}
		}
	}
	return current
}

// applySchemaDefaults fills missing values with schema defaults.
func applySchemaDefaults(body map[string]any, schema *SchemaInfo) map[string]any {
	if schema == nil || schema.Properties == nil {
		return body
	}

	for propName, propSchema := range schema.Properties {
		if _, exists := body[propName]; !exists {
			// Apply default or create nested structure
			if propSchema.Type == "object" && propSchema.Properties != nil {
				body[propName] = applySchemaDefaults(make(map[string]any), propSchema)
			} else if propSchema.Type == "array" {
				body[propName] = make([]any, 0)
			}
		} else if propSchema.Type == "object" && propSchema.Properties != nil {
			// Recursively apply to nested objects
			if nested, ok := body[propName].(map[string]any); ok {
				body[propName] = applySchemaDefaults(nested, propSchema)
			}
		}
	}

	return body
}

// ============================================================================
// Sales Order Request Builder - Complex body structure example
// ============================================================================

// SalesOrderItem represents a line item in a sales order.
type SalesOrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// SalesOrderRequest represents a complete sales order creation request.
type SalesOrderRequest struct {
	CustomerID      string           `json:"customer_id"`
	Items           []SalesOrderItem `json:"items"`
	Notes           string           `json:"notes,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// ShippingAddress represents a shipping address.
type ShippingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// BuildSalesOrderFromPool constructs a complete sales order request from pool values.
// This demonstrates building a complex body with nested objects and arrays.
func (rb *RequestBuilder) BuildSalesOrderFromPool() (*http.Request, error) {
	// Define the endpoint with typed InputPins
	unit := &ExtendedEndpointUnit{
		Name:   "create-sales-order",
		Path:   "/trade/sales-orders",
		Method: "POST",
		InputPins: []InputPin{
			{
				Name:         "customer_id",
				Location:     paramLocationBody,
				SemanticType: circuit.EntityCustomerID,
				Required:     true,
			},
			{
				Name:     "items",
				Location: paramLocationBody,
				Type:     "array",
				Required: true,
				Items: &SchemaInfo{
					Type: "object",
					Properties: map[string]*SchemaInfo{
						"product_id": {Type: "string"},
						"quantity":   {Type: "integer"},
						"unit_price": {Type: "number"},
					},
				},
			},
			{
				Name:         "notes",
				Location:     paramLocationBody,
				SemanticType: circuit.CommonNote,
				Required:     false,
			},
		},
		BodySchema: &SchemaInfo{
			Type: "object",
			Properties: map[string]*SchemaInfo{
				"customer_id": {Type: "string"},
				"items": {
					Type: "array",
					Items: &SchemaInfo{
						Type: "object",
						Properties: map[string]*SchemaInfo{
							"product_id": {Type: "string"},
							"quantity":   {Type: "integer"},
							"unit_price": {Type: "number"},
						},
					},
				},
				"notes": {Type: "string"},
				"shipping_address": {
					Type: "object",
					Properties: map[string]*SchemaInfo{
						"street":      {Type: "string"},
						"city":        {Type: "string"},
						"state":       {Type: "string"},
						"postal_code": {Type: "string"},
						"country":     {Type: "string"},
					},
				},
			},
		},
	}

	// Get customer ID
	customerValue, err := rb.pool.Get(circuit.EntityCustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer_id required: %w", err)
	}

	// Get order items from pool
	items, err := rb.buildOrderItems()
	if err != nil {
		return nil, fmt.Errorf("failed to build order items: %w", err)
	}

	// Get optional notes
	var notes string
	if noteValue, err := rb.pool.Get(circuit.CommonNote); err == nil {
		notes = fmt.Sprintf("%v", noteValue.Data)
	}

	// Build request body manually for complex structure
	body := map[string]any{
		"customer_id": customerValue.Data,
		"items":       items,
	}
	if notes != "" {
		body["notes"] = notes
	}

	// Add shipping address if available
	if address := rb.buildShippingAddress(); address != nil {
		body["shipping_address"] = address
	}

	// Serialize and create request
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := http.NewRequest(unit.Method, unit.Path, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// buildOrderItems builds order items array from pool values.
func (rb *RequestBuilder) buildOrderItems() ([]map[string]any, error) {
	// Try to get items as an array
	itemsValue, err := rb.pool.Get(circuit.SemanticType("order.items"))
	if err == nil {
		if items, ok := itemsValue.Data.([]any); ok {
			result := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if itemMap, ok := item.(map[string]any); ok {
					result = append(result, itemMap)
				}
			}
			if len(result) > 0 {
				return result, nil
			}
		}
	}

	// Build single item from individual fields
	item := make(map[string]any)

	// Product ID
	if productValue, err := rb.pool.Get(circuit.EntityProductID); err == nil {
		item["product_id"] = productValue.Data
	} else {
		return nil, fmt.Errorf("product_id required for order items")
	}

	// Quantity
	if qtyValue, err := rb.pool.Get(circuit.OrderItemQuantity); err == nil {
		item["quantity"] = qtyValue.Data
	} else {
		item["quantity"] = 1 // Default quantity
	}

	// Unit price
	if priceValue, err := rb.pool.Get(circuit.OrderItemPrice); err == nil {
		item["unit_price"] = priceValue.Data
	} else if priceValue, err := rb.pool.Get(circuit.CommonPrice); err == nil {
		item["unit_price"] = priceValue.Data
	}

	return []map[string]any{item}, nil
}

// buildShippingAddress builds a shipping address from pool values if available.
func (rb *RequestBuilder) buildShippingAddress() map[string]any {
	address := make(map[string]any)
	hasAny := false

	// Try to get address as a structured value
	if addrValue, err := rb.pool.Get(circuit.CommonAddress); err == nil {
		if addr, ok := addrValue.Data.(map[string]any); ok {
			return addr
		}
		// If it's a string, try to parse or use as street
		if addrStr, ok := addrValue.Data.(string); ok {
			address["street"] = addrStr
			hasAny = true
		}
	}

	// Try individual address components
	fields := []struct {
		name string
		key  string
	}{
		{"street", "street"},
		{"city", "city"},
		{"state", "state"},
		{"postal_code", "postal_code"},
		{"country", "country"},
	}

	for _, field := range fields {
		semanticType := circuit.SemanticType(fmt.Sprintf("common.%s", field.name))
		if value, err := rb.pool.Get(semanticType); err == nil {
			address[field.key] = value.Data
			hasAny = true
		}
	}

	if !hasAny {
		return nil
	}

	return address
}
