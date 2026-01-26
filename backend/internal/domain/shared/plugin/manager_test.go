package plugin

import (
	"testing"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlugin is a test implementation of IndustryPlugin
type mockPlugin struct {
	name        string
	displayName string
	attributes  []AttributeDefinition
}

func (p *mockPlugin) Name() string {
	return p.name
}

func (p *mockPlugin) DisplayName() string {
	return p.displayName
}

func (p *mockPlugin) RegisterStrategies(registry StrategyRegistrar) {
	// No-op for testing
}

func (p *mockPlugin) GetRequiredProductAttributes() []AttributeDefinition {
	return p.attributes
}

// mockRegistrar is a test implementation of StrategyRegistrar
type mockRegistrar struct{}

func (m *mockRegistrar) RegisterCostStrategy(s any) error       { return nil }
func (m *mockRegistrar) RegisterPricingStrategy(s any) error    { return nil }
func (m *mockRegistrar) RegisterAllocationStrategy(s any) error { return nil }
func (m *mockRegistrar) RegisterBatchStrategy(s any) error      { return nil }
func (m *mockRegistrar) RegisterValidationStrategy(s any) error { return nil }

func TestNewPluginManager(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())
}

func TestPluginManager_Register_Success(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{
		name:        "test",
		displayName: "Test Plugin",
		attributes:  []AttributeDefinition{{Key: "test_attr", Label: "Test Attribute"}},
	}

	err := manager.Register(plugin)
	require.NoError(t, err)

	assert.Equal(t, 1, manager.Count())
	assert.Contains(t, manager.ListPlugins(), "test")
}

func TestPluginManager_Register_NilPlugin(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	err := manager.Register(nil)
	assert.ErrorIs(t, err, shared.ErrInvalidInput)
}

func TestPluginManager_Register_EmptyName(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{
		name: "",
	}

	err := manager.Register(plugin)
	assert.ErrorIs(t, err, shared.ErrInvalidInput)
}

func TestPluginManager_Register_Duplicate(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{name: "test"}

	err := manager.Register(plugin)
	require.NoError(t, err)

	err = manager.Register(plugin)
	assert.ErrorIs(t, err, shared.ErrAlreadyExists)
}

func TestPluginManager_GetPlugin_Found(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{name: "test", displayName: "Test Plugin"}
	err := manager.Register(plugin)
	require.NoError(t, err)

	retrieved, ok := manager.GetPlugin("test")
	assert.True(t, ok)
	assert.Equal(t, "test", retrieved.Name())
	assert.Equal(t, "Test Plugin", retrieved.DisplayName())
}

func TestPluginManager_GetPlugin_NotFound(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	_, ok := manager.GetPlugin("nonexistent")
	assert.False(t, ok)
}

func TestPluginManager_ListPlugins(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	// Register multiple plugins
	for _, name := range []string{"charlie", "alpha", "beta"} {
		err := manager.Register(&mockPlugin{name: name})
		require.NoError(t, err)
	}

	// List should be sorted
	list := manager.ListPlugins()
	assert.Equal(t, []string{"alpha", "beta", "charlie"}, list)
}

func TestPluginManager_GetAllPlugins(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin1 := &mockPlugin{name: "plugin1"}
	plugin2 := &mockPlugin{name: "plugin2"}

	_ = manager.Register(plugin1)
	_ = manager.Register(plugin2)

	all := manager.GetAllPlugins()
	assert.Len(t, all, 2)
}

func TestPluginManager_GetRequiredAttributes(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{
		name: "test",
		attributes: []AttributeDefinition{
			{Key: "attr1", Label: "Attribute 1"},
			{Key: "attr2", Label: "Attribute 2"},
		},
	}
	_ = manager.Register(plugin)

	attrs := manager.GetRequiredAttributes()
	assert.Contains(t, attrs, "test")
	assert.Len(t, attrs["test"], 2)
}

func TestPluginManager_GetAttributesForPlugin_Found(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{
		name: "test",
		attributes: []AttributeDefinition{
			{Key: "attr1", Label: "Attribute 1", Required: true},
		},
	}
	_ = manager.Register(plugin)

	attrs, err := manager.GetAttributesForPlugin("test")
	require.NoError(t, err)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "attr1", attrs[0].Key)
	assert.True(t, attrs[0].Required)
}

func TestPluginManager_GetAttributesForPlugin_NotFound(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	_, err := manager.GetAttributesForPlugin("nonexistent")
	assert.ErrorIs(t, err, shared.ErrNotFound)
}

func TestPluginManager_Unregister_Success(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	plugin := &mockPlugin{name: "test"}
	_ = manager.Register(plugin)
	assert.Equal(t, 1, manager.Count())

	err := manager.Unregister("test")
	require.NoError(t, err)
	assert.Equal(t, 0, manager.Count())
}

func TestPluginManager_Unregister_NotFound(t *testing.T) {
	registry := &mockRegistrar{}
	manager := NewPluginManager(registry)

	err := manager.Unregister("nonexistent")
	assert.ErrorIs(t, err, shared.ErrNotFound)
}

func TestAttributeDefinition(t *testing.T) {
	attr := AttributeDefinition{
		Key:           "registration_number",
		Label:         "Registration Number",
		Required:      true,
		Regex:         `^PD\d{8}$`,
		CategoryCodes: []string{"PESTICIDE"},
	}

	assert.Equal(t, "registration_number", attr.Key)
	assert.Equal(t, "Registration Number", attr.Label)
	assert.True(t, attr.Required)
	assert.Equal(t, `^PD\d{8}$`, attr.Regex)
	assert.Contains(t, attr.CategoryCodes, "PESTICIDE")
}
