package plugin

import (
	domainPlugin "github.com/erp/backend/internal/domain/shared/plugin"
)

// Re-export domain plugin types for easier access from infrastructure
type (
	// IndustryPlugin is a re-export of the domain IndustryPlugin interface
	IndustryPlugin = domainPlugin.IndustryPlugin
	// AttributeDefinition is a re-export of the domain AttributeDefinition type
	AttributeDefinition = domainPlugin.AttributeDefinition
	// PluginManager is a re-export of the domain PluginManager type
	PluginManager = domainPlugin.PluginManager
	// StrategyRegistrar is a re-export of the domain StrategyRegistrar interface
	StrategyRegistrar = domainPlugin.StrategyRegistrar
)

// NewPluginManager creates a new plugin manager (re-export from domain)
func NewPluginManager(registry StrategyRegistrar) *PluginManager {
	return domainPlugin.NewPluginManager(registry)
}
