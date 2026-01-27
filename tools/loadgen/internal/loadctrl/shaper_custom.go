// Package loadctrl provides load control components including traffic shaping.
package loadctrl

import (
	"fmt"
	"sort"
	"time"
)

// CustomShaper implements a user-defined traffic pattern using custom data points.
// QPS is linearly interpolated between the defined points.
//
// Example points:
//
//	Time: 0s, QPS: 10
//	Time: 30s, QPS: 100
//	Time: 60s, QPS: 50
//
// Results in: ramp from 10 to 100 over 30s, then ramp down to 50 over next 30s.
//
// Thread Safety: Safe for concurrent use by multiple goroutines (read-only after creation).
type CustomShaper struct {
	config        ShaperConfig
	points        []CustomPoint
	totalDuration time.Duration
}

// NewCustomShaper creates a new custom traffic shaper.
// Points must be provided in chronological order with at least 2 points.
// The first point should ideally be at Time: 0, but if not, the shaper will
// use the first point's QPS for times before it.
func NewCustomShaper(config ShaperConfig) (*CustomShaper, error) {
	if config.Type != "custom" {
		return nil, fmt.Errorf("expected type 'custom', got '%s'", config.Type)
	}

	if len(config.CustomPoints) < 2 {
		return nil, fmt.Errorf("at least 2 custom points are required, got: %d", len(config.CustomPoints))
	}

	// Validate and sort points by time
	points := make([]CustomPoint, len(config.CustomPoints))
	copy(points, config.CustomPoints)
	sort.Slice(points, func(i, j int) bool {
		return points[i].Time < points[j].Time
	})

	// Verify all points have non-negative QPS
	for i, pt := range points {
		if pt.QPS < 0 {
			return nil, fmt.Errorf("point %d: QPS cannot be negative: %f", i, pt.QPS)
		}
	}

	// Calculate total duration (from first to last point)
	totalDuration := points[len(points)-1].Time

	return &CustomShaper{
		config:        config,
		points:        points,
		totalDuration: totalDuration,
	}, nil
}

// GetTargetQPS returns the target QPS for the given elapsed time.
// Uses linear interpolation between the defined points.
// Before the first point: uses first point's QPS.
// After the last point: uses last point's QPS.
func (s *CustomShaper) GetTargetQPS(elapsed time.Duration) float64 {
	if len(s.points) == 0 {
		return 0
	}

	// Before first point
	if elapsed <= s.points[0].Time {
		return clampQPS(s.points[0].QPS, s.config.MinQPS, s.config.MaxQPS)
	}

	// After last point
	if elapsed >= s.points[len(s.points)-1].Time {
		return clampQPS(s.points[len(s.points)-1].QPS, s.config.MinQPS, s.config.MaxQPS)
	}

	// Find the two points to interpolate between
	for i := 1; i < len(s.points); i++ {
		if elapsed <= s.points[i].Time {
			// Interpolate between points[i-1] and points[i]
			p1 := s.points[i-1]
			p2 := s.points[i]

			// Calculate interpolation factor (0 to 1)
			t := float64(elapsed-p1.Time) / float64(p2.Time-p1.Time)

			// Linear interpolation
			qps := p1.QPS + t*(p2.QPS-p1.QPS)

			return clampQPS(qps, s.config.MinQPS, s.config.MaxQPS)
		}
	}

	// Should not reach here, but return last point's QPS as fallback
	return clampQPS(s.points[len(s.points)-1].QPS, s.config.MinQPS, s.config.MaxQPS)
}

// GetPhase returns a human-readable description of the current phase.
func (s *CustomShaper) GetPhase(elapsed time.Duration) string {
	if len(s.points) == 0 {
		return "no points defined"
	}

	// Before first point
	if elapsed <= s.points[0].Time {
		return fmt.Sprintf("before curve start (%.1fs until point 1)", (s.points[0].Time - elapsed).Seconds())
	}

	// After last point
	if elapsed >= s.points[len(s.points)-1].Time {
		overtime := elapsed - s.points[len(s.points)-1].Time
		return fmt.Sprintf("curve complete, holding at %.0f QPS (+%.1fs)", s.points[len(s.points)-1].QPS, overtime.Seconds())
	}

	// Find current segment
	for i := 1; i < len(s.points); i++ {
		if elapsed <= s.points[i].Time {
			p1 := s.points[i-1]
			p2 := s.points[i]
			segmentDuration := p2.Time - p1.Time
			posInSegment := elapsed - p1.Time
			progress := float64(posInSegment) / float64(segmentDuration) * 100

			direction := "ramping up"
			if p2.QPS < p1.QPS {
				direction = "ramping down"
			} else if p2.QPS == p1.QPS {
				direction = "holding steady"
			}

			return fmt.Sprintf("segment %d/%d: %s from %.0f to %.0f QPS (%.1f%%)",
				i, len(s.points)-1, direction, p1.QPS, p2.QPS, progress)
		}
	}

	return "unknown phase"
}

// Name returns the name of this shaper type.
func (s *CustomShaper) Name() string {
	return "custom"
}

// Config returns a copy of the shaper's configuration.
func (s *CustomShaper) Config() ShaperConfig {
	return s.config
}

// GetTotalDuration returns the duration from first to last point.
func (s *CustomShaper) GetTotalDuration() time.Duration {
	return s.totalDuration
}

// GetPointCount returns the number of custom points.
func (s *CustomShaper) GetPointCount() int {
	return len(s.points)
}

// GetPoint returns the point at the given index.
func (s *CustomShaper) GetPoint(idx int) (CustomPoint, bool) {
	if idx < 0 || idx >= len(s.points) {
		return CustomPoint{}, false
	}
	return s.points[idx], true
}

// GetPoints returns a copy of all points.
func (s *CustomShaper) GetPoints() []CustomPoint {
	result := make([]CustomPoint, len(s.points))
	copy(result, s.points)
	return result
}

// CurrentSegment returns the indices of the two points defining the current segment.
// Returns (0, 0) if before first point, (n-1, n-1) if after last point.
func (s *CustomShaper) CurrentSegment(elapsed time.Duration) (startIdx, endIdx int) {
	if len(s.points) == 0 {
		return 0, 0
	}

	// Before first point
	if elapsed <= s.points[0].Time {
		return 0, 0
	}

	// After last point
	if elapsed >= s.points[len(s.points)-1].Time {
		lastIdx := len(s.points) - 1
		return lastIdx, lastIdx
	}

	// Find current segment
	for i := 1; i < len(s.points); i++ {
		if elapsed <= s.points[i].Time {
			return i - 1, i
		}
	}

	lastIdx := len(s.points) - 1
	return lastIdx, lastIdx
}

// GetMinMaxQPS returns the minimum and maximum QPS values from all points.
func (s *CustomShaper) GetMinMaxQPS() (min, max float64) {
	if len(s.points) == 0 {
		return 0, 0
	}

	min = s.points[0].QPS
	max = s.points[0].QPS

	for _, pt := range s.points {
		if pt.QPS < min {
			min = pt.QPS
		}
		if pt.QPS > max {
			max = pt.QPS
		}
	}

	return min, max
}

// TimeUntilNextPoint returns the duration until the next point.
// Returns 0 if at or past the last point.
func (s *CustomShaper) TimeUntilNextPoint(elapsed time.Duration) time.Duration {
	if len(s.points) == 0 {
		return 0
	}

	for _, pt := range s.points {
		if elapsed < pt.Time {
			return pt.Time - elapsed
		}
	}

	return 0
}
