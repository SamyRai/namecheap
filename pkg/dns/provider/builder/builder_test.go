package builder

import (
	"testing"

	"zonekit/pkg/dns/provider/openapi"

	"github.com/stretchr/testify/require"
)

func TestBuildProvider_FromCloudflareSpec(t *testing.T) {
	specPath := "../cloudflare/openapi.yaml"
	spec, err := openapi.LoadSpec(specPath)
	require.NoError(t, err)

	cfg, err := spec.ToProviderConfig("cloudflare")
	require.NoError(t, err)

	prov, err := BuildProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, prov)

	// Validate provider
	require.NoError(t, prov.Validate())
}
