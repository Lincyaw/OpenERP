// Package scenario provides scenario management for the load generator.
// Scenarios allow predefined test configurations that can override duration,
// traffic shaping, and focus on specific endpoints for targeted testing.
package scenario

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/example/erp/tools/loadgen/internal/loadctrl"
	"gopkg.in/yaml.v3"
)

// Errors returned by the scenario package.
var (
	// ErrScenarioNotFound is returned when a scenario cannot be found.
	ErrScenarioNotFound = errors.New("scenario: not found")
	// ErrInvalidScenario is returned when a scenario configuration is invalid.
	ErrInvalidScenario = errors.New("scenario: invalid configuration")
	// ErrNoScenariosDirectory is returned when the scenarios directory doesn't exist.
	ErrNoScenariosDirectory = errors.New("scenario: scenarios directory not found")
)

// Definition represents a complete scenario definition that can be loaded
// from a YAML file in the scenarios directory.
type Definition struct {
	// Name is the unique identifier for this scenario.
	Name string `yaml:"name" json:"name"`

	// Description provides context about what this scenario tests.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Duration overrides the base test duration when this scenario is active.
	// If zero, the base config duration is used.
	Duration time.Duration `yaml:"duration,omitempty" json:"duration,omitempty"`

	// TrafficShaper overrides the traffic shaping configuration.
	// If nil, the base config traffic shaper is used.
	TrafficShaper *loadctrl.ShaperConfig `yaml:"trafficShaper,omitempty" json:"trafficShaper,omitempty"`

	// FocusEndpoints lists endpoint names to focus on during this scenario.
	// When specified, only these endpoints will receive traffic.
	// If empty, all enabled endpoints are used.
	FocusEndpoints []string `yaml:"focusEndpoints,omitempty" json:"focusEndpoints,omitempty"`

	// EndpointWeights allows overriding weights for specific endpoints.
	// Key is the endpoint name, value is the new weight.
	EndpointWeights map[string]int `yaml:"endpointWeights,omitempty" json:"endpointWeights,omitempty"`

	// DisableEndpoints lists endpoints to disable during this scenario.
	DisableEndpoints []string `yaml:"disableEndpoints,omitempty" json:"disableEndpoints,omitempty"`

	// Tags can be used to filter or categorize scenarios.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Warmup overrides warmup configuration for this scenario.
	Warmup *WarmupOverride `yaml:"warmup,omitempty" json:"warmup,omitempty"`

	// Assertions overrides assertion thresholds for this scenario.
	Assertions *AssertionOverride `yaml:"assertions,omitempty" json:"assertions,omitempty"`

	// Variables provides scenario-specific variables that can be used in templates.
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// WarmupOverride allows overriding warmup configuration in a scenario.
type WarmupOverride struct {
	// Enabled controls whether warmup is performed. If nil, uses base config.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Iterations overrides the warmup iteration count.
	Iterations int `yaml:"iterations,omitempty" json:"iterations,omitempty"`

	// Timeout overrides the warmup timeout.
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// AssertionOverride allows overriding assertion thresholds in a scenario.
type AssertionOverride struct {
	// MaxErrorRate overrides the maximum error rate threshold.
	MaxErrorRate *float64 `yaml:"maxErrorRate,omitempty" json:"maxErrorRate,omitempty"`

	// MinSuccessRate overrides the minimum success rate threshold.
	MinSuccessRate *float64 `yaml:"minSuccessRate,omitempty" json:"minSuccessRate,omitempty"`

	// MaxP95Latency overrides the P95 latency threshold.
	MaxP95Latency time.Duration `yaml:"maxP95Latency,omitempty" json:"maxP95Latency,omitempty"`

	// MaxP99Latency overrides the P99 latency threshold.
	MaxP99Latency time.Duration `yaml:"maxP99Latency,omitempty" json:"maxP99Latency,omitempty"`
}

// Validate validates the scenario definition.
func (d *Definition) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidScenario)
	}

	if d.Duration < 0 {
		return fmt.Errorf("%w: duration cannot be negative", ErrInvalidScenario)
	}

	// Validate endpoint weights
	for name, weight := range d.EndpointWeights {
		if weight < 0 {
			return fmt.Errorf("%w: endpoint weight for %s cannot be negative", ErrInvalidScenario, name)
		}
	}

	// Validate traffic shaper if provided
	if d.TrafficShaper != nil {
		if err := d.TrafficShaper.Validate(); err != nil {
			return fmt.Errorf("%w: traffic shaper: %v", ErrInvalidScenario, err)
		}
	}

	return nil
}

// HasFocusEndpoints returns true if this scenario focuses on specific endpoints.
func (d *Definition) HasFocusEndpoints() bool {
	return len(d.FocusEndpoints) > 0
}

// HasTrafficOverride returns true if this scenario overrides traffic shaping.
func (d *Definition) HasTrafficOverride() bool {
	return d.TrafficShaper != nil
}

// HasDurationOverride returns true if this scenario overrides duration.
func (d *Definition) HasDurationOverride() bool {
	return d.Duration > 0
}

// IsEndpointFocused returns true if the given endpoint is in the focus list,
// or if no focus list is specified (all endpoints are focused).
func (d *Definition) IsEndpointFocused(name string) bool {
	if len(d.FocusEndpoints) == 0 {
		return true
	}
	return slices.Contains(d.FocusEndpoints, name)
}

// IsEndpointDisabled returns true if the given endpoint is in the disable list.
func (d *Definition) IsEndpointDisabled(name string) bool {
	return slices.Contains(d.DisableEndpoints, name)
}

// GetEndpointWeight returns the overridden weight for an endpoint,
// or -1 if no override is specified.
func (d *Definition) GetEndpointWeight(name string) int {
	if weight, ok := d.EndpointWeights[name]; ok {
		return weight
	}
	return -1
}

// File represents a scenario file that can contain multiple scenario definitions.
type File struct {
	// Version is the scenario file format version.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Scenarios contains the scenario definitions in this file.
	Scenarios []Definition `yaml:"scenarios" json:"scenarios"`
}

// LoadFromFile loads a scenario definition from a YAML file.
func LoadFromFile(path string) (*Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, path)
		}
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}

	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing scenario file: %w", err)
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}

	return &def, nil
}

// LoadMultipleFromFile loads multiple scenarios from a single file.
func LoadMultipleFromFile(path string) ([]Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, path)
		}
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err == nil && len(file.Scenarios) > 0 {
		// Successfully parsed as multi-scenario file
		for i := range file.Scenarios {
			if err := file.Scenarios[i].Validate(); err != nil {
				return nil, fmt.Errorf("scenario[%d]: %w", i, err)
			}
		}
		return file.Scenarios, nil
	}

	// Try single definition format
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing scenario file: %w", err)
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}
	return []Definition{def}, nil
}

// Registry manages available scenarios loaded from files and configuration.
type Registry struct {
	scenarios map[string]*Definition
	directory string
}

// NewRegistry creates a new scenario registry.
func NewRegistry() *Registry {
	return &Registry{
		scenarios: make(map[string]*Definition),
	}
}

// SetDirectory sets the scenarios directory path.
func (r *Registry) SetDirectory(dir string) {
	r.directory = dir
}

// Register adds a scenario to the registry.
func (r *Registry) Register(def *Definition) error {
	if err := def.Validate(); err != nil {
		return err
	}
	r.scenarios[def.Name] = def
	return nil
}

// Get retrieves a scenario by name.
func (r *Registry) Get(name string) (*Definition, error) {
	if def, ok := r.scenarios[name]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrScenarioNotFound, name)
}

// List returns all registered scenario names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.scenarios))
	for name := range r.scenarios {
		names = append(names, name)
	}
	return names
}

// All returns all registered scenarios.
func (r *Registry) All() []*Definition {
	defs := make([]*Definition, 0, len(r.scenarios))
	for _, def := range r.scenarios {
		defs = append(defs, def)
	}
	return defs
}

// Count returns the number of registered scenarios.
func (r *Registry) Count() int {
	return len(r.scenarios)
}

// LoadFromDirectory loads all scenarios from the configured directory.
// It scans for .yaml and .yml files and loads all valid scenario definitions.
func (r *Registry) LoadFromDirectory() error {
	if r.directory == "" {
		return nil // No directory configured, skip
	}

	info, err := os.Stat(r.directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, skip silently
		}
		return fmt.Errorf("accessing scenarios directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrNoScenariosDirectory, r.directory)
	}

	// Scan for scenario files
	entries, err := os.ReadDir(r.directory)
	if err != nil {
		return fmt.Errorf("reading scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(r.directory, name)
		defs, err := LoadMultipleFromFile(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", name, err)
		}

		for i := range defs {
			if err := r.Register(&defs[i]); err != nil {
				return fmt.Errorf("registering scenario from %s: %w", name, err)
			}
		}
	}

	return nil
}

// LoadFromConfig loads scenarios from inline configuration.
// This is used for scenarios defined in the main config file's scenarios block.
func (r *Registry) LoadFromConfig(scenarios []InlineScenario) error {
	for i, s := range scenarios {
		def := &Definition{
			Name:           s.Name,
			Description:    s.Description,
			FocusEndpoints: s.Endpoints,
			Tags:           s.Tags,
		}

		// Convert sequential flag to a tag if needed
		if s.Sequential {
			def.Tags = append(def.Tags, "sequential")
		}

		if err := r.Register(def); err != nil {
			return fmt.Errorf("scenario[%d]: %w", i, err)
		}
	}
	return nil
}

// InlineScenario represents a scenario defined inline in the main config.
// This matches the existing ScenarioConfig in config.go.
type InlineScenario struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Endpoints   []string `yaml:"endpoints" json:"endpoints"`
	Weight      int      `yaml:"weight,omitempty" json:"weight,omitempty"`
	Sequential  bool     `yaml:"sequential,omitempty" json:"sequential,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}
