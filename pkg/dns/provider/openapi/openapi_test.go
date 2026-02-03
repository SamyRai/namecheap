package openapi

import (
	"os"
	"path/filepath"
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

	// Mappings
	require.NotNil(t, cfg.Mappings)
	require.Equal(t, "id", cfg.Mappings.Response.ID)
	// Cloudflare spec has priority
	require.Equal(t, "priority", cfg.Mappings.Response.Priority)

	// List path should be detected
	require.Equal(t, "result", cfg.Mappings.ListPath)
}

func TestSRVFieldDetection(t *testing.T) {
	// Create a temporary spec file with SRV fields
	content := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /records:
    get:
      operationId: listRecords
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ListResponse'
components:
  schemas:
    ListResponse:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/Record'
    Record:
      type: object
      properties:
        name:
          type: string
        type:
          type: string
        data:
          type: string
        priority:
          type: integer
        weight:
          type: integer
        port:
          type: integer
        target:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "openapi.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	spec, err := LoadSpec(tmpFile)
	require.NoError(t, err)

	cfg, err := spec.ToProviderConfig("test")
	require.NoError(t, err)

	// Check mappings
	require.NotNil(t, cfg.Mappings)
	require.Equal(t, "name", cfg.Mappings.Response.HostName)
	require.Equal(t, "type", cfg.Mappings.Response.RecordType)
	require.Equal(t, "data", cfg.Mappings.Response.Address)
	require.Equal(t, "priority", cfg.Mappings.Response.Priority)
	require.Equal(t, "weight", cfg.Mappings.Response.Weight)
	require.Equal(t, "port", cfg.Mappings.Response.Port)
	require.Equal(t, "target", cfg.Mappings.Response.Target)

	// List path detection
	require.Equal(t, "items", cfg.Mappings.ListPath)
}
