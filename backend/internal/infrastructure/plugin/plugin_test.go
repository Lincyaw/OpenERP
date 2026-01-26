package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStrategyRegistrar is a mock implementation for testing
type MockStrategyRegistrar struct {
	costStrategies       []any
	pricingStrategies    []any
	allocationStrategies []any
	batchStrategies      []any
	validationStrategies []any
}

func NewMockStrategyRegistrar() *MockStrategyRegistrar {
	return &MockStrategyRegistrar{
		costStrategies:       make([]any, 0),
		pricingStrategies:    make([]any, 0),
		allocationStrategies: make([]any, 0),
		batchStrategies:      make([]any, 0),
		validationStrategies: make([]any, 0),
	}
}

func (m *MockStrategyRegistrar) RegisterCostStrategy(s any) error {
	m.costStrategies = append(m.costStrategies, s)
	return nil
}

func (m *MockStrategyRegistrar) RegisterPricingStrategy(s any) error {
	m.pricingStrategies = append(m.pricingStrategies, s)
	return nil
}

func (m *MockStrategyRegistrar) RegisterAllocationStrategy(s any) error {
	m.allocationStrategies = append(m.allocationStrategies, s)
	return nil
}

func (m *MockStrategyRegistrar) RegisterBatchStrategy(s any) error {
	m.batchStrategies = append(m.batchStrategies, s)
	return nil
}

func (m *MockStrategyRegistrar) RegisterValidationStrategy(s any) error {
	m.validationStrategies = append(m.validationStrategies, s)
	return nil
}

func TestPluginManager_Register(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	// Verify plugin is registered
	assert.Equal(t, 1, manager.Count())
	assert.Contains(t, manager.ListPlugins(), "agricultural")

	// Verify strategies were registered
	assert.Len(t, registry.validationStrategies, 1)
}

func TestPluginManager_Register_NilPlugin(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	err := manager.Register(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestPluginManager_Register_Duplicate(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	// Try to register same plugin again
	err = manager.Register(plugin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPluginManager_GetPlugin(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	// Get existing plugin
	retrieved, ok := manager.GetPlugin("agricultural")
	assert.True(t, ok)
	assert.Equal(t, "agricultural", retrieved.Name())

	// Get non-existent plugin
	_, ok = manager.GetPlugin("nonexistent")
	assert.False(t, ok)
}

func TestPluginManager_GetAllPlugins(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	plugins := manager.GetAllPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "agricultural", plugins[0].Name())
}

func TestPluginManager_GetRequiredAttributes(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	attrs := manager.GetRequiredAttributes()
	assert.Contains(t, attrs, "agricultural")
	assert.NotEmpty(t, attrs["agricultural"])
}

func TestPluginManager_GetAttributesForPlugin(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	// Get attributes for existing plugin
	attrs, err := manager.GetAttributesForPlugin("agricultural")
	require.NoError(t, err)
	assert.NotEmpty(t, attrs)

	// Get attributes for non-existent plugin
	_, err = manager.GetAttributesForPlugin("nonexistent")
	assert.Error(t, err)
}

func TestPluginManager_Unregister(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	manager := NewPluginManager(registry)

	plugin := NewAgriculturalPlugin()
	err := manager.Register(plugin)
	require.NoError(t, err)

	assert.Equal(t, 1, manager.Count())

	// Unregister plugin
	err = manager.Unregister("agricultural")
	require.NoError(t, err)
	assert.Equal(t, 0, manager.Count())

	// Unregister non-existent plugin
	err = manager.Unregister("nonexistent")
	assert.Error(t, err)
}

func TestAgriculturalPlugin_Name(t *testing.T) {
	plugin := NewAgriculturalPlugin()
	assert.Equal(t, "agricultural", plugin.Name())
}

func TestAgriculturalPlugin_DisplayName(t *testing.T) {
	plugin := NewAgriculturalPlugin()
	assert.Equal(t, "农资行业", plugin.DisplayName())
}

func TestAgriculturalPlugin_GetRequiredProductAttributes(t *testing.T) {
	plugin := NewAgriculturalPlugin()
	attrs := plugin.GetRequiredProductAttributes()

	assert.NotEmpty(t, attrs)

	// Check for expected attributes
	attrKeys := make([]string, len(attrs))
	for i, attr := range attrs {
		attrKeys[i] = attr.Key
	}

	assert.Contains(t, attrKeys, "registration_number")
	assert.Contains(t, attrKeys, "variety_approval_number")
	assert.Contains(t, attrKeys, "manufacturer")
}

func TestAgriculturalPlugin_RegisterStrategies(t *testing.T) {
	registry := NewMockStrategyRegistrar()
	plugin := NewAgriculturalPlugin()

	plugin.RegisterStrategies(registry)

	// Should register validation strategy
	assert.Len(t, registry.validationStrategies, 1)
}
