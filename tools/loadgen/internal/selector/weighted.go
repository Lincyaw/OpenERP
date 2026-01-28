// Package selector provides endpoint selection strategies for the load generator.
package selector

import (
	"crypto/rand"
	"errors"
	"maps"
	"math/big"
	"slices"
	"sync"
)

// Errors returned by the selector package.
var (
	// ErrNoEndpoints is returned when there are no endpoints to select from.
	ErrNoEndpoints = errors.New("selector: no endpoints available")
	// ErrInvalidWeight is returned when an endpoint has invalid weight.
	ErrInvalidWeight = errors.New("selector: invalid weight")
	// ErrInvalidRatio is returned when read/write ratio is invalid.
	ErrInvalidRatio = errors.New("selector: invalid read/write ratio")
	// ErrEndpointNotFound is returned when an endpoint is not found.
	ErrEndpointNotFound = errors.New("selector: endpoint not found")
)

// OperationType represents HTTP operation types.
type OperationType string

// Supported operation types.
const (
	OpGET    OperationType = "GET"
	OpPOST   OperationType = "POST"
	OpPUT    OperationType = "PUT"
	OpDELETE OperationType = "DELETE"
	OpPATCH  OperationType = "PATCH"
)

// IsRead returns true if the operation is a read operation (GET).
func (o OperationType) IsRead() bool {
	return o == OpGET
}

// IsWrite returns true if the operation is a write operation (POST, PUT, DELETE, PATCH).
func (o OperationType) IsWrite() bool {
	return o == OpPOST || o == OpPUT || o == OpDELETE || o == OpPATCH
}

// Endpoint represents an API endpoint with selection metadata.
type Endpoint struct {
	// Name is the unique identifier for this endpoint.
	Name string
	// Path is the URL path.
	Path string
	// Method is the HTTP method.
	Method string
	// Category is the endpoint category (e.g., "catalog", "trade", "finance").
	Category string
	// Tags are additional categorization labels.
	Tags []string
	// BaseWeight is the original weight from configuration.
	BaseWeight int
}

// Operation returns the operation type based on HTTP method.
func (e *Endpoint) Operation() OperationType {
	return OperationType(e.Method)
}

// WeightedSelectorConfig configures the weighted selector.
type WeightedSelectorConfig struct {
	// ReadWriteRatio is the global read/write ratio (0.0 to 1.0).
	// 0.8 means 80% read operations, 20% write operations.
	// Default: 0.8 (use a negative value to indicate "unset" and apply default)
	// Set to 0.0 explicitly for 100% write operations.
	ReadWriteRatio float64 `yaml:"readWriteRatio" json:"readWriteRatio"`

	// ReadWriteRatioSet indicates whether ReadWriteRatio was explicitly set.
	// This allows distinguishing between "unset" and "set to 0".
	ReadWriteRatioSet bool `yaml:"-" json:"-"`

	// Categories configures category-level weights.
	// Key: category name, Value: weight (higher = more frequent).
	Categories map[string]int `yaml:"categories,omitempty" json:"categories,omitempty"`

	// CategoryRatios configures per-category read/write ratios.
	// Key: category name, Value: read ratio (0.0 to 1.0).
	// If set, overrides the global ReadWriteRatio for that category.
	CategoryRatios map[string]float64 `yaml:"categoryRatios,omitempty" json:"categoryRatios,omitempty"`

	// Weights configures endpoint-specific weights.
	// Key: endpoint name, Value: weight (higher = more frequent).
	Weights map[string]int `yaml:"weights,omitempty" json:"weights,omitempty"`

	// Operations configures operation type weights.
	// Key: operation type (GET, POST, PUT, DELETE), Value: weight.
	Operations map[OperationType]int `yaml:"operations,omitempty" json:"operations,omitempty"`
}

// Validate validates the configuration.
func (c *WeightedSelectorConfig) Validate() error {
	if c.ReadWriteRatio < 0 || c.ReadWriteRatio > 1 {
		return ErrInvalidRatio
	}

	for cat, ratio := range c.CategoryRatios {
		if ratio < 0 || ratio > 1 {
			return errors.New("selector: invalid category ratio for " + cat)
		}
	}

	for name, weight := range c.Categories {
		if weight < 0 {
			return errors.New("selector: invalid category weight for " + name)
		}
	}

	for name, weight := range c.Weights {
		if weight < 0 {
			return errors.New("selector: invalid endpoint weight for " + name)
		}
	}

	for op, weight := range c.Operations {
		if weight < 0 {
			return errors.New("selector: invalid operation weight for " + string(op))
		}
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *WeightedSelectorConfig) ApplyDefaults() {
	// Only apply default ReadWriteRatio if not explicitly set
	if !c.ReadWriteRatioSet && c.ReadWriteRatio == 0 {
		c.ReadWriteRatio = 0.8 // 80% read, 20% write
	}

	if c.Categories == nil {
		c.Categories = make(map[string]int)
	}

	if c.CategoryRatios == nil {
		c.CategoryRatios = make(map[string]float64)
	}

	if c.Weights == nil {
		c.Weights = make(map[string]int)
	}

	if c.Operations == nil {
		c.Operations = make(map[OperationType]int)
	}
}

// weightedEntry represents an entry in the weighted selection pool.
type weightedEntry struct {
	endpoint         *Endpoint
	effectiveWeight  int
	cumulativeWeight int
}

// WeightedSelector selects endpoints based on configured weights.
type WeightedSelector struct {
	mu     sync.RWMutex
	config WeightedSelectorConfig

	// endpoints holds all registered endpoints.
	endpoints map[string]*Endpoint

	// readEntries holds weighted entries for read operations.
	readEntries []weightedEntry

	// writeEntries holds weighted entries for write operations.
	writeEntries []weightedEntry

	// totalReadWeight is the sum of all read endpoint weights.
	totalReadWeight int

	// totalWriteWeight is the sum of all write endpoint weights.
	totalWriteWeight int

	// categoryTotalWeights stores the total weight per category.
	categoryTotalWeights map[string]int
}

// NewWeightedSelector creates a new weighted selector.
func NewWeightedSelector(config WeightedSelectorConfig) (*WeightedSelector, error) {
	configCopy := config
	if err := configCopy.Validate(); err != nil {
		return nil, err
	}
	configCopy.ApplyDefaults()

	return &WeightedSelector{
		config:               configCopy,
		endpoints:            make(map[string]*Endpoint),
		readEntries:          make([]weightedEntry, 0),
		writeEntries:         make([]weightedEntry, 0),
		categoryTotalWeights: make(map[string]int),
	}, nil
}

// NewConfigWithRatio creates a config with an explicit read/write ratio.
// Use this when you want to set the ratio to 0 (100% write).
func NewConfigWithRatio(ratio float64) WeightedSelectorConfig {
	return WeightedSelectorConfig{
		ReadWriteRatio:    ratio,
		ReadWriteRatioSet: true,
	}
}

// Register adds an endpoint to the selector.
func (s *WeightedSelector) Register(endpoint *Endpoint) error {
	if endpoint == nil {
		return ErrNoEndpoints
	}

	if endpoint.Name == "" {
		return errors.New("selector: endpoint name is required")
	}

	if endpoint.BaseWeight < 0 {
		return ErrInvalidWeight
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the endpoint
	s.endpoints[endpoint.Name] = endpoint

	// Rebuild the weighted pools
	s.rebuildPools()

	return nil
}

// RegisterAll adds multiple endpoints to the selector.
func (s *WeightedSelector) RegisterAll(endpoints []*Endpoint) error {
	if len(endpoints) == 0 {
		return ErrNoEndpoints
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate all endpoints first
	for _, ep := range endpoints {
		if ep == nil {
			return ErrNoEndpoints
		}
		if ep.Name == "" {
			return errors.New("selector: endpoint name is required")
		}
		if ep.BaseWeight < 0 {
			return ErrInvalidWeight
		}
	}

	// Store all endpoints
	for _, ep := range endpoints {
		s.endpoints[ep.Name] = ep
	}

	// Rebuild the weighted pools
	s.rebuildPools()

	return nil
}

// Unregister removes an endpoint from the selector.
func (s *WeightedSelector) Unregister(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.endpoints[name]; !exists {
		return ErrEndpointNotFound
	}

	delete(s.endpoints, name)
	s.rebuildPools()

	return nil
}

// rebuildPools recalculates the weighted entry pools.
// Must be called with s.mu held.
func (s *WeightedSelector) rebuildPools() {
	s.readEntries = s.readEntries[:0]
	s.writeEntries = s.writeEntries[:0]
	s.totalReadWeight = 0
	s.totalWriteWeight = 0
	s.categoryTotalWeights = make(map[string]int)

	// Sort endpoint names for deterministic ordering
	names := make([]string, 0, len(s.endpoints))
	for name := range s.endpoints {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		ep := s.endpoints[name]
		weight := s.calculateEffectiveWeight(ep)
		if weight == 0 {
			continue
		}

		entry := weightedEntry{
			endpoint:        ep,
			effectiveWeight: weight,
		}

		if ep.Operation().IsRead() {
			s.totalReadWeight += weight
			entry.cumulativeWeight = s.totalReadWeight
			s.readEntries = append(s.readEntries, entry)
		} else {
			s.totalWriteWeight += weight
			entry.cumulativeWeight = s.totalWriteWeight
			s.writeEntries = append(s.writeEntries, entry)
		}

		// Track category weights
		if ep.Category != "" {
			s.categoryTotalWeights[ep.Category] += weight
		}
	}
}

// calculateEffectiveWeight computes the final weight for an endpoint.
// Weight is influenced by:
// 1. Endpoint-specific weight (highest priority)
// 2. Category weight
// 3. Operation type weight
// 4. Base weight (from endpoint config)
func (s *WeightedSelector) calculateEffectiveWeight(ep *Endpoint) int {
	baseWeight := ep.BaseWeight
	if baseWeight == 0 {
		baseWeight = 1
	}

	// Check for endpoint-specific weight override (highest priority)
	if weight, exists := s.config.Weights[ep.Name]; exists {
		baseWeight = weight
	}

	// Apply category weight multiplier
	categoryMultiplier := 1.0
	if ep.Category != "" {
		if catWeight, exists := s.config.Categories[ep.Category]; exists {
			categoryMultiplier = float64(catWeight)
		}
	}

	// Apply operation weight multiplier
	operationMultiplier := 1.0
	if opWeight, exists := s.config.Operations[ep.Operation()]; exists {
		operationMultiplier = float64(opWeight)
	}

	// Calculate effective weight (minimum 1 if any multiplier > 0)
	effectiveWeight := float64(baseWeight) * categoryMultiplier * operationMultiplier

	// Handle edge cases
	if effectiveWeight < 1 && effectiveWeight > 0 {
		return 1
	}

	// Overflow protection: cap at max int32 for portability
	const maxWeight = 1<<31 - 1
	if effectiveWeight > float64(maxWeight) {
		return maxWeight
	}

	return int(effectiveWeight)
}

// Select chooses an endpoint based on weighted random selection.
// It first decides between read/write based on the ratio, then selects
// from the appropriate pool.
func (s *WeightedSelector) Select() (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Determine if this should be a read or write operation
	isRead, err := s.shouldBeRead()
	if err != nil {
		return nil, err
	}

	if isRead {
		return s.selectFromPool(s.readEntries, s.totalReadWeight)
	}
	return s.selectFromPool(s.writeEntries, s.totalWriteWeight)
}

// SelectRead selects only from read endpoints.
func (s *WeightedSelector) SelectRead() (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.readEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	return s.selectFromPool(s.readEntries, s.totalReadWeight)
}

// SelectWrite selects only from write endpoints.
func (s *WeightedSelector) SelectWrite() (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.writeEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	return s.selectFromPool(s.writeEntries, s.totalWriteWeight)
}

// SelectByCategory selects an endpoint from a specific category.
func (s *WeightedSelector) SelectByCategory(category string) (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build a temporary pool for this category
	var entries []weightedEntry
	totalWeight := 0

	// Get the read/write ratio for this category
	ratio := s.config.ReadWriteRatio
	if catRatio, exists := s.config.CategoryRatios[category]; exists {
		ratio = catRatio // Category ratio overrides global
	}

	// Decide read or write
	isRead, err := s.shouldBeReadWithRatio(ratio)
	if err != nil {
		return nil, err
	}

	for _, ep := range s.endpoints {
		if ep.Category != category {
			continue
		}

		// Filter by read/write
		if isRead && !ep.Operation().IsRead() {
			continue
		}
		if !isRead && ep.Operation().IsRead() {
			continue
		}

		weight := s.calculateEffectiveWeight(ep)
		if weight == 0 {
			continue
		}

		totalWeight += weight
		entries = append(entries, weightedEntry{
			endpoint:         ep,
			effectiveWeight:  weight,
			cumulativeWeight: totalWeight,
		})
	}

	if len(entries) == 0 {
		return nil, ErrNoEndpoints
	}

	return s.selectFromPool(entries, totalWeight)
}

// SelectByOperation selects an endpoint of a specific operation type.
func (s *WeightedSelector) SelectByOperation(op OperationType) (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build a temporary pool for this operation
	var entries []weightedEntry
	totalWeight := 0

	for _, ep := range s.endpoints {
		if ep.Operation() != op {
			continue
		}

		weight := s.calculateEffectiveWeight(ep)
		if weight == 0 {
			continue
		}

		totalWeight += weight
		entries = append(entries, weightedEntry{
			endpoint:         ep,
			effectiveWeight:  weight,
			cumulativeWeight: totalWeight,
		})
	}

	if len(entries) == 0 {
		return nil, ErrNoEndpoints
	}

	return s.selectFromPool(entries, totalWeight)
}

// shouldBeRead determines if the next selection should be a read operation.
func (s *WeightedSelector) shouldBeRead() (bool, error) {
	return s.shouldBeReadWithRatio(s.config.ReadWriteRatio)
}

// shouldBeReadWithRatio determines read/write based on given ratio.
func (s *WeightedSelector) shouldBeReadWithRatio(ratio float64) (bool, error) {
	// Handle edge cases
	if len(s.readEntries) == 0 {
		return false, nil // Force write if no read endpoints
	}
	if len(s.writeEntries) == 0 {
		return true, nil // Force read if no write endpoints
	}

	// Handle boundary ratios
	if ratio >= 1.0 {
		return true, nil
	}
	if ratio <= 0.0 {
		return false, nil
	}

	// Generate random number between 0 and 1000 for 0.1% precision
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return false, err
	}

	// If random value < ratio * 1000, it's a read
	threshold := int64(ratio * 1000)
	return n.Int64() < threshold, nil
}

// selectFromPool selects an endpoint from a weighted pool.
func (s *WeightedSelector) selectFromPool(entries []weightedEntry, totalWeight int) (*Endpoint, error) {
	if len(entries) == 0 || totalWeight == 0 {
		return nil, ErrNoEndpoints
	}

	// Generate random number in range [0, totalWeight)
	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return nil, err
	}
	target := int(n.Int64())

	// Binary search for the entry
	low, high := 0, len(entries)-1
	for low < high {
		mid := (low + high) / 2
		if entries[mid].cumulativeWeight <= target {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return entries[low].endpoint, nil
}

// GetEndpoint returns an endpoint by name.
func (s *WeightedSelector) GetEndpoint(name string) (*Endpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ep, exists := s.endpoints[name]
	if !exists {
		return nil, ErrEndpointNotFound
	}
	return ep, nil
}

// GetEndpoints returns all registered endpoints.
func (s *WeightedSelector) GetEndpoints() []*Endpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	endpoints := make([]*Endpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// GetEffectiveWeight returns the effective weight of an endpoint.
func (s *WeightedSelector) GetEffectiveWeight(name string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ep, exists := s.endpoints[name]
	if !exists {
		return 0, ErrEndpointNotFound
	}
	return s.calculateEffectiveWeight(ep), nil
}

// Stats returns statistics about the selector's configuration.
type Stats struct {
	TotalEndpoints   int
	ReadEndpoints    int
	WriteEndpoints   int
	TotalReadWeight  int
	TotalWriteWeight int
	Categories       map[string]int        // category -> count
	Operations       map[OperationType]int // operation -> count
}

// GetStats returns statistics about the current selector state.
func (s *WeightedSelector) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		TotalEndpoints:   len(s.endpoints),
		ReadEndpoints:    len(s.readEntries),
		WriteEndpoints:   len(s.writeEntries),
		TotalReadWeight:  s.totalReadWeight,
		TotalWriteWeight: s.totalWriteWeight,
		Categories:       make(map[string]int),
		Operations:       make(map[OperationType]int),
	}

	for _, ep := range s.endpoints {
		if ep.Category != "" {
			stats.Categories[ep.Category]++
		}
		stats.Operations[ep.Operation()]++
	}

	return stats
}

// UpdateConfig updates the selector configuration and rebuilds pools.
func (s *WeightedSelector) UpdateConfig(config WeightedSelectorConfig) error {
	configCopy := config
	if err := configCopy.Validate(); err != nil {
		return err
	}
	configCopy.ApplyDefaults()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = configCopy
	s.rebuildPools()

	return nil
}

// GetConfig returns a copy of the current configuration.
func (s *WeightedSelector) GetConfig() WeightedSelectorConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	config := s.config
	config.Categories = make(map[string]int)
	maps.Copy(config.Categories, s.config.Categories)
	config.CategoryRatios = make(map[string]float64)
	maps.Copy(config.CategoryRatios, s.config.CategoryRatios)
	config.Weights = make(map[string]int)
	maps.Copy(config.Weights, s.config.Weights)
	config.Operations = make(map[OperationType]int)
	maps.Copy(config.Operations, s.config.Operations)

	return config
}
