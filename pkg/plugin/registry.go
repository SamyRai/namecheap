package plugin

import (
	"fmt"
	"sync"
)

var (
	registry     = make(map[string]Plugin)
	registryLock sync.RWMutex
)

// Register registers a plugin with the registry
func Register(plugin Plugin) error {
	registryLock.Lock()
	defer registryLock.Unlock()

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := registry[name]; exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	registry[name] = plugin
	return nil
}

// Get retrieves a plugin by name
func Get(name string) (Plugin, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	plugin, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// List returns all registered plugins
func List() []Plugin {
	registryLock.RLock()
	defer registryLock.RUnlock()

	plugins := make([]Plugin, 0, len(registry))
	for _, plugin := range registry {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// Names returns the names of all registered plugins
func Names() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}

	return names
}

// Unregister removes a plugin from the registry (mainly for testing)
func Unregister(name string) {
	registryLock.Lock()
	defer registryLock.Unlock()

	delete(registry, name)
}

// Clear removes all plugins from the registry (mainly for testing)
func Clear() {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry = make(map[string]Plugin)
}
