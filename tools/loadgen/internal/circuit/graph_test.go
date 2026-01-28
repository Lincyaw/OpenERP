// Package circuit provides circuit-board-like components for the load generator.
package circuit

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencyGraph(t *testing.T) {
	g := NewDependencyGraph()

	assert.NotNil(t, g)
	assert.NotNil(t, g.endpoints)
	assert.NotNil(t, g.producers)
	assert.NotNil(t, g.consumers)
	assert.NotNil(t, g.adjacency)
	assert.NotNil(t, g.reverseAdjacency)
	assert.Equal(t, 0, g.Size())
}

func TestDependencyGraph_AddEndpoint(t *testing.T) {
	t.Run("adds endpoint correctly", func(t *testing.T) {
		g := NewDependencyGraph()

		unit := &EndpointUnit{
			Name:       "create-customer",
			Path:       "/customers",
			Method:     "POST",
			InputPins:  []SemanticType{},
			OutputPins: []SemanticType{EntityCustomerID},
		}

		g.AddEndpoint(unit)

		assert.Equal(t, 1, g.Size())
		assert.Equal(t, unit, g.GetEndpoint("create-customer"))
	})

	t.Run("registers producers correctly", func(t *testing.T) {
		g := NewDependencyGraph()

		unit := &EndpointUnit{
			Name:       "create-customer",
			Path:       "/customers",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID, EntityCustomerCode},
		}

		g.AddEndpoint(unit)

		producers := g.GetProducers(EntityCustomerID)
		assert.Len(t, producers, 1)
		assert.Equal(t, "create-customer", producers[0].Name)

		producers = g.GetProducers(EntityCustomerCode)
		assert.Len(t, producers, 1)
		assert.Equal(t, "create-customer", producers[0].Name)
	})

	t.Run("registers consumers correctly", func(t *testing.T) {
		g := NewDependencyGraph()

		unit := &EndpointUnit{
			Name:      "get-customer",
			Path:      "/customers/{id}",
			Method:    "GET",
			InputPins: []SemanticType{EntityCustomerID},
		}

		g.AddEndpoint(unit)

		consumers := g.GetConsumers(EntityCustomerID)
		assert.Len(t, consumers, 1)
		assert.Equal(t, "get-customer", consumers[0].Name)
	})

	t.Run("ignores nil endpoint", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEndpoint(nil)
		assert.Equal(t, 0, g.Size())
	})

	t.Run("ignores endpoint with empty name", func(t *testing.T) {
		g := NewDependencyGraph()
		g.AddEndpoint(&EndpointUnit{Name: ""})
		assert.Equal(t, 0, g.Size())
	})

	t.Run("ignores unknown semantic types", func(t *testing.T) {
		g := NewDependencyGraph()

		unit := &EndpointUnit{
			Name:       "test",
			Path:       "/test",
			Method:     "GET",
			InputPins:  []SemanticType{UnknownSemanticType, ""},
			OutputPins: []SemanticType{UnknownSemanticType, ""},
		}

		g.AddEndpoint(unit)

		assert.Equal(t, 1, g.Size())
		assert.Empty(t, g.GetProducers(UnknownSemanticType))
		assert.Empty(t, g.GetConsumers(UnknownSemanticType))
	})
}

func TestDependencyGraph_BuildDependencies(t *testing.T) {
	t.Run("builds producer-consumer dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		// Producer: creates customers
		g.AddEndpoint(&EndpointUnit{
			Name:       "create-customer",
			Path:       "/customers",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID},
		})

		// Consumer: gets customer by ID
		g.AddEndpoint(&EndpointUnit{
			Name:      "get-customer",
			Path:      "/customers/{id}",
			Method:    "GET",
			InputPins: []SemanticType{EntityCustomerID},
		})

		g.BuildDependencies()

		// get-customer should depend on create-customer
		deps := g.GetDependencies("get-customer")
		assert.Contains(t, deps, "create-customer")

		// create-customer should have get-customer as dependent
		dependents := g.GetDependents("create-customer")
		assert.Contains(t, dependents, "get-customer")
	})

	t.Run("handles explicit dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:   "setup",
			Path:   "/setup",
			Method: "POST",
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "main",
			Path:      "/main",
			Method:    "POST",
			DependsOn: []string{"setup"},
		})

		g.BuildDependencies()

		deps := g.GetDependencies("main")
		assert.Contains(t, deps, "setup")
	})

	t.Run("ignores self-dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		// Endpoint that both produces and consumes the same type
		g.AddEndpoint(&EndpointUnit{
			Name:       "self-ref",
			Path:       "/self",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.BuildDependencies()

		deps := g.GetDependencies("self-ref")
		assert.NotContains(t, deps, "self-ref")
	})

	t.Run("ignores non-existent explicit dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:      "main",
			Path:      "/main",
			Method:    "POST",
			DependsOn: []string{"non-existent"},
		})

		g.BuildDependencies()

		deps := g.GetDependencies("main")
		assert.NotContains(t, deps, "non-existent")
	})
}

func TestDependencyGraph_DetectCycles(t *testing.T) {
	t.Run("no cycles in acyclic graph", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:       "b",
			Path:       "/b",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityProductID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "c",
			Path:      "/c",
			Method:    "GET",
			InputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		cycles := g.DetectCycles()
		assert.Empty(t, cycles)
		assert.False(t, g.HasCycles())
	})

	t.Run("detects simple cycle", func(t *testing.T) {
		g := NewDependencyGraph()

		// A produces X, consumes Y
		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			InputPins:  []SemanticType{EntityProductID},
			OutputPins: []SemanticType{EntityCustomerID},
		})

		// B produces Y, consumes X
		g.AddEndpoint(&EndpointUnit{
			Name:       "b",
			Path:       "/b",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		cycles := g.DetectCycles()
		assert.NotEmpty(t, cycles)
		assert.True(t, g.HasCycles())
	})

	t.Run("detects cycle with explicit dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:      "a",
			Path:      "/a",
			Method:    "POST",
			DependsOn: []string{"b"},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "b",
			Path:      "/b",
			Method:    "POST",
			DependsOn: []string{"a"},
		})

		g.BuildDependencies()

		cycles := g.DetectCycles()
		assert.NotEmpty(t, cycles)
	})

	t.Run("empty graph has no cycles", func(t *testing.T) {
		g := NewDependencyGraph()
		cycles := g.DetectCycles()
		assert.Empty(t, cycles)
	})
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	t.Run("sorts acyclic graph correctly", func(t *testing.T) {
		g := NewDependencyGraph()

		// c depends on b, b depends on a
		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:       "b",
			Path:       "/b",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityProductID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "c",
			Path:      "/c",
			Method:    "GET",
			InputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		sorted, err := g.TopologicalSort()
		require.NoError(t, err)
		require.Len(t, sorted, 3)

		// Find positions
		positions := make(map[string]int)
		for i, unit := range sorted {
			positions[unit.Name] = i
		}

		// a should come before b, b should come before c
		assert.Less(t, positions["a"], positions["b"])
		assert.Less(t, positions["b"], positions["c"])
	})

	t.Run("returns error for cyclic graph", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			InputPins:  []SemanticType{EntityProductID},
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:       "b",
			Path:       "/b",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		_, err := g.TopologicalSort()
		assert.ErrorIs(t, err, ErrCycleDetected)
	})

	t.Run("handles empty graph", func(t *testing.T) {
		g := NewDependencyGraph()

		sorted, err := g.TopologicalSort()
		require.NoError(t, err)
		assert.Empty(t, sorted)
	})

	t.Run("handles independent endpoints", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{Name: "a", Path: "/a", Method: "GET"})
		g.AddEndpoint(&EndpointUnit{Name: "b", Path: "/b", Method: "GET"})
		g.AddEndpoint(&EndpointUnit{Name: "c", Path: "/c", Method: "GET"})

		g.BuildDependencies()

		sorted, err := g.TopologicalSort()
		require.NoError(t, err)
		assert.Len(t, sorted, 3)
	})
}

func TestDependencyGraph_GetExecutionPlan(t *testing.T) {
	t.Run("generates plan for target with dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:       "create-customer",
			Path:       "/customers",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:       "create-order",
			Path:       "/orders",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{OrderSalesID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "get-order",
			Path:      "/orders/{id}",
			Method:    "GET",
			InputPins: []SemanticType{OrderSalesID},
		})

		g.BuildDependencies()

		plan, err := g.GetExecutionPlan("get-order")
		require.NoError(t, err)

		assert.Equal(t, "get-order", plan.Target)
		assert.Len(t, plan.Steps, 3)

		// Verify order: create-customer -> create-order -> get-order
		positions := make(map[string]int)
		for i, step := range plan.Steps {
			positions[step.Name] = i
		}

		assert.Less(t, positions["create-customer"], positions["create-order"])
		assert.Less(t, positions["create-order"], positions["get-order"])
	})

	t.Run("returns error for non-existent target", func(t *testing.T) {
		g := NewDependencyGraph()

		_, err := g.GetExecutionPlan("non-existent")
		assert.ErrorIs(t, err, ErrEndpointNotFound)
	})

	t.Run("handles target with no dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:   "standalone",
			Path:   "/standalone",
			Method: "GET",
		})

		g.BuildDependencies()

		plan, err := g.GetExecutionPlan("standalone")
		require.NoError(t, err)

		assert.Equal(t, "standalone", plan.Target)
		assert.Len(t, plan.Steps, 1)
		assert.Equal(t, "standalone", plan.Steps[0].Name)
	})

	t.Run("excludes unrelated endpoints", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:      "b",
			Path:      "/b",
			Method:    "GET",
			InputPins: []SemanticType{EntityCustomerID},
		})

		// Unrelated endpoint
		g.AddEndpoint(&EndpointUnit{
			Name:       "unrelated",
			Path:       "/unrelated",
			Method:     "POST",
			OutputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		plan, err := g.GetExecutionPlan("b")
		require.NoError(t, err)

		// Should only include a and b, not unrelated
		assert.Len(t, plan.Steps, 2)

		names := make([]string, len(plan.Steps))
		for i, step := range plan.Steps {
			names[i] = step.Name
		}
		assert.Contains(t, names, "a")
		assert.Contains(t, names, "b")
		assert.NotContains(t, names, "unrelated")
	})

	t.Run("returns error for cyclic dependencies", func(t *testing.T) {
		g := NewDependencyGraph()

		g.AddEndpoint(&EndpointUnit{
			Name:       "a",
			Path:       "/a",
			Method:     "POST",
			InputPins:  []SemanticType{EntityProductID},
			OutputPins: []SemanticType{EntityCustomerID},
		})

		g.AddEndpoint(&EndpointUnit{
			Name:       "b",
			Path:       "/b",
			Method:     "POST",
			InputPins:  []SemanticType{EntityCustomerID},
			OutputPins: []SemanticType{EntityProductID},
		})

		g.BuildDependencies()

		_, err := g.GetExecutionPlan("a")
		assert.ErrorIs(t, err, ErrCycleDetected)
	})
}

func TestDependencyGraph_GetRootEndpoints(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:       "root1",
		Path:       "/root1",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:       "root2",
		Path:       "/root2",
		Method:     "POST",
		OutputPins: []SemanticType{EntityProductID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "dependent",
		Path:      "/dependent",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID, EntityProductID},
	})

	g.BuildDependencies()

	roots := g.GetRootEndpoints()
	assert.Len(t, roots, 2)

	names := []string{roots[0].Name, roots[1].Name}
	assert.Contains(t, names, "root1")
	assert.Contains(t, names, "root2")
}

func TestDependencyGraph_GetLeafEndpoints(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:       "producer",
		Path:       "/producer",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "leaf1",
		Path:      "/leaf1",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "leaf2",
		Path:      "/leaf2",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID},
	})

	g.BuildDependencies()

	leaves := g.GetLeafEndpoints()
	assert.Len(t, leaves, 2)

	names := []string{leaves[0].Name, leaves[1].Name}
	assert.Contains(t, names, "leaf1")
	assert.Contains(t, names, "leaf2")
}

func TestDependencyGraph_Stats(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:       "a",
		Path:       "/a",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:       "b",
		Path:       "/b",
		Method:     "POST",
		InputPins:  []SemanticType{EntityCustomerID},
		OutputPins: []SemanticType{EntityProductID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "c",
		Path:      "/c",
		Method:    "GET",
		InputPins: []SemanticType{EntityProductID},
	})

	g.BuildDependencies()

	stats := g.Stats()

	assert.Equal(t, 3, stats.TotalEndpoints)
	assert.Equal(t, 2, stats.Connections) // a->b, b->c
	assert.Equal(t, 1, stats.RootEndpoints)
	assert.Equal(t, 1, stats.LeafEndpoints)
	assert.Equal(t, 2, stats.MaxDepth) // c -> b -> a
}

func TestDependencyGraph_ProducersMap(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:       "producer1",
		Path:       "/p1",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:       "producer2",
		Path:       "/p2",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID, EntityProductID},
	})

	producersMap := g.ProducersMap()

	assert.Len(t, producersMap[EntityCustomerID], 2)
	assert.Len(t, producersMap[EntityProductID], 1)
}

func TestDependencyGraph_ConsumersMap(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:      "consumer1",
		Path:      "/c1",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "consumer2",
		Path:      "/c2",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID, EntityProductID},
	})

	consumersMap := g.ConsumersMap()

	assert.Len(t, consumersMap[EntityCustomerID], 2)
	assert.Len(t, consumersMap[EntityProductID], 1)
}

func TestDependencyGraph_Clear(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{
		Name:       "test",
		Path:       "/test",
		Method:     "GET",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.BuildDependencies()

	assert.Equal(t, 1, g.Size())

	g.Clear()

	assert.Equal(t, 0, g.Size())
	assert.Empty(t, g.GetAllEndpoints())
	assert.Empty(t, g.ProducersMap())
	assert.Empty(t, g.ConsumersMap())
}

func TestDependencyGraph_GetAllEndpoints(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEndpoint(&EndpointUnit{Name: "c", Path: "/c", Method: "GET"})
	g.AddEndpoint(&EndpointUnit{Name: "a", Path: "/a", Method: "GET"})
	g.AddEndpoint(&EndpointUnit{Name: "b", Path: "/b", Method: "GET"})

	endpoints := g.GetAllEndpoints()

	assert.Len(t, endpoints, 3)
	// Should be sorted by name
	assert.Equal(t, "a", endpoints[0].Name)
	assert.Equal(t, "b", endpoints[1].Name)
	assert.Equal(t, "c", endpoints[2].Name)
}

func TestDependencyGraph_Concurrent(t *testing.T) {
	g := NewDependencyGraph()

	// Add some initial endpoints
	for i := 0; i < 10; i++ {
		g.AddEndpoint(&EndpointUnit{
			Name:       "endpoint-" + string(rune('a'+i)),
			Path:       "/endpoint-" + string(rune('a'+i)),
			Method:     "GET",
			OutputPins: []SemanticType{EntityCustomerID},
		})
	}

	g.BuildDependencies()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = g.GetAllEndpoints()
			_ = g.GetProducers(EntityCustomerID)
			_ = g.GetConsumers(EntityCustomerID)
			_ = g.Stats()
			_ = g.Size()
			_ = g.GetRootEndpoints()
			_ = g.GetLeafEndpoints()
		}()
	}

	wg.Wait()
}

func TestCycle_String(t *testing.T) {
	t.Run("formats cycle correctly", func(t *testing.T) {
		cycle := Cycle{Path: []string{"a", "b", "c"}}
		assert.Equal(t, "a -> b -> c -> a", cycle.String())
	})

	t.Run("handles empty cycle", func(t *testing.T) {
		cycle := Cycle{Path: []string{}}
		assert.Equal(t, "empty cycle", cycle.String())
	})

	t.Run("handles single element cycle", func(t *testing.T) {
		cycle := Cycle{Path: []string{"a"}}
		assert.Equal(t, "a -> a", cycle.String())
	})
}

// TestERPSystemNoCycles verifies that a typical ERP system configuration has no cycles.
func TestERPSystemNoCycles(t *testing.T) {
	g := NewDependencyGraph()

	// Simulate typical ERP endpoints

	// Auth - no dependencies, produces access token
	g.AddEndpoint(&EndpointUnit{
		Name:       "login",
		Path:       "/auth/login",
		Method:     "POST",
		OutputPins: []SemanticType{SystemAccessToken},
	})

	// Master data - produces entity IDs
	g.AddEndpoint(&EndpointUnit{
		Name:       "create-customer",
		Path:       "/customers",
		Method:     "POST",
		OutputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:       "create-product",
		Path:       "/products",
		Method:     "POST",
		OutputPins: []SemanticType{EntityProductID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:       "create-warehouse",
		Path:       "/warehouses",
		Method:     "POST",
		OutputPins: []SemanticType{EntityWarehouseID},
	})

	// Read operations - consume entity IDs
	g.AddEndpoint(&EndpointUnit{
		Name:      "get-customer",
		Path:      "/customers/{id}",
		Method:    "GET",
		InputPins: []SemanticType{EntityCustomerID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "get-product",
		Path:      "/products/{id}",
		Method:    "GET",
		InputPins: []SemanticType{EntityProductID},
	})

	// Orders - consume customer and product, produce order ID
	g.AddEndpoint(&EndpointUnit{
		Name:       "create-sales-order",
		Path:       "/sales-orders",
		Method:     "POST",
		InputPins:  []SemanticType{EntityCustomerID, EntityProductID},
		OutputPins: []SemanticType{OrderSalesID},
	})

	g.AddEndpoint(&EndpointUnit{
		Name:      "get-sales-order",
		Path:      "/sales-orders/{id}",
		Method:    "GET",
		InputPins: []SemanticType{OrderSalesID},
	})

	// Inventory - consume product and warehouse
	g.AddEndpoint(&EndpointUnit{
		Name:       "create-stock-movement",
		Path:       "/stock-movements",
		Method:     "POST",
		InputPins:  []SemanticType{EntityProductID, EntityWarehouseID},
		OutputPins: []SemanticType{InventoryMovementID},
	})

	// Payments - consume order ID
	g.AddEndpoint(&EndpointUnit{
		Name:       "create-payment",
		Path:       "/payments",
		Method:     "POST",
		InputPins:  []SemanticType{OrderSalesID},
		OutputPins: []SemanticType{FinancePaymentID},
	})

	g.BuildDependencies()

	// Verify no cycles
	cycles := g.DetectCycles()
	assert.Empty(t, cycles, "ERP system should have no circular dependencies")

	// Verify topological sort works
	sorted, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, sorted, 10)

	// Verify execution plan for payment (should include order and customer)
	plan, err := g.GetExecutionPlan("create-payment")
	require.NoError(t, err)

	stepNames := make([]string, len(plan.Steps))
	for i, step := range plan.Steps {
		stepNames[i] = step.Name
	}

	assert.Contains(t, stepNames, "create-customer")
	assert.Contains(t, stepNames, "create-product")
	assert.Contains(t, stepNames, "create-sales-order")
	assert.Contains(t, stepNames, "create-payment")

	// Verify stats
	stats := g.Stats()
	assert.Equal(t, 10, stats.TotalEndpoints)
	assert.Greater(t, stats.Connections, 0)
}
