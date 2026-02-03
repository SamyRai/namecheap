package conformance

import (
	"context"
	"testing"
	"zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProviderFactory creates a new provider instance for testing
type ProviderFactory func() (provider.Provider, error)

// RunConformanceTests runs standard tests against a provider implementation
func RunConformanceTests(t *testing.T, factory ProviderFactory) {
	p, err := factory()
	require.NoError(t, err, "Failed to create provider")

	ctx := context.Background()

	t.Run("Capabilities", func(t *testing.T) {
		caps := p.Capabilities()
		t.Logf("Provider %s capabilities: %+v", p.Name(), caps)
		assert.NotEmpty(t, caps.SupportedRecordTypes)
	})

	t.Run("ZoneOperations", func(t *testing.T) {
		zones, err := p.ListZones(ctx)
		require.NoError(t, err, "ListZones failed")

		if len(zones) == 0 {
			t.Log("No zones available to test CRUD operations")
			return
		}

		zone := zones[0]
		t.Logf("Testing against zone: %s (%s)", zone.Name, zone.ID)

		fetchedZone, err := p.GetZone(ctx, zone.Name)
		require.NoError(t, err, "GetZone failed")
		assert.Equal(t, zone.ID, fetchedZone.ID)
		assert.Equal(t, zone.Name, fetchedZone.Name)

		// Record CRUD Tests
		t.Run("RecordCRUD", func(t *testing.T) {
			testRecordCRUD(t, ctx, p, zone.ID)
		})
	})
}

func testRecordCRUD(t *testing.T, ctx context.Context, p provider.Provider, zoneID string) {
	// Create
	newRecord := dnsrecord.Record{
		HostName:   "conformance-test",
		RecordType: dnsrecord.RecordTypeTXT,
		Address:    "test-value",
		TTL:        300,
	}

	created, err := p.CreateRecord(ctx, zoneID, newRecord)
	require.NoError(t, err, "CreateRecord failed")
	require.NotNil(t, created, "Created record is nil")

	// If the provider supports IDs, it must return one.
	// If not, we still need something to reference it by for Update/Delete?
	// The interface requires recordID for Update/Delete.
	assert.NotEmpty(t, created.ID, "Created record ID is empty")

	assert.Equal(t, newRecord.HostName, created.HostName)
	assert.Equal(t, newRecord.RecordType, created.RecordType)
	assert.Equal(t, newRecord.Address, created.Address)

	// Read (List)
	records, err := p.ListRecords(ctx, zoneID)
	require.NoError(t, err, "ListRecords failed")
	found := false
	for _, r := range records {
		if r.ID == created.ID {
			found = true
			assert.Equal(t, created.Address, r.Address)
			break
		}
	}
	assert.True(t, found, "Created record not found in ListRecords")

	// Update
	created.Address = "updated-value"
	updated, err := p.UpdateRecord(ctx, zoneID, created.ID, *created)
	require.NoError(t, err, "UpdateRecord failed")
	assert.Equal(t, "updated-value", updated.Address)

	// Delete
	err = p.DeleteRecord(ctx, zoneID, created.ID)
	require.NoError(t, err, "DeleteRecord failed")

	// Verify Delete
	records, err = p.ListRecords(ctx, zoneID)
	require.NoError(t, err, "ListRecords failed")
	for _, r := range records {
		if r.ID == created.ID {
			t.Errorf("Record %s still exists after deletion", created.ID)
		}
	}
}
