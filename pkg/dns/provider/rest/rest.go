package rest

import (
	"context"
	"fmt"
	"strings"

	dnsprovider "zonekit/pkg/dns/provider"
	httpprovider "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	"zonekit/pkg/dnsrecord"
	"zonekit/pkg/errors"
)

// RESTProvider is a generic REST-based DNS provider
type RESTProvider struct {
	name      string
	client    *httpprovider.Client
	mappings  mapper.Mappings
	endpoints map[string]string
	settings  map[string]interface{}
}

// NewRESTProvider creates a new REST-based DNS provider
func NewRESTProvider(
	name string,
	client *httpprovider.Client,
	mappings mapper.Mappings,
	endpoints map[string]string,
	settings map[string]interface{},
) *RESTProvider {
	return &RESTProvider{
		name:      name,
		client:    client,
		mappings:  mappings,
		endpoints: endpoints,
		settings:  settings,
	}
}

// Name returns the provider name
func (p *RESTProvider) Name() string {
	return p.name
}

// ListZones retrieves all zones managed by the provider
func (p *RESTProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	// check for list_zones or zones endpoint
	endpoint, ok := p.endpoints["list_zones"]
	if !ok {
		endpoint, ok = p.endpoints["zones"]
	}
	if !ok {
		// If no endpoint, return empty list (stub behavior)
		return []dnsprovider.Zone{}, nil
	}

	resp, err := p.client.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPI("ListZones", "failed to list zones", err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// TODO: Implement zone mapping
	return []dnsprovider.Zone{}, nil
}

// GetZone retrieves a specific zone by ID
func (p *RESTProvider) GetZone(ctx context.Context, zoneID string) (dnsprovider.Zone, error) {
	// Stub: return a zone with the ID and Name = ID
	return dnsprovider.Zone{ID: zoneID, Name: zoneID}, nil
}

// ListRecords retrieves all DNS records for a zone
func (p *RESTProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["get_records"]
	if !ok {
		endpoint, ok = p.endpoints["list_records"]
	}
	if !ok {
		return nil, fmt.Errorf("get_records endpoint not configured")
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)

	resp, err := p.client.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPI("ListRecords", fmt.Sprintf("failed to get DNS records for zone %s", zoneID), err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	recordMaps, err := mapper.ExtractRecords(responseData, p.mappings.ListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract records: %w", err)
	}

	records := make([]dnsrecord.Record, 0, len(recordMaps))
	for _, recordMap := range recordMaps {
		record, err := mapper.FromProviderFormat(recordMap, p.mappings.Response)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

// CreateRecord creates a new DNS record
func (p *RESTProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["create_record"]
	if !ok {
		return dnsrecord.Record{}, fmt.Errorf("create_record endpoint not configured")
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)

	body := mapper.ToProviderFormat(record, p.mappings.Request)

	resp, err := p.client.Post(ctx, endpoint, body)
	if err != nil {
		return dnsrecord.Record{}, errors.NewAPI("CreateRecord", "failed to create DNS record", err)
	}
	defer resp.Body.Close()

	// TODO: Parse response to get ID
	return record, nil
}

// UpdateRecord updates an existing DNS record
func (p *RESTProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["update_record"]
	if !ok {
		return dnsrecord.Record{}, fmt.Errorf("update_record endpoint not configured")
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{record_id}", recordID)
	endpoint = strings.ReplaceAll(endpoint, "{id}", recordID)

	body := mapper.ToProviderFormat(record, p.mappings.Request)

	resp, err := p.client.Put(ctx, endpoint, body)
	if err != nil {
		return dnsrecord.Record{}, errors.NewAPI("UpdateRecord", "failed to update DNS record", err)
	}
	defer resp.Body.Close()

	return record, nil
}

// DeleteRecord deletes a DNS record
func (p *RESTProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	endpoint, ok := p.endpoints["delete_record"]
	if !ok {
		return fmt.Errorf("delete_record endpoint not configured")
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{record_id}", recordID)
	endpoint = strings.ReplaceAll(endpoint, "{id}", recordID)

	resp, err := p.client.Delete(ctx, endpoint)
	if err != nil {
		return errors.NewAPI("DeleteRecord", "failed to delete DNS record", err)
	}
	defer resp.Body.Close()

	return nil
}

// BulkReplaceRecords replaces all records in a zone with the provided set
func (p *RESTProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	// Naive implementation
	existing, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}

	for _, r := range existing {
		if r.ID != "" {
			_ = p.DeleteRecord(ctx, zoneID, r.ID)
		}
	}

	for _, r := range records {
		_, err := p.CreateRecord(ctx, zoneID, r)
		if err != nil {
			return err
		}
	}
	return nil
}

// Capabilities returns the provider's capabilities
func (p *RESTProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{
		CanListZones:    p.hasEndpoint("list_zones") || p.hasEndpoint("zones"),
		CanGetZone:      true,
		CanCreateRecord: p.hasEndpoint("create_record"),
		CanUpdateRecord: p.hasEndpoint("update_record"),
		CanDeleteRecord: p.hasEndpoint("delete_record"),
		CanBulkReplace:  true,
	}
}

func (p *RESTProvider) hasEndpoint(name string) bool {
	_, ok := p.endpoints[name]
	return ok
}

// Validate checks if the provider is properly configured
func (p *RESTProvider) Validate() error {
	if p.client == nil {
		return fmt.Errorf("HTTP client is not initialized")
	}
	if p.name == "" {
		return fmt.Errorf("provider name is empty")
	}
	if len(p.endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	return nil
}

var _ dnsprovider.Provider = (*RESTProvider)(nil)
