package namecheap

import (
	"fmt"

	"github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"zonekit/pkg/client"
	"zonekit/pkg/dnsrecord"
	dnsprovider "zonekit/pkg/dns/provider"
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

// GetRecords retrieves all DNS records for a domain
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

// SetRecords sets DNS records for a domain (replaces all existing records)
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

