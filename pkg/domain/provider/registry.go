package provider

import (
	"fmt"
	"strings"
	"sync"

	"zonekit/pkg/client"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]Factory)
)

// Factory is a function that creates a new Provider instance
type Factory func(client *client.Client) (Provider, error)

// Register registers a new provider factory
func Register(name string, factory Factory) {
	providersMu.Lock()
	defer providersMu.Unlock()
	
	name = strings.ToLower(name)
	if factory == nil {
		panic("provider: Register provider is nil")
	}
	if _, dup := providers[name]; dup {
		panic("provider: Register called twice for provider " + name)
	}
	providers[name] = factory
}

// Get returns a provider instance by name
func Get(name string, client *client.Client) (Provider, error) {
	providersMu.RLock()
	factory, ok := providers[strings.ToLower(name)]
	providersMu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	
	return factory(client)
}

// List returns a list of registered provider names
func List() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	
	var list []string
	for name := range providers {
		list = append(list, name)
	}
	return list
}

// Unregister removes a provider from the registry (primarily for testing)
func Unregister(name string) {
	providersMu.Lock()
	defer providersMu.Unlock()
	
	delete(providers, strings.ToLower(name))
}

// Clear removes all providers from the registry (primarily for testing)
func Clear() {
	providersMu.Lock()
	defer providersMu.Unlock()
	
	providers = make(map[string]Factory)
}
