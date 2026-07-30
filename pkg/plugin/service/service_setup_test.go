package service

import (
	"strings"
	"testing"

	"zonekit/pkg/dns"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/plugin"
)

// fakeDNS is a plugin.Service that captures the exact record slice handed to
// SetRecords, so a test can assert on the zone that would actually be published.
type fakeDNS struct {
	existing []dnsrecord.Record
	written  []dnsrecord.Record
	wrote    bool
}

func (f *fakeDNS) GetRecords(string) ([]dnsrecord.Record, error) { return f.existing, nil }
func (f *fakeDNS) SetRecords(_ string, records []dnsrecord.Record) error {
	f.written = records
	f.wrote = true
	return nil
}
func (f *fakeDNS) GetRecordsByType(string, string) ([]dnsrecord.Record, error) { return nil, nil }
func (f *fakeDNS) AddRecord(string, dnsrecord.Record) error                    { return nil }
func (f *fakeDNS) UpdateRecord(string, string, string, dnsrecord.Record) error { return nil }
func (f *fakeDNS) DeleteRecord(string, string, string) error                   { return nil }
func (f *fakeDNS) DeleteAllRecords(string) error                               { return nil }
func (f *fakeDNS) ValidateRecord(dnsrecord.Record) error                       { return nil }
func (f *fakeDNS) BulkUpdate(string, []dns.BulkOperation) error                { return nil }

type discardOutput struct{}

func (discardOutput) Printf(string, ...interface{}) {}
func (discardOutput) Println(...interface{})        {}
func (discardOutput) Print(...interface{})          {}

// testPlugin returns a plugin with a minimal Migadu-shaped config: one MX and
// one apex SPF TXT, enough to exercise both collision paths.
func testPlugin() *ServicePlugin {
	return NewServicePlugin(map[string]*Config{
		"migadu": {
			Name:        "migadu",
			DisplayName: "Migadu",
			Records: Records{
				MX: []MXRecord{
					{Hostname: "@", Server: "aspmx1.migadu.com", Priority: 10},
				},
				SPF: &TXTRecord{Hostname: "@", Value: "v=spf1 include:spf.migadu.com -all"},
			},
		},
	})
}

func setupCtx(f *fakeDNS, flags map[string]interface{}) *plugin.Context {
	return &plugin.Context{
		Domain: "tercul.com",
		DNS:    f,
		Args:   []string{"migadu", "tercul.com"},
		Flags:  flags,
		Output: discardOutput{},
	}
}

func has(records []dnsrecord.Record, host, rtype string) bool {
	for _, r := range records {
		if r.HostName == host && r.RecordType == rtype {
			return true
		}
	}
	return false
}

// The regression this guards: --replace previously skipped the GetRecords call
// and handed SetRecords only the service's own records. Namecheap applies that
// as a whole-zone setHosts, so every unrelated record in the zone was deleted.
func TestSetupReplacePreservesUnrelatedRecords(t *testing.T) {
	f := &fakeDNS{existing: []dnsrecord.Record{
		{HostName: "@", RecordType: "A", Address: "45.141.36.203"},
		{HostName: "api", RecordType: "A", Address: "45.141.36.203"},
		{HostName: "app", RecordType: "A", Address: "45.141.36.203"},
		{HostName: "_acme-challenge", RecordType: "TXT", Address: "nucZmd7RJ9ZVNh"},
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 include:old ~all"},
	}}

	if err := testPlugin().setup(setupCtx(f, map[string]interface{}{"replace": true})); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !f.wrote {
		t.Fatal("SetRecords was never called")
	}

	for _, keep := range []struct{ host, rtype string }{
		{"@", "A"}, {"api", "A"}, {"app", "A"}, {"_acme-challenge", "TXT"},
	} {
		if !has(f.written, keep.host, keep.rtype) {
			t.Errorf("--replace dropped unrelated record %s %s (zone wipe)", keep.host, keep.rtype)
		}
	}

	if !has(f.written, "@", "MX") {
		t.Error("missing the MX record the service owns")
	}

	// Exactly one apex SPF survives (RFC 7208 allows only one), and it extends
	// the existing policy rather than discarding it.
	var spf []string
	for _, r := range f.written {
		if r.HostName == "@" && r.RecordType == "TXT" && txtKind(r.Address) == "spf" {
			spf = append(spf, r.Address)
		}
	}
	if len(spf) != 1 {
		t.Fatalf("expected exactly one apex SPF after replace, got %d: %v", len(spf), spf)
	}
	for _, want := range []string{"include:spf.migadu.com", "include:old", "~all"} {
		if !strings.Contains(spf[0], want) {
			t.Errorf("merged SPF %q lost %q", spf[0], want)
		}
	}
}

// TXT is multi-valued: publishing an SPF record must not evict unrelated
// ownership proofs that share its hostname.
func TestSetupReplacePreservesUnrelatedTXTAtSameHost(t *testing.T) {
	f := &fakeDNS{existing: []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "google-site-verification=zRNLAhp1gv"},
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 include:spf.efwd.registrar-servers.com ~all"},
	}}

	if err := testPlugin().setup(setupCtx(f, map[string]interface{}{"replace": true})); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var google, migaduSPF int
	for _, r := range f.written {
		if r.HostName != "@" || r.RecordType != "TXT" {
			continue
		}
		switch {
		case r.Address == "google-site-verification=zRNLAhp1gv":
			google++
		case txtKind(r.Address) == "spf":
			migaduSPF++
			// The pre-existing forwarding sender must survive the merge.
			for _, want := range []string{"include:spf.migadu.com", "include:spf.efwd.registrar-servers.com"} {
				if !strings.Contains(r.Address, want) {
					t.Errorf("merged SPF %q lost %q", r.Address, want)
				}
			}
		default:
			t.Errorf("unexpected leftover apex TXT: %q", r.Address)
		}
	}
	if google != 1 {
		t.Errorf("google-site-verification TXT dropped (got %d copies)", google)
	}
	if migaduSPF != 1 {
		t.Errorf("expected exactly one apex SPF TXT, got %d", migaduSPF)
	}
}

// Without --replace a genuine same-kind collision is reported and nothing is
// written at all.
func TestSetupWithoutReplaceRefusesOnConflict(t *testing.T) {
	f := &fakeDNS{existing: []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 include:spf.efwd.registrar-servers.com ~all"},
	}}

	if err := testPlugin().setup(setupCtx(f, map[string]interface{}{})); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if f.wrote {
		t.Fatalf("expected no write on unresolved conflict, wrote %d records", len(f.written))
	}
}

// A verification token at the same hostname as the service's SPF is not a
// conflict, so setup proceeds and merges rather than refusing.
func TestSetupWithoutReplaceIgnoresUnrelatedTXT(t *testing.T) {
	f := &fakeDNS{existing: []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "google-site-verification=zRNLAhp1gv"},
	}}

	if err := testPlugin().setup(setupCtx(f, map[string]interface{}{})); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if !f.wrote {
		t.Fatal("expected a write: an unrelated TXT is not a conflict")
	}
	if !has(f.written, "@", "MX") {
		t.Error("service MX record missing")
	}
	if !has(f.written, "@", "TXT") {
		t.Error("apex TXT records missing entirely")
	}
}

func TestTxtKind(t *testing.T) {
	cases := map[string]string{
		`v=spf1 include:spf.migadu.com -all`: "spf",
		`"v=SPF1 -all"`:                      "spf",
		`v=DMARC1; p=quarantine;`:            "dmarc",
		`v=DKIM1;k=rsa;p=MIIB`:               "dkim",
		`google-site-verification=abc`:       "literal:google-site-verification=abc",
	}
	for in, want := range cases {
		if got := txtKind(in); got != want {
			t.Errorf("txtKind(%q) = %q, want %q", in, got, want)
		}
	}
}
