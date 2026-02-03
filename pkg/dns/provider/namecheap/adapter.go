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

// Capabilities returns the provider capabilities
func (p *NamecheapProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{}
}

// ListZones returns a list of zones
func (p *NamecheapProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetZone returns details for a specific zone
func (p *NamecheapProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListRecords retrieves all DNS records for a zone
func (p *NamecheapProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	return p.GetRecords(zoneID)
}

// CreateRecord creates a new record in the zone
func (p *NamecheapProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdateRecord updates an existing record
func (p *NamecheapProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	return nil, fmt.Errorf("not implemented")
}

// DeleteRecord deletes a record
func (p *NamecheapProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return fmt.Errorf("not implemented")
}

// BulkReplaceRecords replaces all records in a zone
func (p *NamecheapProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	return p.SetRecords(zoneID, records)
}

// GetRecords retrieves all DNS records for a domain (Legacy helper)
func (p *NamecheapProvider) GetRecords(domainName string) ([]dnsrecord.Record, error) {
	nc := p.client.GetNamecheapClient()

	resp, err := nc.DomainsDNS.GetHosts(domainName)
	if err != nil {
		return nil, errors.NewAPI("GetHosts", fmt.Sprintf("failed to get DNS records for %s", domainName), err)
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

// SetRecords sets DNS records for a domain (replaces all existing records) (Legacy helper)
func (p *NamecheapProvider) SetRecords(domainName string, records []dnsrecord.Record) error {
	nc := p.client.GetNamecheapClient()

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
	// This is required by the Namecheap API when MX records are present
	if hasMXRecords {
		args.EmailType = namecheap.String("MX")
	}

	_, err := nc.DomainsDNS.SetHosts(args)
	if err != nil {
		return errors.NewAPI("SetHosts", fmt.Sprintf("failed to set DNS records for %s", domainName), err)
	}

	return nil
}

// Validate checks if the provider is properly configured (Legacy)
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
