package strategy

import (
	"context"
	"sync"
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/domain/shared/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock cost strategy for testing
type mockCostStrategy struct {
	strategy.BaseStrategy
}

func newMockCostStrategy(name string) *mockCostStrategy {
	return &mockCostStrategy{
		BaseStrategy: strategy.NewBaseStrategy(name, strategy.StrategyTypeCost, "Mock cost strategy"),
	}
}

func (s *mockCostStrategy) Method() strategy.CostMethod {
	return strategy.CostMethodMovingAverage
}

func (s *mockCostStrategy) CalculateCost(ctx context.Context, costCtx strategy.CostContext, entries []strategy.StockEntry) (strategy.CostResult, error) {
	return strategy.CostResult{}, nil
}

func (s *mockCostStrategy) CalculateAverageCost(ctx context.Context, entries []strategy.StockEntry) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// Mock pricing strategy for testing
type mockPricingStrategy struct {
	strategy.BaseStrategy
}

func newMockPricingStrategy(name string) *mockPricingStrategy {
	return &mockPricingStrategy{
		BaseStrategy: strategy.NewBaseStrategy(name, strategy.StrategyTypePricing, "Mock pricing strategy"),
	}
}

func (s *mockPricingStrategy) CalculatePrice(ctx context.Context, pricingCtx strategy.PricingContext) (strategy.PricingResult, error) {
	return strategy.PricingResult{}, nil
}

func (s *mockPricingStrategy) SupportsPromotion() bool {
	return false
}

func (s *mockPricingStrategy) SupportsTieredPricing() bool {
	return false
}

// Mock allocation strategy for testing
type mockAllocationStrategy struct {
	strategy.BaseStrategy
}

func newMockAllocationStrategy(name string) *mockAllocationStrategy {
	return &mockAllocationStrategy{
		BaseStrategy: strategy.NewBaseStrategy(name, strategy.StrategyTypeAllocation, "Mock allocation strategy"),
	}
}

func (s *mockAllocationStrategy) Allocate(ctx context.Context, allocCtx strategy.AllocationContext, invoices []strategy.Invoice) (strategy.AllocationResult, error) {
	return strategy.AllocationResult{}, nil
}

func (s *mockAllocationStrategy) SupportsPartialAllocation() bool {
	return true
}

// Mock batch strategy for testing
type mockBatchStrategy struct {
	strategy.BaseStrategy
}

func newMockBatchStrategy(name string) *mockBatchStrategy {
	return &mockBatchStrategy{
		BaseStrategy: strategy.NewBaseStrategy(name, strategy.StrategyTypeBatch, "Mock batch strategy"),
	}
}

func (s *mockBatchStrategy) SelectBatches(ctx context.Context, selCtx strategy.BatchSelectionContext, batches []strategy.Batch) (strategy.BatchSelectionResult, error) {
	return strategy.BatchSelectionResult{}, nil
}

func (s *mockBatchStrategy) ConsidersExpiry() bool {
	return false
}

func (s *mockBatchStrategy) SupportsFEFO() bool {
	return false
}

// Mock validation strategy for testing
type mockValidationStrategy struct {
	strategy.BaseStrategy
}

func newMockValidationStrategy(name string) *mockValidationStrategy {
	return &mockValidationStrategy{
		BaseStrategy: strategy.NewBaseStrategy(name, strategy.StrategyTypeValidation, "Mock validation strategy"),
	}
}

func (s *mockValidationStrategy) Validate(ctx context.Context, valCtx strategy.ValidationContext, data strategy.ProductData) (strategy.ValidationResult, error) {
	return strategy.ValidationResult{IsValid: true}, nil
}

func (s *mockValidationStrategy) ValidateField(ctx context.Context, field string, value any) ([]strategy.ValidationError, error) {
	return nil, nil
}

func TestNewStrategyRegistry(t *testing.T) {
	r := NewStrategyRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.costStrategies)
	assert.NotNil(t, r.pricingStrategies)
	assert.NotNil(t, r.allocationStrategies)
	assert.NotNil(t, r.batchStrategies)
	assert.NotNil(t, r.validationStrategies)
	assert.NotNil(t, r.defaults)
}

func TestRegisterCostStrategy(t *testing.T) {
	r := NewStrategyRegistry()

	t.Run("successful registration", func(t *testing.T) {
		s := newMockCostStrategy("test_cost")
		err := r.RegisterCostStrategy(s)
		assert.NoError(t, err)
		assert.True(t, r.IsRegistered(strategy.StrategyTypeCost, "test_cost"))
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		s := newMockCostStrategy("duplicate_cost")
		err := r.RegisterCostStrategy(s)
		require.NoError(t, err)

		err = r.RegisterCostStrategy(s)
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrAlreadyExists)
	})
}

func TestGetCostStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockCostStrategy("get_cost")
	require.NoError(t, r.RegisterCostStrategy(s))

	t.Run("get by name", func(t *testing.T) {
		got, err := r.GetCostStrategy("get_cost")
		assert.NoError(t, err)
		assert.Equal(t, "get_cost", got.Name())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.GetCostStrategy("nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})

	t.Run("get default when name is empty", func(t *testing.T) {
		require.NoError(t, r.SetDefault(strategy.StrategyTypeCost, "get_cost"))
		got, err := r.GetCostStrategy("")
		assert.NoError(t, err)
		assert.Equal(t, "get_cost", got.Name())
	})

	t.Run("no default set", func(t *testing.T) {
		r2 := NewStrategyRegistry()
		_, err := r2.GetCostStrategy("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})
}

func TestGetCostStrategyOrDefault(t *testing.T) {
	r := NewStrategyRegistry()
	defaultS := newMockCostStrategy("default_cost")
	otherS := newMockCostStrategy("other_cost")
	require.NoError(t, r.RegisterCostStrategy(defaultS))
	require.NoError(t, r.RegisterCostStrategy(otherS))
	require.NoError(t, r.SetDefault(strategy.StrategyTypeCost, "default_cost"))

	t.Run("get existing by name", func(t *testing.T) {
		got := r.GetCostStrategyOrDefault("other_cost")
		assert.Equal(t, "other_cost", got.Name())
	})

	t.Run("fallback to default when not found", func(t *testing.T) {
		got := r.GetCostStrategyOrDefault("nonexistent")
		assert.Equal(t, "default_cost", got.Name())
	})

	t.Run("fallback to default when empty name", func(t *testing.T) {
		got := r.GetCostStrategyOrDefault("")
		assert.Equal(t, "default_cost", got.Name())
	})
}

func TestListCostStrategies(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("b_cost")))
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("a_cost")))
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("c_cost")))

	list := r.ListCostStrategies()
	assert.Equal(t, []string{"a_cost", "b_cost", "c_cost"}, list)
}

func TestUnregisterCostStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockCostStrategy("unregister_cost")
	require.NoError(t, r.RegisterCostStrategy(s))
	require.NoError(t, r.SetDefault(strategy.StrategyTypeCost, "unregister_cost"))

	t.Run("successful unregister", func(t *testing.T) {
		err := r.UnregisterCostStrategy("unregister_cost")
		assert.NoError(t, err)
		assert.False(t, r.IsRegistered(strategy.StrategyTypeCost, "unregister_cost"))
		assert.False(t, r.HasDefault(strategy.StrategyTypeCost))
	})

	t.Run("unregister nonexistent", func(t *testing.T) {
		err := r.UnregisterCostStrategy("nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})
}

func TestRegisterPricingStrategy(t *testing.T) {
	r := NewStrategyRegistry()

	t.Run("successful registration", func(t *testing.T) {
		s := newMockPricingStrategy("test_pricing")
		err := r.RegisterPricingStrategy(s)
		assert.NoError(t, err)
		assert.True(t, r.IsRegistered(strategy.StrategyTypePricing, "test_pricing"))
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		s := newMockPricingStrategy("dup_pricing")
		require.NoError(t, r.RegisterPricingStrategy(s))
		err := r.RegisterPricingStrategy(s)
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrAlreadyExists)
	})
}

func TestGetPricingStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockPricingStrategy("get_pricing")
	require.NoError(t, r.RegisterPricingStrategy(s))

	t.Run("get by name", func(t *testing.T) {
		got, err := r.GetPricingStrategy("get_pricing")
		assert.NoError(t, err)
		assert.Equal(t, "get_pricing", got.Name())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.GetPricingStrategy("nonexistent")
		assert.Error(t, err)
	})
}

func TestListPricingStrategies(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterPricingStrategy(newMockPricingStrategy("pricing_b")))
	require.NoError(t, r.RegisterPricingStrategy(newMockPricingStrategy("pricing_a")))

	list := r.ListPricingStrategies()
	assert.Equal(t, []string{"pricing_a", "pricing_b"}, list)
}

func TestUnregisterPricingStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterPricingStrategy(newMockPricingStrategy("unreg_pricing")))

	err := r.UnregisterPricingStrategy("unreg_pricing")
	assert.NoError(t, err)
	assert.False(t, r.IsRegistered(strategy.StrategyTypePricing, "unreg_pricing"))
}

func TestRegisterAllocationStrategy(t *testing.T) {
	r := NewStrategyRegistry()

	t.Run("successful registration", func(t *testing.T) {
		s := newMockAllocationStrategy("test_alloc")
		err := r.RegisterAllocationStrategy(s)
		assert.NoError(t, err)
		assert.True(t, r.IsRegistered(strategy.StrategyTypeAllocation, "test_alloc"))
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		s := newMockAllocationStrategy("dup_alloc")
		require.NoError(t, r.RegisterAllocationStrategy(s))
		err := r.RegisterAllocationStrategy(s)
		assert.Error(t, err)
	})
}

func TestGetAllocationStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockAllocationStrategy("get_alloc")
	require.NoError(t, r.RegisterAllocationStrategy(s))

	got, err := r.GetAllocationStrategy("get_alloc")
	assert.NoError(t, err)
	assert.Equal(t, "get_alloc", got.Name())
}

func TestListAllocationStrategies(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterAllocationStrategy(newMockAllocationStrategy("alloc_z")))
	require.NoError(t, r.RegisterAllocationStrategy(newMockAllocationStrategy("alloc_a")))

	list := r.ListAllocationStrategies()
	assert.Equal(t, []string{"alloc_a", "alloc_z"}, list)
}

func TestUnregisterAllocationStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterAllocationStrategy(newMockAllocationStrategy("unreg_alloc")))

	err := r.UnregisterAllocationStrategy("unreg_alloc")
	assert.NoError(t, err)
	assert.False(t, r.IsRegistered(strategy.StrategyTypeAllocation, "unreg_alloc"))
}

func TestRegisterBatchStrategy(t *testing.T) {
	r := NewStrategyRegistry()

	t.Run("successful registration", func(t *testing.T) {
		s := newMockBatchStrategy("test_batch")
		err := r.RegisterBatchStrategy(s)
		assert.NoError(t, err)
		assert.True(t, r.IsRegistered(strategy.StrategyTypeBatch, "test_batch"))
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		s := newMockBatchStrategy("dup_batch")
		require.NoError(t, r.RegisterBatchStrategy(s))
		err := r.RegisterBatchStrategy(s)
		assert.Error(t, err)
	})
}

func TestGetBatchStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockBatchStrategy("get_batch")
	require.NoError(t, r.RegisterBatchStrategy(s))

	got, err := r.GetBatchStrategy("get_batch")
	assert.NoError(t, err)
	assert.Equal(t, "get_batch", got.Name())
}

func TestListBatchStrategies(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterBatchStrategy(newMockBatchStrategy("batch_2")))
	require.NoError(t, r.RegisterBatchStrategy(newMockBatchStrategy("batch_1")))

	list := r.ListBatchStrategies()
	assert.Equal(t, []string{"batch_1", "batch_2"}, list)
}

func TestUnregisterBatchStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterBatchStrategy(newMockBatchStrategy("unreg_batch")))

	err := r.UnregisterBatchStrategy("unreg_batch")
	assert.NoError(t, err)
	assert.False(t, r.IsRegistered(strategy.StrategyTypeBatch, "unreg_batch"))
}

func TestRegisterValidationStrategy(t *testing.T) {
	r := NewStrategyRegistry()

	t.Run("successful registration", func(t *testing.T) {
		s := newMockValidationStrategy("test_validation")
		err := r.RegisterValidationStrategy(s)
		assert.NoError(t, err)
		assert.True(t, r.IsRegistered(strategy.StrategyTypeValidation, "test_validation"))
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		s := newMockValidationStrategy("dup_validation")
		require.NoError(t, r.RegisterValidationStrategy(s))
		err := r.RegisterValidationStrategy(s)
		assert.Error(t, err)
	})
}

func TestGetValidationStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	s := newMockValidationStrategy("get_validation")
	require.NoError(t, r.RegisterValidationStrategy(s))

	got, err := r.GetValidationStrategy("get_validation")
	assert.NoError(t, err)
	assert.Equal(t, "get_validation", got.Name())
}

func TestListValidationStrategies(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterValidationStrategy(newMockValidationStrategy("val_b")))
	require.NoError(t, r.RegisterValidationStrategy(newMockValidationStrategy("val_a")))

	list := r.ListValidationStrategies()
	assert.Equal(t, []string{"val_a", "val_b"}, list)
}

func TestUnregisterValidationStrategy(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterValidationStrategy(newMockValidationStrategy("unreg_val")))

	err := r.UnregisterValidationStrategy("unreg_val")
	assert.NoError(t, err)
	assert.False(t, r.IsRegistered(strategy.StrategyTypeValidation, "unreg_val"))
}

func TestSetDefault(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("default_test")))

	t.Run("set default successfully", func(t *testing.T) {
		err := r.SetDefault(strategy.StrategyTypeCost, "default_test")
		assert.NoError(t, err)
		assert.Equal(t, "default_test", r.GetDefault(strategy.StrategyTypeCost))
		assert.True(t, r.HasDefault(strategy.StrategyTypeCost))
	})

	t.Run("set default for nonexistent strategy fails", func(t *testing.T) {
		err := r.SetDefault(strategy.StrategyTypeCost, "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, shared.ErrNotFound)
	})
}

func TestIsRegistered(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("registered")))

	assert.True(t, r.IsRegistered(strategy.StrategyTypeCost, "registered"))
	assert.False(t, r.IsRegistered(strategy.StrategyTypeCost, "not_registered"))
	assert.False(t, r.IsRegistered(strategy.StrategyTypePricing, "registered"))
	assert.False(t, r.IsRegistered("invalid_type", "registered"))
}

func TestStats(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("cost1")))
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("cost2")))
	require.NoError(t, r.RegisterPricingStrategy(newMockPricingStrategy("pricing1")))

	stats := r.Stats()
	assert.Equal(t, 2, stats[strategy.StrategyTypeCost])
	assert.Equal(t, 1, stats[strategy.StrategyTypePricing])
	assert.Equal(t, 0, stats[strategy.StrategyTypeAllocation])
	assert.Equal(t, 0, stats[strategy.StrategyTypeBatch])
	assert.Equal(t, 0, stats[strategy.StrategyTypeValidation])
}

func TestConcurrentAccess(t *testing.T) {
	r := NewStrategyRegistry()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := string(rune('a' + (idx % 26)))
			// Each goroutine tries to register - some will succeed, some will get duplicate error
			_ = r.RegisterCostStrategy(newMockCostStrategy(name))
		}(i)
	}

	wg.Wait()

	// Verify no panic and consistent state
	list := r.ListCostStrategies()
	assert.NotEmpty(t, list)
	assert.LessOrEqual(t, len(list), 26) // At most 26 unique strategies (a-z)
}

func TestConcurrentReadWrite(t *testing.T) {
	r := NewStrategyRegistry()
	require.NoError(t, r.RegisterCostStrategy(newMockCostStrategy("concurrent_test")))

	var wg sync.WaitGroup
	numReaders := 50
	numWriters := 10

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = r.GetCostStrategy("concurrent_test")
				r.ListCostStrategies()
				r.Stats()
			}
		}()
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := string(rune('A' + idx))
			_ = r.RegisterCostStrategy(newMockCostStrategy(name))
		}(i)
	}

	wg.Wait()
}

func TestNewRegistryWithDefaults(t *testing.T) {
	r, err := NewRegistryWithDefaults()
	require.NoError(t, err)
	require.NotNil(t, r)

	// Verify cost strategies
	costList := r.ListCostStrategies()
	assert.Contains(t, costList, "moving_average")
	assert.Contains(t, costList, "fifo")
	assert.True(t, r.HasDefault(strategy.StrategyTypeCost))
	assert.Equal(t, "moving_average", r.GetDefault(strategy.StrategyTypeCost))

	// Verify pricing strategies
	pricingList := r.ListPricingStrategies()
	assert.Contains(t, pricingList, "standard")
	assert.True(t, r.HasDefault(strategy.StrategyTypePricing))

	// Verify allocation strategies
	allocList := r.ListAllocationStrategies()
	assert.Contains(t, allocList, "fifo")
	assert.True(t, r.HasDefault(strategy.StrategyTypeAllocation))

	// Verify batch strategies
	batchList := r.ListBatchStrategies()
	assert.Contains(t, batchList, "standard")
	assert.True(t, r.HasDefault(strategy.StrategyTypeBatch))

	// Verify validation strategies
	valList := r.ListValidationStrategies()
	assert.Contains(t, valList, "standard")
	assert.True(t, r.HasDefault(strategy.StrategyTypeValidation))

	// Verify getting default strategies
	costStrategy, err := r.GetCostStrategy("")
	assert.NoError(t, err)
	assert.Equal(t, "moving_average", costStrategy.Name())

	pricingStrategy, err := r.GetPricingStrategy("")
	assert.NoError(t, err)
	assert.Equal(t, "standard", pricingStrategy.Name())
}

func TestGetPricingStrategyOrDefault(t *testing.T) {
	r := NewStrategyRegistry()
	defaultS := newMockPricingStrategy("default_pricing")
	require.NoError(t, r.RegisterPricingStrategy(defaultS))
	require.NoError(t, r.SetDefault(strategy.StrategyTypePricing, "default_pricing"))

	got := r.GetPricingStrategyOrDefault("nonexistent")
	assert.Equal(t, "default_pricing", got.Name())
}

func TestGetAllocationStrategyOrDefault(t *testing.T) {
	r := NewStrategyRegistry()
	defaultS := newMockAllocationStrategy("default_alloc")
	require.NoError(t, r.RegisterAllocationStrategy(defaultS))
	require.NoError(t, r.SetDefault(strategy.StrategyTypeAllocation, "default_alloc"))

	got := r.GetAllocationStrategyOrDefault("nonexistent")
	assert.Equal(t, "default_alloc", got.Name())
}

func TestGetBatchStrategyOrDefault(t *testing.T) {
	r := NewStrategyRegistry()
	defaultS := newMockBatchStrategy("default_batch")
	require.NoError(t, r.RegisterBatchStrategy(defaultS))
	require.NoError(t, r.SetDefault(strategy.StrategyTypeBatch, "default_batch"))

	got := r.GetBatchStrategyOrDefault("nonexistent")
	assert.Equal(t, "default_batch", got.Name())
}

func TestGetValidationStrategyOrDefault(t *testing.T) {
	r := NewStrategyRegistry()
	defaultS := newMockValidationStrategy("default_val")
	require.NoError(t, r.RegisterValidationStrategy(defaultS))
	require.NoError(t, r.SetDefault(strategy.StrategyTypeValidation, "default_val"))

	got := r.GetValidationStrategyOrDefault("nonexistent")
	assert.Equal(t, "default_val", got.Name())
}
