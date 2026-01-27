package warmup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/erp/tools/loadgen/internal/circuit"
	"github.com/example/erp/tools/loadgen/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPool is a mock implementation of pool.ParameterPool for testing.
type mockPool struct {
	values   map[circuit.SemanticType][]any
	closed   bool
	addCalls int
	sizeMock func(semantic circuit.SemanticType) int
}

func newMockPool() *mockPool {
	return &mockPool{
		values: make(map[circuit.SemanticType][]any),
	}
}

func (m *mockPool) Add(semantic circuit.SemanticType, value any, source pool.ValueSource) {
	m.addCalls++
	if m.values[semantic] == nil {
		m.values[semantic] = []any{}
	}
	m.values[semantic] = append(m.values[semantic], value)
}

func (m *mockPool) AddWithTTL(semantic circuit.SemanticType, value any, source pool.ValueSource, ttl time.Duration) {
	m.Add(semantic, value, source)
}

func (m *mockPool) Get(semantic circuit.SemanticType) (*pool.Value, error) {
	vals := m.values[semantic]
	if len(vals) == 0 {
		return nil, pool.ErrNoValue
	}
	return &pool.Value{Data: vals[0], SemanticType: semantic}, nil
}

func (m *mockPool) GetAll(semantic circuit.SemanticType) []pool.Value {
	vals := m.values[semantic]
	result := make([]pool.Value, len(vals))
	for i, v := range vals {
		result[i] = pool.Value{Data: v, SemanticType: semantic}
	}
	return result
}

func (m *mockPool) Size(semantic circuit.SemanticType) int {
	if m.sizeMock != nil {
		return m.sizeMock(semantic)
	}
	return len(m.values[semantic])
}

func (m *mockPool) TotalSize() int {
	total := 0
	for _, vals := range m.values {
		total += len(vals)
	}
	return total
}

func (m *mockPool) Types() []circuit.SemanticType {
	types := make([]circuit.SemanticType, 0, len(m.values))
	for t := range m.values {
		types = append(types, t)
	}
	return types
}

func (m *mockPool) Clear(semantic *circuit.SemanticType) {
	if semantic == nil {
		m.values = make(map[circuit.SemanticType][]any)
	} else {
		delete(m.values, *semantic)
	}
}

func (m *mockPool) Cleanup() {}

func (m *mockPool) Stats() pool.Stats {
	stats := pool.Stats{
		ValuesByType: make(map[circuit.SemanticType]int),
	}
	for t, vals := range m.values {
		stats.ValuesByType[t] = len(vals)
		stats.TotalValues += int64(len(vals))
	}
	return stats
}

func (m *mockPool) Close() {
	m.closed = true
}

func (m *mockPool) IsClosed() bool {
	return m.closed
}

func TestNewExecutor(t *testing.T) {
	t.Run("success with valid config", func(t *testing.T) {
		mockP := newMockPool()
		producers := map[circuit.SemanticType]ProducerFunc{
			"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
				return []any{"cust-1"}, nil
			},
		}

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 5,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
			},
			Producers: producers,
		})

		require.NoError(t, err)
		assert.NotNil(t, executor)
	})

	t.Run("error when pool is nil", func(t *testing.T) {
		executor, err := NewExecutor(ExecutorConfig{
			Pool: nil,
		})

		assert.Error(t, err)
		assert.Nil(t, executor)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("error when fill configured but no producers", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 5,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
			},
			Producers: nil,
		})

		assert.Error(t, err)
		assert.Nil(t, executor)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("success when no fill configured without producers", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 0,
				Fill:       nil,
			},
		})

		require.NoError(t, err)
		assert.NotNil(t, executor)
	})

	t.Run("applies default concurrency", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool:        mockP,
			Concurrency: 0, // Should default to 1
		})

		require.NoError(t, err)
		assert.Equal(t, 1, executor.config.Concurrency)
	})
}

func TestExecutor_Execute(t *testing.T) {
	t.Run("successful warmup with login and fill", func(t *testing.T) {
		mockP := newMockPool()
		loginCalled := false
		producerCalls := 0

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  3,
				Fill:        []circuit.SemanticType{"entity.customer.id"},
				MinPoolSize: 3,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					producerCalls++
					return []any{"cust-" + string(rune('0'+producerCalls))}, nil
				},
			},
			Login: func(ctx context.Context) (string, error) {
				loginCalled = true
				return "test-token", nil
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.True(t, loginCalled)
		assert.Equal(t, "test-token", result.Token)
		assert.Equal(t, 3, producerCalls)
		assert.Equal(t, 3, result.PoolSizes["entity.customer.id"])
		assert.Empty(t, result.Errors)
	})

	t.Run("successful warmup without login", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  2,
				Fill:        []circuit.SemanticType{"entity.product.id"},
				MinPoolSize: 2,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.product.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"prod-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Token)
	})

	t.Run("login failure", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 1,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
				RetryCount: 0,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
			},
			Login: func(ctx context.Context) (string, error) {
				return "", errors.New("auth failed")
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrLoginFailed)
		assert.False(t, result.Success)
	})

	t.Run("producer failure with ContinueOnError=true", func(t *testing.T) {
		mockP := newMockPool()
		calls := 0

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:      2,
				Fill:            []circuit.SemanticType{"entity.customer.id"},
				MinPoolSize:     0, // Don't verify pool size
				RetryCount:      0,
				ContinueOnError: true,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("producer error")
					}
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		// Should not return error due to ContinueOnError
		require.NoError(t, err)
		// Pool should have 1 value from second iteration
		assert.Equal(t, 1, result.PoolSizes["entity.customer.id"])
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("producer failure with ContinueOnError=false", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:      2,
				Fill:            []circuit.SemanticType{"entity.customer.id"},
				MinPoolSize:     2,
				RetryCount:      0,
				ContinueOnError: false,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return nil, errors.New("producer error")
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		assert.Error(t, err)
		assert.False(t, result.Success)
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockP := newMockPool()
		ctx, cancel := context.WithCancel(context.Background())

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 100,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					cancel() // Cancel after first call
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(ctx)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, result.Success)
	})

	t.Run("timeout", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 100,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
				Timeout:    10 * time.Millisecond,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					time.Sleep(20 * time.Millisecond)
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.False(t, result.Success)
	})

	t.Run("multiple semantic types", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 2,
				Fill: []circuit.SemanticType{
					"entity.customer.id",
					"entity.product.id",
					"entity.warehouse.id",
				},
				MinPoolSize: 2,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
				"entity.product.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"prod-1"}, nil
				},
				"entity.warehouse.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"wh-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.PoolSizes["entity.customer.id"])
		assert.Equal(t, 2, result.PoolSizes["entity.product.id"])
		assert.Equal(t, 2, result.PoolSizes["entity.warehouse.id"])
	})

	t.Run("producer returns multiple values", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  1,
				Fill:        []circuit.SemanticType{"entity.customer.id"},
				MinPoolSize: 3,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1", "cust-2", "cust-3"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, result.PoolSizes["entity.customer.id"])
	})

	t.Run("pool size verification failure", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  1,
				Fill:        []circuit.SemanticType{"entity.customer.id"},
				MinPoolSize: 10, // Require 10, but only produce 1
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err) // No fatal error
		assert.False(t, result.Success)
		assert.Contains(t, result.SkippedTypes, circuit.SemanticType("entity.customer.id"))
	})

	t.Run("cannot run twice concurrently", func(t *testing.T) {
		mockP := newMockPool()
		started := make(chan struct{})
		proceed := make(chan struct{})

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 1,
				Fill:       []circuit.SemanticType{"entity.customer.id"},
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					close(started)
					<-proceed
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		// Start first execution
		go func() {
			_, _ = executor.Execute(context.Background())
		}()

		// Wait for first execution to start
		<-started

		// Try to start second execution
		result, err := executor.Execute(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)

		// Clean up
		close(proceed)
	})

	t.Run("empty fill list", func(t *testing.T) {
		mockP := newMockPool()

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 0,
				Fill:       nil,
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
	})
}

func TestExecutor_Progress(t *testing.T) {
	t.Run("progress callback is called", func(t *testing.T) {
		mockP := newMockPool()
		progressCalls := atomic.Int32{}

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 2,
				Fill: []circuit.SemanticType{
					"entity.customer.id",
				},
				MinPoolSize: 2,
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
			},
			OnProgress: func(progress Progress) {
				progressCalls.Add(1)
				// Verify progress data
				assert.NotEmpty(t, progress.Phase)
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		// Should have progress calls for each fill iteration + verify phase
		assert.Greater(t, progressCalls.Load(), int32(0))
	})

	t.Run("progress includes login phase", func(t *testing.T) {
		mockP := newMockPool()
		loginProgressSeen := false

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations: 1,
				Fill: []circuit.SemanticType{
					"entity.customer.id",
				},
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
			},
			Login: func(ctx context.Context) (string, error) {
				return "token", nil
			},
			OnProgress: func(progress Progress) {
				if progress.Phase == "login" {
					loginProgressSeen = true
				}
			},
		})
		require.NoError(t, err)

		_, err = executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, loginProgressSeen)
	})
}

func TestExecutor_Token(t *testing.T) {
	mockP := newMockPool()

	executor, err := NewExecutor(ExecutorConfig{
		Pool: mockP,
		Warmup: Config{
			Iterations: 1,
			Fill:       []circuit.SemanticType{"entity.customer.id"},
		},
		Producers: map[circuit.SemanticType]ProducerFunc{
			"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
				return []any{"cust-1"}, nil
			},
		},
		Login: func(ctx context.Context) (string, error) {
			return "my-secret-token", nil
		},
	})
	require.NoError(t, err)

	// Token should be empty before execution
	assert.Empty(t, executor.Token())

	_, err = executor.Execute(context.Background())
	require.NoError(t, err)

	// Token should be available after execution
	assert.Equal(t, "my-secret-token", executor.Token())
}

func TestExecutor_Retries(t *testing.T) {
	t.Run("producer retries on failure", func(t *testing.T) {
		mockP := newMockPool()
		attempts := 0

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  1,
				Fill:        []circuit.SemanticType{"entity.customer.id"},
				RetryCount:  2,
				RetryDelay:  time.Millisecond,
				MinPoolSize: 1, // Set explicitly to pass verification
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					attempts++
					if attempts < 3 {
						return nil, errors.New("temporary error")
					}
					return []any{"cust-1"}, nil
				},
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, attempts) // 1 initial + 2 retries
	})

	t.Run("login retries on failure", func(t *testing.T) {
		mockP := newMockPool()
		attempts := 0

		executor, err := NewExecutor(ExecutorConfig{
			Pool: mockP,
			Warmup: Config{
				Iterations:  1,
				Fill:        []circuit.SemanticType{"entity.customer.id"},
				RetryCount:  2,
				RetryDelay:  time.Millisecond,
				MinPoolSize: 1, // Set explicitly to pass verification
			},
			Producers: map[circuit.SemanticType]ProducerFunc{
				"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
					return []any{"cust-1"}, nil
				},
			},
			Login: func(ctx context.Context) (string, error) {
				attempts++
				if attempts < 3 {
					return "", errors.New("auth error")
				}
				return "token", nil
			},
		})
		require.NoError(t, err)

		result, err := executor.Execute(context.Background())

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 3, attempts)
	})
}

func TestCheckPoolReady(t *testing.T) {
	t.Run("pool is ready", func(t *testing.T) {
		mockP := newMockPool()
		mockP.values["entity.customer.id"] = []any{"c1", "c2", "c3", "c4", "c5"}
		mockP.values["entity.product.id"] = []any{"p1", "p2", "p3", "p4", "p5"}

		err := CheckPoolReady(mockP, []circuit.SemanticType{
			"entity.customer.id",
			"entity.product.id",
		}, 5)

		assert.NoError(t, err)
	})

	t.Run("pool is not ready", func(t *testing.T) {
		mockP := newMockPool()
		mockP.values["entity.customer.id"] = []any{"c1", "c2"} // Only 2 values
		mockP.values["entity.product.id"] = []any{"p1", "p2", "p3", "p4", "p5"}

		err := CheckPoolReady(mockP, []circuit.SemanticType{
			"entity.customer.id",
			"entity.product.id",
		}, 5)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPoolNotReady)
	})

	t.Run("zero minSize always passes", func(t *testing.T) {
		mockP := newMockPool()
		// Empty pool

		err := CheckPoolReady(mockP, []circuit.SemanticType{
			"entity.customer.id",
		}, 0)

		assert.NoError(t, err)
	})
}

func TestExecutor_ExecuteWarmupOnly(t *testing.T) {
	mockP := newMockPool()

	executor, err := NewExecutor(ExecutorConfig{
		Pool: mockP,
		Warmup: Config{
			Iterations:  2,
			Fill:        []circuit.SemanticType{"entity.customer.id"},
			MinPoolSize: 2,
		},
		Producers: map[circuit.SemanticType]ProducerFunc{
			"entity.customer.id": func(ctx context.Context, st circuit.SemanticType) ([]any, error) {
				return []any{"cust-1"}, nil
			},
		},
	})
	require.NoError(t, err)

	result, err := executor.ExecuteWarmupOnly(context.Background())

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.Iterations)
}
