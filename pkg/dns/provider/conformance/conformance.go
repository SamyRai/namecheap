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
	caps := p.Capabilities()

	t.Run("Capabilities", func(t *testing.T) {
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
			testRecordCRUD(t, ctx, p, caps, zone.ID)
		})
	})
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func testRecordCRUD(t *testing.T, ctx context.Context, p provider.Provider, caps provider.ProviderCapabilities, zoneID string) {
	// Create (TXT) with metadata
	newRecord := dnsrecord.Record{
		HostName:   "conformance-test",
		RecordType: dnsrecord.RecordTypeTXT,
		Address:    "test-value",
		TTL:        300,
		Metadata: map[string]interface{}{
			"conformance": "true",
		},
	}

	created, err := p.CreateRecord(ctx, zoneID, newRecord)
	require.NoError(t, err, "CreateRecord failed")
	require.NotNil(t, created, "Created record is nil")

	if caps.SupportsRecordID {
		assert.NotEmpty(t, created.ID, "Created record ID is empty but provider supports IDs")
	}

	assert.Equal(t, newRecord.HostName, created.HostName)
	assert.Equal(t, newRecord.RecordType, created.RecordType)
	assert.Equal(t, newRecord.Address, created.Address)

	// Metadata preservation (best-effort)
	if newRecord.Metadata != nil {
		if created.Metadata != nil {
			assert.Equal(t, "true", created.Metadata["conformance"])
		} else {
			t.Log("Provider did not return metadata for created record; skipping metadata assertion")
		}
	}

	// Read (List)
	records, err := p.ListRecords(ctx, zoneID)
	require.NoError(t, err, "ListRecords failed")
	found := false
	for _, r := range records {
		// Match by ID if supported, otherwise by content
		if caps.SupportsRecordID {
			if r.ID == created.ID {
				found = true
				break
			}
		} else {
			if r.HostName == created.HostName && r.RecordType == created.RecordType && r.Address == created.Address {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Created record not found in ListRecords")

	// SRV specific test
	if containsString(caps.SupportedRecordTypes, dnsrecord.RecordTypeSRV) {
		srv := dnsrecord.Record{
			HostName:   "_sip._tcp",
			RecordType: dnsrecord.RecordTypeSRV,
			Priority:   10,
			Weight:     5,
			Port:       5060,
			Target:     "sip.example.com",
			TTL:        300,
		}

		createdSRV, err := p.CreateRecord(ctx, zoneID, srv)
		require.NoError(t, err, "CreateRecord (SRV) failed")
		if createdSRV != nil {
			assert.Equal(t, srv.HostName, createdSRV.HostName)
			assert.Equal(t, srv.RecordType, createdSRV.RecordType)
			assert.Equal(t, srv.Priority, createdSRV.Priority)
			assert.Equal(t, srv.Weight, createdSRV.Weight)
			assert.Equal(t, srv.Port, createdSRV.Port)
			assert.Equal(t, srv.Target, createdSRV.Target)
		}
	}

	if caps.SupportsRecordID {
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
	} else {
		t.Log("Skipping Update/Delete by ID as provider does not support it")

		// Test BulkReplace if supported
		if caps.SupportsBulkReplace {
			// Try to remove all records in the zone via BulkReplaceRecords
			if err := p.BulkReplaceRecords(ctx, zoneID, []dnsrecord.Record{}); err != nil {
				t.Fatalf("BulkReplaceRecords failed: %v", err)
			}

			recs, err := p.ListRecords(ctx, zoneID)
			require.NoError(t, err)
			assert.Len(t, recs, 0, "Expected zero records after BulkReplaceRecords([])")
		}
	}
}
