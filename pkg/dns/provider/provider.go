package provider

import (
	"context"

	"zonekit/pkg/dnsrecord"
)

// Zone represents a DNS zone
type Zone struct {
	ID   string
	Name string
}

// ProviderCapabilities describes what a provider supports
type ProviderCapabilities struct {
	CanListZones    bool
	CanGetZone      bool
	CanCreateRecord bool
	CanUpdateRecord bool
	CanDeleteRecord bool
	CanBulkReplace  bool
}

// Provider defines the interface that all DNS providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "namecheap", "cloudflare", "godaddy")
	Name() string

	// ListZones retrieves all zones managed by the provider
	ListZones(ctx context.Context) ([]Zone, error)

	// GetZone retrieves a specific zone by ID
	GetZone(ctx context.Context, zoneID string) (Zone, error)

	// ListRecords retrieves all DNS records for a zone
	ListRecords(ctx context.Context, zoneID string) ([]dnsrecord.Record, error)

	// CreateRecord creates a new DNS record
	CreateRecord(ctx context.Context, zoneID string, record dnsrecord.Record) (dnsrecord.Record, error)

	// UpdateRecord updates an existing DNS record
	UpdateRecord(ctx context.Context, zoneID string, recordID string, record dnsrecord.Record) (dnsrecord.Record, error)

	// DeleteRecord deletes a DNS record
	DeleteRecord(ctx context.Context, zoneID string, recordID string) error

	// BulkReplaceRecords replaces all records in a zone with the provided set
	BulkReplaceRecords(ctx context.Context, zoneID string, records []dnsrecord.Record) error

	// Capabilities returns the provider's capabilities
	Capabilities() ProviderCapabilities

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
		Priority   string `yaml:"priority,omitempty"`
		Weight     string `yaml:"weight,omitempty"`
		Port       string `yaml:"port,omitempty"`
		Target     string `yaml:"target,omitempty"`
	} `yaml:"request,omitempty"`

	// Response mappings (provider format -> our format)
	Response struct {
		HostName   string `yaml:"hostname,omitempty"`
		RecordType string `yaml:"record_type,omitempty"`
		Address    string `yaml:"address,omitempty"`
		TTL        string `yaml:"ttl,omitempty"`
		MXPref     string `yaml:"mx_pref,omitempty"`
		ID         string `yaml:"id,omitempty"` // provider record ID field
		Priority   string `yaml:"priority,omitempty"`
		Weight     string `yaml:"weight,omitempty"`
		Port       string `yaml:"port,omitempty"`
		Target     string `yaml:"target,omitempty"`
	} `yaml:"response,omitempty"`

	// List response structure (for REST providers)
	ListPath string `yaml:"list_path,omitempty"` // JSON path to records array, e.g., "data.records"
}
