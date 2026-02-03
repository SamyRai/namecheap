package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoDaddySpec_ToProviderConfig(t *testing.T) {
	specPath := "../godaddy/openapi.yaml"
	spec, err := LoadSpec(specPath)
	require.NoError(t, err)

	cfg, err := spec.ToProviderConfig("godaddy")
	require.NoError(t, err)
	require.Equal(t, "https://api.godaddy.com/v1", cfg.API.BaseURL)

	// Endpoints
	require.Contains(t, cfg.API.Endpoints, "get_records")
	require.Contains(t, cfg.API.Endpoints, "update_record")
	require.Contains(t, cfg.API.Endpoints, "delete_record")

	// Mappings
	require.NotNil(t, cfg.Mappings)
	require.Equal(t, "name", cfg.Mappings.Response.HostName)
	require.Equal(t, "data", cfg.Mappings.Response.Address)
	require.Equal(t, "priority", cfg.Mappings.Response.Priority)
}

func TestDigitalOceanSpec_ToProviderConfig(t *testing.T) {
	specPath := "../digitalocean/openapi.yaml"
	spec, err := LoadSpec(specPath)
	require.NoError(t, err)

	cfg, err := spec.ToProviderConfig("digitalocean")
	require.NoError(t, err)
	require.Equal(t, "https://api.digitalocean.com/v2", cfg.API.BaseURL)

	// Endpoints
	require.Contains(t, cfg.API.Endpoints, "get_records")
	require.Contains(t, cfg.API.Endpoints, "create_record")
	require.Contains(t, cfg.API.Endpoints, "update_record")
	require.Contains(t, cfg.API.Endpoints, "delete_record")

	// Mappings
	require.NotNil(t, cfg.Mappings)
	require.Equal(t, "id", cfg.Mappings.Response.ID)
	require.Equal(t, "name", cfg.Mappings.Response.HostName)
	require.Equal(t, "data", cfg.Mappings.Response.Address)
	require.Equal(t, "priority", cfg.Mappings.Response.Priority)
	require.Equal(t, "port", cfg.Mappings.Response.Port)
	require.Equal(t, "weight", cfg.Mappings.Response.Weight)

	// List Path
	require.Equal(t, "domain_records", cfg.Mappings.ListPath)
}
