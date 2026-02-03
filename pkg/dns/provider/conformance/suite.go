package conformance

import (
	"context"
	"testing"

	"zonekit/pkg/dns/provider"

	"github.com/stretchr/testify/require"
)

// RunConformanceTests runs a set of tests to verify provider compliance
func RunConformanceTests(t *testing.T, p provider.Provider) {
	ctx := context.Background()

	t.Run("Capabilities", func(t *testing.T) {
		caps := p.Capabilities()
		t.Logf("Provider %s capabilities: %+v", p.Name(), caps)
	})

	t.Run("ZoneOperations", func(t *testing.T) {
		if !p.Capabilities().CanListZones {
			t.Skip("Provider does not support listing zones")
		}

		zones, err := p.ListZones(ctx)
		require.NoError(t, err)

		if len(zones) > 0 && p.Capabilities().CanGetZone {
			zone, err := p.GetZone(ctx, zones[0].ID)
			require.NoError(t, err)
			require.Equal(t, zones[0].ID, zone.ID)
		}
	})
}
