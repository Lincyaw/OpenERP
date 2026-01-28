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