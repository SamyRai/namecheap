package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	"zonekit/pkg/dnsrecord"

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

	err := p.deleteRecord(context.Background(), "example.com", dnsrecord.Record{ID: "abc123"})
	require.NoError(t, err)
}

func TestDeleteRecord_MissingID_Error(t *testing.T) {
	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: "http://example.invalid"})
	mappings := mapper.DefaultMappings()
	p := NewRESTProvider("test", client, mappings, map[string]string{"delete_record": "/records/{record_id}"}, nil)

	err := p.deleteRecord(context.Background(), "example.com", dnsrecord.Record{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires record_id")
}
