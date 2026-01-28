// Package workflow implements business workflow support for the load generator.
package workflow

import (
	"crypto/rand"
	"math/big"
	"sort"
	"sync"
)

// Selector selects workflows based on their weights.
type Selector struct {
	mu          sync.RWMutex
	workflows   []weightedWorkflow
	totalWeight int64
}

type weightedWorkflow struct {
	name             string
	def              Definition
	cumulativeWeight int64
}

// NewSelector creates a new workflow selector from a configuration.
func NewSelector(config *Config) *Selector {
	s := &Selector{
		workflows: make([]weightedWorkflow, 0),
	}

	if config == nil || config.Workflows == nil {
		return s
	}

	s.updateWorkflows(config.Workflows)
	return s
}

// updateWorkflows updates the selector with new workflow definitions.
func (s *Selector) updateWorkflows(workflows map[string]Definition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get enabled workflows with defaults applied
	enabledWorkflows := make([]weightedWorkflow, 0, len(workflows))
	var cumulative int64 = 0

	// Sort workflow names for deterministic ordering
	names := make([]string, 0, len(workflows))
	for name := range workflows {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		def := workflows[name]
		if def.Disabled {
			continue
		}

		// Apply defaults
		def.ApplyDefaults(name)

		weight := int64(def.Weight)
		if weight <= 0 {
			weight = 1
		}

		cumulative += weight
		enabledWorkflows = append(enabledWorkflows, weightedWorkflow{
			name:             name,
			def:              def,
			cumulativeWeight: cumulative,
		})
	}

	s.workflows = enabledWorkflows
	s.totalWeight = cumulative
}

// Select randomly selects a workflow based on weights.
// Returns the workflow name and definition, or empty values if no workflows are available.
func (s *Selector) Select() (string, Definition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.workflows) == 0 || s.totalWeight == 0 {
		return "", Definition{}, false
	}

	// Generate random number in range [0, totalWeight)
	randomVal, err := rand.Int(rand.Reader, big.NewInt(s.totalWeight))
	if err != nil {
		// Fallback to first workflow on error
		return s.workflows[0].name, s.workflows[0].def, true
	}

	target := randomVal.Int64()

	// Binary search for the workflow
	idx := sort.Search(len(s.workflows), func(i int) bool {
		return s.workflows[i].cumulativeWeight > target
	})

	if idx >= len(s.workflows) {
		idx = len(s.workflows) - 1
	}

	return s.workflows[idx].name, s.workflows[idx].def, true
}

// SelectByName returns a specific workflow by name.
func (s *Selector) SelectByName(name string) (Definition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ww := range s.workflows {
		if ww.name == name {
			return ww.def, true
		}
	}

	return Definition{}, false
}

// GetAll returns all enabled workflows.
func (s *Selector) GetAll() map[string]Definition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]Definition, len(s.workflows))
	for _, ww := range s.workflows {
		result[ww.name] = ww.def
	}

	return result
}

// Count returns the number of enabled workflows.
func (s *Selector) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.workflows)
}

// TotalWeight returns the total weight of all enabled workflows.
func (s *Selector) TotalWeight() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalWeight
}

// Update updates the selector with new workflow definitions.
func (s *Selector) Update(workflows map[string]Definition) {
	s.updateWorkflows(workflows)
}

// Names returns the names of all enabled workflows.
func (s *Selector) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, len(s.workflows))
	for i, ww := range s.workflows {
		names[i] = ww.name
	}

	return names
}

// GetWeight returns the weight of a specific workflow.
func (s *Selector) GetWeight(name string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ww := range s.workflows {
		if ww.name == name {
			return ww.def.Weight, true
		}
	}

	return 0, false
}
