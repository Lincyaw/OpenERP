// Package circuit provides circuit-board-like components for the load generator.
// This file implements the DependencyGraph for managing producer-consumer relationships.
package circuit

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Errors returned by DependencyGraph operations.
var (
	// ErrCycleDetected is returned when a circular dependency is found.
	ErrCycleDetected = errors.New("dependency graph: circular dependency detected")
	// ErrEndpointNotFound is returned when a referenced endpoint doesn't exist.
	ErrEndpointNotFound = errors.New("dependency graph: endpoint not found")
	// ErrNoProducers is returned when no producers exist for a required semantic type.
	ErrNoProducers = errors.New("dependency graph: no producers for semantic type")
)

// EndpointUnit represents an API endpoint in the dependency graph.
// It wraps endpoint configuration with input/output pin information.
type EndpointUnit struct {
	// Name is the unique identifier for this endpoint.
	Name string

	// Path is the URL path of the endpoint.
	Path string

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH).
	Method string

	// InputPins are the semantic types this endpoint consumes.
	InputPins []SemanticType

	// OutputPins are the semantic types this endpoint produces.
	OutputPins []SemanticType

	// DependsOn lists explicit endpoint dependencies (by name).
	DependsOn []string

	// Weight is the endpoint weight for selection.
	Weight int

	// Disabled indicates whether this endpoint is disabled.
	Disabled bool
}

// DependencyGraph manages producer-consumer relationships between endpoints.
// It provides cycle detection, topological sorting, and execution plan generation.
//
// Thread Safety: All public methods are safe for concurrent use.
type DependencyGraph struct {
	mu sync.RWMutex

	// endpoints maps endpoint name to EndpointUnit.
	endpoints map[string]*EndpointUnit

	// producers maps SemanticType to endpoints that produce it.
	producers map[SemanticType][]*EndpointUnit

	// consumers maps SemanticType to endpoints that consume it.
	consumers map[SemanticType][]*EndpointUnit

	// adjacency is the dependency adjacency list (endpoint -> dependencies).
	adjacency map[string][]string

	// reverseAdjacency is the reverse adjacency list (endpoint -> dependents).
	reverseAdjacency map[string][]string
}

// NewDependencyGraph creates a new empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		endpoints:        make(map[string]*EndpointUnit),
		producers:        make(map[SemanticType][]*EndpointUnit),
		consumers:        make(map[SemanticType][]*EndpointUnit),
		adjacency:        make(map[string][]string),
		reverseAdjacency: make(map[string][]string),
	}
}

// AddEndpoint adds an endpoint to the graph.
// It automatically builds producer-consumer relationships based on semantic types.
func (g *DependencyGraph) AddEndpoint(unit *EndpointUnit) {
	if unit == nil || unit.Name == "" {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Store the endpoint
	g.endpoints[unit.Name] = unit

	// Register as producer for each output semantic type
	for _, outputType := range unit.OutputPins {
		if outputType != "" && outputType != UnknownSemanticType {
			g.producers[outputType] = append(g.producers[outputType], unit)
		}
	}

	// Register as consumer for each input semantic type
	for _, inputType := range unit.InputPins {
		if inputType != "" && inputType != UnknownSemanticType {
			g.consumers[inputType] = append(g.consumers[inputType], unit)
		}
	}

	// Initialize adjacency lists
	if _, exists := g.adjacency[unit.Name]; !exists {
		g.adjacency[unit.Name] = []string{}
	}
	if _, exists := g.reverseAdjacency[unit.Name]; !exists {
		g.reverseAdjacency[unit.Name] = []string{}
	}
}

// BuildDependencies builds the dependency graph based on producer-consumer relationships.
// This should be called after all endpoints have been added.
// It creates edges from consumers to their producers (consumer depends on producer).
func (g *DependencyGraph) BuildDependencies() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Clear existing adjacency lists but keep endpoints
	g.adjacency = make(map[string][]string)
	g.reverseAdjacency = make(map[string][]string)

	// Initialize adjacency lists for all endpoints
	for name := range g.endpoints {
		g.adjacency[name] = []string{}
		g.reverseAdjacency[name] = []string{}
	}

	// Build dependencies based on semantic types
	for _, unit := range g.endpoints {
		deps := make(map[string]bool)

		// Add explicit dependencies
		for _, depName := range unit.DependsOn {
			if _, exists := g.endpoints[depName]; exists {
				deps[depName] = true
			}
		}

		// Add implicit dependencies based on consumed semantic types
		for _, inputType := range unit.InputPins {
			producers := g.producers[inputType]
			for _, producer := range producers {
				// Don't add self-dependency
				if producer.Name != unit.Name {
					deps[producer.Name] = true
				}
			}
		}

		// Convert to sorted slice for deterministic ordering
		depList := make([]string, 0, len(deps))
		for dep := range deps {
			depList = append(depList, dep)
		}
		sort.Strings(depList)

		g.adjacency[unit.Name] = depList

		// Build reverse adjacency
		for _, dep := range depList {
			g.reverseAdjacency[dep] = append(g.reverseAdjacency[dep], unit.Name)
		}
	}

	// Sort reverse adjacency lists for deterministic ordering
	for name := range g.reverseAdjacency {
		sort.Strings(g.reverseAdjacency[name])
	}
}

// GetProducers returns all endpoints that produce the given semantic type.
func (g *DependencyGraph) GetProducers(semanticType SemanticType) []*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	producers := g.producers[semanticType]
	// Return a copy to prevent external modification
	result := make([]*EndpointUnit, len(producers))
	copy(result, producers)
	return result
}

// GetConsumers returns all endpoints that consume the given semantic type.
func (g *DependencyGraph) GetConsumers(semanticType SemanticType) []*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	consumers := g.consumers[semanticType]
	// Return a copy to prevent external modification
	result := make([]*EndpointUnit, len(consumers))
	copy(result, consumers)
	return result
}

// GetEndpoint returns an endpoint by name.
func (g *DependencyGraph) GetEndpoint(name string) *EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.endpoints[name]
}

// GetAllEndpoints returns all endpoints in the graph.
func (g *DependencyGraph) GetAllEndpoints() []*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]*EndpointUnit, 0, len(g.endpoints))
	for _, unit := range g.endpoints {
		result = append(result, unit)
	}

	// Sort by name for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetDependencies returns the direct dependencies of an endpoint.
func (g *DependencyGraph) GetDependencies(name string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	deps := g.adjacency[name]
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// GetDependents returns endpoints that depend on the given endpoint.
func (g *DependencyGraph) GetDependents(name string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	deps := g.reverseAdjacency[name]
	result := make([]string, len(deps))
	copy(result, deps)
	return result
}

// Cycle represents a detected circular dependency.
type Cycle struct {
	// Path is the sequence of endpoint names forming the cycle.
	Path []string
}

// String returns a human-readable representation of the cycle.
func (c Cycle) String() string {
	if len(c.Path) == 0 {
		return "empty cycle"
	}
	result := c.Path[0]
	for i := 1; i < len(c.Path); i++ {
		result += " -> " + c.Path[i]
	}
	result += " -> " + c.Path[0] // Complete the cycle
	return result
}

// DetectCycles detects all circular dependencies in the graph.
// Returns nil if no cycles are found.
func (g *DependencyGraph) DetectCycles() []Cycle {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var cycles []Cycle
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	// Get sorted endpoint names for deterministic ordering
	names := make([]string, 0, len(g.endpoints))
	for name := range g.endpoints {
		names = append(names, name)
	}
	sort.Strings(names)

	var dfs func(name string) bool
	dfs = func(name string) bool {
		visited[name] = true
		recStack[name] = true
		path = append(path, name)

		for _, dep := range g.adjacency[name] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				// Found a cycle - extract the cycle path
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := make([]string, len(path)-cycleStart)
					copy(cyclePath, path[cycleStart:])
					cycles = append(cycles, Cycle{Path: cyclePath})
				}
				return true
			}
		}

		path = path[:len(path)-1]
		recStack[name] = false
		return false
	}

	for _, name := range names {
		if !visited[name] {
			dfs(name)
		}
	}

	return cycles
}

// HasCycles returns true if the graph contains any circular dependencies.
func (g *DependencyGraph) HasCycles() bool {
	cycles := g.DetectCycles()
	return len(cycles) > 0
}

// TopologicalSort returns endpoints in topological order (dependencies first).
// Returns an error if the graph contains cycles.
func (g *DependencyGraph) TopologicalSort() ([]*EndpointUnit, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)
	for name := range g.endpoints {
		inDegree[name] = len(g.adjacency[name])
	}

	// Find all nodes with no dependencies (in-degree 0)
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Deterministic ordering

	result := make([]*EndpointUnit, 0, len(g.endpoints))
	processed := 0

	for len(queue) > 0 {
		// Sort queue for deterministic ordering
		sort.Strings(queue)

		// Take the first element
		current := queue[0]
		queue = queue[1:]

		result = append(result, g.endpoints[current])
		processed++

		// Reduce in-degree for all dependents
		for _, dependent := range g.reverseAdjacency[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if processed != len(g.endpoints) {
		return nil, ErrCycleDetected
	}

	return result, nil
}

// ExecutionPlan represents a plan for executing endpoints in dependency order.
type ExecutionPlan struct {
	// Target is the target endpoint name.
	Target string

	// Steps are the endpoints to execute in order.
	Steps []*EndpointUnit

	// Dependencies maps each endpoint to its direct dependencies.
	Dependencies map[string][]string
}

// GetExecutionPlan generates an execution plan for the target endpoint.
// The plan includes all transitive dependencies in topological order.
func (g *DependencyGraph) GetExecutionPlan(target string) (*ExecutionPlan, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check if target exists
	if _, exists := g.endpoints[target]; !exists {
		return nil, fmt.Errorf("%w: %s", ErrEndpointNotFound, target)
	}

	// Find all transitive dependencies using BFS
	required := make(map[string]bool)
	queue := []string{target}
	required[target] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range g.adjacency[current] {
			if !required[dep] {
				required[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	// Build a subgraph with only required endpoints
	subgraphInDegree := make(map[string]int)
	subgraphDeps := make(map[string][]string)

	for name := range required {
		deps := []string{}
		for _, dep := range g.adjacency[name] {
			if required[dep] {
				deps = append(deps, dep)
			}
		}
		subgraphDeps[name] = deps
		subgraphInDegree[name] = len(deps)
	}

	// Topological sort on the subgraph
	queue = make([]string, 0)
	for name, degree := range subgraphInDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	steps := make([]*EndpointUnit, 0, len(required))
	processed := 0

	for len(queue) > 0 {
		sort.Strings(queue)
		current := queue[0]
		queue = queue[1:]

		steps = append(steps, g.endpoints[current])
		processed++

		// Find dependents in the subgraph
		for name := range required {
			for i, dep := range subgraphDeps[name] {
				if dep == current {
					// Remove this dependency
					subgraphDeps[name] = append(subgraphDeps[name][:i], subgraphDeps[name][i+1:]...)
					subgraphInDegree[name]--
					if subgraphInDegree[name] == 0 {
						queue = append(queue, name)
					}
					break
				}
			}
		}
	}

	if processed != len(required) {
		return nil, ErrCycleDetected
	}

	// Build dependencies map for the plan
	planDeps := make(map[string][]string)
	for name := range required {
		deps := []string{}
		for _, dep := range g.adjacency[name] {
			if required[dep] {
				deps = append(deps, dep)
			}
		}
		sort.Strings(deps)
		planDeps[name] = deps
	}

	return &ExecutionPlan{
		Target:       target,
		Steps:        steps,
		Dependencies: planDeps,
	}, nil
}

// GetRootEndpoints returns endpoints with no dependencies (can be executed first).
func (g *DependencyGraph) GetRootEndpoints() []*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var roots []*EndpointUnit
	for name, deps := range g.adjacency {
		if len(deps) == 0 {
			roots = append(roots, g.endpoints[name])
		}
	}

	// Sort for deterministic ordering
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})

	return roots
}

// GetLeafEndpoints returns endpoints with no dependents (nothing depends on them).
func (g *DependencyGraph) GetLeafEndpoints() []*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var leaves []*EndpointUnit
	for name, deps := range g.reverseAdjacency {
		if len(deps) == 0 {
			leaves = append(leaves, g.endpoints[name])
		}
	}

	// Sort for deterministic ordering
	sort.Slice(leaves, func(i, j int) bool {
		return leaves[i].Name < leaves[j].Name
	})

	return leaves
}

// Stats returns statistics about the dependency graph.
func (g *DependencyGraph) Stats() GraphStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := GraphStats{
		TotalEndpoints:     len(g.endpoints),
		TotalSemanticTypes: len(g.producers) + len(g.consumers),
		ProducerTypes:      len(g.producers),
		ConsumerTypes:      len(g.consumers),
		Connections:        0,
	}

	// Count total edges
	for _, deps := range g.adjacency {
		stats.Connections += len(deps)
	}

	// Count roots and leaves
	for _, deps := range g.adjacency {
		if len(deps) == 0 {
			stats.RootEndpoints++
		}
	}
	for _, deps := range g.reverseAdjacency {
		if len(deps) == 0 {
			stats.LeafEndpoints++
		}
	}

	// Calculate max depth
	stats.MaxDepth = g.calculateMaxDepthLocked()

	return stats
}

// calculateMaxDepthLocked calculates the maximum dependency depth.
// Must be called with read lock held.
// Handles cyclic graphs safely by tracking visiting state.
func (g *DependencyGraph) calculateMaxDepthLocked() int {
	if len(g.endpoints) == 0 {
		return 0
	}

	depths := make(map[string]int)
	visiting := make(map[string]bool) // Track nodes being visited for cycle detection

	var calculateDepth func(name string) int
	calculateDepth = func(name string) int {
		if d, ok := depths[name]; ok {
			return d
		}

		// Cycle detection: if we're already visiting this node, return 0
		if visiting[name] {
			return 0
		}
		visiting[name] = true
		defer func() { visiting[name] = false }()

		deps := g.adjacency[name]
		if len(deps) == 0 {
			depths[name] = 0
			return 0
		}

		maxDep := 0
		for _, dep := range deps {
			d := calculateDepth(dep)
			if d > maxDep {
				maxDep = d
			}
		}

		depths[name] = maxDep + 1
		return maxDep + 1
	}

	maxDepth := 0
	for name := range g.endpoints {
		d := calculateDepth(name)
		if d > maxDepth {
			maxDepth = d
		}
	}

	return maxDepth
}

// GraphStats holds statistics about the dependency graph.
type GraphStats struct {
	// TotalEndpoints is the number of endpoints in the graph.
	TotalEndpoints int

	// TotalSemanticTypes is the number of unique semantic types.
	TotalSemanticTypes int

	// ProducerTypes is the number of semantic types with producers.
	ProducerTypes int

	// ConsumerTypes is the number of semantic types with consumers.
	ConsumerTypes int

	// Connections is the total number of dependency edges.
	Connections int

	// RootEndpoints is the number of endpoints with no dependencies.
	RootEndpoints int

	// LeafEndpoints is the number of endpoints with no dependents.
	LeafEndpoints int

	// MaxDepth is the maximum dependency chain depth.
	MaxDepth int
}

// ProducersMap returns a copy of the producers map.
func (g *DependencyGraph) ProducersMap() map[SemanticType][]*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[SemanticType][]*EndpointUnit, len(g.producers))
	for k, v := range g.producers {
		copied := make([]*EndpointUnit, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

// ConsumersMap returns a copy of the consumers map.
func (g *DependencyGraph) ConsumersMap() map[SemanticType][]*EndpointUnit {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[SemanticType][]*EndpointUnit, len(g.consumers))
	for k, v := range g.consumers {
		copied := make([]*EndpointUnit, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

// Clear removes all endpoints and relationships from the graph.
func (g *DependencyGraph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.endpoints = make(map[string]*EndpointUnit)
	g.producers = make(map[SemanticType][]*EndpointUnit)
	g.consumers = make(map[SemanticType][]*EndpointUnit)
	g.adjacency = make(map[string][]string)
	g.reverseAdjacency = make(map[string][]string)
}

// Size returns the number of endpoints in the graph.
func (g *DependencyGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.endpoints)
}
