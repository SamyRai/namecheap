package godaddy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	provider "zonekit/pkg/dns/provider"
	conformance "zonekit/pkg/dns/provider/conformance"
	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/require"
)

func TestGoDaddyConformance(t *testing.T) {
	// In-memory state
	zoneName := "example.com"
	recs := make(map[string][]map[string]interface{}) // zoneName -> list of records
	recs[zoneName] = []map[string]interface{}{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/domains":
			json.NewEncoder(w).Encode([]map[string]interface{}{{"domainId": 123, "domain": zoneName, "status": "ACTIVE"}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/domains/"+zoneName:
			json.NewEncoder(w).Encode(map[string]interface{}{"domainId": 123, "domain": zoneName, "status": "ACTIVE"})
			return
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/domains/"+zoneName+"/records"):
			json.NewEncoder(w).Encode(recs[zoneName])
			return
		case r.Method == http.MethodPost && r.URL.Path == "/domains/"+zoneName+"/records":
			// GoDaddy adds records (expects array)
			var body []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			// Append to existing
			recs[zoneName] = append(recs[zoneName], body...)
			w.WriteHeader(http.StatusOK)
			return
		case r.Method == http.MethodPut && r.URL.Path == "/domains/"+zoneName+"/records":
			// Bulk replace (expects array)
			var body []map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			recs[zoneName] = body
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	prov := New(client)

	// Run conformance tests
	conformance.RunConformanceTests(t, func() (provider.Provider, error) {
		// Reset state
		recs[zoneName] = []map[string]interface{}{}
		return prov, nil
	})

	// Manual test for CreateRecord input wrapping
	newRec := dnsrecord.Record{HostName: "manual", RecordType: "A", Address: "1.2.3.4", TTL: 600}
	_, err := prov.CreateRecord(context.Background(), zoneName, newRec)
	require.NoError(t, err)

	// Verify it was added (RunConformanceTests might have left some records, but manual should be there)
	found := false
	for _, r := range recs[zoneName] {
		if r["name"] == "manual" {
			found = true
			break
		}
	}
	require.True(t, found, "Manual record not found")
}
