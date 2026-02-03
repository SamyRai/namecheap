package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"

	"github.com/stretchr/testify/require"
)

func TestResolveZoneID_FromSettings(t *testing.T) {
	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: "http://example.invalid"})
	p := NewRESTProvider("test", client, mapper.DefaultMappings(), map[string]string{}, map[string]interface{}{"zone_id": "z-123"})

	id, err := p.resolveZoneID(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "z-123", id)
}

func TestResolveZoneID_FromListEndpoint(t *testing.T) {
	// Test server returns zone list
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[{"id":"z-1","name":"example.com"}]}`))
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	// Default mappings assume "records", we need "result" for list path?
	// But ListZones uses "ListPath" mapping which defaults to "records".
	// The test data uses "result".
	mappings := mapper.DefaultMappings()
	mappings.ListPath = "result"

	p := NewRESTProvider("test", client, mappings, map[string]string{"list_zones": "/zones"}, nil)

	id, err := p.resolveZoneID(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "z-1", id)
}

func TestResolveZoneID_NoMatch_ReturnsError(t *testing.T) {
	// Test server returns unrelated zone
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":[{"id":"z-1","name":"other.com"}]}`))
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	mappings := mapper.DefaultMappings()
	mappings.ListPath = "result"

	p := NewRESTProvider("test", client, mappings, map[string]string{"list_zones": "/zones"}, nil)

	_, err := p.resolveZoneID(context.Background(), "example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve zone ID")
}
