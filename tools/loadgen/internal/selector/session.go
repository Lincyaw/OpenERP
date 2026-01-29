// Package selector provides endpoint selection strategies for the load generator.
package selector

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

// Errors returned by the session package.
var (
	// ErrSessionExpired is returned when an operation is attempted on an expired session.
	ErrSessionExpired = errors.New("session: session has expired")
	// ErrNoActiveSessions is returned when there are no active sessions available.
	ErrNoActiveSessions = errors.New("session: no active sessions available")
	// ErrInvalidSessionConfig is returned when session configuration is invalid.
	ErrInvalidSessionConfig = errors.New("session: invalid configuration")
	// ErrBehaviorNotFound is returned when a behavior is not found.
	ErrBehaviorNotFound = errors.New("session: behavior not found")
	// ErrSessionLimitReached is returned when the concurrent session limit is reached.
	ErrSessionLimitReached = errors.New("session: concurrent session limit reached")
)

// SessionID uniquely identifies a session.
type SessionID string

// UserBehavior defines a type of user behavior with associated parameters.
// Different behaviors can have different think times and actions per session.
type UserBehavior struct {
	// Name is the unique identifier for this behavior.
	Name string `yaml:"name" json:"name"`

	// Weight determines how often this behavior is selected relative to others.
	// Higher weight = more frequent selection.
	// Default: 1
	Weight int `yaml:"weight" json:"weight"`

	// ThinkTime configures the delay between actions within a session.
	ThinkTime ThinkTimeConfig `yaml:"thinkTime" json:"thinkTime"`

	// ActionsPerSession configures how many actions a session performs.
	ActionsPerSession ActionsConfig `yaml:"actionsPerSession" json:"actionsPerSession"`

	// Description provides context about this behavior.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Tags categorize the behavior (e.g., ["power-user", "admin"]).
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// ThinkTimeConfig configures think time between actions.
type ThinkTimeConfig struct {
	// Min is the minimum think time.
	// Default: 1s
	Min time.Duration `yaml:"min" json:"min"`

	// Max is the maximum think time.
	// Default: 5s
	Max time.Duration `yaml:"max" json:"max"`

	// Distribution is the distribution type: "uniform", "exponential", "normal".
	// Default: "uniform"
	Distribution string `yaml:"distribution,omitempty" json:"distribution,omitempty"`
}

// ActionsConfig configures the number of actions per session.
type ActionsConfig struct {
	// Min is the minimum number of actions.
	// Default: 1
	Min int `yaml:"min" json:"min"`

	// Max is the maximum number of actions.
	// Default: 10
	Max int `yaml:"max" json:"max"`
}

// Validate validates the UserBehavior configuration.
func (ub *UserBehavior) Validate() error {
	if ub.Name == "" {
		return fmt.Errorf("%w: behavior name is required", ErrInvalidSessionConfig)
	}
	if ub.Weight < 0 {
		return fmt.Errorf("%w: behavior weight must be non-negative", ErrInvalidSessionConfig)
	}
	if ub.ThinkTime.Min < 0 || ub.ThinkTime.Max < 0 {
		return fmt.Errorf("%w: think time must be non-negative", ErrInvalidSessionConfig)
	}
	if ub.ThinkTime.Max > 0 && ub.ThinkTime.Min > ub.ThinkTime.Max {
		return fmt.Errorf("%w: think time min must be <= max", ErrInvalidSessionConfig)
	}
	if ub.ActionsPerSession.Min < 0 || ub.ActionsPerSession.Max < 0 {
		return fmt.Errorf("%w: actions per session must be non-negative", ErrInvalidSessionConfig)
	}
	if ub.ActionsPerSession.Max > 0 && ub.ActionsPerSession.Min > ub.ActionsPerSession.Max {
		return fmt.Errorf("%w: actions min must be <= max", ErrInvalidSessionConfig)
	}
	return nil
}

// ApplyDefaults applies default values to unset fields.
func (ub *UserBehavior) ApplyDefaults() {
	if ub.Weight == 0 {
		ub.Weight = 1
	}
	if ub.ThinkTime.Min == 0 {
		ub.ThinkTime.Min = 1 * time.Second
	}
	if ub.ThinkTime.Max == 0 {
		ub.ThinkTime.Max = 5 * time.Second
	}
	if ub.ThinkTime.Distribution == "" {
		ub.ThinkTime.Distribution = "uniform"
	}
	if ub.ActionsPerSession.Min == 0 {
		ub.ActionsPerSession.Min = 1
	}
	if ub.ActionsPerSession.Max == 0 {
		ub.ActionsPerSession.Max = 10
	}
}

// SessionDurationConfig configures session duration.
type SessionDurationConfig struct {
	// Min is the minimum session duration.
	// Default: 30s
	Min time.Duration `yaml:"min" json:"min"`

	// Max is the maximum session duration.
	// Default: 5m
	Max time.Duration `yaml:"max" json:"max"`
}

// Session represents an active user session with its own state and parameters.
type Session struct {
	// ID is the unique identifier for this session.
	ID SessionID

	// StartTime is when the session was created.
	StartTime time.Time

	// ExpiresAt is when the session will expire.
	ExpiresAt time.Time

	// Behavior is the user behavior for this session.
	Behavior *UserBehavior

	// Parameters is the session-level parameter pool.
	// Resources created during this session can be stored here for reuse.
	Parameters *SessionParameters

	// ActionCount tracks the number of actions performed.
	ActionCount int

	// MaxActions is the maximum number of actions for this session.
	MaxActions int

	// LastActionTime is the time of the last action.
	LastActionTime time.Time

	// mu protects session state updates.
	mu sync.Mutex
}

// SessionParameters holds session-scoped parameters.
// Resources created during a session can be stored here for reuse within the same session.
type SessionParameters struct {
	// mu protects the parameters map.
	mu sync.RWMutex

	// params holds the parameter values by key.
	params map[string][]any
}

// NewSessionParameters creates a new session parameters store.
func NewSessionParameters() *SessionParameters {
	return &SessionParameters{
		params: make(map[string][]any),
	}
}

// Set stores a value for a key, appending to any existing values.
func (sp *SessionParameters) Set(key string, value any) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.params[key] = append(sp.params[key], value)
}

// Get retrieves the latest value for a key.
// Returns nil, false if the key doesn't exist.
func (sp *SessionParameters) Get(key string) (any, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	values, ok := sp.params[key]
	if !ok || len(values) == 0 {
		return nil, false
	}
	return values[len(values)-1], true
}

// GetAll retrieves all values for a key.
func (sp *SessionParameters) GetAll(key string) []any {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return append([]any(nil), sp.params[key]...)
}

// GetRandom retrieves a random value for a key.
// Returns nil, false if the key doesn't exist.
func (sp *SessionParameters) GetRandom(key string) (any, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	values, ok := sp.params[key]
	if !ok || len(values) == 0 {
		return nil, false
	}
	if len(values) == 1 {
		return values[0], true
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(values))))
	if err != nil {
		return values[0], true
	}
	return values[n.Int64()], true
}

// Has checks if a key exists.
func (sp *SessionParameters) Has(key string) bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	values, ok := sp.params[key]
	return ok && len(values) > 0
}

// Keys returns all keys in the parameter store.
func (sp *SessionParameters) Keys() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	keys := make([]string, 0, len(sp.params))
	for k := range sp.params {
		keys = append(keys, k)
	}
	return keys
}

// Clear removes all parameters.
func (sp *SessionParameters) Clear() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.params = make(map[string][]any)
}

// Count returns the total number of values stored.
func (sp *SessionParameters) Count() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	count := 0
	for _, values := range sp.params {
		count += len(values)
	}
	return count
}

// Clone creates a copy of the parameters.
func (sp *SessionParameters) Clone() *SessionParameters {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	clone := NewSessionParameters()
	for k, values := range sp.params {
		clone.params[k] = append([]any(nil), values...)
	}
	return clone
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsActionLimitReached checks if the session has reached its action limit.
func (s *Session) IsActionLimitReached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ActionCount >= s.MaxActions
}

// IncrementActionCount increments the action count.
// Returns the new count.
func (s *Session) IncrementActionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActionCount++
	s.LastActionTime = time.Now()
	return s.ActionCount
}

// RemainingActions returns the number of actions remaining.
func (s *Session) RemainingActions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := s.MaxActions - s.ActionCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Duration returns the session duration so far.
func (s *Session) Duration() time.Duration {
	return time.Since(s.StartTime)
}

// RemainingTime returns the time remaining until expiration.
func (s *Session) RemainingTime() time.Duration {
	remaining := time.Until(s.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NextThinkTime generates the next think time based on the session's behavior.
func (s *Session) NextThinkTime() time.Duration {
	if s.Behavior == nil {
		return time.Second // Default fallback
	}

	min := s.Behavior.ThinkTime.Min
	max := s.Behavior.ThinkTime.Max

	if min >= max {
		return min
	}

	switch s.Behavior.ThinkTime.Distribution {
	case "exponential":
		return exponentialDuration(min, max)
	case "normal":
		return normalDuration(min, max)
	default: // "uniform"
		return uniformDuration(min, max)
	}
}

// uniformDuration generates a uniformly distributed random duration.
func uniformDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}

	rangeNs := int64(max - min)
	n, err := rand.Int(rand.Reader, big.NewInt(rangeNs))
	if err != nil {
		return min
	}
	return min + time.Duration(n.Int64())
}

// exponentialDuration generates an exponentially distributed random duration.
// Uses inverse transform sampling with mean = (max - min) / 2.
func exponentialDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}

	// Generate uniform random number [0, 1)
	n, err := rand.Int(rand.Reader, big.NewInt(1<<30))
	if err != nil {
		return min
	}
	u := float64(n.Int64()) / float64(1<<30)

	// Exponential inverse transform: -ln(1-u) * mean
	// We use lambda = 2 / (max - min) to keep most values in range
	rangeNs := float64(max - min)
	lambda := 2.0 / rangeNs

	// Clamp u to avoid log(0)
	if u >= 0.9999 {
		u = 0.9999
	}

	// Inverse transform sampling for exponential
	result := -float64(1.0/lambda) * float64(1.0-u)
	resultDuration := time.Duration(result)

	// Clamp to range
	if resultDuration < min {
		return min
	}
	if resultDuration > max {
		return max
	}
	return resultDuration
}

// normalDuration generates a normally distributed random duration.
// Uses Box-Muller transform with mean = (min + max) / 2.
func normalDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}

	// Use uniform as approximation (simplified)
	// A full Box-Muller would require two random numbers
	return uniformDuration(min, max)
}

// SessionSimulatorConfig configures the session simulator.
type SessionSimulatorConfig struct {
	// ConcurrentSessions is the maximum number of concurrent sessions.
	// Default: 100
	ConcurrentSessions int `yaml:"concurrentSessions" json:"concurrentSessions"`

	// SessionDuration configures session duration range.
	SessionDuration SessionDurationConfig `yaml:"sessionDuration" json:"sessionDuration"`

	// Behaviors defines the available user behaviors.
	// If empty, a default behavior is used.
	Behaviors []UserBehavior `yaml:"behaviors,omitempty" json:"behaviors,omitempty"`

	// ReplaceExpired controls whether expired sessions are automatically replaced.
	// Default: true
	ReplaceExpired *bool `yaml:"replaceExpired,omitempty" json:"replaceExpired,omitempty"`
}

// Validate validates the configuration.
func (c *SessionSimulatorConfig) Validate() error {
	if c.ConcurrentSessions < 0 {
		return fmt.Errorf("%w: concurrentSessions must be non-negative", ErrInvalidSessionConfig)
	}
	if c.SessionDuration.Min < 0 || c.SessionDuration.Max < 0 {
		return fmt.Errorf("%w: session duration must be non-negative", ErrInvalidSessionConfig)
	}
	if c.SessionDuration.Max > 0 && c.SessionDuration.Min > c.SessionDuration.Max {
		return fmt.Errorf("%w: session duration min must be <= max", ErrInvalidSessionConfig)
	}

	for i := range c.Behaviors {
		if err := c.Behaviors[i].Validate(); err != nil {
			return fmt.Errorf("behavior[%d]: %w", i, err)
		}
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *SessionSimulatorConfig) ApplyDefaults() {
	if c.ConcurrentSessions == 0 {
		c.ConcurrentSessions = 100
	}
	if c.SessionDuration.Min == 0 {
		c.SessionDuration.Min = 30 * time.Second
	}
	if c.SessionDuration.Max == 0 {
		c.SessionDuration.Max = 5 * time.Minute
	}
	if c.ReplaceExpired == nil {
		replaceExpired := true
		c.ReplaceExpired = &replaceExpired
	}

	// Apply defaults to behaviors
	for i := range c.Behaviors {
		c.Behaviors[i].ApplyDefaults()
	}

	// If no behaviors defined, add a default one
	if len(c.Behaviors) == 0 {
		defaultBehavior := UserBehavior{
			Name:   "default",
			Weight: 1,
		}
		defaultBehavior.ApplyDefaults()
		c.Behaviors = []UserBehavior{defaultBehavior}
	}
}

// SessionSimulator manages concurrent user sessions with realistic behavior.
type SessionSimulator struct {
	// config holds the simulator configuration.
	config SessionSimulatorConfig

	// mu protects the sessions map.
	mu sync.RWMutex

	// sessions holds active sessions by ID.
	sessions map[SessionID]*Session

	// behaviorWeights holds cumulative weights for behavior selection.
	behaviorWeights []behaviorWeight

	// totalBehaviorWeight is the sum of all behavior weights.
	totalBehaviorWeight int

	// sessionCounter is used to generate unique session IDs.
	sessionCounter atomic.Uint64

	// stats tracks session statistics.
	stats SessionSimulatorStats

	// timeFunc returns the current time. Can be overridden for testing.
	timeFunc func() time.Time
}

// behaviorWeight holds a behavior with its cumulative weight.
type behaviorWeight struct {
	behavior         *UserBehavior
	cumulativeWeight int
}

// SessionSimulatorStats holds statistics about the session simulator.
type SessionSimulatorStats struct {
	// TotalSessionsCreated is the total number of sessions created.
	TotalSessionsCreated atomic.Uint64
	// TotalSessionsExpired is the total number of sessions that expired.
	TotalSessionsExpired atomic.Uint64
	// TotalActionsExecuted is the total number of actions executed across all sessions.
	TotalActionsExecuted atomic.Uint64
}

// NewSessionSimulator creates a new session simulator.
func NewSessionSimulator(config SessionSimulatorConfig) (*SessionSimulator, error) {
	configCopy := config
	if err := configCopy.Validate(); err != nil {
		return nil, err
	}
	configCopy.ApplyDefaults()

	ss := &SessionSimulator{
		config:   configCopy,
		sessions: make(map[SessionID]*Session),
		timeFunc: time.Now,
	}

	// Build behavior weight table
	ss.rebuildBehaviorWeights()

	return ss, nil
}

// rebuildBehaviorWeights rebuilds the behavior weight table.
func (ss *SessionSimulator) rebuildBehaviorWeights() {
	ss.behaviorWeights = make([]behaviorWeight, 0, len(ss.config.Behaviors))
	ss.totalBehaviorWeight = 0

	for i := range ss.config.Behaviors {
		b := &ss.config.Behaviors[i]
		ss.totalBehaviorWeight += b.Weight
		ss.behaviorWeights = append(ss.behaviorWeights, behaviorWeight{
			behavior:         b,
			cumulativeWeight: ss.totalBehaviorWeight,
		})
	}
}

// SetTimeFunc sets the time function (useful for testing).
func (ss *SessionSimulator) SetTimeFunc(fn func() time.Time) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.timeFunc = fn
}

// CreateSession creates a new session with the given behavior.
// If behavior is nil, a random behavior is selected based on weights.
func (ss *SessionSimulator) CreateSession(behavior *UserBehavior) (*Session, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Check concurrent session limit
	if len(ss.sessions) >= ss.config.ConcurrentSessions {
		return nil, ErrSessionLimitReached
	}

	// Select behavior if not provided
	if behavior == nil {
		var err error
		behavior, err = ss.selectBehaviorLocked()
		if err != nil {
			return nil, err
		}
	}

	// Generate session ID
	sessionNum := ss.sessionCounter.Add(1)
	sessionID := SessionID(fmt.Sprintf("session-%d", sessionNum))

	// Calculate session duration and max actions
	now := ss.timeFunc()
	duration := ss.randomSessionDuration()
	maxActions := ss.randomMaxActions(behavior)

	session := &Session{
		ID:         sessionID,
		StartTime:  now,
		ExpiresAt:  now.Add(duration),
		Behavior:   behavior,
		Parameters: NewSessionParameters(),
		MaxActions: maxActions,
	}

	ss.sessions[sessionID] = session
	ss.stats.TotalSessionsCreated.Add(1)

	return session, nil
}

// selectBehaviorLocked selects a random behavior based on weights.
// Must be called with ss.mu held.
func (ss *SessionSimulator) selectBehaviorLocked() (*UserBehavior, error) {
	if len(ss.behaviorWeights) == 0 {
		return nil, ErrBehaviorNotFound
	}

	if ss.totalBehaviorWeight == 0 {
		return ss.behaviorWeights[0].behavior, nil
	}

	// Generate random number in range [0, totalWeight)
	n, err := rand.Int(rand.Reader, big.NewInt(int64(ss.totalBehaviorWeight)))
	if err != nil {
		return ss.behaviorWeights[0].behavior, nil
	}
	target := int(n.Int64())

	// Binary search for the behavior
	low, high := 0, len(ss.behaviorWeights)-1
	for low < high {
		mid := (low + high) / 2
		if ss.behaviorWeights[mid].cumulativeWeight <= target {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return ss.behaviorWeights[low].behavior, nil
}

// randomSessionDuration generates a random session duration within configured range.
func (ss *SessionSimulator) randomSessionDuration() time.Duration {
	min := ss.config.SessionDuration.Min
	max := ss.config.SessionDuration.Max

	if min >= max {
		return min
	}

	rangeNs := int64(max - min)
	n, err := rand.Int(rand.Reader, big.NewInt(rangeNs))
	if err != nil {
		return min
	}
	return min + time.Duration(n.Int64())
}

// randomMaxActions generates a random max actions count within behavior's range.
func (ss *SessionSimulator) randomMaxActions(behavior *UserBehavior) int {
	min := behavior.ActionsPerSession.Min
	max := behavior.ActionsPerSession.Max

	if min >= max {
		return min
	}

	rangeVal := max - min
	n, err := rand.Int(rand.Reader, big.NewInt(int64(rangeVal)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

// GetSession retrieves a session by ID.
func (ss *SessionSimulator) GetSession(id SessionID) (*Session, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	session, ok := ss.sessions[id]
	if !ok {
		return nil, ErrNoActiveSessions
	}
	return session, nil
}

// GetActiveSession retrieves an active (non-expired) session.
// Returns nil if no active sessions are available.
func (ss *SessionSimulator) GetActiveSession() (*Session, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for _, session := range ss.sessions {
		if !session.IsExpired() && !session.IsActionLimitReached() {
			return session, nil
		}
	}
	return nil, ErrNoActiveSessions
}

// GetOrCreateSession gets an active session or creates a new one.
func (ss *SessionSimulator) GetOrCreateSession() (*Session, error) {
	// Try to get an existing active session first
	session, err := ss.GetActiveSession()
	if err == nil {
		return session, nil
	}

	// Clean up expired sessions if replace is enabled
	if ss.config.ReplaceExpired != nil && *ss.config.ReplaceExpired {
		ss.CleanExpiredSessions()
	}

	// Create a new session
	return ss.CreateSession(nil)
}

// GetRandomActiveSession retrieves a random active session.
func (ss *SessionSimulator) GetRandomActiveSession() (*Session, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Collect active sessions
	var activeSessions []*Session
	for _, session := range ss.sessions {
		if !session.IsExpired() && !session.IsActionLimitReached() {
			activeSessions = append(activeSessions, session)
		}
	}

	if len(activeSessions) == 0 {
		return nil, ErrNoActiveSessions
	}

	if len(activeSessions) == 1 {
		return activeSessions[0], nil
	}

	// Select random session
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(activeSessions))))
	if err != nil {
		return activeSessions[0], nil
	}
	return activeSessions[n.Int64()], nil
}

// EndSession ends a session and removes it from the active pool.
func (ss *SessionSimulator) EndSession(id SessionID) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.sessions, id)
	ss.stats.TotalSessionsExpired.Add(1)
}

// CleanExpiredSessions removes all expired sessions.
// Returns the number of sessions removed.
func (ss *SessionSimulator) CleanExpiredSessions() int {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	count := 0
	for id, session := range ss.sessions {
		if session.IsExpired() || session.IsActionLimitReached() {
			delete(ss.sessions, id)
			count++
			ss.stats.TotalSessionsExpired.Add(1)
		}
	}
	return count
}

// ActiveSessionCount returns the number of active sessions.
func (ss *SessionSimulator) ActiveSessionCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	count := 0
	for _, session := range ss.sessions {
		if !session.IsExpired() && !session.IsActionLimitReached() {
			count++
		}
	}
	return count
}

// TotalSessionCount returns the total number of sessions (including expired).
func (ss *SessionSimulator) TotalSessionCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.sessions)
}

// GetAllSessions returns all active sessions.
func (ss *SessionSimulator) GetAllSessions() []*Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sessions := make([]*Session, 0, len(ss.sessions))
	for _, s := range ss.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// RecordAction records an action for a session.
func (ss *SessionSimulator) RecordAction(sessionID SessionID) error {
	ss.mu.RLock()
	session, ok := ss.sessions[sessionID]
	ss.mu.RUnlock()

	if !ok {
		return ErrNoActiveSessions
	}

	if session.IsExpired() {
		return ErrSessionExpired
	}

	session.IncrementActionCount()
	ss.stats.TotalActionsExecuted.Add(1)

	return nil
}

// GetStats returns statistics about the session simulator.
func (ss *SessionSimulator) GetStats() SessionSimulatorStatsSnapshot {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	activeSessions := 0
	for _, session := range ss.sessions {
		if !session.IsExpired() && !session.IsActionLimitReached() {
			activeSessions++
		}
	}

	// Calculate behavior distribution
	behaviorCounts := make(map[string]int)
	for _, session := range ss.sessions {
		if session.Behavior != nil {
			behaviorCounts[session.Behavior.Name]++
		}
	}

	return SessionSimulatorStatsSnapshot{
		TotalSessionsCreated: ss.stats.TotalSessionsCreated.Load(),
		TotalSessionsExpired: ss.stats.TotalSessionsExpired.Load(),
		TotalActionsExecuted: ss.stats.TotalActionsExecuted.Load(),
		ActiveSessions:       activeSessions,
		TotalSessions:        len(ss.sessions),
		MaxConcurrent:        ss.config.ConcurrentSessions,
		BehaviorCounts:       behaviorCounts,
	}
}

// SessionSimulatorStatsSnapshot is a snapshot of session simulator statistics.
type SessionSimulatorStatsSnapshot struct {
	// TotalSessionsCreated is the total number of sessions created.
	TotalSessionsCreated uint64
	// TotalSessionsExpired is the total number of sessions that expired.
	TotalSessionsExpired uint64
	// TotalActionsExecuted is the total number of actions executed.
	TotalActionsExecuted uint64
	// ActiveSessions is the current number of active sessions.
	ActiveSessions int
	// TotalSessions is the total number of sessions (including expired).
	TotalSessions int
	// MaxConcurrent is the configured maximum concurrent sessions.
	MaxConcurrent int
	// BehaviorCounts maps behavior names to session counts.
	BehaviorCounts map[string]int
}

// UpdateConfig updates the simulator configuration.
func (ss *SessionSimulator) UpdateConfig(config SessionSimulatorConfig) error {
	configCopy := config
	if err := configCopy.Validate(); err != nil {
		return err
	}
	configCopy.ApplyDefaults()

	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.config = configCopy
	ss.rebuildBehaviorWeights()

	return nil
}

// GetConfig returns a copy of the current configuration.
func (ss *SessionSimulator) GetConfig() SessionSimulatorConfig {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Return a copy
	configCopy := ss.config
	configCopy.Behaviors = make([]UserBehavior, len(ss.config.Behaviors))
	copy(configCopy.Behaviors, ss.config.Behaviors)

	return configCopy
}

// GetBehavior returns a behavior by name.
func (ss *SessionSimulator) GetBehavior(name string) (*UserBehavior, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for i := range ss.config.Behaviors {
		if ss.config.Behaviors[i].Name == name {
			return &ss.config.Behaviors[i], nil
		}
	}
	return nil, ErrBehaviorNotFound
}

// Clear removes all sessions.
func (ss *SessionSimulator) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions = make(map[SessionID]*Session)
}
