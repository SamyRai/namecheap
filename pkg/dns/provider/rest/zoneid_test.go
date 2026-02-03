package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"

	"github.com/stretchr/testify/require"
)

func TestGetZoneID_FromSettings(t *testing.T) {
	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: "http://example.invalid"})
	p := NewRESTProvider("test", client, mapper.DefaultMappings(), map[string]string{}, map[string]interface{}{"zone_id": "z-123"})

	id, err := p.getZoneID("example.com")
	require.NoError(t, err)
	require.Equal(t, "z-123", id)
}

func TestGetZoneID_FromListEndpoint(t *testing.T) {
	// Test server returns zone list
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[{"id":"z-1","name":"example.com"}]}`))
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	p := NewRESTProvider("test", client, mapper.DefaultMappings(), map[string]string{"list_zones": "/zones"}, nil)

	id, err := p.getZoneID("example.com")
	require.NoError(t, err)
	require.Equal(t, "z-1", id)
}

func TestGetZoneID_NoMatch_ReturnsEmpty(t *testing.T) {
	// Test server returns unrelated zone
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[{"id":"z-1","name":"other.com"}]}`))
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	p := NewRESTProvider("test", client, mapper.DefaultMappings(), map[string]string{"list_zones": "/zones"}, nil)

	id, err := p.getZoneID("example.com")
	require.NoError(t, err)
	require.Equal(t, "", id)
}
