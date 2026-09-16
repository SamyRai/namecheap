package conformance

import (
	"testing"

	"zonekit/pkg/dns/provider"
)

func TestMockProviderConformance(t *testing.T) {
	p := NewMockProvider()
	p.Zones["zone-1"] = provider.Zone{ID: "zone-1", Name: "example.com"}

	RunConformanceTests(t, p)
}
