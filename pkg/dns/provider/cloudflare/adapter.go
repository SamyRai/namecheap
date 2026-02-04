package cloudflare

import (
	"context"
	"fmt"

	dnsprovider "zonekit/pkg/dns/provider"
	httpprovider "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	rest "zonekit/pkg/dns/provider/rest"
	"zonekit/pkg/dnsrecord"
)

// New creates a Cloudflare provider backed by the generic REST adapter
func New(client *httpprovider.Client) *RESTCloudflareProvider {
	m := mapper.DefaultMappings()
	// Cloudflare responses are wrapped in `result` and the record fields are: id, name, type, content
	m.ListPath = "result"
	m.Response.ID = "id"
	m.Response.HostName = "name"
	m.Response.RecordType = "type"
	m.Response.Address = "content"

	m.Request.ID = "id"
	m.Request.HostName = "name"
	m.Request.RecordType = "type"
	m.Request.Address = "content"
	// SRV fields
	m.Request.Priority = "priority"
	m.Request.Weight = "weight"
	m.Request.Port = "port"
	m.Request.Target = "target"

	m.Response.Priority = "priority"
	m.Response.Weight = "weight"
	m.Response.Port = "port"
	m.Response.Target = "target"

	endpoints := map[string]string{
		"list_zones":    "/zones",
		"get_records":   "/zones/{zone_id}/dns_records",
		"create_record": "/zones/{zone_id}/dns_records",
		"update_record": "/zones/{zone_id}/dns_records/{id}",
		"delete_record": "/zones/{zone_id}/dns_records/{id}",
	}

	restProv := rest.NewRESTProvider("cloudflare", client, m, endpoints, nil)
	return &RESTCloudflareProvider{rest: restProv}
}

// RESTCloudflareProvider is a thin wrapper around the REST provider
type RESTCloudflareProvider struct {
	rest *rest.RESTProvider
}

// Ensure interface
var _ dnsprovider.Provider = (*RESTCloudflareProvider)(nil)

func (p *RESTCloudflareProvider) Name() string {
	return p.rest.Name()
}

func (p *RESTCloudflareProvider) Capabilities() dnsprovider.ProviderCapabilities {
	return p.rest.Capabilities()
}

func (p *RESTCloudflareProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	return p.rest.ListZones(ctx)
}

func (p *RESTCloudflareProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	return p.rest.GetZone(ctx, domain)
}

func (p *RESTCloudflareProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	return p.rest.ListRecords(ctx, zoneID)
}

func (p *RESTCloudflareProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	return p.rest.CreateRecord(ctx, zoneID, record)
}

func (p *RESTCloudflareProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	return p.rest.UpdateRecord(ctx, zoneID, recordID, record)
}

func (p *RESTCloudflareProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	return p.rest.DeleteRecord(ctx, zoneID, recordID)
}

func (p *RESTCloudflareProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	return p.rest.BulkReplaceRecords(ctx, zoneID, records)
}

// Register registers the Cloudflare provider using a configured HTTP client
func Register(client *httpprovider.Client) error {
	if client == nil {
		return fmt.Errorf("http client is nil")
	}
	prov := New(client)
	return dnsprovider.Register(prov)
}

// Validate helper
func (p *RESTCloudflareProvider) Validate() error {
	if p == nil || p.rest == nil {
		return fmt.Errorf("provider not initialized")
	}
	return nil
}
