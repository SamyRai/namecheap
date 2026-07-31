package migadu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zonekit/pkg/dnsrecord"
)

// liveBundle is a verbatim capture of GET /v1/domains/tercul.com/records from
// the production API, so the parser is pinned to the real payload shape rather
// than an assumed one.
const liveBundle = `{
  "spf": {"name": "@", "type": "txt", "value": "v=spf1 include:spf.migadu.com -all"},
  "dkim": [
    {"name": "key1._domainkey", "type": "cname", "value": "key1.tercul.com._domainkey.migadu.com."},
    {"name": "key2._domainkey", "type": "cname", "value": "key2.tercul.com._domainkey.migadu.com."},
    {"name": "key3._domainkey", "type": "cname", "value": "key3.tercul.com._domainkey.migadu.com."}
  ],
  "domain_name": "tercul.com",
  "dmarc": {"name": "_dmarc", "type": "txt", "value": "v=DMARC1; p=quarantine;"},
  "dns_verification": {"name": "@", "type": "txt", "value": "hosted-email-verify=gvmelozw"},
  "mx_records": [
    {"name": "@", "priority": 10, "type": "mx", "value": "aspmx1.migadu.com"},
    {"name": "@", "priority": 20, "type": "mx", "value": "aspmx2.migadu.com"}
  ]
}`

func testServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New("account@example.com", "key")
	c.BaseURL = srv.URL
	return c
}

func TestGetDNSBundleParsesLivePayload(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains/tercul.com/records" {
			t.Errorf("unexpected path %q -- the bundle lives at /records", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "account@example.com" || pass != "key" {
			t.Error("request did not carry basic auth")
		}
		_, _ = w.Write([]byte(liveBundle))
	})

	b, err := c.GetDNSBundle(context.Background(), "tercul.com")
	if err != nil {
		t.Fatalf("GetDNSBundle: %v", err)
	}
	if got := b.VerificationToken(); got != "gvmelozw" {
		t.Errorf("VerificationToken = %q, want %q", got, "gvmelozw")
	}
	if len(b.DKIM) != 3 {
		t.Errorf("expected 3 DKIM records, got %d", len(b.DKIM))
	}
	if b.DMARC.Name != "_dmarc" {
		t.Errorf("DMARC must be at _dmarc, got %q", b.DMARC.Name)
	}
}

// The whole point of sourcing from the API: no hand-pasted token, and the
// generated set matches what the provider says the domain needs.
func TestBundleToRecords(t *testing.T) {
	var b DNSBundle
	if err := json.Unmarshal([]byte(liveBundle), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	records := b.ToRecords()

	want := map[string]string{
		"@|TXT|hosted-email-verify=gvmelozw":                           "verification",
		"@|MX|aspmx1.migadu.com.":                                      "primary MX",
		"@|MX|aspmx2.migadu.com.":                                      "secondary MX",
		"@|TXT|v=spf1 include:spf.migadu.com -all":                     "spf",
		"_dmarc|TXT|v=DMARC1; p=quarantine;":                           "dmarc",
		"key1._domainkey|CNAME|key1.tercul.com._domainkey.migadu.com.": "dkim1",
	}
	got := map[string]bool{}
	for _, r := range records {
		got[r.HostName+"|"+r.RecordType+"|"+r.Address] = true
		if strings.Contains(r.Address, "{") {
			t.Errorf("record still contains a placeholder: %q", r.Address)
		}
	}
	for key, label := range want {
		if !got[key] {
			t.Errorf("missing %s record (%s)", label, key)
		}
	}

	// MX priorities must survive, since Migadu reports them separately.
	for _, r := range records {
		if r.RecordType == dnsrecord.RecordTypeMX && r.MXPref == 0 {
			t.Errorf("MX record lost its priority: %+v", r)
		}
	}
}

// Migadu returns MX targets without a trailing dot and DKIM targets with one.
func TestToRecordsNormalizesTrailingDots(t *testing.T) {
	var b DNSBundle
	_ = json.Unmarshal([]byte(liveBundle), &b)
	for _, r := range b.ToRecords() {
		if r.RecordType == dnsrecord.RecordTypeMX || r.RecordType == dnsrecord.RecordTypeCNAME {
			if !strings.HasSuffix(r.Address, ".") {
				t.Errorf("%s target not fully qualified: %q", r.RecordType, r.Address)
			}
		}
	}
}

func TestDiagnostics(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"failed","checks":{"verify":"failed","mx":"ok","spf":"ok","dkim":"ok","nameservers":"ok"}}`))
	})
	d, err := c.GetDiagnostics(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("GetDiagnostics: %v", err)
	}
	if d.OK() {
		t.Error("status was 'failed' but OK() returned true")
	}
	failed := d.Failed()
	if len(failed) != 1 || !strings.HasPrefix(failed[0], "verify=") {
		t.Errorf("Failed() should name the failing check, got %v", failed)
	}
}

// A 401 is nearly always the label-prefix or expired-key mistake; the error
// should say so instead of surfacing a bare status code.
func TestUnauthorizedErrorIsActionable(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"the token is invalid or expired"}`))
	})
	_, err := c.ListDomains(context.Background())
	if err == nil {
		t.Fatal("expected an error on 401")
	}
	if !strings.Contains(err.Error(), "label prefix") {
		t.Errorf("401 error should hint at the cause, got: %v", err)
	}
}

// Mailboxes are created through the invitation flow so the tool never handles
// a password.
func TestCreateMailboxUsesInvitationByDefault(t *testing.T) {
	var received map[string]interface{}
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_, _ = w.Write([]byte(`{"address":"noreply@example.com","is_active":false,"may_send":true}`))
	})

	_, err := c.CreateMailbox(context.Background(), "example.com", MailboxSpec{
		LocalPart:     "noreply",
		RecoveryEmail: "owner@example.com",
		MaySend:       true,
	})
	if err != nil {
		t.Fatalf("CreateMailbox: %v", err)
	}
	if received["password_method"] != "invitation" {
		t.Errorf("password_method = %v, want invitation", received["password_method"])
	}
	if _, leaked := received["password"]; leaked {
		t.Error("a password field must never be sent by this client")
	}
}

func TestDomainIsActive(t *testing.T) {
	if (Domain{State: "inactive"}).IsActive() {
		t.Error("inactive domain reported as active")
	}
	if !(Domain{State: "active"}).IsActive() {
		t.Error("active domain reported as inactive")
	}
}
