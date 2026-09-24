package cloudflare

import (
	"testing"

	"go.glpx.pro/zonekit/pkg/dns/provider/conformance"
)

// TestConformance runs zonekit's shared provider conformance suite
// against the typed Cloudflare provider backed by the httptest fake, so
// the same checks applied to the mock provider also cover Cloudflare.
func TestConformance(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")

	p := newTestProvider(t, fs)
	conformance.RunConformanceTests(t, p)
}
