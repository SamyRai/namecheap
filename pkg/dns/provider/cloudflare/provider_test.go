package cloudflare

import (
	"context"
	"strconv"
	"testing"

	dnsprovider "go.glpx.pro/zonekit/pkg/dns/provider"
	"go.glpx.pro/zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/require"
)

func TestNew_RequiresAPIToken(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
}

func TestValidate_RequiresAccountIDForAccountScope(t *testing.T) {
	p, err := New(Config{APIToken: testAPIToken})
	require.NoError(t, err)
	require.Error(t, p.Validate())

	p2, err := New(Config{APIToken: testAPIToken, TokenScope: TokenScopeUser})
	require.NoError(t, err)
	require.NoError(t, p2.Validate())
}

func TestListZones_Pagination(t *testing.T) {
	fs := newFakeServer(t)
	// Seed more zones than a single page to force the client through
	// more than one GET.
	for i := 0; i < listPerPage+5; i++ {
		fs.addZone(idFor(i), nameFor(i))
	}

	p := newTestProvider(t, fs)
	zones, err := p.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, listPerPage+5)

	require.GreaterOrEqual(t, fs.callCount("GET", "/zones"), 2, "expected pagination to require more than one request")
}

func TestGetZone_AndZoneByName(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")

	p := newTestProvider(t, fs)

	z, err := p.GetZone(context.Background(), "zone-1")
	require.NoError(t, err)
	require.Equal(t, "example.com", z.Name)

	byName, err := p.ZoneByName(context.Background(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "zone-1", byName.ID)

	_, err = p.ZoneByName(context.Background(), "nope.example.com")
	require.Error(t, err)
}

func TestRecordCRUD(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")
	p := newTestProvider(t, fs)
	ctx := context.Background()

	created, err := p.CreateRecord(ctx, "zone-1", dnsrecord.Record{
		HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: "1.2.3.4", TTL: 300,
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "1.2.3.4", created.Address)

	list, err := p.ListRecords(ctx, "zone-1")
	require.NoError(t, err)
	require.Len(t, list, 1)

	updated, err := p.UpdateRecord(ctx, "zone-1", created.ID, dnsrecord.Record{
		HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: "5.6.7.8", TTL: 300,
	})
	require.NoError(t, err)
	require.Equal(t, "5.6.7.8", updated.Address)

	err = p.DeleteRecord(ctx, "zone-1", created.ID)
	require.NoError(t, err)

	list, err = p.ListRecords(ctx, "zone-1")
	require.NoError(t, err)
	require.Empty(t, list)
}

// TestBulkReplaceRecords_DiffBased proves BulkReplaceRecords issues only
// the create/update/delete calls the diff requires, never a delete-all
// followed by a full re-create.
func TestBulkReplaceRecords_DiffBased(t *testing.T) {
	fs := newFakeServer(t)
	fs.addZone("zone-1", "example.com")

	kept := fs.seedRecord("zone-1", dnsRecordWire{Type: "A", Name: testHostWWW, Content: testAddrA1, TTL: 300})
	toChange := fs.seedRecord("zone-1", dnsRecordWire{Type: "A", Name: "app", Content: testAddrA2, TTL: 300})
	toRemove := fs.seedRecord("zone-1", dnsRecordWire{Type: "A", Name: "old", Content: "3.3.3.3", TTL: 300})
	_ = toRemove

	p := newTestProvider(t, fs)
	ctx := context.Background()

	desired := []dnsrecord.Record{
		{HostName: testHostWWW, RecordType: "A", Address: testAddrA1, TTL: 300}, // unchanged
		{HostName: "app", RecordType: "A", Address: testAddrA2, TTL: 600},       // TTL changed -> update
		{HostName: "new", RecordType: "A", Address: "9.9.9.9", TTL: 300},        // new -> create
	}

	err := p.BulkReplaceRecords(ctx, "zone-1", desired)
	require.NoError(t, err)

	require.Equal(t, 1, fs.callCount("DELETE", "/zones/zone-1/dns_records/"+toRemove.ID), "exactly the removed record must be deleted")
	require.Equal(t, 0, fs.callCount("DELETE", "/zones/zone-1/dns_records/"+kept.ID), "an unchanged record must never be deleted")
	require.Equal(t, 0, fs.callCount("DELETE", "/zones/zone-1/dns_records/"+toChange.ID), "a metadata-only change must be an update, not a delete")
	require.Equal(t, 1, fs.callCount("PUT", "/zones/zone-1/dns_records/"+toChange.ID), "the changed record must be updated in place")
	require.Equal(t, 1, fs.callCount("POST", "/zones/zone-1/dns_records"), "only the genuinely new record must be created")

	final, err := p.ListRecords(ctx, "zone-1")
	require.NoError(t, err)
	require.Len(t, final, 3)
}

func TestCapabilities_AdvertisesZoneConfigurer(t *testing.T) {
	p, err := New(Config{APIToken: testAPIToken, TokenScope: TokenScopeUser})
	require.NoError(t, err)

	caps := p.Capabilities()
	require.True(t, caps.CanConfigureZoneSettings)
	require.True(t, caps.CanConfigureBotManagement)
	require.True(t, caps.CanConfigureSecurityTXT)
	require.True(t, caps.CanConfigureRedirectRules)
	require.True(t, caps.CanConfigureDNSSEC)

	_, ok := dnsprovider.Provider(p).(dnsprovider.ZoneConfigurer)
	require.True(t, ok, "a provider advertising zone-configuration capabilities must implement ZoneConfigurer")
}

func idFor(i int) string   { return "zone-" + strconv.Itoa(i) }
func nameFor(i int) string { return strconv.Itoa(i) + ".example.com" }
