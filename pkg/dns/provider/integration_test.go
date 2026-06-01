package provider_test

import (
	"context"
	"os"
	"testing"
	"time"

	"zonekit/pkg/dns/provider"
	// Import the providers so they can register themselves if needed
	_ "zonekit/pkg/dns/provider/cloudflare"
	_ "zonekit/pkg/dns/provider/namecheap"
)

// TestIntegration runs real API calls against providers.
// It is skipped unless ZONEKIT_INTEGRATION_TEST=1 is set.
func TestIntegration(t *testing.T) {
	if os.Getenv("ZONEKIT_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration tests; set ZONEKIT_INTEGRATION_TEST=1 to run")
	}

	// This is a scaffolding for integration tests.
	// You would typically read credentials from environment variables,
	// instantiate the specific provider adapters, and run CRUD operations.

	providers := []struct {
		name       string
		domain     string
		authEnvKey string
		factory    func() (provider.Provider, error)
	}{
		{
			name:       "cloudflare",
			domain:     os.Getenv("CF_TEST_DOMAIN"),
			authEnvKey: "CF_API_TOKEN",
			factory: func() (provider.Provider, error) {
				// Initialize Cloudflare provider with env vars
				return provider.Get("cloudflare")
			},
		},
		{
			name:       "namecheap",
			domain:     os.Getenv("NC_TEST_DOMAIN"),
			authEnvKey: "NC_API_KEY",
			factory: func() (provider.Provider, error) {
				// Initialize Namecheap provider with env vars
				return provider.Get("namecheap")
			},
		},
		// Add GoDaddy and DigitalOcean here similarly
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			if os.Getenv(p.authEnvKey) == "" || p.domain == "" {
				t.Skipf("Skipping %s integration test; missing %s or test domain", p.name, p.authEnvKey)
			}

			prov, err := p.factory()
			if err != nil {
				t.Fatalf("Failed to initialize provider: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Test Zone Discovery (if supported)
			if prov.Capabilities().SupportsZoneDiscovery {
				zone, err := prov.GetZone(ctx, p.domain)
				if err != nil {
					t.Fatalf("GetZone failed: %v", err)
				}
				if zone == nil || zone.Name != p.domain {
					t.Fatalf("Expected zone %s, got %v", p.domain, zone)
				}
			}

			// Test ListRecords
			// Assuming there is at least an SOA/NS record in the test domain
			// Actually we can't test much without hardcoding, so we just check it doesn't error
			// If we wanted to test full CRUD, we would create a test record and delete it.
			t.Logf("Successfully executed integration test scaffolding for %s", p.name)
		})
	}
}
