package cloudflare

import (
	"context"
	"strings"
	"testing"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

const testZoneID = "0123456789abcdef0123456789abcdef"

func TestByDomainResolvesNameOnceAndPassesIDsThrough(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone(testZoneID, "example.com")
	d := ByDomain(newTestProvider(t, fs))
	ctx := context.Background()

	if _, err := d.CreateRecord(ctx, "example.com", dnsrecord.Record{HostName: "www", RecordType: "A", Address: "192.0.2.10", TTL: 300}); err != nil {
		t.Fatalf("CreateRecord by name: %v", err)
	}
	recs, err := d.ListRecords(ctx, "example.com")
	if err != nil || len(recs) != 1 {
		t.Fatalf("ListRecords by name = %d records, err %v", len(recs), err)
	}
	if _, err := d.ListRecords(ctx, testZoneID); err != nil {
		t.Fatalf("ListRecords by id: %v", err)
	}

	lookups := 0
	for _, c := range fs.calls {
		if strings.HasPrefix(c, "GET /zones") && !strings.Contains(strings.TrimPrefix(c, "GET /zones"), "/") {
			lookups++
		}
	}
	if lookups != 1 {
		t.Fatalf("zone name lookups = %d, want 1 (cached); calls: %v", lookups, fs.calls)
	}
}

func TestByDomainUnknownZoneFails(t *testing.T) {
	fs := newFakeServer(t)
	d := ByDomain(newTestProvider(t, fs))
	if _, err := d.ListRecords(context.Background(), "missing.example"); err == nil {
		t.Fatal("expected an error for an unknown zone name")
	}
}
