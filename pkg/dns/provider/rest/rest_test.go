package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteRecord_ByID_Success(t *testing.T) {
	// Start test server expecting DELETE /records/abc123
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/records/abc123" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	mappings := mapper.DefaultMappings()
	p := NewRESTProvider("test", client, mappings, map[string]string{"delete_record": "/records/{record_id}"}, nil)

	err := p.DeleteRecord(context.Background(), "example.com", "abc123")
	require.NoError(t, err)
}

func TestDeleteRecord_MissingID_Error(t *testing.T) {
	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: "http://example.invalid"})
	mappings := mapper.DefaultMappings()
	p := NewRESTProvider("test", client, mappings, map[string]string{"delete_record": "/records/{record_id}"}, nil)

	err := p.DeleteRecord(context.Background(), "example.com", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "record ID is required")
}

func TestCreateRecord_ReturnsRecord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/records" {
			w.Header().Set("Content-Type", "application/json")
			// Return created record
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "new-id-123",
				"hostname":    "test",
				"record_type": "A",
				"address":     "1.2.3.4",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	mappings := mapper.DefaultMappings()
	p := NewRESTProvider("test", client, mappings, map[string]string{"create_record": "/records"}, nil)

	rec := dnsrecord.Record{
		HostName:   "test",
		RecordType: "A",
		Address:    "1.2.3.4",
	}

	created, err := p.CreateRecord(context.Background(), "example.com", rec)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "new-id-123", created.ID)
	assert.Equal(t, "test", created.HostName)
}

func TestListZones(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": []interface{}{
					map[string]interface{}{"id": "zone-1", "name": "example.com"},
					map[string]interface{}{"id": "zone-2", "name": "test.com"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	mappings := mapper.DefaultMappings()
	mappings.ListPath = "result" // Set list path

	p := NewRESTProvider("test", client, mappings, map[string]string{"list_zones": "/zones"}, nil)

	zones, err := p.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, 2)
	assert.Equal(t, "zone-1", zones[0].ID)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestUpdateRecord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/records/rec-123" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          "rec-123",
				"hostname":    "updated",
				"record_type": "A",
				"address":     "5.6.7.8",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	mappings := mapper.DefaultMappings()
	p := NewRESTProvider("test", client, mappings, map[string]string{"update_record": "/records/{id}"}, nil)

	rec := dnsrecord.Record{
		HostName:   "updated",
		RecordType: "A",
		Address:    "5.6.7.8",
	}

	updated, err := p.UpdateRecord(context.Background(), "example.com", "rec-123", rec)
	require.NoError(t, err)
	assert.Equal(t, "rec-123", updated.ID)
	assert.Equal(t, "updated", updated.HostName)
}

func TestCapabilities(t *testing.T) {
	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: "http://example.com"})
	mappings := mapper.DefaultMappings()
	mappings.Response.ID = "id_field" // Supports ID

	p := NewRESTProvider("test", client, mappings, map[string]string{"list_zones": "/zones"}, nil)

	caps := p.Capabilities()
	assert.True(t, caps.SupportsRecordID)
	assert.True(t, caps.SupportsZoneDiscovery)
}
