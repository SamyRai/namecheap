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

// GetRecords retrieves all DNS records for a domain
func (p *RESTProvider) GetRecords(domainName string) ([]dnsrecord.Record, error) {
	endpoint, ok := p.endpoints["get_records"]
	if !ok {
		return nil, fmt.Errorf("get_records endpoint not configured")
	}

	// Replace placeholders in endpoint (e.g., {zone_id}, {domain})
	endpoint = p.replacePlaceholders(endpoint, domainName)

	// Get zone ID if required
	zoneID, err := p.getZoneID(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}
	if zoneID != "" {
		endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	}

	ctx := context.Background()
	resp, err := p.client.Get(ctx, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPI("GetRecords", fmt.Sprintf("failed to get DNS records for %s", domainName), err)
	}

	// Parse response (this will close the body)
	var responseData interface{}
	if err := httpprovider.ParseJSONResponse(resp, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract records using list path
	recordMaps, err := mapper.ExtractRecords(responseData, p.mappings.ListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract records: %w", err)
	}

	// Convert to dnsrecord.Record
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

// SetRecords sets DNS records for a domain (replaces all existing records)
func (p *RESTProvider) SetRecords(domainName string, records []dnsrecord.Record) error {
	// Most REST APIs don't support bulk replace, so we need to:
	// 1. Get existing records
	// 2. Delete all existing records
	// 3. Create new records

	existingRecords, err := p.GetRecords(domainName)
	if err != nil {
		return fmt.Errorf("failed to get existing records: %w", err)
	}

	ctx := context.Background()

	// Delete existing records
	for _, record := range existingRecords {
		if err := p.deleteRecord(ctx, domainName, record); err != nil {
			// Log but continue - some records might not exist
			continue
		}
	}

	// Create new records
	for _, record := range records {
		if err := p.createRecord(ctx, domainName, record); err != nil {
			return fmt.Errorf("failed to create record: %w", err)
		}
	}

	return nil
}

// createRecord creates a single DNS record
func (p *RESTProvider) createRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	endpoint, ok := p.endpoints["create_record"]
	if !ok {
		return fmt.Errorf("create_record endpoint not configured")
	}

	endpoint = p.replacePlaceholders(endpoint, domainName)
	zoneID, _ := p.getZoneID(domainName)
	if zoneID != "" {
		endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	}

	// Convert record to provider format
	body := mapper.ToProviderFormat(record, p.mappings.Request)

	resp, err := p.client.Post(ctx, endpoint, body)
	if err != nil {
		return errors.NewAPI("CreateRecord", "failed to create DNS record", err)
	}
	defer resp.Body.Close()

	return nil
}

// deleteRecord deletes a single DNS record
func (p *RESTProvider) deleteRecord(ctx context.Context, domainName string, record dnsrecord.Record) error {
	endpoint, ok := p.endpoints["delete_record"]
	if !ok {
		// If delete endpoint not configured, try to use record ID
		// For now, skip if not configured
		return nil
	}

	endpoint = p.replacePlaceholders(endpoint, domainName)
	zoneID, _ := p.getZoneID(domainName)
	if zoneID != "" {
		endpoint = strings.ReplaceAll(endpoint, "{zone_id}", zoneID)
	}

	// Replace {record_id} or {id} placeholders with the record's ID if provided
	if strings.Contains(endpoint, "{record_id}") || strings.Contains(endpoint, "{id}") || strings.Contains(endpoint, "{recordId}") {
		// Prefer record.ID
		if record.ID == "" {
			return fmt.Errorf("delete_record requires record_id - record is missing ID")
		}
		endpoint = strings.ReplaceAll(endpoint, "{record_id}", record.ID)
		endpoint = strings.ReplaceAll(endpoint, "{id}", record.ID)
		endpoint = strings.ReplaceAll(endpoint, "{recordId}", record.ID)
	}

	resp, err := p.client.Delete(ctx, endpoint)
	if err != nil {
		return errors.NewAPI("DeleteRecord", "failed to delete DNS record", err)
	}
	defer resp.Body.Close()

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
	if len(p.endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	return nil
}

// Helper methods

func (p *RESTProvider) replacePlaceholders(endpoint, domainName string) string {
	endpoint = strings.ReplaceAll(endpoint, "{domain}", domainName)
	return endpoint
}

func (p *RESTProvider) getZoneID(domainName string) (string, error) {
	// Check if zone_id is in settings
	if zoneID, ok := p.settings["zone_id"].(string); ok && zoneID != "" {
		return zoneID, nil
	}

	// Some providers require zone_id to be fetched
	// This would need provider-specific implementation
	// For now, return empty if not in settings
	return "", nil
}

// Ensure RESTProvider implements Provider interface
var _ dnsprovider.Provider = (*RESTProvider)(nil)
