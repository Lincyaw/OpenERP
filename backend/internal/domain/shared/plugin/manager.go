package plugin

import (
	"fmt"
	"sort"
	"sync"

	"github.com/erp/backend/internal/domain/shared"
)

// PluginManager manages industry plugin registrations
type PluginManager struct {
	mu       sync.RWMutex
	plugins  map[string]IndustryPlugin
	registry StrategyRegistrar
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(registry StrategyRegistrar) *PluginManager {
	return &PluginManager{
		plugins:  make(map[string]IndustryPlugin),
		registry: registry,
	}
}

// Register registers an industry plugin
// This also triggers the plugin's strategy registration
func (m *PluginManager) Register(plugin IndustryPlugin) error {
	if plugin == nil {
		return fmt.Errorf("%w: plugin cannot be nil", shared.ErrInvalidInput)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("%w: plugin name cannot be empty", shared.ErrInvalidInput)
	}

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("%w: plugin '%s' already registered", shared.ErrAlreadyExists, name)
	}

	// Register plugin's strategies with the registry
	plugin.RegisterStrategies(m.registry)

	m.plugins[name] = plugin
	return nil
}

// GetPlugin returns a plugin by name
func (m *PluginManager) GetPlugin(name string) (IndustryPlugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	return plugin, exists
}

// ListPlugins returns all registered plugin names
func (m *PluginManager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAllPlugins returns all registered plugins
func (m *PluginManager) GetAllPlugins() []IndustryPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]IndustryPlugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetRequiredAttributes returns all required attributes from all registered plugins
func (m *PluginManager) GetRequiredAttributes() map[string][]AttributeDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]AttributeDefinition)
	for name, plugin := range m.plugins {
		result[name] = plugin.GetRequiredProductAttributes()
	}
	return result
}

// GetAttributesForPlugin returns required attributes for a specific plugin
func (m *PluginManager) GetAttributesForPlugin(pluginName string) ([]AttributeDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("%w: plugin '%s' not found", shared.ErrNotFound, pluginName)
	}

	return plugin.GetRequiredProductAttributes(), nil
}

// Unregister removes a plugin (useful for testing)
func (m *PluginManager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; !exists {
		return fmt.Errorf("%w: plugin '%s' not found", shared.ErrNotFound, name)
	}

	delete(m.plugins, name)
	return nil
}

// Count returns the number of registered plugins
func (m *PluginManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins)
}
