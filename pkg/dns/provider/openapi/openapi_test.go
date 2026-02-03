package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloudflareSpec_ToProviderConfig(t *testing.T) {
	specPath := "../cloudflare/openapi.yaml"
	spec, err := LoadSpec(specPath)
	require.NoError(t, err)

	cfg, err := spec.ToProviderConfig("cloudflare")
	require.NoError(t, err)
	require.Equal(t, "https://api.cloudflare.com/client/v4", cfg.API.BaseURL)

	// Endpoints
	require.Contains(t, cfg.API.Endpoints, "get_records")
	t.Logf("endpoints: %+v", cfg.API.Endpoints)
	require.Equal(t, "/zones/{zone_id}/dns_records", cfg.API.Endpoints["get_records"])

	require.Contains(t, cfg.API.Endpoints, "delete_record")
	require.Equal(t, "/zones/{zone_id}/dns_records/{dns_record_id}", cfg.API.Endpoints["delete_record"])

	// Zone endpoints
	require.Contains(t, cfg.API.Endpoints, "list_zones")
	require.Equal(t, "/zones", cfg.API.Endpoints["list_zones"])

	// Mappings
	require.NotNil(t, cfg.Mappings)
	require.Equal(t, "id", cfg.Mappings.Response.ID)

	// List path should be detected
	require.Equal(t, "result", cfg.Mappings.ListPath)

	// Zone mappings
	require.Equal(t, "id", cfg.Mappings.ZoneID)
	require.Equal(t, "name", cfg.Mappings.ZoneName)
	require.Equal(t, "result", cfg.Mappings.ZoneListPath)
}
