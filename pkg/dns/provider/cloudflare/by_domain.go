package cloudflare

import (
	"context"
	"regexp"
	"sync"

	dnsprovider "zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
)

var zoneIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// DomainAddressed adapts a Provider so callers can pass a domain name wherever
// the Provider interface expects a zone ID, as the Namecheap-shaped DNS service
// and CLI do. A 32-hex argument is used as-is; anything else is resolved with
// ZoneByName and cached for the adapter's lifetime.
type DomainAddressed struct {
	*Provider
	mu    sync.Mutex
	cache map[string]string
}

// ByDomain wraps p so that zone arguments may be domain names.
func ByDomain(p *Provider) *DomainAddressed {
	return &DomainAddressed{Provider: p, cache: map[string]string{}}
}

func (d *DomainAddressed) zoneID(ctx context.Context, zone string) (string, error) {
	if zoneIDPattern.MatchString(zone) {
		return zone, nil
	}
	d.mu.Lock()
	id, ok := d.cache[zone]
	d.mu.Unlock()
	if ok {
		return id, nil
	}
	z, err := d.Provider.ZoneByName(ctx, zone)
	if err != nil {
		return "", err
	}
	d.mu.Lock()
	d.cache[zone] = z.ID
	d.mu.Unlock()
	return z.ID, nil
}

// GetZone resolves zone and delegates.
func (d *DomainAddressed) GetZone(ctx context.Context, zone string) (dnsprovider.Zone, error) {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return dnsprovider.Zone{}, err
	}
	return d.Provider.GetZone(ctx, id)
}

// ListRecords resolves zone and delegates.
func (d *DomainAddressed) ListRecords(ctx context.Context, zone string) ([]dnsrecord.Record, error) {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return nil, err
	}
	return d.Provider.ListRecords(ctx, id)
}

// CreateRecord resolves zone and delegates.
func (d *DomainAddressed) CreateRecord(ctx context.Context, zone string, record dnsrecord.Record) (dnsrecord.Record, error) {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return dnsrecord.Record{}, err
	}
	return d.Provider.CreateRecord(ctx, id, record)
}

// UpdateRecord resolves zone and delegates.
func (d *DomainAddressed) UpdateRecord(ctx context.Context, zone, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return dnsrecord.Record{}, err
	}
	return d.Provider.UpdateRecord(ctx, id, recordID, record)
}

// DeleteRecord resolves zone and delegates.
func (d *DomainAddressed) DeleteRecord(ctx context.Context, zone, recordID string) error {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return err
	}
	return d.Provider.DeleteRecord(ctx, id, recordID)
}

// BulkReplaceRecords resolves zone and delegates.
func (d *DomainAddressed) BulkReplaceRecords(ctx context.Context, zone string, records []dnsrecord.Record) error {
	id, err := d.zoneID(ctx, zone)
	if err != nil {
		return err
	}
	return d.Provider.BulkReplaceRecords(ctx, id, records)
}

var _ dnsprovider.Provider = (*DomainAddressed)(nil)
