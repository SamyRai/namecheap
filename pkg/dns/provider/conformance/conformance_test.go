package conformance

import (
	"testing"
	"zonekit/pkg/dns/provider"
)

func TestMockProviderConformance(t *testing.T) {
	mock := NewMockProvider()

    // Seed some data
    mock.AddZone(provider.Zone{
        ID:   "zone-1",
        Name: "example.com",
    })

	factory := func() (provider.Provider, error) {
		return mock, nil
	}

	RunConformanceTests(t, factory)
}
