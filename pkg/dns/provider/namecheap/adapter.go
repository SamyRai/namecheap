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
	client *client.Client
}

// New creates a new Namecheap DNS provider
func New(client *client.Client) *NamecheapProvider {
	return &NamecheapProvider{
		client: client,
	}
}

// Name returns the provider name
func (p *NamecheapProvider) Name() string {
	return "namecheap"
}

// ListZones retrieves all zones managed by the provider
func (p *NamecheapProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	nc := p.client.GetNamecheapClient()
	res, err := nc.Domains.GetList(&namecheap.DomainsGetListArgs{})
	if err != nil {
		return nil, errors.NewAPI("ListZones", "failed to list zones", err)
	}

	if res == nil || res.Domains == nil {
		return []dnsprovider.Zone{}, nil
	}

	var zones []dnsprovider.Zone
	for _, d := range *res.Domains {
		zones = append(zones, dnsprovider.Zone{
			ID:   *d.Name,
			Name: *d.Name,
		})
	}
	return zones, nil
}

// GetZone retrieves a specific zone by ID
func (p *NamecheapProvider) GetZone(ctx context.Context, zoneID string) (dnsprovider.Zone, error) {
	nc := p.client.GetNamecheapClient()
	res, err := nc.Domains.GetInfo(zoneID)
	if err != nil {
		return dnsprovider.Zone{}, errors.NewAPI("GetZone", fmt.Sprintf("failed to get zone %s", zoneID), err)
	}
	if res == nil || res.DomainDNSGetListResult == nil || res.DomainDNSGetListResult.DomainName == nil {
		return dnsprovider.Zone{}, fmt.Errorf("zone not found")
	}

	return dnsprovider.Zone{
		ID:   *res.DomainDNSGetListResult.DomainName,
		Name: *res.DomainDNSGetListResult.DomainName,
	}, nil
}

// ListRecords retrieves all DNS records for a zone
func (p *NamecheapProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	nc := p.client.GetNamecheapClient()

	resp, err := nc.DomainsDNS.GetHosts(zoneID)
	if err != nil {
		return nil, errors.NewAPI("GetHosts", fmt.Sprintf("failed to get DNS records for %s", zoneID), err)
	}

	// Safety check for nil response
	if resp == nil || resp.DomainDNSGetHostsResult == nil || resp.DomainDNSGetHostsResult.Hosts == nil {
		return []dnsrecord.Record{}, nil
	}

	records := make([]dnsrecord.Record, 0, len(*resp.DomainDNSGetHostsResult.Hosts))
	for _, host := range *resp.DomainDNSGetHostsResult.Hosts {
		record := dnsrecord.Record{
			HostName:   pointer.String(host.Name),
			RecordType: pointer.String(host.Type),
			Address:    pointer.String(host.Address),
			TTL:        pointer.Int(host.TTL),
			MXPref:     pointer.Int(host.MXPref),
		}
		records = append(records, record)
	}

	return records, nil
}

// CreateRecord creates a new DNS record
func (p *NamecheapProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	records, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return dnsrecord.Record{}, err
	}

	records = append(records, record)

	if err := p.BulkReplaceRecords(ctx, zoneID, records); err != nil {
		return dnsrecord.Record{}, err
	}

	return record, nil
}

// UpdateRecord updates an existing DNS record
func (p *NamecheapProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	// Stub: Namecheap doesn't support granular update easily without ID
	return dnsrecord.Record{}, fmt.Errorf("UpdateRecord not implemented for Namecheap (requires BulkReplace)")
}

// DeleteRecord deletes a DNS record
func (p *NamecheapProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	// Stub
	return fmt.Errorf("DeleteRecord not implemented for Namecheap (requires BulkReplace)")
}

// BulkReplaceRecords sets DNS records for a domain (replaces all existing records)
func (p *NamecheapProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	nc := p.client.GetNamecheapClient()
	domainName := zoneID

	// Convert records to Namecheap format
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

		// Check if this is an MX record
		if record.RecordType == dnsrecord.RecordTypeMX {
			hasMXRecords = true
		}

		hostRecords[i] = hostRecord
	}

	// Build SetHostsArgs
	args := &namecheap.DomainsDNSSetHostsArgs{
		Domain:  namecheap.String(domainName),
		Records: &hostRecords,
	}

	// Set EmailType to MX if there are any MX records
	if hasMXRecords {
		args.EmailType = namecheap.String("MX")
	}

	_, err := nc.DomainsDNS.SetHosts(args)
	if err != nil {
		return errors.NewAPI("SetHosts", fmt.Sprintf("failed to set DNS records for %s", domainName), err)
	}

	return nil
}

// Capabilities returns the provider's capabilities
func (p *NamecheapProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{
		CanListZones:    true,
		CanGetZone:      true,
		CanCreateRecord: true,
		CanUpdateRecord: false,
		CanDeleteRecord: false,
		CanBulkReplace:  true,
	}
}

// Validate checks if the provider is properly configured
func (p *NamecheapProvider) Validate() error {
	if p.client == nil {
		return fmt.Errorf("namecheap client is not initialized")
	}
	// Additional validation can be added here
	return nil
}

// Register registers the Namecheap provider
func Register(client *client.Client) error {
	provider := New(client)
	return dnsprovider.Register(provider)
}
