package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResponseParser_JSONPath tests basic JSONPath extraction.
func TestResponseParser_JSONPath(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected any
		wantErr  bool
	}{
		{
			name:     "simple field",
			json:     `{"name": "test"}`,
			path:     "$.name",
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "nested field",
			json:     `{"data": {"id": "123"}}`,
			path:     "$.data.id",
			expected: "123",
			wantErr:  false,
		},
		{
			name:     "deeply nested",
			json:     `{"response": {"data": {"user": {"name": "admin"}}}}`,
			path:     "$.response.data.user.name",
			expected: "admin",
			wantErr:  false,
		},
		{
			name:     "array element",
			json:     `{"items": ["a", "b", "c"]}`,
			path:     "$.items[1]",
			expected: "b",
			wantErr:  false,
		},
		{
			name:     "nested array",
			json:     `{"data": {"list": [{"id": 1}, {"id": 2}]}}`,
			path:     "$.data.list[0].id",
			expected: float64(1),
			wantErr:  false,
		},
		{
			name:     "numeric value",
			json:     `{"count": 42}`,
			path:     "$.count",
			expected: float64(42),
			wantErr:  false,
		},
		{
			name:     "boolean value",
			json:     `{"active": true}`,
			path:     "$.active",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "without dollar sign",
			json:     `{"key": "value"}`,
			path:     "key",
			expected: "value",
			wantErr:  false,
		},
		{
			name:    "missing field",
			json:    `{"name": "test"}`,
			path:    "$.missing",
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `not valid json`,
			path:    "$.name",
			wantErr: true,
		},
		{
			name:    "array out of bounds",
			json:    `{"items": [1, 2]}`,
			path:    "$.items[5]",
			wantErr: true,
		},
		{
			name:    "empty path",
			json:    `{"name": "test"}`,
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.JSONPath([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestResponseParser_ExtractString tests string extraction.
func TestResponseParser_ExtractString(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "string value",
			json:     `{"name": "test"}`,
			path:     "$.name",
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "number as string",
			json:     `{"count": 42}`,
			path:     "$.count",
			expected: "42",
			wantErr:  false,
		},
		{
			name:     "float as string",
			json:     `{"price": 19.99}`,
			path:     "$.price",
			expected: "19.99",
			wantErr:  false,
		},
		{
			name:     "boolean as string",
			json:     `{"active": true}`,
			path:     "$.active",
			expected: "true",
			wantErr:  false,
		},
		{
			name:    "missing field",
			json:    `{"name": "test"}`,
			path:    "$.missing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractString([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestResponseParser_ExtractInt tests integer extraction.
func TestResponseParser_ExtractInt(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected int
		wantErr  bool
	}{
		{
			name:     "integer value",
			json:     `{"count": 42}`,
			path:     "$.count",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "float to int",
			json:     `{"value": 42.7}`,
			path:     "$.value",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "string to int",
			json:     `{"num": "123"}`,
			path:     "$.num",
			expected: 123,
			wantErr:  false,
		},
		{
			name:    "non-numeric string",
			json:    `{"value": "abc"}`,
			path:    "$.value",
			wantErr: true,
		},
		{
			name:    "boolean value",
			json:    `{"flag": true}`,
			path:    "$.flag",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractInt([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestResponseParser_ExtractFloat tests float extraction.
func TestResponseParser_ExtractFloat(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected float64
		wantErr  bool
	}{
		{
			name:     "float value",
			json:     `{"price": 19.99}`,
			path:     "$.price",
			expected: 19.99,
			wantErr:  false,
		},
		{
			name:     "integer to float",
			json:     `{"count": 42}`,
			path:     "$.count",
			expected: 42.0,
			wantErr:  false,
		},
		{
			name:     "string to float",
			json:     `{"value": "3.14"}`,
			path:     "$.value",
			expected: 3.14,
			wantErr:  false,
		},
		{
			name:    "non-numeric string",
			json:    `{"value": "abc"}`,
			path:    "$.value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractFloat([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.001)
			}
		})
	}
}

// TestResponseParser_ExtractBool tests boolean extraction.
func TestResponseParser_ExtractBool(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "true value",
			json:     `{"active": true}`,
			path:     "$.active",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "false value",
			json:     `{"active": false}`,
			path:     "$.active",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "string true",
			json:     `{"flag": "true"}`,
			path:     "$.flag",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "string false",
			json:     `{"flag": "false"}`,
			path:     "$.flag",
			expected: false,
			wantErr:  false,
		},
		{
			name:    "non-boolean value",
			json:    `{"value": 123}`,
			path:    "$.value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractBool([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestResponseParser_ExtractArray tests array extraction.
func TestResponseParser_ExtractArray(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected int // length of array
		wantErr  bool
	}{
		{
			name:     "simple array",
			json:     `{"items": [1, 2, 3]}`,
			path:     "$.items",
			expected: 3,
			wantErr:  false,
		},
		{
			name:     "empty array",
			json:     `{"items": []}`,
			path:     "$.items",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "nested array",
			json:     `{"data": {"list": ["a", "b"]}}`,
			path:     "$.data.list",
			expected: 2,
			wantErr:  false,
		},
		{
			name:    "non-array value",
			json:    `{"value": "string"}`,
			path:    "$.value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractArray([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expected)
			}
		})
	}
}

// TestResponseParser_ExtractObject tests object extraction.
func TestResponseParser_ExtractObject(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name        string
		json        string
		path        string
		expectedKey string
		wantErr     bool
	}{
		{
			name:        "simple object",
			json:        `{"data": {"id": "123"}}`,
			path:        "$.data",
			expectedKey: "id",
			wantErr:     false,
		},
		{
			name:        "nested object",
			json:        `{"response": {"user": {"name": "admin"}}}`,
			path:        "$.response.user",
			expectedKey: "name",
			wantErr:     false,
		},
		{
			name:    "non-object value",
			json:    `{"value": "string"}`,
			path:    "$.value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractObject([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				_, ok := result[tt.expectedKey]
				assert.True(t, ok, "expected key %s not found", tt.expectedKey)
			}
		})
	}
}

// TestResponseParser_ExtractMultiple tests extracting multiple values.
func TestResponseParser_ExtractMultiple(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		path     string
		expected int // length of result
		wantErr  bool
	}{
		{
			name:     "array of values",
			json:     `{"items": [1, 2, 3]}`,
			path:     "$.items",
			expected: 3,
			wantErr:  false,
		},
		{
			name:     "single value wrapped",
			json:     `{"value": "single"}`,
			path:     "$.value",
			expected: 1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractMultiple([]byte(tt.json), tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expected)
			}
		})
	}
}

// TestResponseParser_ParseErrorResponse tests error response parsing.
func TestResponseParser_ParseErrorResponse(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "error field",
			json:     `{"error": "something went wrong"}`,
			expected: "something went wrong",
		},
		{
			name:     "message field",
			json:     `{"message": "access denied"}`,
			expected: "access denied",
		},
		{
			name:     "msg field",
			json:     `{"msg": "invalid request"}`,
			expected: "invalid request",
		},
		{
			name:     "nested error object converted to string",
			json:     `{"error": {"message": "internal error"}}`,
			expected: "map[message:internal error]",
		},
		{
			name:     "fallback to raw json",
			json:     `{"unknown": "format"}`,
			expected: `{"unknown": "format"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseErrorResponse([]byte(tt.json))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestResponseParser_ERPLoginResponse tests parsing ERP login response.
func TestResponseParser_ERPLoginResponse(t *testing.T) {
	parser := NewResponseParser()

	// Simulate ERP login response
	erpResponse := `{
		"success": true,
		"data": {
			"token": {
				"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
				"refresh_token": "refresh-token-123",
				"access_token_expires_at": "2025-01-29T12:00:00Z",
				"token_type": "Bearer"
			}
		}
	}`

	t.Run("extract access token", func(t *testing.T) {
		token, err := parser.ExtractString([]byte(erpResponse), "$.data.token.access_token")
		require.NoError(t, err)
		assert.Contains(t, token, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
	})

	t.Run("extract refresh token", func(t *testing.T) {
		token, err := parser.ExtractString([]byte(erpResponse), "$.data.token.refresh_token")
		require.NoError(t, err)
		assert.Equal(t, "refresh-token-123", token)
	})

	t.Run("extract token type", func(t *testing.T) {
		tokenType, err := parser.ExtractString([]byte(erpResponse), "$.data.token.token_type")
		require.NoError(t, err)
		assert.Equal(t, "Bearer", tokenType)
	})

	t.Run("extract success flag", func(t *testing.T) {
		success, err := parser.ExtractBool([]byte(erpResponse), "$.success")
		require.NoError(t, err)
		assert.True(t, success)
	})
}

// TestResponseParser_ERPListResponse tests parsing ERP list response.
func TestResponseParser_ERPListResponse(t *testing.T) {
	parser := NewResponseParser()

	// Simulate ERP list response
	erpResponse := `{
		"success": true,
		"data": {
			"items": [
				{"id": "prod-001", "name": "Product A"},
				{"id": "prod-002", "name": "Product B"},
				{"id": "prod-003", "name": "Product C"}
			]
		},
		"meta": {
			"total": 100,
			"page": 1,
			"limit": 10
		}
	}`

	t.Run("extract items count", func(t *testing.T) {
		items, err := parser.ExtractArray([]byte(erpResponse), "$.data.items")
		require.NoError(t, err)
		assert.Len(t, items, 3)
	})

	t.Run("extract first item id", func(t *testing.T) {
		id, err := parser.ExtractString([]byte(erpResponse), "$.data.items[0].id")
		require.NoError(t, err)
		assert.Equal(t, "prod-001", id)
	})

	t.Run("extract total count", func(t *testing.T) {
		total, err := parser.ExtractInt([]byte(erpResponse), "$.meta.total")
		require.NoError(t, err)
		assert.Equal(t, 100, total)
	})

	t.Run("extract page number", func(t *testing.T) {
		page, err := parser.ExtractInt([]byte(erpResponse), "$.meta.page")
		require.NoError(t, err)
		assert.Equal(t, 1, page)
	})
}
