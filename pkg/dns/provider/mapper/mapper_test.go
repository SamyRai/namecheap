package mapper

import (
	"testing"

	"zonekit/pkg/dnsrecord"

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
