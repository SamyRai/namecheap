package service

import (
	"path/filepath"
	"strings"
	"testing"

	"zonekit/pkg/dnsrecord"
)

// TestMigaduTemplateMatchesProviderDocs pins the shipped template to Migadu's
// published "DNS Setup Instructions". Each expectation below corresponds to a
// section of that page; if Migadu changes a hostname or target, this fails
// rather than silently publishing a stale zone.
func TestMigaduTemplateMatchesProviderDocs(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("services", "migadu.yaml"))
	if err != nil {
		t.Fatalf("load migadu.yaml: %v", err)
	}

	p := NewServicePlugin(map[string]*Config{"migadu": cfg})
	records := p.generateRecordsWithOpts(cfg, "example.com", generateOpts{
		Vars:         map[string]string{"token": "jyersruu"},
		WithWildcard: true,
	})

	if missing := unresolvedPlaceholders(records); len(missing) > 0 {
		t.Fatalf("template left placeholders unresolved: %v", missing)
	}

	type want struct {
		host, rtype, contains string
		port, priority        int
	}
	expected := []want{
		// Verification TXT -- the record that actually activates the domain.
		{host: "@", rtype: "TXT", contains: "hosted-email-verify=jyersruu"},
		// MX, aspmx1 at a lower (more preferred) priority than aspmx2.
		{host: "@", rtype: "MX", contains: "aspmx1.migadu.com", priority: 10},
		{host: "@", rtype: "MX", contains: "aspmx2.migadu.com", priority: 20},
		// DKIM+ARC: three CNAMEs so Migadu can rotate keys without a DNS change.
		{host: "key1._domainkey", rtype: "CNAME", contains: "key1.example.com._domainkey.migadu.com"},
		{host: "key2._domainkey", rtype: "CNAME", contains: "key2.example.com._domainkey.migadu.com"},
		{host: "key3._domainkey", rtype: "CNAME", contains: "key3.example.com._domainkey.migadu.com"},
		// SPF.
		{host: "@", rtype: "TXT", contains: "include:spf.migadu.com"},
		// DMARC lives at _dmarc, never the apex (RFC 7489 6.1).
		{host: "_dmarc", rtype: "TXT", contains: "v=DMARC1"},
		// Subdomain addressing (opt-in).
		{host: "*", rtype: "MX", contains: "aspmx1.migadu.com", priority: 10},
		// Thunderbird autoconfig.
		{host: "autoconfig", rtype: "CNAME", contains: "autoconfig.migadu.com"},
		// Service discovery. Submission uses 465 (implicit TLS), not 587.
		{host: "_autodiscover._tcp", rtype: "SRV", contains: "autodiscover.migadu.com", port: 443},
		{host: "_submissions._tcp", rtype: "SRV", contains: "smtp.migadu.com", port: 465},
		{host: "_imaps._tcp", rtype: "SRV", contains: "imap.migadu.com", port: 993},
		{host: "_pop3s._tcp", rtype: "SRV", contains: "pop.migadu.com", port: 995},
	}

	for _, w := range expected {
		if !hasMatching(records, w.host, w.rtype, w.contains, w.port, w.priority) {
			t.Errorf("template missing %s %s -> %s (port=%d prio=%d)",
				w.host, w.rtype, w.contains, w.port, w.priority)
		}
	}

	// The apex must never carry a DMARC policy: it would be inert and would
	// collide with the SPF record's hostname+type.
	for _, r := range records {
		if r.HostName == "@" && r.RecordType == dnsrecord.RecordTypeTXT &&
			strings.HasPrefix(strings.ToLower(r.Address), "v=dmarc1") {
			t.Error("DMARC published at the apex; it belongs at _dmarc")
		}
	}
}

func hasMatching(records []dnsrecord.Record, host, rtype, contains string, port, priority int) bool {
	for _, r := range records {
		if r.HostName != host || r.RecordType != rtype {
			continue
		}
		haystack := r.Address
		if r.RecordType == dnsrecord.RecordTypeSRV {
			haystack = r.Target
		}
		if !strings.Contains(haystack, contains) {
			continue
		}
		if port != 0 && r.Port != port {
			continue
		}
		if priority != 0 && r.MXPref != priority && r.Priority != priority {
			continue
		}
		return true
	}
	return false
}

// Wildcard MX must stay absent unless explicitly requested.
func TestMigaduTemplateWildcardIsOptIn(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("services", "migadu.yaml"))
	if err != nil {
		t.Fatalf("load migadu.yaml: %v", err)
	}
	p := NewServicePlugin(map[string]*Config{"migadu": cfg})
	records := p.generateRecordsWithOpts(cfg, "example.com", generateOpts{
		Vars: map[string]string{"token": "t"},
	})
	for _, r := range records {
		if r.HostName == "*" {
			t.Errorf("wildcard record published without opt-in: %+v", r)
		}
	}
}
