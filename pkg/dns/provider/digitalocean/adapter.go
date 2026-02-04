package digitalocean

import (
	"context"
	"fmt"

	dnsprovider "zonekit/pkg/dns/provider"
	httpprovider "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	rest "zonekit/pkg/dns/provider/rest"
	"zonekit/pkg/dnsrecord"
)

// New creates a DigitalOcean provider backed by the generic REST adapter
func New(client *httpprovider.Client) *RESTDigitalOceanProvider {
	m := mapper.DefaultMappings()

	// DigitalOcean mappings for records
	m.ListPath = "domain_records"

	m.Response.ID = "id"
	m.Response.HostName = "name"
	m.Response.RecordType = "type"
	m.Response.Address = "data"
	m.Response.Priority = "priority"
	m.Response.Port = "port"
	m.Response.Weight = "weight"
	m.Response.TTL = "ttl"

	m.Request.ID = "id"
	m.Request.HostName = "name"
	m.Request.RecordType = "type"
	m.Request.Address = "data"
	m.Request.Priority = "priority"
	m.Request.Port = "port"
	m.Request.Weight = "weight"
	m.Request.TTL = "ttl"

	endpoints := map[string]string{
		"list_zones":    "/domains",
		"get_records":   "/domains/{zone_id}/records",
		"create_record": "/domains/{zone_id}/records",
		"update_record": "/domains/{zone_id}/records/{id}",
		"delete_record": "/domains/{zone_id}/records/{id}",
	}

	restProv := rest.NewRESTProvider("digitalocean", client, m, endpoints, nil)
	return &RESTDigitalOceanProvider{rest: restProv, client: client}
}

// RESTDigitalOceanProvider is a thin wrapper around the REST provider
type RESTDigitalOceanProvider struct {
	rest   *rest.RESTProvider
	client *httpprovider.Client
}

// Ensure interface
var _ dnsprovider.Provider = (*RESTDigitalOceanProvider)(nil)

func (p *RESTDigitalOceanProvider) Name() string {
	return p.rest.Name()
}

func (p *RESTDigitalOceanProvider) Capabilities() dnsprovider.ProviderCapabilities {
	// DigitalOcean supports record IDs and zone discovery
	return dnsprovider.ProviderCapabilities{
		SupportsRecordID:      true,
		SupportsZoneDiscovery: true,
		SupportedRecordTypes:  []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"},
	}
}

func (p *RESTDigitalOceanProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	// Custom implementation because ListPath differs for zones ("domains") vs records ("domain_records")
	resp, err := p.client.Get(ctx, "/domains", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	zoneMaps, err := mapper.ExtractRecords(responseData, "domains")
	if err != nil {
		return nil, fmt.Errorf("failed to extract zones: %w", err)
	}

	zones := make([]dnsprovider.Zone, 0, len(zoneMaps))
	for _, zm := range zoneMaps {
		name, _ := zm["name"].(string)
		if name != "" {
			zones = append(zones, dnsprovider.Zone{
				ID:       name, // DigitalOcean uses domain name as ID
				Name:     name,
				Metadata: zm,
			})
		}
	}

	return zones, nil
}

func (p *RESTDigitalOceanProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	// We can use the generic one which calls ListZones, but specialized implementation is more efficient
	resp, err := p.client.Get(ctx, fmt.Sprintf("/domains/%s", domain), nil)
	if err != nil {
		// Generic REST provider returns Not Found if API returns 404
		// But client.Get returns error on non-2xx usually?
		// Check client implementation if needed. For now assume error means failure.
		return nil, fmt.Errorf("failed to get zone %s: %w", domain, err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Response is { "domain": { ... } }
	var zoneMap map[string]interface{}
	if m, ok := responseData.(map[string]interface{}); ok {
		if z, ok := m["domain"]; ok {
			if zMap, ok := z.(map[string]interface{}); ok {
				zoneMap = zMap
			}
		}
	}

	if zoneMap == nil {
		return nil, fmt.Errorf("failed to extract zone data")
	}

	name, _ := zoneMap["name"].(string)

	return &dnsprovider.Zone{
		ID:       name,
		Name:     name,
		Metadata: zoneMap,
	}, nil
}

func (p *RESTDigitalOceanProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	records, err := p.rest.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	// Fix up SRV targets (data field -> Target)
	for i := range records {
		if records[i].RecordType == "SRV" && records[i].Target == "" && records[i].Address != "" {
			records[i].Target = records[i].Address
		}
	}
	return records, nil
}

func (p *RESTDigitalOceanProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	// Custom implementation to handle "domain_record" response wrapper
	endpoint := fmt.Sprintf("/domains/%s/records", zoneID)

	// DigitalOcean uses 'data' for everything. For SRV, target is in 'data'.
	// If Address is empty but Target is set (SRV), copy Target to Address.
	if record.RecordType == "SRV" && record.Address == "" && record.Target != "" {
		record.Address = record.Target
	}

	body := mapper.ToProviderFormat(record, p.rest.Mappings().Request)

	resp, err := p.client.Post(ctx, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer resp.Body.Close()

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		// Return best effort
		return &record, nil
	}

	// Unwrap "domain_record"
	var recordData map[string]interface{}
	if m, ok := responseData.(map[string]interface{}); ok {
		if r, ok := m["domain_record"]; ok {
			if rMap, ok := r.(map[string]interface{}); ok {
				recordData = rMap
			}
		}
	}

	if recordData != nil {
		created, err := mapper.FromProviderFormat(recordData, p.rest.Mappings().Response)
		if err == nil {
			// Fix up SRV target
			if created.RecordType == "SRV" && created.Target == "" && created.Address != "" {
				created.Target = created.Address
			}
			return &created, nil
		}
	}

	return &record, nil
}

func (p *RESTDigitalOceanProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	// Custom implementation to handle "domain_record" response wrapper
	endpoint := fmt.Sprintf("/domains/%s/records/%s", zoneID, recordID)
	record.ID = recordID

	// DigitalOcean uses 'data' for everything. For SRV, target is in 'data'.
	// If Address is empty but Target is set (SRV), copy Target to Address.
	if record.RecordType == "SRV" && record.Address == "" && record.Target != "" {
		record.Address = record.Target
	}

	body := mapper.ToProviderFormat(record, p.rest.Mappings().Request)

	resp, err := p.client.Put(ctx, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	defer resp.Body.Close()

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return &record, nil
	}

	// Unwrap "domain_record"
	var recordData map[string]interface{}
	if m, ok := responseData.(map[string]interface{}); ok {
		if r, ok := m["domain_record"]; ok {
			if rMap, ok := r.(map[string]interface{}); ok {
				recordData = rMap
			}
		}
	}

	if recordData != nil {
		updated, err := mapper.FromProviderFormat(recordData, p.rest.Mappings().Response)
		if err == nil {
			// Fix up SRV target
			if updated.RecordType == "SRV" && updated.Target == "" && updated.Address != "" {
				updated.Target = updated.Address
			}
			return &updated, nil
		}
	}

	return &record, nil
}

func (p *RESTDigitalOceanProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return p.rest.DeleteRecord(ctx, zoneID, recordID)
}

func (p *RESTDigitalOceanProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	return p.rest.BulkReplaceRecords(ctx, zoneID, records)
}

// Register registers the DigitalOcean provider using a configured HTTP client
func Register(client *httpprovider.Client) error {
	if client == nil {
		return fmt.Errorf("http client is nil")
	}
	prov := New(client)
	return dnsprovider.Register(prov)
}

// Validate helper
func (p *RESTDigitalOceanProvider) Validate() error {
	if p == nil || p.rest == nil {
		return fmt.Errorf("provider not initialized")
	}
	return nil
}
