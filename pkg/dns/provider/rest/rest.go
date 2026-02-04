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

// Mappings returns the provider mappings
func (p *RESTProvider) Mappings() mapper.Mappings {
	return p.mappings
}

// Capabilities returns the provider capabilities
func (p *RESTProvider) Capabilities() dnsprovider.ProviderCapabilities {
	// We assume REST providers support Record ID if mappings include ID
	supportsRecordID := p.mappings.Response.ID != ""

	return dnsprovider.ProviderCapabilities{
		SupportsRecordID:      supportsRecordID,
		SupportsZoneDiscovery: len(p.endpoints) > 0, // Rudimentary check
		SupportedRecordTypes:  []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV"}, // Generic assumption
	}
}

// ListZones returns a list of zones
func (p *RESTProvider) ListZones(ctx context.Context) ([]dnsprovider.Zone, error) {
	// Try configured endpoints that may list zones
	candidates := []string{"list_zones", "zones", "get_zones"}
	var endpoint string
	for _, key := range candidates {
		if path, ok := p.endpoints[key]; ok && path != "" {
			endpoint = path
			break
		}
	}

	if endpoint == "" {
		return nil, fmt.Errorf("list_zones endpoint not configured")
	}

	resp, err := p.client.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPI("ListZones", "failed to list zones", err)
	}

	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// We don't have a specific mapping for Zones yet, so we use heuristics
	// or look for "result" / "data" containing arrays
	// For now, let's reuse ExtractRecords logic but for zones?
	// Or just look for an array in likely places

	// Assumption: Response structure for zones is similar to records
	zoneMaps, err := mapper.ExtractRecords(responseData, p.mappings.ListPath)
	if err != nil {
		// Try root if default path fails
		zoneMaps, err = mapper.ExtractRecords(responseData, "")
		if err != nil {
			return nil, fmt.Errorf("failed to extract zones: %w", err)
		}
	}

	zones := make([]dnsprovider.Zone, 0, len(zoneMaps))
	for _, zm := range zoneMaps {
		// Heuristics to find ID and Name
		id := extractField(zm, []string{"id", "zone_id", "uuid", "_id"})
		name := extractField(zm, []string{"name", "domain", "zone_name", "host"})

		if id != "" && name != "" {
			zones = append(zones, dnsprovider.Zone{
				ID:   id,
				Name: strings.TrimSuffix(name, "."), // Normalize
				Metadata: zm,
			})
		}
	}

	return zones, nil
}

// GetZone returns details for a specific zone
func (p *RESTProvider) GetZone(ctx context.Context, domain string) (*dnsprovider.Zone, error) {
	// First try to list zones and find it
	zones, err := p.ListZones(ctx)
	if err == nil {
		for _, z := range zones {
			if strings.EqualFold(z.Name, domain) || strings.EqualFold(z.Name+".", domain) {
				return &z, nil
			}
		}
	}

	// Fallback: try get_zone endpoint if available
	// ... (implementation complexity omitted for now, ListZones is usually enough)

	// If we can't find it, but we have a static configuration or can derive ID?
	// For now return not found
	return nil, errors.NewNotFound("Zone", domain)
}

// ListRecords retrieves all DNS records for a zone
func (p *RESTProvider) ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["get_records"]
	if !ok {
		return nil, fmt.Errorf("get_records endpoint not configured")
	}

	// Resolve Zone ID if it looks like a domain and endpoint requires ID
	realZoneID := zoneID
	if strings.Contains(endpoint, "{zone_id}") && strings.Contains(zoneID, ".") {
		// Try to resolve domain to ID
		resolved, err := p.resolveZoneID(ctx, zoneID)
		if err == nil && resolved != "" {
			realZoneID = resolved
		}
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", realZoneID)
	// Also replace {domain} if present (assuming zoneID was the domain)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)

	resp, err := p.client.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPI("ListRecords", fmt.Sprintf("failed to get records for zone %s", zoneID), err)
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

// CreateRecord creates a new record in the zone
func (p *RESTProvider) CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["create_record"]
	if !ok {
		return nil, fmt.Errorf("create_record endpoint not configured")
	}

	// Resolve Zone ID
	realZoneID := zoneID
	if strings.Contains(endpoint, "{zone_id}") && strings.Contains(zoneID, ".") {
		resolved, err := p.resolveZoneID(ctx, zoneID)
		if err == nil && resolved != "" {
			realZoneID = resolved
		}
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", realZoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)

	body := mapper.ToProviderFormat(record, p.mappings.Request)

	resp, err := p.client.Post(ctx, endpoint, body)
	if err != nil {
		return nil, errors.NewAPI("CreateRecord", "failed to create DNS record", err)
	}
	defer resp.Body.Close()

	// Parse response to get the created record (ID, etc)
	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		// If parsing fails, return the original record (best effort)
		return &record, nil
	}

	// Try to extract the record from response
	// The response might be the record itself, or wrapped (e.g. {result: record})
	// We use ListPath heuristics or Response mapping heuristics?
	// usually create returns the object or {result: object}

	// Try to unwrap if wrapped
	var recordData map[string]interface{}
	if m, ok := responseData.(map[string]interface{}); ok {
		// Check if it's wrapped in "result" or "data"
		if val, ok := m["result"]; ok {
			if vMap, ok := val.(map[string]interface{}); ok {
				recordData = vMap
			}
		} else if val, ok := m["data"]; ok {
			if vMap, ok := val.(map[string]interface{}); ok {
				recordData = vMap
			}
		} else {
			// Assume root is the record
			recordData = m
		}
	}

	if recordData != nil {
		createdRecord, err := mapper.FromProviderFormat(recordData, p.mappings.Response)
		if err == nil {
			return &createdRecord, nil
		}
	}

	return &record, nil
}

// UpdateRecord updates an existing record
func (p *RESTProvider) UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (*dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["update_record"]
	if !ok {
		return nil, fmt.Errorf("update_record endpoint not configured")
	}

	// Ensure record ID is present
	if recordID == "" {
		return nil, fmt.Errorf("record ID is required for update")
	}
	record.ID = recordID // Ensure record has the ID

	// Resolve Zone ID
	realZoneID := zoneID
	if strings.Contains(endpoint, "{zone_id}") && strings.Contains(zoneID, ".") {
		resolved, err := p.resolveZoneID(ctx, zoneID)
		if err == nil && resolved != "" {
			realZoneID = resolved
		}
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", realZoneID)
	endpoint = strings.ReplaceAll(endpoint, "{domain}", zoneID)
	endpoint = strings.ReplaceAll(endpoint, "{record_id}", recordID)
	endpoint = strings.ReplaceAll(endpoint, "{id}", recordID)

	body := mapper.ToProviderFormat(record, p.mappings.Request)

	// Determine method (PUT or PATCH) - usually PUT for full update
	// But OpenAPI spec might say otherwise.
	// For now default to PUT, but maybe we should support config?
	// Config doesn't specify method for endpoints.
	// Spec extractor puts method in key? No.
	// We assume standard REST: PUT or PATCH.
	// Let's use PUT by default.

	resp, err := p.client.Put(ctx, endpoint, body)
	if err != nil {
		// Fallback to PATCH if PUT fails? No, that's risky.
		return nil, errors.NewAPI("UpdateRecord", "failed to update DNS record", err)
	}
	defer resp.Body.Close()

	// Similar response parsing as CreateRecord
	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return &record, nil
	}

	var recordData map[string]interface{}
	if m, ok := responseData.(map[string]interface{}); ok {
		if val, ok := m["result"]; ok {
			if vMap, ok := val.(map[string]interface{}); ok {
				recordData = vMap
			}
		} else if val, ok := m["data"]; ok {
			if vMap, ok := val.(map[string]interface{}); ok {
				recordData = vMap
			}
		} else {
			recordData = m
		}
	}

	if recordData != nil {
		updatedRecord, err := mapper.FromProviderFormat(recordData, p.mappings.Response)
		if err == nil {
			return &updatedRecord, nil
		}
	}

	return &record, nil
}

// DeleteRecord deletes a record
func (p *RESTProvider) DeleteRecord(ctx context.Context, zoneID string, recordID string) error {
	endpoint, ok := p.endpoints["delete_record"]
	if !ok {
		return fmt.Errorf("delete_record endpoint not configured")
	}

	if recordID == "" {
		return fmt.Errorf("record ID is required for delete")
	}

	// Resolve Zone ID
	realZoneID := zoneID
	if strings.Contains(endpoint, "{zone_id}") && strings.Contains(zoneID, ".") {
		resolved, err := p.resolveZoneID(ctx, zoneID)
		if err == nil && resolved != "" {
			realZoneID = resolved
		}
	}

	endpoint = strings.ReplaceAll(endpoint, "{zone_id}", realZoneID)
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

// BulkReplaceRecords replaces all records in a zone
func (p *RESTProvider) BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error {
	// Naive implementation: List, Delete All, Create New
	// This is dangerous but standard for providers without bulk ops

	existing, err := p.ListRecords(ctx, zoneID)
	if err != nil {
		return err
	}

	for _, r := range existing {
		// Use ID if available
		if r.ID != "" {
			_ = p.DeleteRecord(ctx, zoneID, r.ID)
		} else {
			// Fallback: we can't delete without ID in generic REST
			// Maybe log warning?
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

// Validate checks if the provider is properly configured
func (p *RESTProvider) Validate() error {
	if p.client == nil {
		return fmt.Errorf("HTTP client is not initialized")
	}
	if p.name == "" {
		return fmt.Errorf("provider name is empty")
	}
	return nil
}

// resolveZoneID tries to resolve a domain name to a zone ID
func (p *RESTProvider) resolveZoneID(ctx context.Context, domain string) (string, error) {
	// Check settings first
	if id, ok := p.settings["zone_id"].(string); ok && id != "" {
		return id, nil
	}

	// Try to find in zone list
	zone, err := p.GetZone(ctx, domain)
	if err == nil && zone != nil {
		return zone.ID, nil
	}

	return "", fmt.Errorf("could not resolve zone ID for domain %s", domain)
}

// Helper to extract field from map with candidates
func extractField(data map[string]interface{}, candidates []string) string {
	for _, candidate := range candidates {
		if val, ok := data[candidate]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// Ensure RESTProvider implements Provider interface
var _ dnsprovider.Provider = (*RESTProvider)(nil)
