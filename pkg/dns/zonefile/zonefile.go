// Package zonefile converts between zonekit's internal DNS record slice and
// a BIND-style master zone file, so `zonekit dns export`/`dns import` can
// round-trip a domain's records through a plain text file.
package zonefile

import (
	"fmt"
	"strings"

	"zonekit/pkg/dns"
	"zonekit/pkg/dnsrecord"
)

// Namecheap-specific pseudo-record types. These configure the registrar's
// own HTTP-redirect/frame forwarding, not a real DNS resource record, so
// they cannot be represented as an "IN <TYPE> ..." zone file line.
const (
	RecordTypeURL    = "URL"
	RecordTypeURL301 = "URL301"
	RecordTypeFrame  = "FRAME"
)

// urlRedirectPrefix marks a comment line that encodes a URL/URL301/FRAME
// pseudo-record. Parse looks for this exact prefix; any other comment is
// treated as inert text and skipped.
const urlRedirectPrefix = "; zonekit:url-redirect "

// maxTXTChunk is the maximum length in bytes of a single BIND
// <character-string>, per RFC 1035 3.3: "the first byte... is a single
// length octet" (max 255).
const maxTXTChunk = 255

// Format renders records as a BIND-style master zone file for domainName.
//
// Every record line carries an explicit owner name ("@" for the zone apex,
// or the relative hostname otherwise). A blank owner is never emitted:
// in master-file syntax a blank owner column inherits the *previous*
// line's owner, so exporting an apex MX/TXT record right after a "www"
// record with a blank owner would silently relocate it onto "www" on
// import elsewhere. See ownerName.
//
// Namecheap's URL/URL301/FRAME entries are not DNS records: they are the
// registrar's own HTTP-redirect/frame configuration. Emitting them as if
// they were e.g. an "IN URL301 ..." resource record produces a file
// standard tools can't parse and other providers can't import. They are
// written as inert "; zonekit:url-redirect" comments instead; Parse
// understands and restores them.
//
// SOA/NS are intentionally omitted rather than fabricated. Namecheap's
// hosted-DNS API never returns real SOA data (serial/refresh/retry/expire
// numbers do not exist on the account side to export), and Namecheap
// manages the zone's authoritative NS set itself -- it is not part of the
// record set `dns list`/`dns export` see. A previous version of this
// exporter synthesized a fake SOA and a hardcoded "ns1/ns2.namecheap.com"
// NS pair on every run, which is neither valid (serial=1 forever) nor
// actually authoritative. Emitting invented data is worse than omitting
// it, so this exporter emits neither, consistently, and says so in the
// header comment.
func Format(domainName string, records []dnsrecord.Record) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "$ORIGIN %s.\n", domainName)
	sb.WriteString("; SOA/NS intentionally omitted: Namecheap hosted DNS does not expose\n")
	sb.WriteString("; SOA data and manages this zone's NS records itself.\n\n")

	for _, record := range records {
		writeRecord(&sb, record)
	}

	return sb.String()
}

func writeRecord(sb *strings.Builder, record dnsrecord.Record) {
	owner := ownerName(record.HostName)

	if isPseudoRedirectType(record.RecordType) {
		writeURLRedirectComment(sb, owner, record)
		return
	}

	ttl := record.TTL
	if ttl <= 0 {
		ttl = dns.DefaultTTL
	}

	switch strings.ToUpper(record.RecordType) {
	case dnsrecord.RecordTypeTXT:
		fmt.Fprintf(sb, "%s\t%d\tIN\tTXT\t%s\n", owner, ttl, quoteTXT(record.Address))
	case dnsrecord.RecordTypeMX:
		mxPref := record.MXPref
		if mxPref == 0 {
			mxPref = dns.DefaultMXPref
		}
		fmt.Fprintf(sb, "%s\t%d\tIN\tMX\t%d %s\n", owner, ttl, mxPref, record.Address)
	default:
		fmt.Fprintf(sb, "%s\t%d\tIN\t%s\t%s\n", owner, ttl, record.RecordType, record.Address)
	}
}

// ownerName returns an explicit, never-blank owner name for hostname: "@"
// for the zone apex (Namecheap returns HostName "@" for apex records; an
// empty HostName is treated the same way defensively), the hostname
// unchanged otherwise.
func ownerName(hostname string) string {
	if hostname == "" || hostname == "@" {
		return "@"
	}
	return hostname
}

func isPseudoRedirectType(recordType string) bool {
	switch strings.ToUpper(recordType) {
	case RecordTypeURL, RecordTypeURL301, RecordTypeFrame:
		return true
	default:
		return false
	}
}

func writeURLRedirectComment(sb *strings.Builder, owner string, record dnsrecord.Record) {
	fmt.Fprintf(sb, "%s%s %s %s\n",
		urlRedirectPrefix, owner, strings.ToUpper(record.RecordType), quoteChunk(record.Address))
}

// quoteTXT renders value as one or more space-separated BIND
// <character-string>s, splitting on maxTXTChunk-byte boundaries so no
// single quoted string exceeds the format's 255-byte limit (e.g. a 408-byte
// DKIM key becomes two chunks). value is taken as raw, unquoted record
// data -- the form dnsrecord.Record.Address holds it in throughout the rest
// of zonekit.
func quoteTXT(value string) string {
	data := []byte(value)
	if len(data) == 0 {
		return `""`
	}

	chunks := make([]string, 0, (len(data)/maxTXTChunk)+1)
	for len(data) > 0 {
		n := maxTXTChunk
		if n > len(data) {
			n = len(data)
		}
		chunks = append(chunks, quoteChunk(string(data[:n])))
		data = data[n:]
	}
	return strings.Join(chunks, " ")
}

var txtEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func quoteChunk(chunk string) string {
	return `"` + txtEscaper.Replace(chunk) + `"`
}
