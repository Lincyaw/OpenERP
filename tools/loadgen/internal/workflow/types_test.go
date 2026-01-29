package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty config is valid",
			config: Config{
				Workflows: nil,
			},
			wantErr: false,
		},
		{
			name: "valid workflow",
			config: Config{
				Workflows: map[string]Definition{
					"test_workflow": {
						Name: "test_workflow",
						Steps: []Step{
							{Endpoint: "POST /api/test"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "workflow with no steps",
			config: Config{
				Workflows: map[string]Definition{
					"empty_workflow": {
						Name:  "empty_workflow",
						Steps: []Step{},
					},
				},
			},
			wantErr: true,
			errMsg:  "has no steps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		def     Definition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid definition",
			def: Definition{
				Steps: []Step{
					{Endpoint: "GET /api/test"},
				},
			},
			wantErr: false,
		},
		{
			name: "no steps",
			def: Definition{
				Steps: []Step{},
			},
			wantErr: true,
			errMsg:  "has no steps",
		},
		{
			name: "invalid step",
			def: Definition{
				Steps: []Step{
					{Endpoint: ""}, // Invalid: empty endpoint
				},
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "valid definition with weight",
			def: Definition{
				Weight: 10,
				Steps: []Step{
					{Endpoint: "POST /api/orders"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.def.Validate("test_workflow")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStep_Validate(t *testing.T) {
	tests := []struct {
		name    string
		step    Step
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid GET step",
			step:    Step{Endpoint: "GET /api/test"},
			wantErr: false,
		},
		{
			name:    "valid POST step",
			step:    Step{Endpoint: "POST /api/test"},
			wantErr: false,
		},
		{
			name:    "valid PUT step",
			step:    Step{Endpoint: "PUT /api/test"},
			wantErr: false,
		},
		{
			name:    "valid DELETE step",
			step:    Step{Endpoint: "DELETE /api/test"},
			wantErr: false,
		},
		{
			name:    "valid PATCH step",
			step:    Step{Endpoint: "PATCH /api/test"},
			wantErr: false,
		},
		{
			name:    "empty endpoint",
			step:    Step{Endpoint: ""},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name:    "invalid endpoint format - no method",
			step:    Step{Endpoint: "/api/test"},
			wantErr: true,
			errMsg:  "must be in format",
		},
		{
			name:    "invalid HTTP method",
			step:    Step{Endpoint: "INVALID /api/test"},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
		{
			name:    "valid onFailure - abort",
			step:    Step{Endpoint: "GET /api/test", OnFailure: "abort"},
			wantErr: false,
		},
		{
			name:    "valid onFailure - continue",
			step:    Step{Endpoint: "GET /api/test", OnFailure: "continue"},
			wantErr: false,
		},
		{
			name:    "valid onFailure - retry",
			step:    Step{Endpoint: "GET /api/test", OnFailure: "retry", RetryCount: 3},
			wantErr: false,
		},
		{
			name:    "invalid onFailure",
			step:    Step{Endpoint: "GET /api/test", OnFailure: "invalid"},
			wantErr: true,
			errMsg:  "invalid onFailure option",
		},
		{
			name: "valid extract JSONPath",
			step: Step{
				Endpoint: "GET /api/test",
				Extract:  map[string]string{"id": "$.data.id"},
			},
			wantErr: false,
		},
		{
			name: "invalid extract - empty variable name",
			step: Step{
				Endpoint: "GET /api/test",
				Extract:  map[string]string{"": "$.data.id"},
			},
			wantErr: true,
			errMsg:  "variable name cannot be empty",
		},
		{
			name: "invalid extract - empty JSONPath",
			step: Step{
				Endpoint: "GET /api/test",
				Extract:  map[string]string{"id": ""},
			},
			wantErr: true,
			errMsg:  "JSONPath for",
		},
		{
			name: "invalid extract - JSONPath must start with $",
			step: Step{
				Endpoint: "GET /api/test",
				Extract:  map[string]string{"id": "data.id"},
			},
			wantErr: true,
			errMsg:  "must start with",
		},
		{
			name: "valid extract with array JSONPath",
			step: Step{
				Endpoint: "GET /api/test",
				Extract:  map[string]string{"ids": "$[0].id"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.step.Validate(0)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefinition_ApplyDefaults(t *testing.T) {
	def := Definition{
		Steps: []Step{
			{Endpoint: "GET /api/test"},
			{Endpoint: "POST /api/test"},
			{Endpoint: "DELETE /api/test"},
		},
	}

	def.ApplyDefaults("test_workflow")

	// Check name is set
	assert.Equal(t, "test_workflow", def.Name)

	// Check weight default
	assert.Equal(t, 1, def.Weight)

	// Check timeout default
	assert.Equal(t, "60s", def.Timeout)

	// Check step defaults
	for _, step := range def.Steps {
		assert.Equal(t, "abort", step.OnFailure)
		assert.Equal(t, "1s", step.RetryDelay)
	}

	// Check expected status defaults by method
	assert.Equal(t, 200, def.Steps[0].ExpectedStatus) // GET
	assert.Equal(t, 201, def.Steps[1].ExpectedStatus) // POST
	assert.Equal(t, 204, def.Steps[2].ExpectedStatus) // DELETE
}

func TestStep_GetMethod(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"GET /api/test", "GET"},
		{"POST /api/test", "POST"},
		{"put /api/test", "PUT"},
		{"Delete /api/test", "DELETE"},
		{"PATCH /api/test", "PATCH"},
		{"", ""},
	}

	for _, tt := range tests {
		step := Step{Endpoint: tt.endpoint}
		assert.Equal(t, tt.expected, step.GetMethod())
	}
}

func TestStep_GetPath(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"GET /api/test", "/api/test"},
		{"POST /api/orders/{id}", "/api/orders/{id}"},
		{"PUT /api/test?query=value", "/api/test?query=value"},
		{"DELETE /", "/"},
		{"GET", ""},
	}

	for _, tt := range tests {
		step := Step{Endpoint: tt.endpoint}
		assert.Equal(t, tt.expected, step.GetPath())
	}
}

func TestStep_GetPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		expected []string
	}{
		{
			name:     "no placeholders",
			step:     Step{Endpoint: "GET /api/test"},
			expected: []string{},
		},
		{
			name:     "single placeholder in path",
			step:     Step{Endpoint: "GET /api/orders/{order_id}"},
			expected: []string{"order_id"},
		},
		{
			name:     "multiple placeholders in path",
			step:     Step{Endpoint: "GET /api/orders/{order_id}/items/{item_id}"},
			expected: []string{"order_id", "item_id"},
		},
		{
			name:     "placeholder in body",
			step:     Step{Endpoint: "POST /api/test", Body: `{"customer_id": "{customer_id}"}`},
			expected: []string{"customer_id"},
		},
		{
			name: "placeholders in both path and body",
			step: Step{
				Endpoint: "POST /api/orders/{order_id}",
				Body:     `{"product_id": "{product_id}"}`,
			},
			expected: []string{"order_id", "product_id"},
		},
		{
			name: "duplicate placeholders are deduplicated",
			step: Step{
				Endpoint: "POST /api/orders/{order_id}",
				Body:     `{"order_id": "{order_id}"}`,
			},
			expected: []string{"order_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.step.GetPlaceholders()
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]any
		expected string
	}{
		{
			name:     "empty template",
			template: "",
			context:  map[string]any{"id": "123"},
			expected: "",
		},
		{
			name:     "nil context",
			template: "/api/orders/{id}",
			context:  nil,
			expected: "/api/orders/{id}",
		},
		{
			name:     "no placeholders",
			template: "/api/test",
			context:  map[string]any{"id": "123"},
			expected: "/api/test",
		},
		{
			name:     "single replacement",
			template: "/api/orders/{order_id}",
			context:  map[string]any{"order_id": "abc123"},
			expected: "/api/orders/abc123",
		},
		{
			name:     "multiple replacements",
			template: "/api/orders/{order_id}/items/{item_id}",
			context:  map[string]any{"order_id": "o1", "item_id": "i1"},
			expected: "/api/orders/o1/items/i1",
		},
		{
			name:     "missing value - keep original",
			template: "/api/orders/{order_id}",
			context:  map[string]any{},
			expected: "/api/orders/{order_id}",
		},
		{
			name:     "integer value",
			template: "/api/orders/{id}",
			context:  map[string]any{"id": 123},
			expected: "/api/orders/123",
		},
		{
			name:     "float value",
			template: "/api/prices/{price}",
			context:  map[string]any{"price": 99.99},
			expected: "/api/prices/99.99",
		},
		{
			name:     "body template replacement",
			template: `{"customer_id": "{customer_id}", "amount": {amount}}`,
			context:  map[string]any{"customer_id": "c1", "amount": 100},
			expected: `{"customer_id": "c1", "amount": 100}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplacePlaceholders(tt.template, tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetEnabledWorkflows(t *testing.T) {
	config := Config{
		Workflows: map[string]Definition{
			"enabled1": {Name: "enabled1", Disabled: false},
			"disabled": {Name: "disabled", Disabled: true},
			"enabled2": {Name: "enabled2", Disabled: false},
		},
	}

	enabled := config.GetEnabledWorkflows()
	assert.Len(t, enabled, 2)
	assert.Contains(t, enabled, "enabled1")
	assert.Contains(t, enabled, "enabled2")
	assert.NotContains(t, enabled, "disabled")
}

func TestConfig_TotalWeight(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name:     "empty config",
			config:   Config{},
			expected: 0,
		},
		{
			name: "single workflow default weight",
			config: Config{
				Workflows: map[string]Definition{
					"w1": {Weight: 0}, // Default should be 1
				},
			},
			expected: 0, // TotalWeight uses actual value, not default
		},
		{
			name: "multiple workflows with weights",
			config: Config{
				Workflows: map[string]Definition{
					"w1": {Weight: 5},
					"w2": {Weight: 10},
					"w3": {Weight: 3},
				},
			},
			expected: 18,
		},
		{
			name: "excludes disabled workflows",
			config: Config{
				Workflows: map[string]Definition{
					"enabled":  {Weight: 10},
					"disabled": {Weight: 100, Disabled: true},
				},
			},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.TotalWeight()
			assert.Equal(t, tt.expected, result)
		})
	}
}
