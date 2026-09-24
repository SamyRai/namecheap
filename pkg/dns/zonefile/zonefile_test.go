package zonefile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"go.glpx.pro/zonekit/pkg/dnsrecord"
)

const (
	testHostnameWWW = "www"
	testSPFValue    = "v=spf1 ip4:134.255.231.180 -all"
)

// ZoneFileTestSuite covers Format/Parse and their round-trip.
type ZoneFileTestSuite struct {
	suite.Suite
}

func TestZoneFileSuite(t *testing.T) {
	suite.Run(t, new(ZoneFileTestSuite))
}

// roundTrip formats records, parses the result back, and returns the
// records Parse produced (or fails the test if either step errors).
func (s *ZoneFileTestSuite) roundTrip(domain string, records []dnsrecord.Record) []dnsrecord.Record {
	content := Format(domain, records)
	parsed, err := Parse(content)
	s.Require().NoError(err, "zone file:\n%s", content)
	return parsed
}

// TestApexRecordsAfterWWWKeepExplicitOwner is the regression test for B1:
// the reviewed export dropped the owner on records following a "www" line,
// which under BIND's owner-inheritance rule silently relocated the apex
// MX/TXT records onto "www". Every record here must round-trip with its
// own HostName intact, in particular the apex ones after "www".
func (s *ZoneFileTestSuite) TestApexRecordsAfterWWWKeepExplicitOwner() {
	records := []dnsrecord.Record{
		{HostName: testHostnameWWW, RecordType: dnsrecord.RecordTypeCNAME, Address: "parkingpage.namecheap.com.", TTL: 1800},
		{HostName: "@", RecordType: dnsrecord.RecordTypeMX, Address: "mx.glpx.pro.", TTL: 300, MXPref: 10},
		{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: testSPFValue, TTL: 1800},
	}

	content := Format("shootme.pro", records)

	// Every non-comment, non-directive line must start with an explicit
	// owner token; the bug produced a leading blank/tab for the apex
	// lines instead of "@".
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "$") {
			continue
		}
		s.Require().False(strings.HasPrefix(line, "\t"), "record line has a blank owner (inherits previous owner): %q", line)
		s.Require().False(strings.HasPrefix(line, " "), "record line has a blank owner (inherits previous owner): %q", line)
	}

	parsed := s.roundTrip("shootme.pro", records)
	s.Require().Equal(records, parsed)

	// The MX and TXT records must still be owned by "@", not "www".
	s.Require().Equal("@", parsed[1].HostName)
	s.Require().Equal("@", parsed[2].HostName)
}

// TestURLPseudoRecordsAreCommentsNotRecords is the regression test for B2:
// Namecheap's URL/URL301/FRAME entries are not DNS records and must not be
// emitted as an "IN URL..." line.
func (s *ZoneFileTestSuite) TestURLPseudoRecordsAreCommentsNotRecords() {
	records := []dnsrecord.Record{
		{HostName: testHostnameWWW, RecordType: RecordTypeURL301, Address: "https://example.com/"},
		{HostName: "old", RecordType: RecordTypeURL, Address: "https://example.com/new"},
		{HostName: "shop", RecordType: RecordTypeFrame, Address: "https://frame.example.com/"},
	}

	content := Format("example.com", records)

	s.Require().NotContains(content, "IN URL301")
	s.Require().NotContains(content, "IN URL ")
	s.Require().NotContains(content, "IN FRAME")
	s.Require().Contains(content, "; zonekit:url-redirect www URL301")
	s.Require().Contains(content, "; zonekit:url-redirect old URL")
	s.Require().Contains(content, "; zonekit:url-redirect shop FRAME")

	parsed := s.roundTrip("example.com", records)
	s.Require().Len(parsed, 3)
	for i := range records {
		s.Require().Equal(records[i].HostName, parsed[i].HostName)
		s.Require().Equal(records[i].RecordType, parsed[i].RecordType)
		s.Require().Equal(records[i].Address, parsed[i].Address)
	}
}

// TestNoFabricatedSOAOrNS is the regression test for B3: the reviewed
// exporter always emitted a fake SOA (serial 1 forever) and a hardcoded
// namecheap NS pair that may not even be the domain's real nameservers.
// Format must emit neither.
func (s *ZoneFileTestSuite) TestNoFabricatedSOAOrNS() {
	content := Format("example.com", []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeA, Address: "1.2.3.4", TTL: 1800},
	})

	// The header documents *why* SOA/NS are absent, so the word "SOA" is
	// expected to appear in that prose -- what must never appear is an
	// actual (fabricated) SOA resource record or NS pair.
	s.Require().NotContains(content, "IN SOA")
	s.Require().NotContains(content, "ns1.namecheap.com")
	s.Require().NotContains(content, "ns2.namecheap.com")
	s.Require().NotContains(content, "IN NS")
	s.Require().Contains(content, "$ORIGIN example.com.")
}

// TestLongTXTIsChunkedAndRoundTrips is the regression test for the TXT
// chunking requirement: a 408-byte value (typical DKIM key length) must be
// split into <=255-byte character-strings and reassembled to the identical
// value on import.
func (s *ZoneFileTestSuite) TestLongTXTIsChunkedAndRoundTrips() {
	value := strings.Repeat("AbCdEfGhIj", 40) + "12345678" // 408 chars
	s.Require().Len(value, 408)

	records := []dnsrecord.Record{
		{HostName: "selector1._domainkey", RecordType: dnsrecord.RecordTypeTXT, Address: value, TTL: 1800},
	}

	content := Format("example.com", records)

	// Two quoted character-strings: 255 + 153 bytes.
	quoteCount := strings.Count(content, `"`)
	s.Require().Equal(4, quoteCount, "expected exactly two quoted chunks (4 quote marks), got:\n%s", content)

	parsed := s.roundTrip("example.com", records)
	s.Require().Equal(records, parsed)
	s.Require().Equal(value, parsed[0].Address)
}

// TestTXTEscapesQuotesAndBackslashes verifies a TXT value that itself
// contains a double quote and a backslash survives Format/Parse unchanged.
func (s *ZoneFileTestSuite) TestTXTEscapesQuotesAndBackslashes() {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: `has "quotes" and \backslash\`, TTL: 300},
	}

	parsed := s.roundTrip("example.com", records)
	s.Require().Equal(records, parsed)
}

// TestShortTXTSingleChunk keeps the common case (SPF-length TXT) to one
// quoted chunk and round-trips it.
func (s *ZoneFileTestSuite) TestShortTXTSingleChunk() {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: testSPFValue, TTL: 1800},
	}

	content := Format("example.com", records)
	s.Require().Equal(2, strings.Count(content, `"`))

	parsed := s.roundTrip("example.com", records)
	s.Require().Equal(records, parsed)
}

// TestCAAAndSRVPassThrough covers the record types the review flagged as
// "if supported": zonekit stores their rdata as a single opaque string, and
// Format/Parse must preserve it exactly.
func (s *ZoneFileTestSuite) TestCAAAndSRVPassThrough() {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: "CAA", Address: "0 issue letsencrypt.org", TTL: 3600},
		{HostName: "_sip._tcp", RecordType: dnsrecord.RecordTypeSRV, Address: "10 60 5060 sipserver.example.com.", TTL: 3600},
	}

	parsed := s.roundTrip("example.com", records)
	s.Require().Equal(records, parsed)
}

// TestFullRoundTrip exercises every record kind together in one file,
// mirroring what a real 17-domain export looks like: an apex A record,
// records after "www", a multi-value TXT, and a URL redirect.
func (s *ZoneFileTestSuite) TestFullRoundTrip() {
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: dnsrecord.RecordTypeA, Address: "134.255.231.180", TTL: 1800},
		{HostName: testHostnameWWW, RecordType: dnsrecord.RecordTypeCNAME, Address: "parkingpage.namecheap.com.", TTL: 1800},
		{HostName: "@", RecordType: dnsrecord.RecordTypeMX, Address: "mx.glpx.pro.", TTL: 300, MXPref: 10},
		{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: testSPFValue, TTL: 1800},
		{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: "google-site-verification=abc123", TTL: 1800},
		{HostName: "shop", RecordType: RecordTypeURL301, Address: "https://shop.example.com/"},
	}

	parsed := s.roundTrip("shootme.pro", records)
	s.Require().Equal(records, parsed)
}

// TestParseRejectsMalformedInput checks a handful of parse error paths so
// import fails loudly on a corrupt/foreign file instead of silently
// dropping or misreading data.
func (s *ZoneFileTestSuite) TestParseRejectsMalformedInput() {
	tests := []struct {
		name    string
		content string
	}{
		{"non-numeric TTL", "@\tabc\tIN\tA\t1.2.3.4\n"},
		{"unsupported class", "@\t300\tCH\tA\t1.2.3.4\n"},
		{"too few fields", "@\t300\tIN\n"},
		{"MX missing preference", "@\t300\tIN\tMX\tmx.example.com.\n"},
		{"TXT missing quotes", "@\t300\tIN\tTXT\tunquoted\n"},
		{"unknown url-redirect type", "; zonekit:url-redirect www ALIAS \"https://example.com/\"\n"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := Parse(tt.content)
			s.Require().Error(err)
		})
	}
}

// TestParseSkipsDirectivesAndComments confirms $ORIGIN and plain comments
// are ignored rather than misparsed as records.
func (s *ZoneFileTestSuite) TestParseSkipsDirectivesAndComments() {
	content := "$ORIGIN example.com.\n; just a note\n\n@\t300\tIN\tA\t1.2.3.4\n"

	records, err := Parse(content)
	s.Require().NoError(err)
	s.Require().Len(records, 1)
	s.Require().Equal("1.2.3.4", records[0].Address)
}
