package godaddy

import (
	"context"
	"fmt"

	dnsprovider "zonekit/pkg/dns/provider"
	httpprovider "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	rest "zonekit/pkg/dns/provider/rest"
	"zonekit/pkg/dnsrecord"
)

// New creates a GoDaddy provider
func New(client *httpprovider.Client) *RESTGoDaddyProvider {
	m := mapper.DefaultMappings()
	m.ListPath = "" // Root array

	m.Response.ID = "" // No ID support
	m.Response.HostName = "name"
	m.Response.RecordType = "type"
	m.Response.Address = "data"
	m.Response.TTL = "ttl"
	m.Response.Priority = "priority"
	m.Response.Port = "port"
	m.Response.Weight = "weight"
	m.Response.MXPref = "priority" // GoDaddy uses priority for MX

	m.Request.ID = ""
	m.Request.HostName = "name"
	m.Request.RecordType = "type"
	m.Request.Address = "data"
	m.Request.TTL = "ttl"
	m.Request.Priority = "priority"
	m.Request.Port = "port"
	m.Request.Weight = "weight"
	m.Request.MXPref = "priority"

	endpoints := map[string]string{
		"list_zones":    "/domains",
		"get_records":   "/domains/{domain}/records",
		"create_record": "/domains/{domain}/records",
		// update/delete not supported individually via ID
	}

	restProv := rest.NewRESTProvider("godaddy", client, m, endpoints, nil)
	return &RESTGoDaddyProvider{rest: restProv, client: client}
}

// RESTGoDaddyProvider is a wrapper around the REST provider
type RESTGoDaddyProvider struct {
	rest   *rest.RESTProvider
	client *httpprovider.Client
}

// Ensure interface
var _ dnsprovider.Provider = (*RESTGoDaddyProvider)(nil)

func (p *RESTGoDaddyProvider) Name() string {
	return p.rest.Name()
}

func (p *RESTGoDaddyProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return dnsprovider.ProviderCapabilities{
		SupportsRecordID:      false,
		SupportsBulkReplace:   true,
		SupportsZoneDiscovery: true,
		SupportedRecordTypes:  []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"},
	}
}

func (p *RESTGoDaddyProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	// Custom implementation to parse /domains response
	// GoDaddy returns array of objects: [{"domainId":..., "domain":"example.com", ...}]
	resp, err := p.client.Get(ctx, "/domains", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract root array
	zoneMaps, err := mapper.ExtractRecords(responseData, "")
	if err != nil {
		return nil, fmt.Errorf("failed to extract zones: %w", err)
	}

	zones := make([]dnsprovider.Zone, 0, len(zoneMaps))
	for _, zm := range zoneMaps {
		// GoDaddy uses "domain" for the name
		name, _ := zm["domain"].(string)
		if name != "" {
			zones = append(zones, dnsprovider.Zone{
				ID:       name, // Use domain name as ID
				Name:     name,
				Metadata: zm,
			})
		}
	}

	return zones, nil
}

func (p *RESTGoDaddyProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	// API: GET /v1/domains/{domain}
	resp, err := p.client.Get(ctx, fmt.Sprintf("/domains/%s", domain), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone %s: %w", domain, err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	zm, ok := responseData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	name, _ := zm["domain"].(string)
	return &dnsprovider.Zone{
		ID:       name,
		Name:     name,
		Metadata: zm,
	}, nil
}

func (p *RESTGoDaddyProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	return p.rest.ListRecords(ctx, zoneID)
}

func (p *RESTGoDaddyProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	// GoDaddy expects array of records for POST
	endpoint := fmt.Sprintf("/domains/%s/records", zoneID)

	// Wrap single record in array
	bodyData := mapper.ToProviderFormat(record, p.rest.Mappings().Request)
	body := []map[string]interface{}{bodyData}

	resp, err := p.client.Post(ctx, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	defer resp.Body.Close()

	// GoDaddy returns empty body on success
	return &record, nil
}

func (p *RESTGoDaddyProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	return nil, fmt.Errorf("UpdateRecord by ID is not supported by GoDaddy")
}

func (p *RESTGoDaddyProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return fmt.Errorf("DeleteRecord by ID is not supported by GoDaddy")
}

func (p *RESTGoDaddyProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	endpoint := fmt.Sprintf("/domains/%s/records", zoneID)

	body := make([]map[string]interface{}, len(records))
	for i, r := range records {
		body[i] = mapper.ToProviderFormat(r, p.rest.Mappings().Request)
	}

	// Use PUT to replace all records
	resp, err := p.client.Put(ctx, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to replace records: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// Register registers the GoDaddy provider
func Register(client *httpprovider.Client) error {
	if client == nil {
		return fmt.Errorf("http client is nil")
	}
	prov := New(client)
	return dnsprovider.Register(prov)
}

// Validate helper
func (p *RESTGoDaddyProvider) Validate() error {
	if p == nil || p.rest == nil {
		return fmt.Errorf("provider not initialized")
	}
	return nil
}
