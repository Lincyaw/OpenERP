// Package workflow implements business workflow support for the load generator.
// Workflows define sequences of API calls that form business processes,
// such as creating an order, confirming it, and then shipping it.
package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Errors returned by the workflow package.
var (
	// ErrInvalidWorkflow is returned when a workflow configuration is invalid.
	ErrInvalidWorkflow = errors.New("workflow: invalid workflow configuration")
	// ErrStepFailed is returned when a workflow step fails to execute.
	ErrStepFailed = errors.New("workflow: step execution failed")
	// ErrExtractionFailed is returned when value extraction from response fails.
	ErrExtractionFailed = errors.New("workflow: value extraction failed")
	// ErrMissingParameter is returned when a required parameter is not available.
	ErrMissingParameter = errors.New("workflow: missing required parameter")
	// ErrWorkflowAborted is returned when a workflow is aborted due to context cancellation.
	ErrWorkflowAborted = errors.New("workflow: workflow aborted")
	// ErrInvalidJSONPath is returned when a JSONPath expression is invalid.
	ErrInvalidJSONPath = errors.New("workflow: invalid JSONPath expression")
)

// Config holds the configuration for all workflows.
type Config struct {
	// Workflows maps workflow names to their definitions.
	Workflows map[string]Definition `yaml:"workflows" json:"workflows"`
}

// Definition defines a single workflow.
type Definition struct {
	// Name is the unique identifier for this workflow.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Description provides context about the workflow's purpose.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Weight determines how often this workflow is selected relative to others.
	// Higher weight = more frequent selection.
	// Default: 1
	Weight int `yaml:"weight,omitempty" json:"weight,omitempty"`

	// Steps is the ordered sequence of API calls to execute.
	Steps []Step `yaml:"steps" json:"steps"`

	// Timeout is the maximum duration for the entire workflow.
	// Default: 60s
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// ContinueOnError determines whether to continue executing steps after a failure.
	// Default: false (abort on first failure)
	ContinueOnError bool `yaml:"continueOnError,omitempty" json:"continueOnError,omitempty"`

	// Disabled indicates whether this workflow is disabled.
	Disabled bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`

	// Tags are used to categorize workflows.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Step defines a single step in a workflow.
type Step struct {
	// Name is an optional identifier for this step (used for logging).
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Endpoint is the API endpoint to call.
	// Can include placeholders like {order_id} that will be replaced with extracted values.
	// Example: "POST /trade/sales-orders/{order_id}/confirm"
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Body is an optional request body template.
	// Can include placeholders that will be replaced with extracted values.
	Body string `yaml:"body,omitempty" json:"body,omitempty"`

	// Headers are additional headers for this step.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// QueryParams are query parameters for this step.
	QueryParams map[string]string `yaml:"queryParams,omitempty" json:"queryParams,omitempty"`

	// Extract defines values to extract from the response for use in subsequent steps.
	// Key is the variable name, value is a JSONPath expression.
	// Example: {"order_id": "$.data.id", "item_ids": "$.data.items[*].id"}
	Extract map[string]string `yaml:"extract,omitempty" json:"extract,omitempty"`

	// ExpectedStatus is the expected HTTP status code.
	// Default: 200 for GET, 201 for POST, 204 for DELETE
	ExpectedStatus int `yaml:"expectedStatus,omitempty" json:"expectedStatus,omitempty"`

	// Delay is the delay before executing this step.
	// Useful for simulating think time between actions.
	Delay string `yaml:"delay,omitempty" json:"delay,omitempty"`

	// Condition is an optional expression that must evaluate to true for this step to execute.
	// Example: "{{.order_id}}" (step runs only if order_id is not empty)
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`

	// OnFailure specifies what to do if this step fails.
	// Options: "abort", "continue", "retry"
	// Default: "abort"
	OnFailure string `yaml:"onFailure,omitempty" json:"onFailure,omitempty"`

	// RetryCount is the number of times to retry on failure (if OnFailure is "retry").
	// Default: 0
	RetryCount int `yaml:"retryCount,omitempty" json:"retryCount,omitempty"`

	// RetryDelay is the delay between retries.
	// Default: 1s
	RetryDelay string `yaml:"retryDelay,omitempty" json:"retryDelay,omitempty"`
}

// placeholderPattern matches placeholders like {variable_name} in endpoints and bodies.
var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// Validate validates the workflow configuration.
func (c *Config) Validate() error {
	if c.Workflows == nil {
		return nil // Empty config is valid
	}

	for name, def := range c.Workflows {
		if err := def.Validate(name); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates a workflow definition.
func (d *Definition) Validate(name string) error {
	if name == "" && d.Name == "" {
		return fmt.Errorf("%w: workflow name is required", ErrInvalidWorkflow)
	}

	if len(d.Steps) == 0 {
		return fmt.Errorf("%w: workflow %q has no steps", ErrInvalidWorkflow, name)
	}

	for i, step := range d.Steps {
		if err := step.Validate(i); err != nil {
			return fmt.Errorf("%w: workflow %q step %d: %v", ErrInvalidWorkflow, name, i+1, err)
		}
	}

	return nil
}

// Validate validates a step configuration.
func (s *Step) Validate(index int) error {
	if s.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	// Validate endpoint format (should be "METHOD /path")
	parts := strings.SplitN(s.Endpoint, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("endpoint must be in format 'METHOD /path', got %q", s.Endpoint)
	}

	method := strings.ToUpper(parts[0])
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	// Validate OnFailure option
	if s.OnFailure != "" {
		validOptions := map[string]bool{
			"abort": true, "continue": true, "retry": true,
		}
		if !validOptions[strings.ToLower(s.OnFailure)] {
			return fmt.Errorf("invalid onFailure option: %s (must be abort, continue, or retry)", s.OnFailure)
		}
	}

	// Validate retry configuration
	if strings.ToLower(s.OnFailure) == "retry" && s.RetryCount < 0 {
		return fmt.Errorf("retryCount must be non-negative")
	}

	// Validate extract map
	for varName, jsonPath := range s.Extract {
		if varName == "" {
			return fmt.Errorf("extract variable name cannot be empty")
		}
		if jsonPath == "" {
			return fmt.Errorf("extract JSONPath for %q cannot be empty", varName)
		}
		if !strings.HasPrefix(jsonPath, "$.") && !strings.HasPrefix(jsonPath, "$[") {
			return fmt.Errorf("extract JSONPath for %q must start with '$.' or '$[', got %q", varName, jsonPath)
		}
	}

	return nil
}

// ApplyDefaults applies default values to a workflow definition.
func (d *Definition) ApplyDefaults(name string) {
	if d.Name == "" {
		d.Name = name
	}
	if d.Weight <= 0 {
		d.Weight = 1
	}
	if d.Timeout == "" {
		d.Timeout = "60s"
	}

	for i := range d.Steps {
		d.Steps[i].ApplyDefaults()
	}
}

// ApplyDefaults applies default values to a step.
func (s *Step) ApplyDefaults() {
	if s.OnFailure == "" {
		s.OnFailure = "abort"
	}
	if s.RetryDelay == "" {
		s.RetryDelay = "1s"
	}

	// Set default expected status based on method
	if s.ExpectedStatus == 0 {
		parts := strings.SplitN(s.Endpoint, " ", 2)
		if len(parts) >= 1 {
			switch strings.ToUpper(parts[0]) {
			case "POST":
				s.ExpectedStatus = 201
			case "DELETE":
				s.ExpectedStatus = 204
			default:
				s.ExpectedStatus = 200
			}
		}
	}
}

// GetMethod returns the HTTP method from the endpoint string.
func (s *Step) GetMethod() string {
	parts := strings.SplitN(s.Endpoint, " ", 2)
	if len(parts) >= 1 {
		return strings.ToUpper(parts[0])
	}
	return ""
}

// GetPath returns the path from the endpoint string.
func (s *Step) GetPath() string {
	parts := strings.SplitN(s.Endpoint, " ", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// GetPlaceholders returns a list of placeholder variable names in the endpoint.
func (s *Step) GetPlaceholders() []string {
	matches := placeholderPattern.FindAllStringSubmatch(s.Endpoint, -1)
	result := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) >= 2 && !seen[match[1]] {
			result = append(result, match[1])
			seen[match[1]] = true
		}
	}

	// Also check body for placeholders
	if s.Body != "" {
		bodyMatches := placeholderPattern.FindAllStringSubmatch(s.Body, -1)
		for _, match := range bodyMatches {
			if len(match) >= 2 && !seen[match[1]] {
				result = append(result, match[1])
				seen[match[1]] = true
			}
		}
	}

	return result
}

// ReplacePlaceholders replaces placeholders in a string with values from the context.
func ReplacePlaceholders(template string, context map[string]any) string {
	if template == "" || context == nil {
		return template
	}

	result := placeholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name from {name}
		varName := match[1 : len(match)-1]
		if val, ok := context[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match // Keep original if not found
	})

	return result
}

// GetEnabledWorkflows returns all non-disabled workflow definitions.
func (c *Config) GetEnabledWorkflows() map[string]Definition {
	result := make(map[string]Definition)
	for name, def := range c.Workflows {
		if !def.Disabled {
			result[name] = def
		}
	}
	return result
}

// TotalWeight returns the sum of all enabled workflow weights.
func (c *Config) TotalWeight() int {
	total := 0
	for _, def := range c.Workflows {
		if !def.Disabled {
			total += def.Weight
		}
	}
	return total
}
