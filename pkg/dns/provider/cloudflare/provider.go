package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"zonekit/pkg/config"
	dnsprovider "zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

// providerName is the name this provider registers under and returns
// from Name().
const providerName = "cloudflare"

// Provider is zonekit's typed Cloudflare DNS provider. It implements
// dnsprovider.Provider and, additionally, dnsprovider.ZoneConfigurer.
type Provider struct {
	client     *apiClient
	accountID  string
	tokenScope TokenScope
}

// New builds a Cloudflare provider from cfg without touching the network
// or the provider registry.
func New(cfg Config) (*Provider, error) {
	c, err := newAPIClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Provider{
		client:     c,
		accountID:  cfg.AccountID,
		tokenScope: cfg.tokenScope(),
	}, nil
}

// NewFromAccount builds a Cloudflare provider from a zonekit account
// configuration.
func NewFromAccount(account *config.AccountConfig) (*Provider, error) {
	cfg, err := FromAccount(account)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// Register builds a Cloudflare provider from cfg and registers it in the
// global dnsprovider registry under the name "cloudflare".
func Register(cfg Config) (*Provider, error) {
	p, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if err := dnsprovider.Register(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{
		CanListZones:    true,
		CanGetZone:      true,
		CanCreateRecord: true,
		CanUpdateRecord: true,
		CanDeleteRecord: true,
		CanBulkReplace:  true,

		CanConfigureZoneSettings:  true,
		CanConfigureBotManagement: true,
		CanConfigureSecurityTXT:   true,
		CanConfigureRedirectRules: true,
		CanConfigureDNSSEC:        true,
	}
}

// Validate checks the provider is configured well enough to make calls.
// It does not touch the network - Cloudflare token verification
// (/accounts/{id}/tokens/verify or /user/tokens/verify) is available via
// VerifyToken for callers who want a live check.
func (p *Provider) Validate() error {
	if p.client == nil {
		return fmt.Errorf("cloudflare provider is not initialized")
	}
	if p.tokenScope == TokenScopeAccount && p.accountID == "" {
		return fmt.Errorf("cloudflare: account_id is required when token_scope is %q", TokenScopeAccount)
	}
	return nil
}

// VerifyToken checks the configured API token against Cloudflare's
// token-verification endpoint for the configured TokenScope. It is a
// live network call and is never made implicitly by Validate.
func (p *Provider) VerifyToken(ctx context.Context) error {
	path := "/user/tokens/verify"
	if p.tokenScope == TokenScopeAccount {
		if p.accountID == "" {
			return fmt.Errorf("cloudflare: account_id is required to verify an account-scoped token")
		}
		path = "/accounts/" + p.accountID + "/tokens/verify"
	}
	return p.client.get(ctx, path, nil)
}

// ListZones retrieves every zone visible to the token.
func (p *Provider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	zones, err := listPaginated[zoneWire](ctx, p.client, "/zones", nil)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: list zones: %w", err)
	}
	out := make([]dnsprovider.Zone, 0, len(zones))
	for _, z := range zones {
		out = append(out, dnsprovider.Zone{ID: z.ID, Name: z.Name})
	}
	return out, nil
}

// GetZone retrieves a zone by its Cloudflare zone ID. Use ZoneByName to
// resolve a domain name instead.
func (p *Provider) GetZone(ctx context.Context, zoneID string) (dnsprovider.Zone, error) {
	var z zoneWire
	if err := p.client.get(ctx, "/zones/"+url.PathEscape(zoneID), &z); err != nil {
		return dnsprovider.Zone{}, fmt.Errorf("cloudflare: get zone %s: %w", zoneID, err)
	}
	return dnsprovider.Zone{ID: z.ID, Name: z.Name}, nil
}

// ZoneByName resolves a zone by its domain name, e.g. "example.com".
func (p *Provider) ZoneByName(ctx context.Context, name string) (dnsprovider.Zone, error) {
	q := url.Values{"name": []string{name}}
	zones, err := listPaginated[zoneWire](ctx, p.client, "/zones", q)
	if err != nil {
		return dnsprovider.Zone{}, fmt.Errorf("cloudflare: lookup zone %q: %w", name, err)
	}
	if len(zones) == 0 {
		return dnsprovider.Zone{}, fmt.Errorf("cloudflare: zone %q not found", name)
	}
	return dnsprovider.Zone{ID: zones[0].ID, Name: zones[0].Name}, nil
}

// ListRecords retrieves every DNS record in a zone.
func (p *Provider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	wires, err := listPaginated[dnsRecordWire](ctx, p.client, "/zones/"+url.PathEscape(zoneID)+"/dns_records", nil)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: list records for zone %s: %w", zoneID, err)
	}
	out := make([]dnsrecord.Record, 0, len(wires))
	for _, w := range wires {
		out = append(out, fromWire(w))
	}
	return out, nil
}

// CreateRecord creates a new DNS record in a zone.
func (p *Provider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	req, err := toWire(record)
	if err != nil {
		return dnsrecord.Record{}, err
	}
	req.ID = ""

	var resp dnsRecordWire
	if err := p.client.post(ctx, "/zones/"+url.PathEscape(zoneID)+"/dns_records", req, &resp); err != nil {
		return dnsrecord.Record{}, fmt.Errorf("cloudflare: create record %s %s: %w", record.HostName, record.RecordType, err)
	}
	return fromWire(resp), nil
}

// UpdateRecord replaces an existing DNS record's fields.
func (p *Provider) UpdateRecord(ctx context.Context, zoneID, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	req, err := toWire(record)
	if err != nil {
		return dnsrecord.Record{}, err
	}
	req.ID = recordID

	var resp dnsRecordWire
	path := "/zones/" + url.PathEscape(zoneID) + "/dns_records/" + url.PathEscape(recordID)
	if err := p.client.put(ctx, path, req, &resp); err != nil {
		return dnsrecord.Record{}, fmt.Errorf("cloudflare: update record %s: %w", recordID, err)
	}
	return fromWire(resp), nil
}

// DeleteRecord deletes a DNS record.
func (p *Provider) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	path := "/zones/" + url.PathEscape(zoneID) + "/dns_records/" + url.PathEscape(recordID)
	if err := p.client.delete(ctx, path); err != nil {
		return fmt.Errorf("cloudflare: delete record %s: %w", recordID, err)
	}
	return nil
}

// BulkReplaceRecords reconciles a zone's DNS records to exactly the
// given set. It computes a diff against the current records and issues
// only the create/update/delete calls the diff requires - it never
// deletes everything and re-creates it, which would cause DNS
// propagation churn and a brief resolution gap for every record in the
// zone instead of just the ones that changed.
func (p *Provider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	existing, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("cloudflare: bulk replace: list existing records: %w", err)
	}

	diff := diffRecords(existing, records)

	var errs []error
	for _, r := range diff.toDelete {
		if err := p.DeleteRecord(ctx, zoneID, r.ID); err != nil {
			errs = append(errs, err)
		}
	}
	for _, r := range diff.toCreate {
		if _, err := p.CreateRecord(ctx, zoneID, r); err != nil {
			errs = append(errs, err)
		}
	}
	for _, u := range diff.toUpdate {
		if _, err := p.UpdateRecord(ctx, zoneID, u.id, u.record); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cloudflare: bulk replace zone %s: %w", zoneID, errors.Join(errs...))
	}
	return nil
}

// zoneWire is the Cloudflare zone wire shape, restricted to the fields
// this package uses.
type zoneWire struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
