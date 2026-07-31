// Package migadu is a client for Migadu's Admin API, covering the operations
// needed to take a domain from "registered" to "sending mail" without manual
// console work: reading the provider's own DNS advisory bundle, running its
// diagnostics, activating the domain, and provisioning mailboxes.
//
// The API is documented at https://www.migadu.com/api/ and is described there
// as early beta. Endpoint paths below are the ones verified against the live
// service; note that the DNS bundle lives at /records, while every intuitively
// named alternative (/dns, /instructions, /verify, /zone) returns 404.
package migadu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is Migadu's Admin API root.
const DefaultBaseURL = "https://api.migadu.com/v1"

// Client talks to the Migadu Admin API. Auth is HTTP Basic where the username
// is the *account* email (not a mailbox) and the password is an API key
// generated under My Account -> API Keys.
type Client struct {
	BaseURL    string
	Account    string
	APIKey     string
	HTTPClient *http.Client
}

// New returns a Client with sensible defaults.
func New(account, apiKey string) *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		Account:    account,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Domain is a domain as Migadu models it. Only the fields this tool acts on
// are mapped; the API returns considerably more.
type Domain struct {
	Name                    string   `json:"name"`
	State                   string   `json:"state"`
	CanSend                 bool     `json:"can_send"`
	CanReceive              bool     `json:"can_receive"`
	CatchallDestinations    []string `json:"catchall_destinations"`
	SubjectRewritingEnabled bool     `json:"subject_rewriting_enabled"`
}

// IsActive reports whether the domain has passed Migadu's DNS checks. Until it
// is active, Migadu "knows nothing of your domain and mailboxes" and refuses
// to send.
func (d Domain) IsActive() bool { return d.State == "active" }

// DNSRecord is one entry of the advisory bundle.
type DNSRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"`
}

// DNSBundle is Migadu's computed, per-domain view of the DNS a domain needs.
// It is authoritative: it carries the domain's unique verification token, which
// is not exposed anywhere else in the API and otherwise has to be copied by
// hand from the admin console.
type DNSBundle struct {
	DomainName      string      `json:"domain_name"`
	DNSVerification DNSRecord   `json:"dns_verification"`
	SPF             DNSRecord   `json:"spf"`
	DMARC           DNSRecord   `json:"dmarc"`
	DKIM            []DNSRecord `json:"dkim"`
	MXRecords       []DNSRecord `json:"mx_records"`
}

// VerificationToken returns the bare token from the hosted-email-verify record,
// stripping the "hosted-email-verify=" prefix that the API includes.
func (b DNSBundle) VerificationToken() string {
	return strings.TrimPrefix(b.DNSVerification.Value, "hosted-email-verify=")
}

// Diagnostics is Migadu's own DNS check result. Activation refuses until
// status is "ok".
type Diagnostics struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// OK reports whether every check passed.
func (d Diagnostics) OK() bool { return d.Status == "ok" }

// Failed lists the checks that are not "ok", so a caller can say precisely
// what is missing rather than "DNS checks failed".
func (d Diagnostics) Failed() []string {
	var out []string
	for name, status := range d.Checks {
		if status != "ok" {
			out = append(out, fmt.Sprintf("%s=%s", name, status))
		}
	}
	return out
}

// Mailbox is a Migadu mailbox.
type Mailbox struct {
	Address    string `json:"address"`
	LocalPart  string `json:"local_part,omitempty"`
	Name       string `json:"name,omitempty"`
	IsActive   bool   `json:"is_active"`
	MaySend    bool   `json:"may_send"`
	MayReceive bool   `json:"may_receive"`
}

// Forwarding is a forwarding target on a mailbox.
type Forwarding struct {
	Address  string `json:"address"`
	IsActive bool   `json:"is_active"`
}

func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(c.Account, c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		// 401 here almost always means the API key was pasted with a label
		// prefix or has expired; say so rather than echoing a bare status.
		hint := ""
		if resp.StatusCode == http.StatusUnauthorized {
			hint = " (check the account email and that the API key has no label prefix)"
		}
		return fmt.Errorf("%s %s: HTTP %d%s: %s",
			method, path, resp.StatusCode, hint, strings.TrimSpace(buf.String()))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}

// ListDomains returns every domain on the account.
func (c *Client) ListDomains(ctx context.Context) ([]Domain, error) {
	var wrapper struct {
		Domains []Domain `json:"domains"`
	}
	if err := c.do(ctx, http.MethodGet, "/domains", nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Domains, nil
}

// GetDomain returns one domain.
func (c *Client) GetDomain(ctx context.Context, domain string) (*Domain, error) {
	var d Domain
	if err := c.do(ctx, http.MethodGet, "/domains/"+url.PathEscape(domain), nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDNSBundle returns Migadu's computed DNS advisory for a domain, including
// the verification token.
func (c *Client) GetDNSBundle(ctx context.Context, domain string) (*DNSBundle, error) {
	var b DNSBundle
	if err := c.do(ctx, http.MethodGet, "/domains/"+url.PathEscape(domain)+"/records", nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetDiagnostics runs Migadu's DNS checks for a domain.
func (c *Client) GetDiagnostics(ctx context.Context, domain string) (*Diagnostics, error) {
	var d Diagnostics
	if err := c.do(ctx, http.MethodGet, "/domains/"+url.PathEscape(domain)+"/diagnostics", nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Activate flips a domain from inactive to active. Migadu rejects this with
// 422 dns_check_failed until diagnostics pass, so callers should check
// GetDiagnostics first to produce a useful message.
func (c *Client) Activate(ctx context.Context, domain string) (*Domain, error) {
	var d Domain
	if err := c.do(ctx, http.MethodGet, "/domains/"+url.PathEscape(domain)+"/activate", nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDomain patches domain-level settings.
//
// NOTE: catch-all destinations must be a mailbox on the same domain. Passing an
// address on another domain returns HTTP 200 but is silently coerced to the
// same local part on this domain -- so use a forwarding on the target mailbox
// to reach another domain, and do not trust the 200 alone.
func (c *Client) UpdateDomain(ctx context.Context, domain string, patch map[string]interface{}) (*Domain, error) {
	var d Domain
	if err := c.do(ctx, http.MethodPatch, "/domains/"+url.PathEscape(domain), patch, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// ListMailboxes returns the mailboxes on a domain.
func (c *Client) ListMailboxes(ctx context.Context, domain string) ([]Mailbox, error) {
	var wrapper struct {
		Mailboxes []Mailbox `json:"mailboxes"`
	}
	if err := c.do(ctx, http.MethodGet, "/domains/"+url.PathEscape(domain)+"/mailboxes", nil, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Mailboxes, nil
}

// MailboxSpec describes a mailbox to create.
//
// Password is deliberately absent. Mailboxes are created with Migadu's
// invitation flow so this tool never generates, transports or logs a
// credential; the recipient sets the password themselves.
type MailboxSpec struct {
	LocalPart      string `json:"local_part"`
	Name           string `json:"name,omitempty"`
	PasswordMethod string `json:"password_method"`
	RecoveryEmail  string `json:"password_recovery_email"`
	MaySend        bool   `json:"may_send"`
	MayReceive     bool   `json:"may_receive"`
	MayAccessIMAP  bool   `json:"may_access_imap"`
	MayAccessPOP3  bool   `json:"may_access_pop3"`
}

// CreateMailbox provisions a mailbox via the invitation flow.
func (c *Client) CreateMailbox(ctx context.Context, domain string, spec MailboxSpec) (*Mailbox, error) {
	if spec.PasswordMethod == "" {
		spec.PasswordMethod = "invitation"
	}
	var m Mailbox
	path := "/domains/" + url.PathEscape(domain) + "/mailboxes"
	if err := c.do(ctx, http.MethodPost, path, spec, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// AddForwarding adds a forwarding target to a mailbox. Targets inside the same
// Migadu account are confirmed automatically; external ones require the
// recipient to confirm.
func (c *Client) AddForwarding(ctx context.Context, domain, localPart, target string) (*Forwarding, error) {
	var f Forwarding
	path := fmt.Sprintf("/domains/%s/mailboxes/%s/forwardings",
		url.PathEscape(domain), url.PathEscape(localPart))
	body := map[string]interface{}{"address": target, "is_active": true}
	if err := c.do(ctx, http.MethodPost, path, body, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
