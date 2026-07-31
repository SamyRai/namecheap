package migadu

import (
	"strings"

	"zonekit/pkg/dnsrecord"
)

// DefaultTTL matches the DNS layer's default; kept local so this package does
// not depend on the DNS service.
const DefaultTTL = 1800

// ToRecords converts Migadu's advisory bundle into zonekit records.
//
// This is the point of provider-backed templates: the values come from the
// provider's own computed view of the domain rather than a hand-maintained
// YAML copy. In particular the per-domain verification token -- which is not
// derivable and appears nowhere else in the API -- no longer has to be pasted
// in by hand, and a change to Migadu's hostnames or key layout propagates
// without a template edit.
//
// Records the bundle does not describe (autoconfig CNAME, SRV service
// discovery, wildcard MX for subdomain addressing) stay in the static template,
// which is why the two are merged rather than one replacing the other.
func (b DNSBundle) ToRecords() []dnsrecord.Record {
	var records []dnsrecord.Record

	if b.DNSVerification.Value != "" {
		records = append(records, dnsrecord.Record{
			HostName:   hostOrApex(b.DNSVerification.Name),
			RecordType: dnsrecord.RecordTypeTXT,
			Address:    b.DNSVerification.Value,
			TTL:        DefaultTTL,
		})
	}

	for _, mx := range b.MXRecords {
		records = append(records, dnsrecord.Record{
			HostName:   hostOrApex(mx.Name),
			RecordType: dnsrecord.RecordTypeMX,
			Address:    ensureTrailingDot(mx.Value),
			TTL:        DefaultTTL,
			MXPref:     mx.Priority,
		})
	}

	if b.SPF.Value != "" {
		records = append(records, dnsrecord.Record{
			HostName:   hostOrApex(b.SPF.Name),
			RecordType: dnsrecord.RecordTypeTXT,
			Address:    b.SPF.Value,
			TTL:        DefaultTTL,
		})
	}

	for _, dkim := range b.DKIM {
		records = append(records, dnsrecord.Record{
			HostName:   hostOrApex(dkim.Name),
			RecordType: recordType(dkim.Type, dnsrecord.RecordTypeCNAME),
			Address:    ensureTrailingDot(dkim.Value),
			TTL:        DefaultTTL,
		})
	}

	if b.DMARC.Value != "" {
		records = append(records, dnsrecord.Record{
			HostName:   hostOrApex(b.DMARC.Name),
			RecordType: dnsrecord.RecordTypeTXT,
			Address:    b.DMARC.Value,
			TTL:        DefaultTTL,
		})
	}

	return records
}

// hostOrApex normalizes an empty hostname to the apex marker used by the DNS
// layer. Migadu reports the apex as "@", but tolerate "" too.
func hostOrApex(name string) string {
	if strings.TrimSpace(name) == "" {
		return "@"
	}
	return name
}

// recordType upper-cases the provider's lowercase type names ("txt", "cname").
func recordType(provided, fallback string) string {
	if provided == "" {
		return fallback
	}
	return strings.ToUpper(provided)
}

// ensureTrailingDot makes a target fully qualified. Migadu returns MX targets
// without a trailing dot but DKIM targets with one, so normalize both.
func ensureTrailingDot(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || strings.HasSuffix(host, ".") {
		return host
	}
	return host + "."
}
