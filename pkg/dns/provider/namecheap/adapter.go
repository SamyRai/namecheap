package namecheap

import (
	"context"
	"fmt"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"zonekit/pkg/client"
	dnsprovider "zonekit/pkg/dns/provider"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/errors"
	"zonekit/pkg/pointer"
)

// NamecheapProvider implements the DNS Provider interface for Namecheap
type NamecheapProvider struct {
	client NamecheapClient
}

// New creates a new Namecheap DNS provider
func New(client *client.Client) *NamecheapProvider {
	return &NamecheapProvider{
		client: NewSDKClient(client.GetNamecheapClient()),
	}
}

// Name returns the provider name
func (p *NamecheapProvider) Name() string {
	return "namecheap"
}

// Capabilities returns the provider capabilities
func (p *NamecheapProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{
		SupportsRecordID:      false,
		SupportsBulkReplace:   true,
		SupportsZoneDiscovery: true,
		SupportedRecordTypes:  []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "URL", "URL301", "FRAME"},
	}
}

// ListZones returns a list of zones
func (p *NamecheapProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	resp, err := p.client.DomainsGetList(&namecheap.DomainsGetListArgs{
		ListType: namecheap.String("ALL"),
		Page:     namecheap.Int(1),
		PageSize: namecheap.Int(100),
	})

	if err != nil {
		return nil, errors.NewAPI("ListZones", "failed to list domains", err)
	}

	if resp == nil || resp.Domains == nil {
		return []dnsprovider.Zone{}, nil
	}

	zones := make([]dnsprovider.Zone, 0, len(*resp.Domains))
	for _, d := range *resp.Domains {
		if d.Name != nil {
			zones = append(zones, dnsprovider.Zone{
				ID:   *d.Name,
				Name: *d.Name,
				Metadata: map[string]interface{}{
					"user":        pointer.String(d.User),
					"created":     safeDateString(d.Created),
					"expires":     safeDateString(d.Expires),
					"is_expired":  pointer.Bool(d.IsExpired),
					"is_locked":   pointer.Bool(d.IsLocked),
					"auto_renew":  pointer.Bool(d.AutoRenew),
					"whois_guard": pointer.String(d.WhoisGuard),
				},
			})
		}
	}

	return zones, nil
}

// GetZone returns details for a specific zone
func (p *NamecheapProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	resp, err := p.client.DomainsGetInfo(domain)
	if err != nil {
		return nil, errors.NewAPI("GetZone", fmt.Sprintf("failed to get info for %s", domain), err)
	}

	if resp == nil || resp.DomainDNSGetListResult == nil {
		return nil, errors.NewNotFound("Zone", domain)
	}

	domainName := pointer.String(resp.DomainDNSGetListResult.DomainName)
	z := &dnsprovider.Zone{
		ID:   domainName,
		Name: domainName,
		Metadata: map[string]interface{}{
			"is_premium": pointer.Bool(resp.DomainDNSGetListResult.IsPremium),
			"is_our_dns": pointer.Bool(resp.DomainDNSGetListResult.DnsDetails.IsUsingOurDNS),
		},
	}

	return z, nil
}

// ListRecords retrieves all DNS records for a zone
func (p *NamecheapProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	resp, err := p.client.DomainsDNSGetHosts(zoneID)
	if err != nil {
		return nil, errors.NewAPI("ListRecords", fmt.Sprintf("failed to get DNS records for %s", zoneID), err)
	}

	if resp == nil || resp.DomainDNSGetHostsResult == nil || resp.DomainDNSGetHostsResult.Hosts == nil {
		return []dnsrecord.Record{}, nil
	}

	records := make([]dnsrecord.Record, 0, len(*resp.DomainDNSGetHostsResult.Hosts))
	for _, host := range *resp.DomainDNSGetHostsResult.Hosts {
		records = append(records, convertToRecord(host))
	}

	return records, nil
}

// CreateRecord creates a new record in the zone
func (p *NamecheapProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	existing, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	existing = append(existing, record)

	if err := p.BulkReplaceRecords(ctx, zoneID, existing); err != nil {
		return nil, err
	}

	return &record, nil
}

// UpdateRecord updates an existing record
func (p *NamecheapProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	existing, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	matchIndex := -1
	matchCount := 0

	for i, r := range existing {
		if r.HostName == record.HostName && r.RecordType == record.RecordType {
			matchIndex = i
			matchCount++
		}
	}

	if matchCount == 0 {
		return nil, errors.NewNotFound("Record", fmt.Sprintf("%s %s", record.HostName, record.RecordType))
	}
	if matchCount > 1 {
		return nil, fmt.Errorf("ambiguous update: multiple records found for %s %s, cannot identify which to update without ID", record.HostName, record.RecordType)
	}

	existing[matchIndex] = record

	if err := p.BulkReplaceRecords(ctx, zoneID, existing); err != nil {
		return nil, err
	}

	return &record, nil
}

// DeleteRecord deletes a record
func (p *NamecheapProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return fmt.Errorf("DeleteRecord by ID is not supported by Namecheap (no persistent record IDs)")
}

// BulkReplaceRecords replaces all records in a zone
func (p *NamecheapProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	hostRecords := make([]namecheap.DomainsDNSHostRecord, len(records))
	hasMXRecords := false
	for i, record := range records {
		hostRecord := namecheap.DomainsDNSHostRecord{
			HostName:   namecheap.String(record.HostName),
			RecordType: namecheap.String(record.RecordType),
			Address:    namecheap.String(record.Address),
		}

		if record.TTL > 0 {
			hostRecord.TTL = namecheap.Int(record.TTL)
		}

		if record.MXPref > 0 {
			hostRecord.MXPref = namecheap.UInt8(uint8(record.MXPref))
		}

		if record.RecordType == dnsrecord.RecordTypeMX {
			hasMXRecords = true
		}

		hostRecords[i] = hostRecord
	}

	args := &namecheap.DomainsDNSSetHostsArgs{
		Domain:  namecheap.String(zoneID),
		Records: &hostRecords,
	}

	if hasMXRecords {
		args.EmailType = namecheap.String("MX")
	}

	_, err := p.client.DomainsDNSSetHosts(args)
	if err != nil {
		return errors.NewAPI("SetHosts", fmt.Sprintf("failed to set DNS records for %s", zoneID), err)
	}

	return nil
}

// Validate checks if the provider is properly configured
func (p *NamecheapProvider) Validate() error {
	if p.client == nil {
		return fmt.Errorf("namecheap client is not initialized")
	}
	return nil
}

// Register registers the Namecheap provider
func Register(client *client.Client) error {
	provider := New(client)
	return dnsprovider.Register(provider)
}

// Helper to convert Namecheap host to our Record
func convertToRecord(host namecheap.DomainsDNSHostRecordDetailed) dnsrecord.Record {
	return dnsrecord.Record{
		HostName:   pointer.String(host.Name),
		RecordType: pointer.String(host.Type),
		Address:    pointer.String(host.Address),
		TTL:        pointer.Int(host.TTL),
		MXPref:     pointer.Int(host.MXPref),

		Metadata: map[string]interface{}{
			"is_active": pointer.Bool(host.IsActive),
			"host_id":   pointer.Int(host.HostId),
		},
	}
}

func safeDateString(dt *namecheap.DateTime) string {
	if dt == nil {
		return ""
	}
	return dt.String()
}
