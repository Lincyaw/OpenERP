// Package selector provides endpoint selection strategies for the load generator.
package selector

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Errors returned by the schedule package.
var (
	// ErrInvalidTimeRange is returned when a time range string is invalid.
	ErrInvalidTimeRange = errors.New("schedule: invalid time range format")
	// ErrInvalidCronExpression is returned when a cron expression is invalid.
	ErrInvalidCronExpression = errors.New("schedule: invalid cron expression")
	// ErrEndTimeBeforeStart is returned when end time is before start time.
	ErrEndTimeBeforeStart = errors.New("schedule: end time must be after start time")
	// ErrInvalidWeight is returned when a weight value is invalid.
	ErrInvalidScheduleWeight = errors.New("schedule: weight must be non-negative")
)

// TimeRange represents a time-of-day range (e.g., 09:00-12:00).
type TimeRange struct {
	// StartHour is the start hour (0-23).
	StartHour int
	// StartMinute is the start minute (0-59).
	StartMinute int
	// EndHour is the end hour (0-23).
	EndHour int
	// EndMinute is the end minute (0-59).
	EndMinute int
}

// Contains checks if the given time falls within this range.
// Note: If the range crosses midnight (e.g., 22:00-02:00), it handles wrapping.
func (tr TimeRange) Contains(t time.Time) bool {
	currentMinutes := t.Hour()*60 + t.Minute()
	startMinutes := tr.StartHour*60 + tr.StartMinute
	endMinutes := tr.EndHour*60 + tr.EndMinute

	// Handle ranges that cross midnight
	if startMinutes > endMinutes {
		// Range crosses midnight (e.g., 22:00-02:00)
		return currentMinutes >= startMinutes || currentMinutes < endMinutes
	}

	// Normal range within the same day
	return currentMinutes >= startMinutes && currentMinutes < endMinutes
}

// String returns the time range in "HH:MM-HH:MM" format.
func (tr TimeRange) String() string {
	return fmt.Sprintf("%02d:%02d-%02d:%02d", tr.StartHour, tr.StartMinute, tr.EndHour, tr.EndMinute)
}

// ParseTimeRange parses a time range string in format "HH:MM-HH:MM".
// Examples: "09:00-12:00", "14:00-18:00", "22:00-02:00" (crosses midnight)
func ParseTimeRange(s string) (TimeRange, error) {
	// Pattern: HH:MM-HH:MM
	pattern := regexp.MustCompile(`^(\d{1,2}):(\d{2})-(\d{1,2}):(\d{2})$`)
	matches := pattern.FindStringSubmatch(strings.TrimSpace(s))
	if matches == nil {
		return TimeRange{}, fmt.Errorf("%w: %s (expected format: HH:MM-HH:MM)", ErrInvalidTimeRange, s)
	}

	startHour, _ := strconv.Atoi(matches[1])
	startMinute, _ := strconv.Atoi(matches[2])
	endHour, _ := strconv.Atoi(matches[3])
	endMinute, _ := strconv.Atoi(matches[4])

	// Validate ranges
	if startHour > 23 || endHour > 23 {
		return TimeRange{}, fmt.Errorf("%w: hour must be 0-23", ErrInvalidTimeRange)
	}
	if startMinute > 59 || endMinute > 59 {
		return TimeRange{}, fmt.Errorf("%w: minute must be 0-59", ErrInvalidTimeRange)
	}

	return TimeRange{
		StartHour:   startHour,
		StartMinute: startMinute,
		EndHour:     endHour,
		EndMinute:   endMinute,
	}, nil
}

// TimeBasedWeight defines a weight adjustment for a specific time period.
type TimeBasedWeight struct {
	// TimeRange is the time range string (e.g., "09:00-12:00").
	TimeRange string `yaml:"time" json:"time"`

	// Weight is the endpoint weight during this time period.
	// This replaces the base weight when the time condition matches.
	Weight int `yaml:"weight" json:"weight"`

	// Cron is an optional cron expression for more complex schedules.
	// When set, it overrides TimeRange.
	// Simplified format: "minute hour day-of-month month day-of-week"
	Cron string `yaml:"cron,omitempty" json:"cron,omitempty"`

	// Modifier is a multiplier applied to the base weight instead of replacing it.
	// If set (> 0), it multiplies the base weight instead of replacing.
	// For example, Modifier: 2.0 doubles the weight.
	Modifier float64 `yaml:"modifier,omitempty" json:"modifier,omitempty"`

	// parsedRange is the cached parsed time range.
	parsedRange *TimeRange

	// parsedCron is the cached parsed cron expression.
	parsedCron *CronSchedule
}

// Validate validates the TimeBasedWeight configuration.
func (tbw *TimeBasedWeight) Validate() error {
	if tbw.Weight < 0 {
		return ErrInvalidScheduleWeight
	}

	if tbw.Cron != "" {
		// Validate cron expression
		_, err := ParseCronSchedule(tbw.Cron)
		if err != nil {
			return err
		}
	} else if tbw.TimeRange != "" {
		// Validate time range
		_, err := ParseTimeRange(tbw.TimeRange)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: either time or cron must be specified", ErrInvalidTimeRange)
	}

	return nil
}

// Parse parses and caches the time range or cron expression.
func (tbw *TimeBasedWeight) Parse() error {
	if tbw.Cron != "" {
		cron, err := ParseCronSchedule(tbw.Cron)
		if err != nil {
			return err
		}
		tbw.parsedCron = cron
	} else if tbw.TimeRange != "" {
		tr, err := ParseTimeRange(tbw.TimeRange)
		if err != nil {
			return err
		}
		tbw.parsedRange = &tr
	}
	return nil
}

// Matches checks if the given time matches this schedule.
func (tbw *TimeBasedWeight) Matches(t time.Time) bool {
	if tbw.parsedCron != nil {
		return tbw.parsedCron.Matches(t)
	}
	if tbw.parsedRange != nil {
		return tbw.parsedRange.Contains(t)
	}
	return false
}

// GetWeight returns the effective weight for the given base weight.
// If Modifier is set, it applies the modifier to the base weight.
// Otherwise, it returns the configured Weight.
func (tbw *TimeBasedWeight) GetWeight(baseWeight int) int {
	if tbw.Modifier > 0 {
		return int(float64(baseWeight) * tbw.Modifier)
	}
	return tbw.Weight
}

// CronSchedule represents a simplified cron schedule.
// Format: "minute hour day-of-month month day-of-week"
// Supports: numbers, ranges (1-5), lists (1,3,5), wildcards (*)
type CronSchedule struct {
	Minutes     []int // 0-59
	Hours       []int // 0-23
	DaysOfMonth []int // 1-31
	Months      []int // 1-12
	DaysOfWeek  []int // 0-6 (Sunday = 0)
}

// Matches checks if the given time matches this cron schedule.
func (cs *CronSchedule) Matches(t time.Time) bool {
	return cs.matchesSlice(cs.Minutes, t.Minute()) &&
		cs.matchesSlice(cs.Hours, t.Hour()) &&
		cs.matchesSlice(cs.DaysOfMonth, t.Day()) &&
		cs.matchesSlice(cs.Months, int(t.Month())) &&
		cs.matchesSlice(cs.DaysOfWeek, int(t.Weekday()))
}

func (cs *CronSchedule) matchesSlice(allowed []int, value int) bool {
	if len(allowed) == 0 {
		return true // Wildcard (*)
	}
	return slices.Contains(allowed, value)
}

// ParseCronSchedule parses a simplified cron expression.
// Format: "minute hour day-of-month month day-of-week"
// Examples:
//   - "* 9-17 * * 1-5" - 9am-5pm on weekdays
//   - "0 9 * * *" - 9:00am every day
//   - "*/15 * * * *" - every 15 minutes
func ParseCronSchedule(expr string) (*CronSchedule, error) {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return nil, fmt.Errorf("%w: expected 5 fields, got %d", ErrInvalidCronExpression, len(parts))
	}

	cs := &CronSchedule{}
	var err error

	cs.Minutes, err = parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("%w: minutes: %v", ErrInvalidCronExpression, err)
	}

	cs.Hours, err = parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("%w: hours: %v", ErrInvalidCronExpression, err)
	}

	cs.DaysOfMonth, err = parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("%w: days of month: %v", ErrInvalidCronExpression, err)
	}

	cs.Months, err = parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("%w: months: %v", ErrInvalidCronExpression, err)
	}

	cs.DaysOfWeek, err = parseCronField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("%w: days of week: %v", ErrInvalidCronExpression, err)
	}

	return cs, nil
}

// parseCronField parses a single cron field.
// Supports: numbers, ranges (1-5), lists (1,3,5), wildcards (*), steps (*/15, 1-10/2)
func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return nil, nil // Wildcard matches all values
	}

	var result []int

	// Handle lists (comma-separated)
	parts := strings.Split(field, ",")
	for _, part := range parts {
		values, err := parseCronPart(part, min, max)
		if err != nil {
			return nil, err
		}
		result = append(result, values...)
	}

	return dedupe(result), nil
}

// parseCronPart parses a single part of a cron field (range, step, or number).
func parseCronPart(part string, min, max int) ([]int, error) {
	// Handle step values (*/15 or 1-10/2)
	if strings.Contains(part, "/") {
		stepParts := strings.SplitN(part, "/", 2)
		step, err := strconv.Atoi(stepParts[1])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step: %s", stepParts[1])
		}

		var rangeMin, rangeMax int
		if stepParts[0] == "*" {
			rangeMin, rangeMax = min, max
		} else {
			rangeMin, rangeMax, err = parseRange(stepParts[0], min, max)
			if err != nil {
				return nil, err
			}
		}

		var result []int
		for i := rangeMin; i <= rangeMax; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// Handle ranges (1-5)
	if strings.Contains(part, "-") {
		rangeMin, rangeMax, err := parseRange(part, min, max)
		if err != nil {
			return nil, err
		}
		var result []int
		for i := rangeMin; i <= rangeMax; i++ {
			result = append(result, i)
		}
		return result, nil
	}

	// Handle single number
	num, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %s", part)
	}
	if num < min || num > max {
		return nil, fmt.Errorf("value %d out of range [%d-%d]", num, min, max)
	}
	return []int{num}, nil
}

// parseRange parses a range expression (e.g., "1-5").
func parseRange(s string, min, max int) (int, int, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range: %s", s)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range start: %s", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range end: %s", parts[1])
	}

	if start < min || start > max || end < min || end > max {
		return 0, 0, fmt.Errorf("range [%d-%d] out of bounds [%d-%d]", start, end, min, max)
	}

	if start > end {
		return 0, 0, fmt.Errorf("range start %d > end %d", start, end)
	}

	return start, end, nil
}

// dedupe removes duplicates from a slice and returns a sorted result.
func dedupe(slice []int) []int {
	if len(slice) == 0 {
		return slice
	}

	seen := make(map[int]struct{})
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	// Sort the result for deterministic behavior using standard library (O(n log n))
	slices.Sort(result)

	return result
}

// TimeSchedule represents a global time-based weight modifier.
// Unlike TimeBasedWeight which is endpoint-specific, TimeSchedule applies globally.
type TimeSchedule struct {
	// Start is the start time of the schedule.
	Start time.Time `yaml:"start" json:"start"`
	// End is the end time of the schedule.
	End time.Time `yaml:"end" json:"end"`
	// Modifier is the weight multiplier (e.g., 1.5 = 50% increase).
	Modifier float64 `yaml:"modifier" json:"modifier"`
}

// Contains checks if the given time falls within this schedule.
func (ts TimeSchedule) Contains(t time.Time) bool {
	return !t.Before(ts.Start) && t.Before(ts.End)
}

// EndpointSchedule holds time-based scheduling for an endpoint.
type EndpointSchedule struct {
	// EndpointName is the name of the endpoint.
	EndpointName string

	// Schedules holds time-based weight configurations.
	Schedules []TimeBasedWeight

	// mu protects the schedules slice during updates.
	mu sync.RWMutex
}

// NewEndpointSchedule creates a new endpoint schedule.
func NewEndpointSchedule(endpointName string, schedules []TimeBasedWeight) (*EndpointSchedule, error) {
	es := &EndpointSchedule{
		EndpointName: endpointName,
		Schedules:    make([]TimeBasedWeight, 0, len(schedules)),
	}

	// Parse and validate all schedules
	for _, s := range schedules {
		schedule := s // Copy
		if err := schedule.Validate(); err != nil {
			return nil, fmt.Errorf("endpoint %s: %w", endpointName, err)
		}
		if err := schedule.Parse(); err != nil {
			return nil, fmt.Errorf("endpoint %s: %w", endpointName, err)
		}
		es.Schedules = append(es.Schedules, schedule)
	}

	return es, nil
}

// GetWeight returns the effective weight for the given time and base weight.
// If no schedule matches, returns the base weight.
// If multiple schedules match, the first matching schedule wins.
func (es *EndpointSchedule) GetWeight(t time.Time, baseWeight int) int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	for i := range es.Schedules {
		if es.Schedules[i].Matches(t) {
			return es.Schedules[i].GetWeight(baseWeight)
		}
	}

	return baseWeight
}

// TimeAwareScheduler wraps a scheduler with time-based weight adjustments.
type TimeAwareScheduler struct {
	// mu protects the endpoint schedules map.
	mu sync.RWMutex

	// endpointSchedules maps endpoint names to their schedules.
	endpointSchedules map[string]*EndpointSchedule

	// globalSchedules holds global time schedules that apply to all endpoints.
	globalSchedules []TimeSchedule

	// timeFunc returns the current time. Defaults to time.Now.
	// Can be overridden for testing.
	timeFunc func() time.Time
}

// NewTimeAwareScheduler creates a new time-aware scheduler.
func NewTimeAwareScheduler() *TimeAwareScheduler {
	return &TimeAwareScheduler{
		endpointSchedules: make(map[string]*EndpointSchedule),
		globalSchedules:   make([]TimeSchedule, 0),
		timeFunc:          time.Now,
	}
}

// SetTimeFunc sets the time function (useful for testing).
func (tas *TimeAwareScheduler) SetTimeFunc(fn func() time.Time) {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	tas.timeFunc = fn
}

// AddEndpointSchedule adds a schedule for an endpoint.
func (tas *TimeAwareScheduler) AddEndpointSchedule(endpointName string, schedules []TimeBasedWeight) error {
	es, err := NewEndpointSchedule(endpointName, schedules)
	if err != nil {
		return err
	}

	tas.mu.Lock()
	defer tas.mu.Unlock()
	tas.endpointSchedules[endpointName] = es

	return nil
}

// AddGlobalSchedule adds a global time schedule.
func (tas *TimeAwareScheduler) AddGlobalSchedule(schedule TimeSchedule) error {
	if !schedule.End.After(schedule.Start) {
		return ErrEndTimeBeforeStart
	}

	tas.mu.Lock()
	defer tas.mu.Unlock()
	tas.globalSchedules = append(tas.globalSchedules, schedule)

	return nil
}

// GetEffectiveWeight calculates the effective weight for an endpoint at the current time.
// The calculation order is:
// 1. Check endpoint-specific schedules (highest priority)
// 2. Apply global schedule modifiers
// 3. Return base weight if no schedules match
func (tas *TimeAwareScheduler) GetEffectiveWeight(endpointName string, baseWeight int) int {
	tas.mu.RLock()
	defer tas.mu.RUnlock()

	currentTime := tas.timeFunc()

	// Check endpoint-specific schedules first
	if es, ok := tas.endpointSchedules[endpointName]; ok {
		weight := es.GetWeight(currentTime, baseWeight)
		if weight != baseWeight {
			// Endpoint schedule matched, apply global modifier if any
			return tas.applyGlobalModifier(currentTime, weight)
		}
	}

	// Apply global modifier to base weight
	return tas.applyGlobalModifier(currentTime, baseWeight)
}

// applyGlobalModifier applies any active global schedule modifiers.
// Must be called with tas.mu held.
func (tas *TimeAwareScheduler) applyGlobalModifier(t time.Time, weight int) int {
	for _, gs := range tas.globalSchedules {
		if gs.Contains(t) {
			return int(float64(weight) * gs.Modifier)
		}
	}
	return weight
}

// GetActiveSchedules returns information about currently active schedules.
func (tas *TimeAwareScheduler) GetActiveSchedules() []string {
	tas.mu.RLock()
	defer tas.mu.RUnlock()

	return tas.getActiveSchedulesLocked()
}

// getActiveSchedulesLocked returns active schedules. Must be called with tas.mu held.
func (tas *TimeAwareScheduler) getActiveSchedulesLocked() []string {
	currentTime := tas.timeFunc()
	var active []string

	// Get sorted endpoint names for deterministic output
	names := make([]string, 0, len(tas.endpointSchedules))
	for name := range tas.endpointSchedules {
		names = append(names, name)
	}
	slices.Sort(names)

	// Check endpoint schedules
	for _, name := range names {
		es := tas.endpointSchedules[name]
		for _, s := range es.Schedules {
			if s.Matches(currentTime) {
				if s.Modifier > 0 {
					active = append(active, fmt.Sprintf("%s: modifier=%.2f", name, s.Modifier))
				} else {
					active = append(active, fmt.Sprintf("%s: weight=%d", name, s.Weight))
				}
			}
		}
	}

	// Check global schedules
	for i, gs := range tas.globalSchedules {
		if gs.Contains(currentTime) {
			active = append(active, fmt.Sprintf("global[%d]: modifier=%.2f", i, gs.Modifier))
		}
	}

	return active
}

// Stats returns statistics about the time-aware scheduler.
type TimeAwareSchedulerStats struct {
	// EndpointScheduleCount is the number of endpoints with schedules.
	EndpointScheduleCount int
	// GlobalScheduleCount is the number of global schedules.
	GlobalScheduleCount int
	// ActiveSchedules is the list of currently active schedule descriptions.
	ActiveSchedules []string
}

// GetStats returns statistics about the scheduler.
func (tas *TimeAwareScheduler) GetStats() TimeAwareSchedulerStats {
	tas.mu.RLock()
	defer tas.mu.RUnlock()

	return TimeAwareSchedulerStats{
		EndpointScheduleCount: len(tas.endpointSchedules),
		GlobalScheduleCount:   len(tas.globalSchedules),
		ActiveSchedules:       tas.getActiveSchedulesLocked(),
	}
}

// ClearSchedules removes all schedules.
func (tas *TimeAwareScheduler) ClearSchedules() {
	tas.mu.Lock()
	defer tas.mu.Unlock()

	tas.endpointSchedules = make(map[string]*EndpointSchedule)
	tas.globalSchedules = make([]TimeSchedule, 0)
}

// RemoveEndpointSchedule removes the schedule for an endpoint.
func (tas *TimeAwareScheduler) RemoveEndpointSchedule(endpointName string) {
	tas.mu.Lock()
	defer tas.mu.Unlock()

	delete(tas.endpointSchedules, endpointName)
}
