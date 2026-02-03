package mapper

import (
	"testing"

	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromProviderFormat_IncludesID(t *testing.T) {
	data := map[string]interface{}{
		"hostname":    "www",
		"record_type": "A",
		"address":     "1.2.3.4",
		"id":          "abc123",
	}

	mapping := FieldMapping{
		HostName:   "hostname",
		RecordType: "record_type",
		Address:    "address",
		ID:         "id",
	}

	rec, err := FromProviderFormat(data, mapping)
	require.NoError(t, err)
	require.Equal(t, "abc123", rec.ID)
	require.Equal(t, "www", rec.HostName)
}

func TestToProviderFormat_IncludesID(t *testing.T) {
	rec := dnsrecord.Record{
		ID:         "abc123",
		HostName:   "www",
		RecordType: "A",
		Address:    "1.2.3.4",
	}

	mapping := FieldMapping{
		HostName:   "hostname",
		RecordType: "record_type",
		Address:    "address",
		ID:         "id",
	}

	m := ToProviderFormat(rec, mapping)
	require.Equal(t, "abc123", m["id"])
	require.Equal(t, "www", m["hostname"])
}

func TestExtendedFields(t *testing.T) {
	// Test SRV Record
	t.Run("SRV Record", func(t *testing.T) {
		data := map[string]interface{}{
			"name":     "_sip._tcp",
			"type":     "SRV",
			"priority": 10,
			"weight":   5,
			"port":     5060,
			"target":   "sip.example.com",
			"extra":    "metadata",
		}

		mapping := FieldMapping{
			HostName:   "name",
			RecordType: "type",
			Priority:   "priority",
			Weight:     "weight",
			Port:       "port",
			Target:     "target",
		}

		// Test FromProviderFormat
		rec, err := FromProviderFormat(data, mapping)
		require.NoError(t, err)
		assert.Equal(t, "_sip._tcp", rec.HostName)
		assert.Equal(t, "SRV", rec.RecordType)
		assert.Equal(t, 10, rec.Priority)
		assert.Equal(t, 5, rec.Weight)
		assert.Equal(t, 5060, rec.Port)
		assert.Equal(t, "sip.example.com", rec.Target)

		// Metadata check
		assert.NotNil(t, rec.Metadata)
		assert.Equal(t, "metadata", rec.Metadata["extra"])
		assert.Equal(t, 10, rec.Metadata["priority"])

		// Raw check
		assert.Equal(t, data, rec.Raw)

		// Test ToProviderFormat
		out := ToProviderFormat(rec, mapping)
		assert.Equal(t, "_sip._tcp", out["name"])
		assert.Equal(t, "SRV", out["type"])
		assert.Equal(t, 10, out["priority"])
		assert.Equal(t, 5, out["weight"])
		assert.Equal(t, 5060, out["port"])
		assert.Equal(t, "sip.example.com", out["target"])
	})
}

func TestExtractRecords(t *testing.T) {
	t.Run("Nested Path", func(t *testing.T) {
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"records": []interface{}{
					map[string]interface{}{"name": "r1"},
					map[string]interface{}{"name": "r2"},
				},
			},
		}

		recs, err := ExtractRecords(data, "data.records")
		require.NoError(t, err)
		assert.Len(t, recs, 2)
		assert.Equal(t, "r1", recs[0]["name"])
	})

	t.Run("Root Array", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{"name": "r1"},
		}

		recs, err := ExtractRecords(data, "")
		require.NoError(t, err)
		assert.Len(t, recs, 1)
	})

	t.Run("Invalid Path", func(t *testing.T) {
		data := map[string]interface{}{"foo": "bar"}
		_, err := ExtractRecords(data, "baz")
		// Depending on implementation, this might return error or empty list
		// Current implementation returns empty list if path not found?
		// No, implementation says "return nil, fmt.Errorf('path not found')" if map key missing
		// But in mapper.go I changed it to return empty list?
		// Let's check mapper.go logic:
		// if !current.IsValid() { return []map...{}, nil }
		assert.NoError(t, err)
	})
}
