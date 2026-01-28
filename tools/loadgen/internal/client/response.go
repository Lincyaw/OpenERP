package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ResponseParser provides utilities for parsing HTTP responses.
type ResponseParser struct{}

// NewResponseParser creates a new response parser.
func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

// JSONPath extracts a value from JSON using a JSONPath expression.
// This is a simplified implementation that supports basic JSONPath expressions.
func (p *ResponseParser) JSONPath(data []byte, path string) (interface{}, error) {
	// Remove leading $
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return nil, fmt.Errorf("empty JSONPath")
	}

	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Split path into parts
	parts := strings.Split(path, ".")
	current := jsonData

	for _, part := range parts {
		// Handle array indices
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			arrayPart := strings.Split(part, "[")
			fieldName := arrayPart[0]
			indexStr := strings.TrimSuffix(arrayPart[1], "]")

			// Navigate to the field
			if fieldName != "" {
				current = p.getField(current, fieldName)
				if current == nil {
					return nil, fmt.Errorf("field not found: %s", fieldName)
				}
			}

			// Parse index
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", indexStr)
			}

			// Get array element
			current = p.getArrayElement(current, index)
			if current == nil {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}
		} else {
			// Regular field access
			current = p.getField(current, part)
			if current == nil {
				return nil, fmt.Errorf("field not found: %s", part)
			}
		}
	}

	return current, nil
}

// getField gets a field from a map or struct.
func (p *ResponseParser) getField(data interface{}, field string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return v[field]
	default:
		return nil
	}
}

// getArrayElement gets an element from an array.
func (p *ResponseParser) getArrayElement(data interface{}, index int) interface{} {
	switch v := data.(type) {
	case []interface{}:
		if index >= 0 && index < len(v) {
			return v[index]
		}
		return nil
	default:
		return nil
	}
}

// ExtractString extracts a string value using JSONPath.
func (p *ResponseParser) ExtractString(data []byte, path string) (string, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// ExtractInt extracts an integer value using JSONPath.
func (p *ResponseParser) ExtractInt(data []byte, path string) (int, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ExtractFloat extracts a float value using JSONPath.
func (p *ResponseParser) ExtractFloat(data []byte, path string) (float64, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ExtractBool extracts a boolean value using JSONPath.
func (p *ResponseParser) ExtractBool(data []byte, path string) (bool, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// ExtractArray extracts an array value using JSONPath.
func (p *ResponseParser) ExtractArray(data []byte, path string) ([]interface{}, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case []interface{}:
		return v, nil
	default:
		return nil, fmt.Errorf("value is not an array: %T", value)
	}
}

// ExtractObject extracts an object value using JSONPath.
func (p *ResponseParser) ExtractObject(data []byte, path string) (map[string]interface{}, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case map[string]interface{}:
		return v, nil
	default:
		return nil, fmt.Errorf("value is not an object: %T", value)
	}
}

// ExtractMultiple extracts multiple values from an array using JSONPath.
func (p *ResponseParser) ExtractMultiple(data []byte, path string) ([]interface{}, error) {
	value, err := p.JSONPath(data, path)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case []interface{}:
		return v, nil
	default:
		// Single value, wrap in array
		return []interface{}{v}, nil
	}
}

// ParseErrorResponse parses an error response from the API.
func (p *ResponseParser) ParseErrorResponse(data []byte) (string, error) {
	// Try to extract error message from common fields
	fields := []string{
		"error",
		"message",
		"msg",
		"error.message",
		"error.description",
	}

	for _, field := range fields {
		value, err := p.ExtractString(data, field)
		if err == nil && value != "" {
			return value, nil
		}
	}

	// If no error message found, return the whole response
	return string(data), nil
}