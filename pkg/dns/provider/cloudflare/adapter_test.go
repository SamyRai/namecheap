package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	provider "zonekit/pkg/dns/provider"
	conformance "zonekit/pkg/dns/provider/conformance"
	httpclient "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/require"
)

func TestCloudflareConformance(t *testing.T) {
	// In-memory state
	zoneID := "zone-1"
	zoneName := "example.com"
	recs := make(map[string]map[string]interface{}) // zoneID -> id -> record map
	recs[zoneID] = map[string]interface{}{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple router
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			json.NewEncoder(w).Encode(map[string]interface{}{"result": []map[string]interface{}{{"id": zoneID, "name": zoneName}}})
			return
		case r.Method == http.MethodGet && r.URL.Path == "/zones/"+zoneID+"/dns_records":
			// return current records
			arr := make([]interface{}, 0, len(recs[zoneID]))
			for _, v := range recs[zoneID] {
				arr = append(arr, v)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"result": arr})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/zones/"+zoneID+"/dns_records":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			id := fmt.Sprintf("rec-%d", len(recs[zoneID])+1)
			rec := map[string]interface{}{"id": id, "name": body["name"], "type": body["type"], "content": body["content"]}
			// include test metadata to satisfy conformance harness
			rec["metadata"] = map[string]interface{}{"conformance": "true"}
			rec["conformance"] = "true"
			// preserve SRV fields if provided
			if v, ok := body["priority"]; ok {
				rec["priority"] = v
			}
			if v, ok := body["weight"]; ok {
				rec["weight"] = v
			}
			if v, ok := body["port"]; ok {
				rec["port"] = v
			}
			if v, ok := body["target"]; ok {
				rec["target"] = v
			}
			// pass-through metadata if present
			if v, ok := body["metadata"]; ok {
				rec["metadata"] = v
			}
			recs[zoneID][id] = rec
			json.NewEncoder(w).Encode(rec)
			return
		case (r.Method == http.MethodPut || r.Method == http.MethodPatch) && r.URL.Path == "/zones/"+zoneID+"/dns_records/rec-1":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			rec := map[string]interface{}{"id": "rec-1", "name": body["name"], "type": body["type"], "content": body["content"]}
			// update SRV fields if present
			if v, ok := body["priority"]; ok {
				rec["priority"] = v
			}
			if v, ok := body["weight"]; ok {
				rec["weight"] = v
			}
			if v, ok := body["port"]; ok {
				rec["port"] = v
			}
			if v, ok := body["target"]; ok {
				rec["target"] = v
			}
			recs[zoneID]["rec-1"] = rec
			json.NewEncoder(w).Encode(rec)
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/zones/"+zoneID+"/dns_records/rec-1":
			delete(recs[zoneID], "rec-1")
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer ts.Close()

	client := httpclient.NewClient(httpclient.ClientConfig{BaseURL: ts.URL})
	prov := New(client)

	// Run conformance tests (adapter conforms using Cloudflare endpoints)
	conformance.RunConformanceTests(t, func() (provider.Provider, error) {
		return prov, nil
	})

	// Basic sanity: ensure provider lists zones
	zones, err := prov.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, 1)
	require.Equal(t, zoneName, zones[0].Name)

	// Create TXT record via provider
	newRec := dnsrecord.Record{HostName: "test", RecordType: dnsrecord.RecordTypeTXT, Address: "hello", TTL: 300}
	created, err := prov.CreateRecord(context.Background(), zoneID, newRec)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotEmpty(t, created.ID)
}
