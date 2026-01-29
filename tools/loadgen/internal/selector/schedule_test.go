package selector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TimeRange
		wantErr  bool
	}{
		{
			name:  "valid range morning",
			input: "09:00-12:00",
			expected: TimeRange{
				StartHour:   9,
				StartMinute: 0,
				EndHour:     12,
				EndMinute:   0,
			},
			wantErr: false,
		},
		{
			name:  "valid range afternoon",
			input: "14:00-18:00",
			expected: TimeRange{
				StartHour:   14,
				StartMinute: 0,
				EndHour:     18,
				EndMinute:   0,
			},
			wantErr: false,
		},
		{
			name:  "valid range with minutes",
			input: "09:30-17:45",
			expected: TimeRange{
				StartHour:   9,
				StartMinute: 30,
				EndHour:     17,
				EndMinute:   45,
			},
			wantErr: false,
		},
		{
			name:  "valid range crossing midnight",
			input: "22:00-02:00",
			expected: TimeRange{
				StartHour:   22,
				StartMinute: 0,
				EndHour:     2,
				EndMinute:   0,
			},
			wantErr: false,
		},
		{
			name:  "valid single digit hour",
			input: "9:00-12:00",
			expected: TimeRange{
				StartHour:   9,
				StartMinute: 0,
				EndHour:     12,
				EndMinute:   0,
			},
			wantErr: false,
		},
		{
			name:    "invalid format - missing dash",
			input:   "09:00 12:00",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong separator",
			input:   "09.00-12.00",
			wantErr: true,
		},
		{
			name:    "invalid hour > 23",
			input:   "25:00-12:00",
			wantErr: true,
		},
		{
			name:    "invalid minute > 59",
			input:   "09:60-12:00",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimeRange(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeRangeContains(t *testing.T) {
	// Test normal range (09:00-12:00)
	normalRange := TimeRange{
		StartHour:   9,
		StartMinute: 0,
		EndHour:     12,
		EndMinute:   0,
	}

	t.Run("normal range contains time within", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.True(t, normalRange.Contains(testTime))
	})

	t.Run("normal range contains start time", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		assert.True(t, normalRange.Contains(testTime))
	})

	t.Run("normal range excludes end time", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		assert.False(t, normalRange.Contains(testTime))
	})

	t.Run("normal range excludes time before", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 8, 59, 0, 0, time.UTC)
		assert.False(t, normalRange.Contains(testTime))
	})

	t.Run("normal range excludes time after", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
		assert.False(t, normalRange.Contains(testTime))
	})

	// Test range crossing midnight (22:00-02:00)
	midnightRange := TimeRange{
		StartHour:   22,
		StartMinute: 0,
		EndHour:     2,
		EndMinute:   0,
	}

	t.Run("midnight range contains time before midnight", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 23, 30, 0, 0, time.UTC)
		assert.True(t, midnightRange.Contains(testTime))
	})

	t.Run("midnight range contains time after midnight", func(t *testing.T) {
		testTime := time.Date(2024, 1, 16, 1, 30, 0, 0, time.UTC)
		assert.True(t, midnightRange.Contains(testTime))
	})

	t.Run("midnight range contains start time", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC)
		assert.True(t, midnightRange.Contains(testTime))
	})

	t.Run("midnight range excludes end time", func(t *testing.T) {
		testTime := time.Date(2024, 1, 16, 2, 0, 0, 0, time.UTC)
		assert.False(t, midnightRange.Contains(testTime))
	})

	t.Run("midnight range excludes afternoon", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		assert.False(t, midnightRange.Contains(testTime))
	})
}

func TestTimeRangeString(t *testing.T) {
	tr := TimeRange{
		StartHour:   9,
		StartMinute: 30,
		EndHour:     17,
		EndMinute:   45,
	}
	assert.Equal(t, "09:30-17:45", tr.String())
}

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *CronSchedule
		wantErr  bool
	}{
		{
			name:  "all wildcards",
			input: "* * * * *",
			expected: &CronSchedule{
				Minutes:     nil,
				Hours:       nil,
				DaysOfMonth: nil,
				Months:      nil,
				DaysOfWeek:  nil,
			},
			wantErr: false,
		},
		{
			name:  "specific time 9:00",
			input: "0 9 * * *",
			expected: &CronSchedule{
				Minutes:     []int{0},
				Hours:       []int{9},
				DaysOfMonth: nil,
				Months:      nil,
				DaysOfWeek:  nil,
			},
			wantErr: false,
		},
		{
			name:  "weekday work hours 9-17",
			input: "* 9-17 * * 1-5",
			expected: &CronSchedule{
				Minutes:     nil,
				Hours:       []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
				DaysOfMonth: nil,
				Months:      nil,
				DaysOfWeek:  []int{1, 2, 3, 4, 5},
			},
			wantErr: false,
		},
		{
			name:  "every 15 minutes",
			input: "*/15 * * * *",
			expected: &CronSchedule{
				Minutes:     []int{0, 15, 30, 45},
				Hours:       nil,
				DaysOfMonth: nil,
				Months:      nil,
				DaysOfWeek:  nil,
			},
			wantErr: false,
		},
		{
			name:  "list of hours",
			input: "0 9,12,18 * * *",
			expected: &CronSchedule{
				Minutes:     []int{0},
				Hours:       []int{9, 12, 18},
				DaysOfMonth: nil,
				Months:      nil,
				DaysOfWeek:  nil,
			},
			wantErr: false,
		},
		{
			name:    "too few fields",
			input:   "* * * *",
			wantErr: true,
		},
		{
			name:    "too many fields",
			input:   "* * * * * *",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			input:   "* 25 * * *",
			wantErr: true,
		},
		{
			name:    "invalid minute",
			input:   "60 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid range end",
			input:   "* 5-3 * * *",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCronSchedule(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCronScheduleMatches(t *testing.T) {
	// Monday, January 15, 2024 at 10:30:00
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		cron     string
		time     time.Time
		expected bool
	}{
		{
			name:     "wildcard matches everything",
			cron:     "* * * * *",
			time:     testTime,
			expected: true,
		},
		{
			name:     "specific minute matches",
			cron:     "30 * * * *",
			time:     testTime,
			expected: true,
		},
		{
			name:     "specific minute doesn't match",
			cron:     "0 * * * *",
			time:     testTime,
			expected: false,
		},
		{
			name:     "hour range matches",
			cron:     "* 9-12 * * *",
			time:     testTime,
			expected: true,
		},
		{
			name:     "hour range doesn't match",
			cron:     "* 14-18 * * *",
			time:     testTime,
			expected: false,
		},
		{
			name:     "weekday matches (Monday = 1)",
			cron:     "* * * * 1",
			time:     testTime,
			expected: true,
		},
		{
			name:     "weekday doesn't match (Sunday = 0)",
			cron:     "* * * * 0",
			time:     testTime,
			expected: false,
		},
		{
			name:     "complex expression matches",
			cron:     "30 10 15 1 1",
			time:     testTime,
			expected: true,
		},
		{
			name:     "complex expression doesn't match month",
			cron:     "30 10 15 2 1",
			time:     testTime,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := ParseCronSchedule(tt.cron)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cs.Matches(tt.time))
		})
	}
}

func TestTimeBasedWeight(t *testing.T) {
	t.Run("validate with valid time range", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Weight:    20,
		}
		assert.NoError(t, tbw.Validate())
	})

	t.Run("validate with valid cron", func(t *testing.T) {
		tbw := TimeBasedWeight{
			Cron:   "* 9-12 * * 1-5",
			Weight: 20,
		}
		assert.NoError(t, tbw.Validate())
	})

	t.Run("validate fails with negative weight", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Weight:    -1,
		}
		assert.Error(t, tbw.Validate())
	})

	t.Run("validate fails with neither time nor cron", func(t *testing.T) {
		tbw := TimeBasedWeight{
			Weight: 10,
		}
		assert.Error(t, tbw.Validate())
	})

	t.Run("validate fails with invalid time range", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "invalid",
			Weight:    10,
		}
		assert.Error(t, tbw.Validate())
	})

	t.Run("validate fails with invalid cron", func(t *testing.T) {
		tbw := TimeBasedWeight{
			Cron:   "invalid",
			Weight: 10,
		}
		assert.Error(t, tbw.Validate())
	})

	t.Run("matches time range", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Weight:    20,
		}
		require.NoError(t, tbw.Parse())

		// 10:30 should match
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.True(t, tbw.Matches(testTime))

		// 14:00 should not match
		testTime = time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
		assert.False(t, tbw.Matches(testTime))
	})

	t.Run("matches cron expression", func(t *testing.T) {
		tbw := TimeBasedWeight{
			Cron:   "* 9-12 * * *",
			Weight: 20,
		}
		require.NoError(t, tbw.Parse())

		// 10:30 should match
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.True(t, tbw.Matches(testTime))

		// 14:00 should not match
		testTime = time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
		assert.False(t, tbw.Matches(testTime))
	})

	t.Run("GetWeight returns configured weight", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Weight:    20,
		}
		assert.Equal(t, 20, tbw.GetWeight(10))
	})

	t.Run("GetWeight applies modifier to base weight", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Modifier:  2.0, // Double the weight
		}
		assert.Equal(t, 20, tbw.GetWeight(10))
	})
}

func TestEndpointSchedule(t *testing.T) {
	t.Run("create valid endpoint schedule", func(t *testing.T) {
		schedules := []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
			{TimeRange: "14:00-18:00", Weight: 15},
		}
		es, err := NewEndpointSchedule("test.endpoint", schedules)
		require.NoError(t, err)
		assert.NotNil(t, es)
		assert.Equal(t, "test.endpoint", es.EndpointName)
		assert.Len(t, es.Schedules, 2)
	})

	t.Run("create fails with invalid schedule", func(t *testing.T) {
		schedules := []TimeBasedWeight{
			{TimeRange: "invalid", Weight: 20},
		}
		es, err := NewEndpointSchedule("test.endpoint", schedules)
		assert.Error(t, err)
		assert.Nil(t, es)
	})

	t.Run("GetWeight returns matched schedule weight", func(t *testing.T) {
		schedules := []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
			{TimeRange: "14:00-18:00", Weight: 15},
		}
		es, err := NewEndpointSchedule("test.endpoint", schedules)
		require.NoError(t, err)

		// 10:30 should return weight 20
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, 20, es.GetWeight(testTime, 10))

		// 16:00 should return weight 15
		testTime = time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)
		assert.Equal(t, 15, es.GetWeight(testTime, 10))

		// 13:00 (lunch) should return base weight 10
		testTime = time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC)
		assert.Equal(t, 10, es.GetWeight(testTime, 10))
	})

	t.Run("first matching schedule wins", func(t *testing.T) {
		schedules := []TimeBasedWeight{
			{TimeRange: "09:00-18:00", Weight: 20}, // Broad range
			{TimeRange: "10:00-11:00", Weight: 30}, // Narrow range (will never match)
		}
		es, err := NewEndpointSchedule("test.endpoint", schedules)
		require.NoError(t, err)

		// 10:30 matches both, but first wins
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Equal(t, 20, es.GetWeight(testTime, 10))
	})
}

func TestTimeAwareScheduler(t *testing.T) {
	t.Run("create scheduler", func(t *testing.T) {
		tas := NewTimeAwareScheduler()
		assert.NotNil(t, tas)
	})

	t.Run("add and use endpoint schedule", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		// Set fixed time to 10:30
		tas.SetTimeFunc(func() time.Time {
			return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		})

		// Add schedule for 09:00-12:00 with weight 20
		err := tas.AddEndpointSchedule("sales.order.create", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
			{TimeRange: "14:00-18:00", Weight: 15},
		})
		require.NoError(t, err)

		// At 10:30, should return 20
		weight := tas.GetEffectiveWeight("sales.order.create", 10)
		assert.Equal(t, 20, weight)

		// Change time to 16:00
		tas.SetTimeFunc(func() time.Time {
			return time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)
		})

		// At 16:00, should return 15
		weight = tas.GetEffectiveWeight("sales.order.create", 10)
		assert.Equal(t, 15, weight)

		// Change time to 13:00 (lunch)
		tas.SetTimeFunc(func() time.Time {
			return time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC)
		})

		// At 13:00, should return base weight 10
		weight = tas.GetEffectiveWeight("sales.order.create", 10)
		assert.Equal(t, 10, weight)
	})

	t.Run("endpoint without schedule returns base weight", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		// No schedules added
		weight := tas.GetEffectiveWeight("unknown.endpoint", 10)
		assert.Equal(t, 10, weight)
	})

	t.Run("global schedule modifier", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		tas.SetTimeFunc(func() time.Time { return now })

		// Add global schedule with 1.5x modifier
		err := tas.AddGlobalSchedule(TimeSchedule{
			Start:    now.Add(-1 * time.Hour),
			End:      now.Add(1 * time.Hour),
			Modifier: 1.5,
		})
		require.NoError(t, err)

		// Base weight 10 should become 15 (10 * 1.5)
		weight := tas.GetEffectiveWeight("any.endpoint", 10)
		assert.Equal(t, 15, weight)
	})

	t.Run("endpoint schedule combined with global modifier", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		tas.SetTimeFunc(func() time.Time { return now })

		// Add endpoint schedule with weight 20
		err := tas.AddEndpointSchedule("sales.order.create", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
		})
		require.NoError(t, err)

		// Add global schedule with 1.5x modifier
		err = tas.AddGlobalSchedule(TimeSchedule{
			Start:    now.Add(-1 * time.Hour),
			End:      now.Add(1 * time.Hour),
			Modifier: 1.5,
		})
		require.NoError(t, err)

		// Endpoint weight 20 * global modifier 1.5 = 30
		weight := tas.GetEffectiveWeight("sales.order.create", 10)
		assert.Equal(t, 30, weight)
	})

	t.Run("global schedule validation", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		now := time.Now()
		err := tas.AddGlobalSchedule(TimeSchedule{
			Start:    now,
			End:      now.Add(-1 * time.Hour), // End before start
			Modifier: 1.5,
		})
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrEndTimeBeforeStart)
	})

	t.Run("GetActiveSchedules", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		tas.SetTimeFunc(func() time.Time { return now })

		err := tas.AddEndpointSchedule("sales.order.create", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
		})
		require.NoError(t, err)

		active := tas.GetActiveSchedules()
		assert.Len(t, active, 1)
		assert.Contains(t, active[0], "sales.order.create")
		assert.Contains(t, active[0], "weight=20")
	})

	t.Run("GetStats", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		tas.SetTimeFunc(func() time.Time { return now })

		err := tas.AddEndpointSchedule("ep1", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
		})
		require.NoError(t, err)

		err = tas.AddEndpointSchedule("ep2", []TimeBasedWeight{
			{TimeRange: "14:00-18:00", Weight: 15},
		})
		require.NoError(t, err)

		err = tas.AddGlobalSchedule(TimeSchedule{
			Start:    now.Add(-1 * time.Hour),
			End:      now.Add(1 * time.Hour),
			Modifier: 1.5,
		})
		require.NoError(t, err)

		stats := tas.GetStats()
		assert.Equal(t, 2, stats.EndpointScheduleCount)
		assert.Equal(t, 1, stats.GlobalScheduleCount)
		assert.Len(t, stats.ActiveSchedules, 2) // ep1 schedule + global schedule
	})

	t.Run("ClearSchedules", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		err := tas.AddEndpointSchedule("ep1", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
		})
		require.NoError(t, err)

		tas.ClearSchedules()

		stats := tas.GetStats()
		assert.Equal(t, 0, stats.EndpointScheduleCount)
		assert.Equal(t, 0, stats.GlobalScheduleCount)
	})

	t.Run("RemoveEndpointSchedule", func(t *testing.T) {
		tas := NewTimeAwareScheduler()

		err := tas.AddEndpointSchedule("ep1", []TimeBasedWeight{
			{TimeRange: "09:00-12:00", Weight: 20},
		})
		require.NoError(t, err)

		tas.RemoveEndpointSchedule("ep1")

		stats := tas.GetStats()
		assert.Equal(t, 0, stats.EndpointScheduleCount)
	})
}

func TestTimeBasedWeightWithModifier(t *testing.T) {
	t.Run("modifier doubles base weight", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Modifier:  2.0,
		}
		assert.Equal(t, 20, tbw.GetWeight(10))
	})

	t.Run("modifier halves base weight", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Modifier:  0.5,
		}
		assert.Equal(t, 5, tbw.GetWeight(10))
	})

	t.Run("weight takes precedence over zero modifier", func(t *testing.T) {
		tbw := TimeBasedWeight{
			TimeRange: "09:00-12:00",
			Weight:    25,
			Modifier:  0, // Zero modifier means use Weight
		}
		assert.Equal(t, 25, tbw.GetWeight(10))
	})
}

// TestAcceptanceCriteria_MorningSalesOrderWeightBoost verifies the acceptance criteria:
// "上午 9-12 点订单端点权重提升生效"
func TestAcceptanceCriteria_MorningSalesOrderWeightBoost(t *testing.T) {
	tas := NewTimeAwareScheduler()

	// Configure sales order endpoint with morning peak schedule
	err := tas.AddEndpointSchedule("POST /trade/sales-orders", []TimeBasedWeight{
		{TimeRange: "09:00-12:00", Weight: 20}, // Morning peak
		{TimeRange: "12:00-14:00", Weight: 5},  // Lunch lull
		{TimeRange: "14:00-18:00", Weight: 15}, // Afternoon peak
	})
	require.NoError(t, err)

	baseWeight := 10

	// Test at 09:00 (start of morning peak)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	})
	assert.Equal(t, 20, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"09:00 should have boosted weight of 20")

	// Test at 10:30 (middle of morning peak)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	})
	assert.Equal(t, 20, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"10:30 should have boosted weight of 20")

	// Test at 11:59 (just before end of morning peak)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 11, 59, 0, 0, time.UTC)
	})
	assert.Equal(t, 20, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"11:59 should have boosted weight of 20")

	// Test at 12:00 (start of lunch lull)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	})
	assert.Equal(t, 5, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"12:00 should have reduced weight of 5")

	// Test at 14:00 (start of afternoon peak)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	})
	assert.Equal(t, 15, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"14:00 should have weight of 15")

	// Test at 08:00 (before any schedule)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	})
	assert.Equal(t, baseWeight, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"08:00 should return base weight")

	// Test at 19:00 (after all schedules)
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 19, 0, 0, 0, time.UTC)
	})
	assert.Equal(t, baseWeight, tas.GetEffectiveWeight("POST /trade/sales-orders", baseWeight),
		"19:00 should return base weight")
}

// Benchmark tests
func BenchmarkTimeRangeContains(b *testing.B) {
	tr := TimeRange{
		StartHour:   9,
		StartMinute: 0,
		EndHour:     17,
		EndMinute:   0,
	}
	testTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Contains(testTime)
	}
}

func BenchmarkCronScheduleMatches(b *testing.B) {
	cs, _ := ParseCronSchedule("* 9-17 * * 1-5")
	testTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.Matches(testTime)
	}
}

func BenchmarkTimeAwareSchedulerGetEffectiveWeight(b *testing.B) {
	tas := NewTimeAwareScheduler()
	tas.SetTimeFunc(func() time.Time {
		return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	})

	_ = tas.AddEndpointSchedule("test.endpoint", []TimeBasedWeight{
		{TimeRange: "09:00-12:00", Weight: 20},
		{TimeRange: "14:00-18:00", Weight: 15},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tas.GetEffectiveWeight("test.endpoint", 10)
	}
}
