package service

import (
	"strings"
	"testing"

	"zonekit/pkg/dnsrecord"
)

// ownershipPlugin mirrors the shape of the real Migadu template: a
// per-domain ownership token, an apex SPF, a _dmarc policy, an opt-in
// wildcard MX and an SRV record.
func ownershipPlugin() *ServicePlugin {
	return NewServicePlugin(map[string]*Config{
		"migadu": {
			Name:        "migadu",
			DisplayName: "Migadu",
			Records: Records{
				Ownership: &TXTRecord{Hostname: "@", Value: "hosted-email-verify={token}"},
				MX: []MXRecord{
					{Hostname: "@", Server: "aspmx1.migadu.com", Priority: 10},
				},
				SPF:   &TXTRecord{Hostname: "@", Value: "v=spf1 include:spf.migadu.com -all"},
				DMARC: &TXTRecord{Hostname: "_dmarc", Value: "v=DMARC1; p=quarantine;"},
				WildcardMX: []MXRecord{
					{Hostname: "*", Server: "aspmx1.migadu.com", Priority: 10},
				},
				SRV: []SRVRecord{
					{Hostname: "_imaps._tcp", Target: "imap.migadu.com", Port: 993, Priority: 0, Weight: 1},
				},
			},
		},
	})
}

// Publishing a literal "{token}" would look like success while leaving the
// domain unverified and mail disabled, so an unresolved placeholder must abort
// before anything is written.
func TestSetupFailsClosedOnUnresolvedPlaceholder(t *testing.T) {
	f := &fakeDNS{}
	err := ownershipPlugin().setup(setupCtx(f, map[string]interface{}{"replace": true}))
	if err == nil {
		t.Fatal("expected setup to fail when {token} is unresolved")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error should name the missing variable, got: %v", err)
	}
	if f.wrote {
		t.Error("nothing may be written when a placeholder is unresolved")
	}
}

func TestSetupResolvesOwnershipToken(t *testing.T) {
	f := &fakeDNS{}
	ctx := setupCtx(f, map[string]interface{}{
		"replace": true,
		"vars":    map[string]string{"token": "jyersruu"},
	})
	if err := ownershipPlugin().setup(ctx); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var found bool
	for _, r := range f.written {
		if r.RecordType == "TXT" && r.Address == "hosted-email-verify=jyersruu" {
			found = true
		}
		if strings.Contains(r.Address, "{") {
			t.Errorf("published record still contains a placeholder: %q", r.Address)
		}
	}
	if !found {
		t.Error("ownership record was not published with the supplied token")
	}
}

// Wildcard MX changes delivery for every subdomain, so it must not appear
// unless explicitly requested.
func TestWildcardMXIsOptIn(t *testing.T) {
	for _, tc := range []struct{ optIn, want bool }{{false, false}, {true, true}} {
		f := &fakeDNS{}
		flags := map[string]interface{}{
			"replace": true,
			"vars":    map[string]string{"token": "abc"},
		}
		if tc.optIn {
			flags["with-wildcard-mx"] = true
		}
		if err := ownershipPlugin().setup(setupCtx(f, flags)); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if got := has(f.written, "*", "MX"); got != tc.want {
			t.Errorf("with-wildcard-mx=%v: wildcard MX present=%v, want %v", tc.optIn, got, tc.want)
		}
	}
}

func TestSRVRecordCarriesPriorityWeightPort(t *testing.T) {
	f := &fakeDNS{}
	ctx := setupCtx(f, map[string]interface{}{
		"replace":  true,
		"with-srv": true,
		"vars":     map[string]string{"token": "abc"},
	})
	if err := ownershipPlugin().setup(ctx); err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, r := range f.written {
		if r.RecordType != dnsrecord.RecordTypeSRV {
			continue
		}
		if r.Port != 993 || r.Weight != 1 || r.Target == "" {
			t.Errorf("SRV record lost its parameters: %+v", r)
		}
		return
	}
	t.Error("no SRV record was generated")
}

// An existing DMARC policy commonly carries an rua= reporting address. The
// template's bare policy must not silently discard it.
func TestDMARCMergePreservesReportingAddress(t *testing.T) {
	existing := "v=DMARC1;p=none;sp=quarantine;pct=100;rua=mailto:info@glowing-pixels.com"
	merged := mergeDMARC(existing, "v=DMARC1; p=quarantine;")

	for _, want := range []string{"rua=mailto:info@glowing-pixels.com", "sp=quarantine", "pct=100"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged DMARC %q lost %q", merged, want)
		}
	}
	// The existing policy strength wins; the service only fills gaps.
	if !strings.Contains(merged, "p=none") {
		t.Errorf("merged DMARC %q should keep the existing p= tag", merged)
	}
}

func TestDMARCMergeAdoptsServicePolicyWhenAbsent(t *testing.T) {
	if got := mergeDMARC("", "v=DMARC1; p=quarantine;"); got != "v=DMARC1; p=quarantine;" {
		t.Errorf("with no existing policy the service value should be used, got %q", got)
	}
}

func TestMergeSPF(t *testing.T) {
	cases := []struct {
		name, existing, service string
		wantContains            []string
		wantNotContains         []string
	}{
		{
			name:         "splices include and keeps existing qualifier",
			existing:     "v=spf1 include:spf.efwd.registrar-servers.com ~all",
			service:      "v=spf1 include:spf.migadu.com -all",
			wantContains: []string{"v=spf1", "include:spf.migadu.com", "include:spf.efwd.registrar-servers.com", "~all"},
			// Only one all-qualifier may appear, and it stays the existing one.
			wantNotContains: []string{"-all"},
		},
		{
			name:         "idempotent when the include is already present",
			existing:     "v=spf1 include:spf.migadu.com -all",
			service:      "v=spf1 include:spf.migadu.com -all",
			wantContains: []string{"include:spf.migadu.com"},
		},
		{
			name:         "hard-fail policy preserved",
			existing:     "v=spf1 -all",
			service:      "v=spf1 include:spf.migadu.com -all",
			wantContains: []string{"include:spf.migadu.com", "-all"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeSPF(tc.existing, tc.service)
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("mergeSPF = %q, missing %q", got, want)
				}
			}
			for _, notWant := range tc.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("mergeSPF = %q, should not contain %q", got, notWant)
				}
			}
			if n := strings.Count(got, "v=spf1"); n != 1 {
				t.Errorf("mergeSPF = %q has %d version tags, want exactly 1", got, n)
			}
		})
	}
}

// --force-policy is the explicit escape hatch: it overwrites rather than merges.
func TestForcePolicyOverwrites(t *testing.T) {
	existing := []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 include:other ~all"},
	}
	records := []dnsrecord.Record{
		{HostName: "@", RecordType: "TXT", Address: "v=spf1 include:spf.migadu.com -all"},
	}

	forced := mergePolicyRecords(records, existing, true)
	if forced[0].Address != "v=spf1 include:spf.migadu.com -all" {
		t.Errorf("--force-policy should not merge, got %q", forced[0].Address)
	}

	merged := mergePolicyRecords(records, existing, false)
	if !strings.Contains(merged[0].Address, "include:other") {
		t.Errorf("without --force-policy the existing sender must survive, got %q", merged[0].Address)
	}
}

// Dry-run output is the only review surface before a whole-zone write, so an
// SRV record must not render as a blank value.
func TestDisplayValueRendersSRVParameters(t *testing.T) {
	got := displayValue(dnsrecord.Record{
		HostName:   "_submissions._tcp",
		RecordType: dnsrecord.RecordTypeSRV,
		Target:     "smtp.migadu.com.",
		Port:       465,
		Priority:   0,
		Weight:     1,
	})
	for _, want := range []string{"smtp.migadu.com.", "465", "weight: 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("displayValue = %q, missing %q", got, want)
		}
	}
	if strings.TrimSpace(got) == "" {
		t.Error("SRV rendered as an empty value")
	}
}
