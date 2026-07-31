package namecheap

import (
	"context"
	"testing"

	"zonekit/pkg/dnsrecord"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// hostsResponse builds a getHosts response with the given EmailType and hosts.
func hostsResponse(emailType string, hosts ...namecheap.DomainsDNSHostRecordDetailed) *namecheap.DomainsDNSGetHostsCommandResponse {
	return &namecheap.DomainsDNSGetHostsCommandResponse{
		DomainDNSGetHostsResult: &namecheap.DomainDNSGetHostsResult{
			EmailType: namecheap.String(emailType),
			Hosts:     &hosts,
		},
	}
}

// encodeAddress still renders the canonical SRV wire form. Namecheap cannot
// accept it today, but the helper is shared with the read path (decode) and is
// what a provider capable of SRV would emit.
func TestSRVEncodedIntoAddress(t *testing.T) {
	got := encodeAddress(dnsrecord.Record{
		HostName:   "_submissions._tcp",
		RecordType: dnsrecord.RecordTypeSRV,
		Target:     "smtp.migadu.com.",
		Port:       465,
		Priority:   0,
		Weight:     1,
	})
	assert.Equal(t, "0 1 465 smtp.migadu.com.", got,
		"SRV must render as 'priority weight port target'")
}

// Reading it back must reconstruct the structured fields, so a get/set cycle
// is lossless rather than degrading SRV into an opaque string.
func TestSRVRoundTrip(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	mockClient.On("DomainsDNSGetHosts", "example.com").Return(hostsResponse("MX",
		namecheap.DomainsDNSHostRecordDetailed{
			Name:    namecheap.String("_imaps._tcp"),
			Type:    namecheap.String("SRV"),
			Address: namecheap.String("0 1 993 imap.migadu.com."),
			TTL:     namecheap.Int(1800),
		},
	), nil)

	records, err := p.ListRecords(context.Background(), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)

	got := records[0]
	assert.Equal(t, 0, got.Priority)
	assert.Equal(t, 1, got.Weight)
	assert.Equal(t, 993, got.Port)
	assert.Equal(t, "imap.migadu.com.", got.Target)

	// Re-encoding the decoded record reproduces the original wire value.
	assert.Equal(t, "0 1 993 imap.migadu.com.", encodeAddress(got))
}

// A malformed or hand-entered SRV value must pass through untouched rather
// than being silently mangled into zeros.
func TestSRVMalformedAddressLeftAlone(t *testing.T) {
	record := dnsrecord.Record{
		HostName:   "_sip._tcp",
		RecordType: dnsrecord.RecordTypeSRV,
		Address:    "sip.example.com",
	}
	decodeSRVAddress(&record)
	assert.Equal(t, "sip.example.com", record.Address)
	assert.Zero(t, record.Port)
	assert.Empty(t, record.Target)

	// With no Target set, encodeAddress preserves whatever the caller supplied.
	assert.Equal(t, "sip.example.com", encodeAddress(record))
}

// EmailType is zone-level state that setHosts overwrites. A rewrite that
// introduces no MX records must not silently disable the zone's existing mail
// routing -- tercul.com sits on FWD (Namecheap email forwarding) today.
func TestEmailTypePreservedWhenNoMXRecords(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	var captured *namecheap.DomainsDNSSetHostsArgs
	mockClient.On("DomainsDNSSetHosts", mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(0).(*namecheap.DomainsDNSSetHostsArgs)
		}).
		Return(&namecheap.DomainsDNSSetHostsCommandResponse{}, nil)
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(hostsResponse("FWD"), nil)

	err := p.BulkReplaceRecords(context.Background(), "example.com", []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeA, Address: "203.0.113.1", TTL: 1800},
	})
	require.NoError(t, err)
	require.NotNil(t, captured.EmailType)
	assert.Equal(t, namecheap.EmailTypeForward, *captured.EmailType,
		"a zone rewrite without MX must not drop existing email forwarding")
}

// Introducing MX records is an explicit switch to MX routing.
func TestEmailTypeSetToMXWhenMXPresent(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	var captured *namecheap.DomainsDNSSetHostsArgs
	mockClient.On("DomainsDNSSetHosts", mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(0).(*namecheap.DomainsDNSSetHostsArgs)
		}).
		Return(&namecheap.DomainsDNSSetHostsCommandResponse{}, nil)
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(hostsResponse("FWD"), nil)

	err := p.BulkReplaceRecords(context.Background(), "example.com", []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeMX, Address: "aspmx1.migadu.com.", MXPref: 10},
	})
	require.NoError(t, err)
	require.NotNil(t, captured.EmailType)
	assert.Equal(t, namecheap.EmailTypeMX, *captured.EmailType)

	records := *captured.Records
	require.Len(t, records, 1)
	require.NotNil(t, records[0].MXPref)
	assert.Equal(t, uint8(10), *records[0].MXPref)
}

// MX priority may arrive in the contract-v2 Priority field rather than MXPref.
func TestMXPriorityFallsBackToPriorityField(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	var captured *namecheap.DomainsDNSSetHostsArgs
	mockClient.On("DomainsDNSSetHosts", mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(0).(*namecheap.DomainsDNSSetHostsArgs)
		}).
		Return(&namecheap.DomainsDNSSetHostsCommandResponse{}, nil)
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(hostsResponse("MX"), nil)

	err := p.BulkReplaceRecords(context.Background(), "example.com", []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeMX, Address: "aspmx2.migadu.com.", Priority: 20},
	})
	require.NoError(t, err)

	records := *captured.Records
	require.NotNil(t, records[0].MXPref)
	assert.Equal(t, uint8(20), *records[0].MXPref)
}

// CAA carries its full "flags tag value" payload in Address, so it must round
// trip verbatim.
func TestCAAPassesThroughVerbatim(t *testing.T) {
	const caa = `0 issue "letsencrypt.org"`
	record := dnsrecord.Record{
		HostName:   "@",
		RecordType: dnsrecord.RecordTypeCAA,
		Address:    caa,
	}
	assert.Equal(t, caa, encodeAddress(record))

	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}
	mockClient.On("DomainsDNSGetHosts", "example.com").Return(hostsResponse("NONE",
		namecheap.DomainsDNSHostRecordDetailed{
			Name:    namecheap.String("@"),
			Type:    namecheap.String("CAA"),
			Address: namecheap.String(caa),
			TTL:     namecheap.Int(1800),
		},
	), nil)

	records, err := p.ListRecords(context.Background(), "example.com")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, caa, records[0].Address)
}

// Namecheap's API cannot create SRV records: its DNS platform supports them,
// but setHosts rejects the type (AllowedRecordTypeValues omits SRV in every
// SDK version, including v2.7.2). The adapter must say so up front instead of
// letting the SDK fail a whole-zone write with an opaque message.
func TestSRVRejectedWithActionableError(t *testing.T) {
	mockClient := new(MockNamecheapClient)
	p := &NamecheapProvider{client: mockClient}

	err := p.BulkReplaceRecords(context.Background(), "example.com", []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeA, Address: "203.0.113.1"},
		{HostName: "_imaps._tcp", RecordType: dnsrecord.RecordTypeSRV, Target: "imap.migadu.com.", Port: 993},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SRV")
	assert.Contains(t, err.Error(), "control panel",
		"the error should tell the operator where SRV records can actually be created")

	// Nothing may be written when the record set cannot be represented.
	mockClient.AssertNotCalled(t, "DomainsDNSSetHosts", mock.Anything)
}

// The advertised capability set must not claim SRV.
func TestCapabilitiesDoNotClaimSRV(t *testing.T) {
	p := &NamecheapProvider{}
	assert.NotContains(t, p.Capabilities().SupportedRecordTypes, "SRV")
	assert.Contains(t, p.Capabilities().SupportedRecordTypes, "CAA")
}
