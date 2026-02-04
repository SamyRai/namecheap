package provider

import (
	"context"
	"zonekit/pkg/dnsrecord"
)

// ProviderCapabilities defines the capabilities of a DNS provider
type ProviderCapabilities struct {
	// IsBulkReplaceAtomic indicates if the provider supports atomic bulk replacement of records
	IsBulkReplaceAtomic bool
}

// Provider defines the interface that all DNS providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "namecheap", "cloudflare", "godaddy")
	Name() string

	// Capabilities returns the provider capabilities
	Capabilities() ProviderCapabilities

	// GetRecords retrieves all DNS records for a domain
	GetRecords(ctx context.Context, domainName string) ([]dnsrecord.Record, error)

	// SetRecords sets DNS records for a domain (replaces all existing records)
	SetRecords(ctx context.Context, domainName string, records []dnsrecord.Record) error

	// AddRecord adds a single record
	AddRecord(ctx context.Context, domainName string, record dnsrecord.Record) error

	// UpdateRecord updates a single record
	UpdateRecord(ctx context.Context, domainName string, record dnsrecord.Record) error

	// DeleteRecord deletes a single record
	DeleteRecord(ctx context.Context, domainName string, record dnsrecord.Record) error

	// Validate checks if the provider is properly configured
	Validate() error
}

// Config represents provider-specific configuration
type Config struct {
	// Provider name (e.g., "namecheap", "cloudflare")
	Name string `yaml:"name"`

	// Display name for UI
	DisplayName string `yaml:"display_name"`

	// Provider type determines which adapter to use
	Type string `yaml:"type"` // "namecheap", "cloudflare", "godaddy", "rest", etc.

	// Authentication configuration
	Auth struct {
		Method string `yaml:"method"` // "api_key", "oauth", "basic", etc.
		// Fields vary by method - stored as map for flexibility
		Credentials map[string]interface{} `yaml:"credentials"`
	} `yaml:"auth"`

	// API configuration
	API struct {
		BaseURL   string            `yaml:"base_url"`
		Endpoints map[string]string `yaml:"endpoints"` // e.g., "get_records": "/api/v1/dns/records"
		Headers   map[string]string `yaml:"headers,omitempty"`
		Timeout   int               `yaml:"timeout,omitempty"` // seconds
		Retries   int               `yaml:"retries,omitempty"`
	} `yaml:"api"`

	// Provider-specific settings
	Settings map[string]interface{} `yaml:"settings,omitempty"`

	// Field mappings for REST providers (optional, for generic REST adapter)
	Mappings *FieldMappings `yaml:"mappings,omitempty"`
}

// FieldMappings defines how to map between our Record structure and provider's API format
type FieldMappings struct {
	// Request mappings (our format -> provider format)
	Request struct {
		HostName   string `yaml:"hostname,omitempty"`    // e.g., "name" or "host"
		RecordType string `yaml:"record_type,omitempty"` // e.g., "type" or "rtype"
		Address    string `yaml:"address,omitempty"`     // e.g., "value" or "content"
		TTL        string `yaml:"ttl,omitempty"`
		MXPref     string `yaml:"mx_pref,omitempty"` // e.g., "priority" or "preference"
		ID         string `yaml:"id,omitempty"`      // provider record ID field
	} `yaml:"request,omitempty"`

	// Response mappings (provider format -> our format)
	Response struct {
		HostName   string `yaml:"hostname,omitempty"`
		RecordType string `yaml:"record_type,omitempty"`
		Address    string `yaml:"address,omitempty"`
		TTL        string `yaml:"ttl,omitempty"`
		MXPref     string `yaml:"mx_pref,omitempty"`
		ID         string `yaml:"id,omitempty"` // provider record ID field
	} `yaml:"response,omitempty"`

	// List response structure (for REST providers)
	ListPath string `yaml:"list_path,omitempty"` // JSON path to records array, e.g., "data.records"
}
