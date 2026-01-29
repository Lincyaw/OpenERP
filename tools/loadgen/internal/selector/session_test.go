package selector

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserBehavior_Validate(t *testing.T) {
	tests := []struct {
		name      string
		behavior  UserBehavior
		wantError bool
	}{
		{
			name: "valid behavior",
			behavior: UserBehavior{
				Name:   "test",
				Weight: 10,
				ThinkTime: ThinkTimeConfig{
					Min: time.Second,
					Max: 5 * time.Second,
				},
				ActionsPerSession: ActionsConfig{
					Min: 1,
					Max: 10,
				},
			},
			wantError: false,
		},
		{
			name: "missing name",
			behavior: UserBehavior{
				Weight: 10,
			},
			wantError: true,
		},
		{
			name: "negative weight",
			behavior: UserBehavior{
				Name:   "test",
				Weight: -1,
			},
			wantError: true,
		},
		{
			name: "negative think time",
			behavior: UserBehavior{
				Name: "test",
				ThinkTime: ThinkTimeConfig{
					Min: -time.Second,
				},
			},
			wantError: true,
		},
		{
			name: "think time min > max",
			behavior: UserBehavior{
				Name: "test",
				ThinkTime: ThinkTimeConfig{
					Min: 10 * time.Second,
					Max: 5 * time.Second,
				},
			},
			wantError: true,
		},
		{
			name: "negative actions",
			behavior: UserBehavior{
				Name: "test",
				ActionsPerSession: ActionsConfig{
					Min: -1,
				},
			},
			wantError: true,
		},
		{
			name: "actions min > max",
			behavior: UserBehavior{
				Name: "test",
				ActionsPerSession: ActionsConfig{
					Min: 20,
					Max: 10,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.behavior.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserBehavior_ApplyDefaults(t *testing.T) {
	behavior := UserBehavior{
		Name: "test",
	}
	behavior.ApplyDefaults()

	assert.Equal(t, 1, behavior.Weight)
	assert.Equal(t, time.Second, behavior.ThinkTime.Min)
	assert.Equal(t, 5*time.Second, behavior.ThinkTime.Max)
	assert.Equal(t, "uniform", behavior.ThinkTime.Distribution)
	assert.Equal(t, 1, behavior.ActionsPerSession.Min)
	assert.Equal(t, 10, behavior.ActionsPerSession.Max)
}

func TestSessionParameters(t *testing.T) {
	params := NewSessionParameters()

	t.Run("Set and Get", func(t *testing.T) {
		params.Set("product_id", "prod-001")

		val, ok := params.Get("product_id")
		assert.True(t, ok)
		assert.Equal(t, "prod-001", val)
	})

	t.Run("Get non-existent", func(t *testing.T) {
		val, ok := params.Get("non_existent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("Multiple values", func(t *testing.T) {
		params.Set("order_id", "order-001")
		params.Set("order_id", "order-002")
		params.Set("order_id", "order-003")

		// Get returns the latest value
		val, ok := params.Get("order_id")
		assert.True(t, ok)
		assert.Equal(t, "order-003", val)

		// GetAll returns all values
		all := params.GetAll("order_id")
		assert.Len(t, all, 3)
	})

	t.Run("Has", func(t *testing.T) {
		assert.True(t, params.Has("product_id"))
		assert.False(t, params.Has("unknown"))
	})

	t.Run("Keys", func(t *testing.T) {
		keys := params.Keys()
		assert.Contains(t, keys, "product_id")
		assert.Contains(t, keys, "order_id")
	})

	t.Run("Count", func(t *testing.T) {
		count := params.Count()
		assert.Equal(t, 4, count) // 1 product_id + 3 order_ids
	})

	t.Run("Clone", func(t *testing.T) {
		clone := params.Clone()
		assert.Equal(t, params.Count(), clone.Count())

		// Modify original
		params.Set("product_id", "prod-002")

		// Clone should not be affected
		val, _ := clone.Get("product_id")
		assert.Equal(t, "prod-001", val)
	})

	t.Run("Clear", func(t *testing.T) {
		testParams := NewSessionParameters()
		testParams.Set("key", "value")
		testParams.Clear()
		assert.Equal(t, 0, testParams.Count())
	})

	t.Run("GetRandom", func(t *testing.T) {
		randomParams := NewSessionParameters()
		randomParams.Set("id", "1")
		randomParams.Set("id", "2")
		randomParams.Set("id", "3")

		// Run multiple times to verify randomness
		hits := make(map[string]int)
		for i := 0; i < 100; i++ {
			val, ok := randomParams.GetRandom("id")
			assert.True(t, ok)
			hits[val.(string)]++
		}

		// All values should be hit at least once
		assert.True(t, hits["1"] > 0)
		assert.True(t, hits["2"] > 0)
		assert.True(t, hits["3"] > 0)
	})
}

func TestSession_IsExpired(t *testing.T) {
	now := time.Now()

	t.Run("not expired", func(t *testing.T) {
		session := &Session{
			ID:        "test-1",
			StartTime: now,
			ExpiresAt: now.Add(time.Hour),
		}
		assert.False(t, session.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		session := &Session{
			ID:        "test-2",
			StartTime: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-time.Hour),
		}
		assert.True(t, session.IsExpired())
	})
}

func TestSession_IsActionLimitReached(t *testing.T) {
	t.Run("limit not reached", func(t *testing.T) {
		session := &Session{
			ActionCount: 5,
			MaxActions:  10,
		}
		assert.False(t, session.IsActionLimitReached())
	})

	t.Run("limit reached", func(t *testing.T) {
		session := &Session{
			ActionCount: 10,
			MaxActions:  10,
		}
		assert.True(t, session.IsActionLimitReached())
	})

	t.Run("limit exceeded", func(t *testing.T) {
		session := &Session{
			ActionCount: 15,
			MaxActions:  10,
		}
		assert.True(t, session.IsActionLimitReached())
	})
}

func TestSession_IncrementActionCount(t *testing.T) {
	session := &Session{
		ActionCount: 0,
		MaxActions:  10,
	}

	count := session.IncrementActionCount()
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, session.ActionCount)
	assert.False(t, session.LastActionTime.IsZero())
}

func TestSession_RemainingActions(t *testing.T) {
	session := &Session{
		ActionCount: 3,
		MaxActions:  10,
	}
	assert.Equal(t, 7, session.RemainingActions())

	session.ActionCount = 15
	assert.Equal(t, 0, session.RemainingActions())
}

func TestSession_NextThinkTime(t *testing.T) {
	t.Run("nil behavior", func(t *testing.T) {
		session := &Session{}
		thinkTime := session.NextThinkTime()
		assert.Equal(t, time.Second, thinkTime)
	})

	t.Run("uniform distribution", func(t *testing.T) {
		behavior := &UserBehavior{
			Name: "test",
			ThinkTime: ThinkTimeConfig{
				Min:          time.Second,
				Max:          2 * time.Second,
				Distribution: "uniform",
			},
		}
		session := &Session{Behavior: behavior}

		// Generate multiple times and verify range
		for i := 0; i < 100; i++ {
			thinkTime := session.NextThinkTime()
			assert.GreaterOrEqual(t, thinkTime, time.Second)
			assert.Less(t, thinkTime, 2*time.Second)
		}
	})

	t.Run("exponential distribution", func(t *testing.T) {
		behavior := &UserBehavior{
			Name: "test",
			ThinkTime: ThinkTimeConfig{
				Min:          time.Second,
				Max:          10 * time.Second,
				Distribution: "exponential",
			},
		}
		session := &Session{Behavior: behavior}

		// Just verify it doesn't panic and returns reasonable values
		thinkTime := session.NextThinkTime()
		assert.GreaterOrEqual(t, thinkTime, time.Second)
		assert.LessOrEqual(t, thinkTime, 10*time.Second)
	})

	t.Run("min equals max", func(t *testing.T) {
		behavior := &UserBehavior{
			Name: "test",
			ThinkTime: ThinkTimeConfig{
				Min:          2 * time.Second,
				Max:          2 * time.Second,
				Distribution: "uniform",
			},
		}
		session := &Session{Behavior: behavior}
		thinkTime := session.NextThinkTime()
		assert.Equal(t, 2*time.Second, thinkTime)
	})
}

func TestSessionSimulatorConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    SessionSimulatorConfig
		wantError bool
	}{
		{
			name:      "empty config (uses defaults)",
			config:    SessionSimulatorConfig{},
			wantError: false,
		},
		{
			name: "valid config",
			config: SessionSimulatorConfig{
				ConcurrentSessions: 100,
				SessionDuration: SessionDurationConfig{
					Min: time.Minute,
					Max: 5 * time.Minute,
				},
			},
			wantError: false,
		},
		{
			name: "negative concurrent sessions",
			config: SessionSimulatorConfig{
				ConcurrentSessions: -1,
			},
			wantError: true,
		},
		{
			name: "negative session duration",
			config: SessionSimulatorConfig{
				SessionDuration: SessionDurationConfig{
					Min: -time.Second,
				},
			},
			wantError: true,
		},
		{
			name: "session duration min > max",
			config: SessionSimulatorConfig{
				SessionDuration: SessionDurationConfig{
					Min: 10 * time.Minute,
					Max: 5 * time.Minute,
				},
			},
			wantError: true,
		},
		{
			name: "invalid behavior",
			config: SessionSimulatorConfig{
				Behaviors: []UserBehavior{
					{Name: ""}, // Invalid - missing name
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionSimulatorConfig_ApplyDefaults(t *testing.T) {
	config := SessionSimulatorConfig{}
	config.ApplyDefaults()

	assert.Equal(t, 100, config.ConcurrentSessions)
	assert.Equal(t, 30*time.Second, config.SessionDuration.Min)
	assert.Equal(t, 5*time.Minute, config.SessionDuration.Max)
	assert.NotNil(t, config.ReplaceExpired)
	assert.True(t, *config.ReplaceExpired)
	assert.Len(t, config.Behaviors, 1)
	assert.Equal(t, "default", config.Behaviors[0].Name)
}

func TestNewSessionSimulator(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		ss, err := NewSessionSimulator(SessionSimulatorConfig{})
		require.NoError(t, err)
		assert.NotNil(t, ss)
		assert.Equal(t, 100, ss.config.ConcurrentSessions)
	})

	t.Run("custom config", func(t *testing.T) {
		config := SessionSimulatorConfig{
			ConcurrentSessions: 50,
			Behaviors: []UserBehavior{
				{
					Name:   "power-user",
					Weight: 2,
				},
				{
					Name:   "casual-user",
					Weight: 8,
				},
			},
		}
		ss, err := NewSessionSimulator(config)
		require.NoError(t, err)
		assert.Equal(t, 50, ss.config.ConcurrentSessions)
		assert.Len(t, ss.config.Behaviors, 2)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := SessionSimulatorConfig{
			ConcurrentSessions: -1,
		}
		ss, err := NewSessionSimulator(config)
		assert.Error(t, err)
		assert.Nil(t, ss)
	})
}

func TestSessionSimulator_CreateSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 10,
	})
	require.NoError(t, err)

	t.Run("create session", func(t *testing.T) {
		session, err := ss.CreateSession(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.NotNil(t, session.Behavior)
		assert.NotNil(t, session.Parameters)
		assert.False(t, session.StartTime.IsZero())
		assert.False(t, session.ExpiresAt.IsZero())
		assert.Greater(t, session.MaxActions, 0)
	})

	t.Run("create session with specific behavior", func(t *testing.T) {
		behavior := &UserBehavior{
			Name:   "custom",
			Weight: 1,
		}
		behavior.ApplyDefaults()

		session, err := ss.CreateSession(behavior)
		require.NoError(t, err)
		assert.Equal(t, "custom", session.Behavior.Name)
	})

	t.Run("session limit reached", func(t *testing.T) {
		ss, err := NewSessionSimulator(SessionSimulatorConfig{
			ConcurrentSessions: 2,
		})
		require.NoError(t, err)

		// Create sessions up to limit
		_, err = ss.CreateSession(nil)
		require.NoError(t, err)
		_, err = ss.CreateSession(nil)
		require.NoError(t, err)

		// Third should fail
		_, err = ss.CreateSession(nil)
		assert.ErrorIs(t, err, ErrSessionLimitReached)
	})
}

func TestSessionSimulator_GetSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{})
	require.NoError(t, err)

	session, err := ss.CreateSession(nil)
	require.NoError(t, err)

	t.Run("get existing session", func(t *testing.T) {
		found, err := ss.GetSession(session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, found.ID)
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := ss.GetSession("non-existent")
		assert.ErrorIs(t, err, ErrNoActiveSessions)
	})
}

func TestSessionSimulator_GetActiveSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{})
	require.NoError(t, err)

	t.Run("no sessions", func(t *testing.T) {
		_, err := ss.GetActiveSession()
		assert.ErrorIs(t, err, ErrNoActiveSessions)
	})

	t.Run("has active session", func(t *testing.T) {
		_, err := ss.CreateSession(nil)
		require.NoError(t, err)

		session, err := ss.GetActiveSession()
		require.NoError(t, err)
		assert.NotNil(t, session)
	})
}

func TestSessionSimulator_EndSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{})
	require.NoError(t, err)

	session, err := ss.CreateSession(nil)
	require.NoError(t, err)

	ss.EndSession(session.ID)

	_, err = ss.GetSession(session.ID)
	assert.ErrorIs(t, err, ErrNoActiveSessions)
}

func TestSessionSimulator_CleanExpiredSessions(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		SessionDuration: SessionDurationConfig{
			Min: time.Millisecond,
			Max: time.Millisecond,
		},
	})
	require.NoError(t, err)

	// Create sessions
	_, err = ss.CreateSession(nil)
	require.NoError(t, err)
	_, err = ss.CreateSession(nil)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Clean up
	count := ss.CleanExpiredSessions()
	assert.Equal(t, 2, count)
	assert.Equal(t, 0, ss.TotalSessionCount())
}

func TestSessionSimulator_RecordAction(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{})
	require.NoError(t, err)

	session, err := ss.CreateSession(nil)
	require.NoError(t, err)

	t.Run("record action", func(t *testing.T) {
		err := ss.RecordAction(session.ID)
		assert.NoError(t, err)

		stats := ss.GetStats()
		assert.Equal(t, uint64(1), stats.TotalActionsExecuted)
	})

	t.Run("record action for non-existent session", func(t *testing.T) {
		err := ss.RecordAction("non-existent")
		assert.ErrorIs(t, err, ErrNoActiveSessions)
	})
}

func TestSessionSimulator_GetStats(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 50,
	})
	require.NoError(t, err)

	session, err := ss.CreateSession(nil)
	require.NoError(t, err)

	err = ss.RecordAction(session.ID)
	require.NoError(t, err)

	stats := ss.GetStats()
	assert.Equal(t, uint64(1), stats.TotalSessionsCreated)
	assert.Equal(t, uint64(1), stats.TotalActionsExecuted)
	assert.Equal(t, 1, stats.ActiveSessions)
	assert.Equal(t, 50, stats.MaxConcurrent)
}

func TestSessionSimulator_BehaviorWeightSelection(t *testing.T) {
	config := SessionSimulatorConfig{
		ConcurrentSessions: 1000,
		Behaviors: []UserBehavior{
			{Name: "heavy", Weight: 80},
			{Name: "light", Weight: 20},
		},
	}
	ss, err := NewSessionSimulator(config)
	require.NoError(t, err)

	// Create many sessions and track behavior distribution
	behaviorCounts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		session, err := ss.CreateSession(nil)
		require.NoError(t, err)
		behaviorCounts[session.Behavior.Name]++
		ss.EndSession(session.ID)
	}

	// Heavy should be selected roughly 80% of the time
	heavyRatio := float64(behaviorCounts["heavy"]) / 1000.0
	lightRatio := float64(behaviorCounts["light"]) / 1000.0

	assert.InDelta(t, 0.8, heavyRatio, 0.1, "heavy behavior should be ~80%")
	assert.InDelta(t, 0.2, lightRatio, 0.1, "light behavior should be ~20%")
}

func TestSessionSimulator_GetBehavior(t *testing.T) {
	config := SessionSimulatorConfig{
		Behaviors: []UserBehavior{
			{Name: "power-user", Weight: 1},
			{Name: "casual-user", Weight: 1},
		},
	}
	ss, err := NewSessionSimulator(config)
	require.NoError(t, err)

	t.Run("get existing behavior", func(t *testing.T) {
		b, err := ss.GetBehavior("power-user")
		require.NoError(t, err)
		assert.Equal(t, "power-user", b.Name)
	})

	t.Run("get non-existent behavior", func(t *testing.T) {
		_, err := ss.GetBehavior("unknown")
		assert.ErrorIs(t, err, ErrBehaviorNotFound)
	})
}

func TestSessionSimulator_UpdateConfig(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 10,
	})
	require.NoError(t, err)

	newConfig := SessionSimulatorConfig{
		ConcurrentSessions: 50,
	}
	err = ss.UpdateConfig(newConfig)
	require.NoError(t, err)

	config := ss.GetConfig()
	assert.Equal(t, 50, config.ConcurrentSessions)
}

func TestSessionSimulator_Clear(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{})
	require.NoError(t, err)

	// Create some sessions
	_, _ = ss.CreateSession(nil)
	_, _ = ss.CreateSession(nil)
	assert.Equal(t, 2, ss.TotalSessionCount())

	ss.Clear()
	assert.Equal(t, 0, ss.TotalSessionCount())
}

func TestSessionSimulator_GetRandomActiveSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 10,
	})
	require.NoError(t, err)

	t.Run("no sessions", func(t *testing.T) {
		_, err := ss.GetRandomActiveSession()
		assert.ErrorIs(t, err, ErrNoActiveSessions)
	})

	t.Run("single session", func(t *testing.T) {
		session, err := ss.CreateSession(nil)
		require.NoError(t, err)

		found, err := ss.GetRandomActiveSession()
		require.NoError(t, err)
		assert.Equal(t, session.ID, found.ID)
	})

	t.Run("multiple sessions", func(t *testing.T) {
		// Create more sessions
		_, _ = ss.CreateSession(nil)
		_, _ = ss.CreateSession(nil)

		// Should return one of the sessions
		found, err := ss.GetRandomActiveSession()
		require.NoError(t, err)
		assert.NotNil(t, found)
	})
}

func TestSessionSimulator_GetOrCreateSession(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 2,
	})
	require.NoError(t, err)

	t.Run("creates new session when none exist", func(t *testing.T) {
		session, err := ss.GetOrCreateSession()
		require.NoError(t, err)
		assert.NotNil(t, session)
	})

	t.Run("returns existing session", func(t *testing.T) {
		existing := ss.GetAllSessions()[0]
		session, err := ss.GetOrCreateSession()
		require.NoError(t, err)
		// Should return an existing session (not necessarily the same one)
		assert.NotNil(t, session)
		assert.Equal(t, existing.ID, session.ID)
	})
}

func TestSessionSimulator_Concurrency(t *testing.T) {
	ss, err := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 100,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	// Spawn goroutines to create sessions concurrently
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, err := ss.CreateSession(nil)
			if err != nil {
				errorCount.Add(1)
				return
			}
			successCount.Add(1)

			// Record some actions
			_ = ss.RecordAction(session.ID)

			// End some sessions
			if successCount.Load()%3 == 0 {
				ss.EndSession(session.ID)
			}
		}()
	}

	wg.Wait()

	// We should have at least 100 successful creations
	assert.GreaterOrEqual(t, successCount.Load(), int32(100))

	// Stats should be consistent
	stats := ss.GetStats()
	assert.Equal(t, uint64(successCount.Load()), stats.TotalSessionsCreated)
}

// TestAcceptanceCriteria_100ConcurrentSessions verifies the acceptance criteria:
// Simulate 100 concurrent sessions with behavior conforming to configuration.
func TestAcceptanceCriteria_100ConcurrentSessions(t *testing.T) {
	config := SessionSimulatorConfig{
		ConcurrentSessions: 100,
		SessionDuration: SessionDurationConfig{
			Min: time.Minute,
			Max: 5 * time.Minute,
		},
		Behaviors: []UserBehavior{
			{
				Name:   "browse",
				Weight: 70,
				ThinkTime: ThinkTimeConfig{
					Min:          500 * time.Millisecond,
					Max:          2 * time.Second,
					Distribution: "uniform",
				},
				ActionsPerSession: ActionsConfig{
					Min: 3,
					Max: 10,
				},
			},
			{
				Name:   "purchase",
				Weight: 20,
				ThinkTime: ThinkTimeConfig{
					Min:          time.Second,
					Max:          3 * time.Second,
					Distribution: "uniform",
				},
				ActionsPerSession: ActionsConfig{
					Min: 5,
					Max: 15,
				},
			},
			{
				Name:   "admin",
				Weight: 10,
				ThinkTime: ThinkTimeConfig{
					Min:          200 * time.Millisecond,
					Max:          500 * time.Millisecond,
					Distribution: "uniform",
				},
				ActionsPerSession: ActionsConfig{
					Min: 10,
					Max: 50,
				},
			},
		},
	}

	ss, err := NewSessionSimulator(config)
	require.NoError(t, err)

	// Create 100 concurrent sessions
	sessions := make([]*Session, 0, 100)
	for i := 0; i < 100; i++ {
		session, err := ss.CreateSession(nil)
		require.NoError(t, err, "failed to create session %d", i)
		sessions = append(sessions, session)
	}

	// Verify 100 concurrent sessions
	assert.Equal(t, 100, ss.ActiveSessionCount())

	// Try to create one more - should fail
	_, err = ss.CreateSession(nil)
	assert.ErrorIs(t, err, ErrSessionLimitReached)

	// Verify behavior distribution (approximately)
	stats := ss.GetStats()
	browseCount := stats.BehaviorCounts["browse"]
	purchaseCount := stats.BehaviorCounts["purchase"]
	adminCount := stats.BehaviorCounts["admin"]

	t.Logf("Behavior distribution: browse=%d, purchase=%d, admin=%d",
		browseCount, purchaseCount, adminCount)

	// Allow some deviation from expected distribution
	assert.InDelta(t, 70, float64(browseCount), 20, "browse should be ~70%")
	assert.InDelta(t, 20, float64(purchaseCount), 15, "purchase should be ~20%")
	assert.InDelta(t, 10, float64(adminCount), 10, "admin should be ~10%")

	// Verify each session has proper configuration
	for _, session := range sessions {
		assert.NotEmpty(t, session.ID)
		assert.NotNil(t, session.Behavior)
		assert.NotNil(t, session.Parameters)
		assert.False(t, session.StartTime.IsZero())
		assert.False(t, session.ExpiresAt.IsZero())
		assert.False(t, session.IsExpired())
		assert.False(t, session.IsActionLimitReached())

		// Verify think time is within behavior bounds
		thinkTime := session.NextThinkTime()
		assert.GreaterOrEqual(t, thinkTime, session.Behavior.ThinkTime.Min)
		assert.LessOrEqual(t, thinkTime, session.Behavior.ThinkTime.Max)

		// Verify max actions is within behavior bounds
		assert.GreaterOrEqual(t, session.MaxActions, session.Behavior.ActionsPerSession.Min)
		assert.LessOrEqual(t, session.MaxActions, session.Behavior.ActionsPerSession.Max)

		// Verify session duration is within bounds
		duration := session.ExpiresAt.Sub(session.StartTime)
		assert.GreaterOrEqual(t, duration, config.SessionDuration.Min)
		assert.LessOrEqual(t, duration, config.SessionDuration.Max)
	}

	// Simulate actions on sessions
	var wg sync.WaitGroup
	for _, session := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			// Perform some actions
			actionsToPerform := s.MaxActions / 2
			for i := 0; i < actionsToPerform; i++ {
				err := ss.RecordAction(s.ID)
				if err != nil {
					break
				}
				// Verify parameters can be stored and retrieved
				s.Parameters.Set("created_resource", i)
			}
		}(session)
	}
	wg.Wait()

	// Verify actions were recorded
	finalStats := ss.GetStats()
	assert.Greater(t, finalStats.TotalActionsExecuted, uint64(0))
	t.Logf("Total actions executed: %d", finalStats.TotalActionsExecuted)
}

// BenchmarkSessionSimulator_CreateSession benchmarks session creation.
func BenchmarkSessionSimulator_CreateSession(b *testing.B) {
	ss, _ := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: b.N + 1,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ss.CreateSession(nil)
	}
}

// BenchmarkSessionSimulator_GetRandomActiveSession benchmarks random session selection.
func BenchmarkSessionSimulator_GetRandomActiveSession(b *testing.B) {
	ss, _ := NewSessionSimulator(SessionSimulatorConfig{
		ConcurrentSessions: 100,
	})

	// Create sessions
	for i := 0; i < 100; i++ {
		_, _ = ss.CreateSession(nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ss.GetRandomActiveSession()
	}
}

// BenchmarkSession_NextThinkTime benchmarks think time generation.
func BenchmarkSession_NextThinkTime(b *testing.B) {
	behavior := &UserBehavior{
		Name: "test",
		ThinkTime: ThinkTimeConfig{
			Min:          time.Second,
			Max:          5 * time.Second,
			Distribution: "uniform",
		},
	}
	session := &Session{Behavior: behavior}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.NextThinkTime()
	}
}

// BenchmarkSessionParameters_SetGet benchmarks parameter operations.
func BenchmarkSessionParameters_SetGet(b *testing.B) {
	params := NewSessionParameters()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params.Set("key", i)
		_, _ = params.Get("key")
	}
}
