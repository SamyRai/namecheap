package provider

import (
	"fmt"
	"sync"
)

var (
	registry     = make(map[string]Provider)
	registryLock sync.RWMutex
)

// Register registers a DNS provider
func Register(provider Provider) error {
	registryLock.Lock()
	defer registryLock.Unlock()

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := registry[name]; exists {
		return fmt.Errorf("provider %s is already registered", name)
	}

	registry[name] = provider
	return nil
}

// Get retrieves a provider by name
func Get(name string) (Provider, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	provider, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("DNS provider %s not found", name)
	}

	return provider, nil
}

// List returns all registered providers
func List() []Provider {
	registryLock.RLock()
	defer registryLock.RUnlock()

	providers := make([]Provider, 0, len(registry))
	for _, provider := range registry {
		providers = append(providers, provider)
	}

	return providers
}

// Names returns the names of all registered providers
func Names() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}

	return names
}

// Unregister removes a provider (mainly for testing)
func Unregister(name string) {
	registryLock.Lock()
	defer registryLock.Unlock()

	delete(registry, name)
}

// Clear removes all providers (mainly for testing)
func Clear() {
	registryLock.Lock()
	defer registryLock.Unlock()

	registry = make(map[string]Provider)
}
