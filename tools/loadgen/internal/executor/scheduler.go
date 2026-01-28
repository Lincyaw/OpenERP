// Package executor provides request building and execution functionality for the load generator.
package executor

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/example/erp/tools/loadgen/internal/circuit"
)

// Errors returned by the scheduler package.
var (
	// ErrNoEndpoints is returned when there are no endpoints to select from.
	ErrNoEndpoints = errors.New("scheduler: no endpoints available")
	// ErrEndpointNotFound is returned when an endpoint is not found.
	ErrEndpointNotFound = errors.New("scheduler: endpoint not found")
	// ErrInvalidWeight is returned when an endpoint has invalid weight.
	ErrInvalidWeight = errors.New("scheduler: invalid weight")
)

// EndpointInfo holds metadata for an endpoint used in scheduling.
type EndpointInfo struct {
	// Name is the unique identifier for this endpoint.
	Name string

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH).
	Method string

	// Path is the URL path.
	Path string

	// Weight determines how often this endpoint is selected (higher = more frequent).
	// Default: 1
	Weight int

	// Tags categorize the endpoint.
	Tags []string

	// Category is the primary category (e.g., "catalog", "trade", "finance").
	Category string

	// Disabled indicates if this endpoint should be excluded from selection.
	Disabled bool

	// Unit is the underlying circuit.EndpointUnit (if available).
	Unit *circuit.EndpointUnit
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	// Weights allows overriding endpoint weights by name.
	// Key: endpoint name, Value: weight
	Weights map[string]int

	// Exclude is a list of endpoint names to exclude from selection.
	Exclude []string

	// CategoryWeights allows setting weights per category.
	// Key: category name, Value: weight multiplier
	CategoryWeights map[string]int

	// TagWeights allows setting weights per tag.
	// Key: tag name, Value: weight multiplier
	TagWeights map[string]int

	// ReadWriteRatio controls the ratio of read (GET) to write (POST/PUT/DELETE) operations.
	// Value from 0.0 (all writes) to 1.0 (all reads).
	// Default: 0.8 (80% reads, 20% writes)
	ReadWriteRatio float64

	// ReadWriteRatioSet indicates whether ReadWriteRatio was explicitly set.
	ReadWriteRatioSet bool
}

// DefaultSchedulerConfig returns a default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Weights:         make(map[string]int),
		Exclude:         make([]string, 0),
		CategoryWeights: make(map[string]int),
		TagWeights:      make(map[string]int),
		ReadWriteRatio:  0.8,
	}
}

// weightedEntry represents an entry in the weighted selection pool.
type weightedEntry struct {
	endpoint         *EndpointInfo
	effectiveWeight  int
	cumulativeWeight int
}

// Scheduler selects endpoints based on weighted random selection.
// It supports:
// - Endpoint-specific weight overrides
// - Category-based weight multipliers
// - Tag-based weight multipliers
// - Exclusion rules
// - Read/Write ratio control
//
// Thread Safety: Safe for concurrent use.
type Scheduler struct {
	mu     sync.RWMutex
	config SchedulerConfig

	// endpoints holds all registered endpoints by name.
	endpoints map[string]*EndpointInfo

	// excludeSet is a set of excluded endpoint names for quick lookup.
	excludeSet map[string]struct{}

	// readEntries holds weighted entries for read operations (GET).
	readEntries []weightedEntry

	// writeEntries holds weighted entries for write operations (POST/PUT/DELETE/PATCH).
	writeEntries []weightedEntry

	// allEntries holds all weighted entries (for selection without read/write filtering).
	allEntries []weightedEntry

	// totalReadWeight is the sum of all read endpoint weights.
	totalReadWeight int

	// totalWriteWeight is the sum of all write endpoint weights.
	totalWriteWeight int

	// totalWeight is the sum of all endpoint weights.
	totalWeight int

	// Statistics
	totalSelections atomic.Int64
	readSelections  atomic.Int64
	writeSelections atomic.Int64
	selectionsPerEp sync.Map // map[string]*atomic.Int64
}

// NewScheduler creates a new scheduler with the given configuration.
func NewScheduler(config SchedulerConfig) *Scheduler {
	if config.Weights == nil {
		config.Weights = make(map[string]int)
	}
	if config.Exclude == nil {
		config.Exclude = make([]string, 0)
	}
	if config.CategoryWeights == nil {
		config.CategoryWeights = make(map[string]int)
	}
	if config.TagWeights == nil {
		config.TagWeights = make(map[string]int)
	}

	// Apply default read/write ratio if not explicitly set
	if !config.ReadWriteRatioSet && config.ReadWriteRatio == 0 {
		config.ReadWriteRatio = 0.8
	}

	s := &Scheduler{
		config:       config,
		endpoints:    make(map[string]*EndpointInfo),
		excludeSet:   make(map[string]struct{}),
		readEntries:  make([]weightedEntry, 0),
		writeEntries: make([]weightedEntry, 0),
		allEntries:   make([]weightedEntry, 0),
	}

	// Build exclude set for quick lookup
	for _, name := range config.Exclude {
		s.excludeSet[name] = struct{}{}
	}

	return s
}

// Register adds an endpoint to the scheduler.
func (s *Scheduler) Register(endpoint *EndpointInfo) error {
	if endpoint == nil {
		return ErrNoEndpoints
	}
	if endpoint.Name == "" {
		return errors.New("scheduler: endpoint name is required")
	}
	if endpoint.Weight < 0 {
		return ErrInvalidWeight
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the endpoint
	s.endpoints[endpoint.Name] = endpoint

	// Rebuild weighted pools
	s.rebuildPools()

	return nil
}

// RegisterAll adds multiple endpoints to the scheduler.
func (s *Scheduler) RegisterAll(endpoints []*EndpointInfo) error {
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
			return errors.New("scheduler: endpoint name is required")
		}
		if ep.Weight < 0 {
			return ErrInvalidWeight
		}
	}

	// Store all endpoints
	for _, ep := range endpoints {
		s.endpoints[ep.Name] = ep
	}

	// Rebuild weighted pools
	s.rebuildPools()

	return nil
}

// RegisterFromUnits converts circuit.EndpointUnits to EndpointInfo and registers them.
func (s *Scheduler) RegisterFromUnits(units []*circuit.EndpointUnit) error {
	if len(units) == 0 {
		return ErrNoEndpoints
	}

	infos := make([]*EndpointInfo, 0, len(units))
	for _, unit := range units {
		if unit == nil || unit.Disabled {
			continue
		}
		info := &EndpointInfo{
			Name:     unit.Name,
			Method:   unit.Method,
			Path:     unit.Path,
			Weight:   unit.Weight,
			Disabled: unit.Disabled,
			Unit:     unit,
		}
		// Infer category from path if possible (e.g., "/catalog/products" -> "catalog")
		info.Category = inferCategoryFromPath(unit.Path)
		infos = append(infos, info)
	}

	if len(infos) == 0 {
		return ErrNoEndpoints
	}

	return s.RegisterAll(infos)
}

// inferCategoryFromPath extracts a category from the URL path.
// For example: "/catalog/products" -> "catalog", "/trade/orders" -> "trade"
func inferCategoryFromPath(path string) string {
	if len(path) == 0 {
		return ""
	}

	// Remove leading slash and split
	if path[0] == '/' {
		path = path[1:]
	}

	// Find the first path segment
	for i, ch := range path {
		if ch == '/' {
			return path[:i]
		}
	}

	return path
}

// Unregister removes an endpoint from the scheduler.
func (s *Scheduler) Unregister(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.endpoints[name]; !exists {
		return ErrEndpointNotFound
	}

	delete(s.endpoints, name)
	s.rebuildPools()

	return nil
}

// Select chooses an endpoint based on weighted random selection.
// It respects the read/write ratio configuration.
func (s *Scheduler) Select() (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.allEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	// Decide if this should be a read or write based on ratio
	isRead, err := s.shouldBeRead()
	if err != nil {
		return nil, err
	}

	var ep *EndpointInfo
	if isRead && len(s.readEntries) > 0 {
		ep, err = s.selectFromPool(s.readEntries, s.totalReadWeight)
		if err == nil {
			s.readSelections.Add(1)
		}
	} else if !isRead && len(s.writeEntries) > 0 {
		ep, err = s.selectFromPool(s.writeEntries, s.totalWriteWeight)
		if err == nil {
			s.writeSelections.Add(1)
		}
	} else {
		// Fallback to all entries if specific pool is empty
		ep, err = s.selectFromPool(s.allEntries, s.totalWeight)
	}

	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// SelectAny selects an endpoint ignoring read/write ratio.
func (s *Scheduler) SelectAny() (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.allEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	ep, err := s.selectFromPool(s.allEntries, s.totalWeight)
	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// SelectRead selects only from read (GET) endpoints.
func (s *Scheduler) SelectRead() (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.readEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	ep, err := s.selectFromPool(s.readEntries, s.totalReadWeight)
	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.readSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// SelectWrite selects only from write (POST/PUT/DELETE/PATCH) endpoints.
func (s *Scheduler) SelectWrite() (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.writeEntries) == 0 {
		return nil, ErrNoEndpoints
	}

	ep, err := s.selectFromPool(s.writeEntries, s.totalWriteWeight)
	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.writeSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// SelectByCategory selects an endpoint from a specific category.
func (s *Scheduler) SelectByCategory(category string) (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build temporary pool for this category
	var entries []weightedEntry
	totalWeight := 0

	for _, entry := range s.allEntries {
		if entry.endpoint.Category == category {
			totalWeight += entry.effectiveWeight
			entries = append(entries, weightedEntry{
				endpoint:         entry.endpoint,
				effectiveWeight:  entry.effectiveWeight,
				cumulativeWeight: totalWeight,
			})
		}
	}

	if len(entries) == 0 {
		return nil, ErrNoEndpoints
	}

	ep, err := s.selectFromPool(entries, totalWeight)
	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// SelectByTag selects an endpoint with a specific tag.
func (s *Scheduler) SelectByTag(tag string) (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build temporary pool for this tag
	var entries []weightedEntry
	totalWeight := 0

	for _, entry := range s.allEntries {
		hasTag := false
		for _, t := range entry.endpoint.Tags {
			if t == tag {
				hasTag = true
				break
			}
		}
		if hasTag {
			totalWeight += entry.effectiveWeight
			entries = append(entries, weightedEntry{
				endpoint:         entry.endpoint,
				effectiveWeight:  entry.effectiveWeight,
				cumulativeWeight: totalWeight,
			})
		}
	}

	if len(entries) == 0 {
		return nil, ErrNoEndpoints
	}

	ep, err := s.selectFromPool(entries, totalWeight)
	if err != nil {
		return nil, err
	}

	s.totalSelections.Add(1)
	s.incrementEndpointCount(ep.Name)

	return ep, nil
}

// GetEndpoint returns an endpoint by name.
func (s *Scheduler) GetEndpoint(name string) (*EndpointInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ep, exists := s.endpoints[name]
	if !exists {
		return nil, ErrEndpointNotFound
	}
	return ep, nil
}

// GetAllEndpoints returns all registered endpoints.
func (s *Scheduler) GetAllEndpoints() []*EndpointInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	endpoints := make([]*EndpointInfo, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// GetEffectiveWeight returns the effective weight of an endpoint.
func (s *Scheduler) GetEffectiveWeight(name string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ep, exists := s.endpoints[name]
	if !exists {
		return 0, ErrEndpointNotFound
	}

	return s.calculateEffectiveWeight(ep), nil
}

// UpdateConfig updates the scheduler configuration and rebuilds pools.
func (s *Scheduler) UpdateConfig(config SchedulerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Weights == nil {
		config.Weights = make(map[string]int)
	}
	if config.Exclude == nil {
		config.Exclude = make([]string, 0)
	}
	if config.CategoryWeights == nil {
		config.CategoryWeights = make(map[string]int)
	}
	if config.TagWeights == nil {
		config.TagWeights = make(map[string]int)
	}

	s.config = config

	// Rebuild exclude set
	s.excludeSet = make(map[string]struct{})
	for _, name := range config.Exclude {
		s.excludeSet[name] = struct{}{}
	}

	s.rebuildPools()
}

// SetWeight sets the weight override for an endpoint.
func (s *Scheduler) SetWeight(name string, weight int) error {
	if weight < 0 {
		return ErrInvalidWeight
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Weights[name] = weight
	s.rebuildPools()

	return nil
}

// SetExclude updates the exclusion list.
func (s *Scheduler) SetExclude(exclude []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.Exclude = exclude
	s.excludeSet = make(map[string]struct{})
	for _, name := range exclude {
		s.excludeSet[name] = struct{}{}
	}

	s.rebuildPools()
}

// AddExclude adds an endpoint to the exclusion list.
func (s *Scheduler) AddExclude(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.excludeSet[name] = struct{}{}
	s.config.Exclude = append(s.config.Exclude, name)
	s.rebuildPools()
}

// RemoveExclude removes an endpoint from the exclusion list.
func (s *Scheduler) RemoveExclude(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.excludeSet, name)

	// Remove from config slice
	newExclude := make([]string, 0, len(s.config.Exclude))
	for _, n := range s.config.Exclude {
		if n != name {
			newExclude = append(newExclude, n)
		}
	}
	s.config.Exclude = newExclude

	s.rebuildPools()
}

// SetReadWriteRatio sets the read/write ratio.
func (s *Scheduler) SetReadWriteRatio(ratio float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clamp to valid range
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	s.config.ReadWriteRatio = ratio
	s.config.ReadWriteRatioSet = true
}

// SchedulerStats holds statistics about the scheduler.
type SchedulerStats struct {
	// TotalEndpoints is the total number of registered endpoints.
	TotalEndpoints int

	// ActiveEndpoints is the number of endpoints available for selection.
	ActiveEndpoints int

	// ReadEndpoints is the number of read (GET) endpoints.
	ReadEndpoints int

	// WriteEndpoints is the number of write endpoints.
	WriteEndpoints int

	// TotalWeight is the sum of all active endpoint weights.
	TotalWeight int

	// TotalReadWeight is the sum of all read endpoint weights.
	TotalReadWeight int

	// TotalWriteWeight is the sum of all write endpoint weights.
	TotalWriteWeight int

	// TotalSelections is the total number of selections made.
	TotalSelections int64

	// ReadSelections is the number of read selections.
	ReadSelections int64

	// WriteSelections is the number of write selections.
	WriteSelections int64

	// ExcludedCount is the number of excluded endpoints.
	ExcludedCount int

	// SelectionsByEndpoint holds selection counts per endpoint.
	SelectionsByEndpoint map[string]int64

	// ReadWriteRatio is the configured read/write ratio.
	ReadWriteRatio float64
}

// Stats returns statistics about the scheduler.
func (s *Scheduler) Stats() SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := SchedulerStats{
		TotalEndpoints:       len(s.endpoints),
		ActiveEndpoints:      len(s.allEntries),
		ReadEndpoints:        len(s.readEntries),
		WriteEndpoints:       len(s.writeEntries),
		TotalWeight:          s.totalWeight,
		TotalReadWeight:      s.totalReadWeight,
		TotalWriteWeight:     s.totalWriteWeight,
		TotalSelections:      s.totalSelections.Load(),
		ReadSelections:       s.readSelections.Load(),
		WriteSelections:      s.writeSelections.Load(),
		ExcludedCount:        len(s.excludeSet),
		SelectionsByEndpoint: make(map[string]int64),
		ReadWriteRatio:       s.config.ReadWriteRatio,
	}

	// Collect per-endpoint selection counts
	s.selectionsPerEp.Range(func(key, value any) bool {
		if counter, ok := value.(*atomic.Int64); ok {
			stats.SelectionsByEndpoint[key.(string)] = counter.Load()
		}
		return true
	})

	return stats
}

// rebuildPools recalculates the weighted entry pools.
// Must be called with s.mu held.
func (s *Scheduler) rebuildPools() {
	s.readEntries = s.readEntries[:0]
	s.writeEntries = s.writeEntries[:0]
	s.allEntries = s.allEntries[:0]
	s.totalReadWeight = 0
	s.totalWriteWeight = 0
	s.totalWeight = 0

	// Sort endpoint names for deterministic ordering
	names := make([]string, 0, len(s.endpoints))
	for name := range s.endpoints {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ep := s.endpoints[name]

		// Skip excluded and disabled endpoints
		if _, excluded := s.excludeSet[name]; excluded {
			continue
		}
		if ep.Disabled {
			continue
		}

		weight := s.calculateEffectiveWeight(ep)
		if weight == 0 {
			continue
		}

		// Add to all entries
		s.totalWeight += weight
		s.allEntries = append(s.allEntries, weightedEntry{
			endpoint:         ep,
			effectiveWeight:  weight,
			cumulativeWeight: s.totalWeight,
		})

		// Add to read or write entries
		if isReadMethod(ep.Method) {
			s.totalReadWeight += weight
			s.readEntries = append(s.readEntries, weightedEntry{
				endpoint:         ep,
				effectiveWeight:  weight,
				cumulativeWeight: s.totalReadWeight,
			})
		} else {
			s.totalWriteWeight += weight
			s.writeEntries = append(s.writeEntries, weightedEntry{
				endpoint:         ep,
				effectiveWeight:  weight,
				cumulativeWeight: s.totalWriteWeight,
			})
		}
	}
}

// calculateEffectiveWeight computes the final weight for an endpoint.
// Weight calculation considers:
// 1. Endpoint-specific weight override (highest priority)
// 2. Category weight multiplier
// 3. Tag weight multiplier (max of all matching tags)
// 4. Base weight from endpoint config
func (s *Scheduler) calculateEffectiveWeight(ep *EndpointInfo) int {
	// Start with base weight
	baseWeight := ep.Weight
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
		if catWeight, exists := s.config.CategoryWeights[ep.Category]; exists && catWeight > 0 {
			categoryMultiplier = float64(catWeight)
		}
	}

	// Apply tag weight multiplier (use max of all matching tags)
	tagMultiplier := 1.0
	for _, tag := range ep.Tags {
		if tagWeight, exists := s.config.TagWeights[tag]; exists && tagWeight > 0 {
			if float64(tagWeight) > tagMultiplier {
				tagMultiplier = float64(tagWeight)
			}
		}
	}

	// Calculate effective weight
	effectiveWeight := float64(baseWeight) * categoryMultiplier * tagMultiplier

	// Handle edge cases
	if effectiveWeight < 1 && effectiveWeight > 0 {
		return 1
	}

	// Overflow protection
	const maxWeight = 1<<31 - 1
	if effectiveWeight > float64(maxWeight) {
		return maxWeight
	}

	return int(effectiveWeight)
}

// shouldBeRead determines if the next selection should be a read operation.
func (s *Scheduler) shouldBeRead() (bool, error) {
	ratio := s.config.ReadWriteRatio

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

// selectFromPool selects an endpoint from a weighted pool using binary search.
func (s *Scheduler) selectFromPool(entries []weightedEntry, totalWeight int) (*EndpointInfo, error) {
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

// incrementEndpointCount increments the selection count for an endpoint.
func (s *Scheduler) incrementEndpointCount(name string) {
	counter, _ := s.selectionsPerEp.LoadOrStore(name, &atomic.Int64{})
	counter.(*atomic.Int64).Add(1)
}

// isReadMethod returns true if the method is a read operation (GET, HEAD, OPTIONS).
func isReadMethod(method string) bool {
	return method == "GET" || method == "HEAD" || method == "OPTIONS"
}
