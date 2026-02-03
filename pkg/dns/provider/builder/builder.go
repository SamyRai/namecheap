package builder

import (
	"fmt"
	"time"

	dnsprovider "zonekit/pkg/dns/provider"
	"zonekit/pkg/dns/provider/auth"
	httpprovider "zonekit/pkg/dns/provider/http"
	"zonekit/pkg/dns/provider/mapper"
	"zonekit/pkg/dns/provider/rest"
)

// BuildProvider creates a DNS provider from configuration
func BuildProvider(config *dnsprovider.Config) (dnsprovider.Provider, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid provider config: %w", err)
	}

	// Create authenticator
	authenticator, err := auth.NewAuthenticator(config.Auth.Method, config.Auth.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	if err := authenticator.Validate(); err != nil {
		return nil, fmt.Errorf("authenticator validation failed: %w", err)
	}

	// Get auth headers
	authHeaders := authenticator.GetHeaders()

	// Merge with configured headers
	headers := make(map[string]string)
	for k, v := range config.API.Headers {
		headers[k] = v
	}
	for k, v := range authHeaders {
		headers[k] = v
	}

	// Create HTTP client
	httpClient := httpprovider.NewClient(httpprovider.ClientConfig{
		BaseURL: config.API.BaseURL,
		Headers: headers,
		Timeout: time.Duration(config.API.Timeout) * time.Second,
		Retries: config.API.Retries,
	})

	// Build provider based on type
	switch config.Type {
	case "rest":
		return buildRESTProvider(config, httpClient)
	case "namecheap":
		// Namecheap uses SOAP, handled separately
		return nil, fmt.Errorf("namecheap provider must be created using namecheap.New()")
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// buildRESTProvider creates a REST-based provider
func buildRESTProvider(config *dnsprovider.Config, client *httpprovider.Client) (dnsprovider.Provider, error) {
	// Build mappings
	mappings := buildMappings(config.Mappings)

	// Create REST provider
	provider := rest.NewRESTProvider(
		config.Name,
		client,
		mappings,
		config.API.Endpoints,
		config.Settings,
	)

	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	return provider, nil
}

// buildMappings builds field mappings from config
func buildMappings(configMappings *dnsprovider.FieldMappings) mapper.Mappings {
	if configMappings == nil {
		return mapper.DefaultMappings()
	}

	m := mapper.Mappings{
		ListPath:     configMappings.ListPath,
		ResponsePath: configMappings.ResponsePath,
		ZoneListPath: configMappings.ZoneListPath,
		ZoneID:       configMappings.ZoneID,
		ZoneName:     configMappings.ZoneName,
	}

	// Set defaults for zone mappings if empty
	if m.ZoneListPath == "" {
		m.ZoneListPath = "zones"
	}
	if m.ZoneID == "" {
		m.ZoneID = "id"
	}
	if m.ZoneName == "" {
		m.ZoneName = "name"
	}

	// Request mappings
	if configMappings.Request.HostName != "" {
		m.Request.HostName = configMappings.Request.HostName
	} else {
		m.Request.HostName = "hostname"
	}

	if configMappings.Request.RecordType != "" {
		m.Request.RecordType = configMappings.Request.RecordType
	} else {
		m.Request.RecordType = "record_type"
	}

	if configMappings.Request.Address != "" {
		m.Request.Address = configMappings.Request.Address
	} else {
		m.Request.Address = "address"
	}

	if configMappings.Request.TTL != "" {
		m.Request.TTL = configMappings.Request.TTL
	} else {
		m.Request.TTL = "ttl"
	}

	if configMappings.Request.MXPref != "" {
		m.Request.MXPref = configMappings.Request.MXPref
	} else {
		m.Request.MXPref = "mx_pref"
	}

	if configMappings.Request.ID != "" {
		m.Request.ID = configMappings.Request.ID
	} else {
		m.Request.ID = ""
	}

	// Response mappings
	if configMappings.Response.HostName != "" {
		m.Response.HostName = configMappings.Response.HostName
	} else {
		m.Response.HostName = "hostname"
	}

	if configMappings.Response.RecordType != "" {
		m.Response.RecordType = configMappings.Response.RecordType
	} else {
		m.Response.RecordType = "record_type"
	}

	if configMappings.Response.Address != "" {
		m.Response.Address = configMappings.Response.Address
	} else {
		m.Response.Address = "address"
	}

	if configMappings.Response.TTL != "" {
		m.Response.TTL = configMappings.Response.TTL
	} else {
		m.Response.TTL = "ttl"
	}

	if configMappings.Response.MXPref != "" {
		m.Response.MXPref = configMappings.Response.MXPref
	} else {
		m.Response.MXPref = "mx_pref"
	}

	if configMappings.Response.ID != "" {
		m.Response.ID = configMappings.Response.ID
	} else {
		m.Response.ID = ""
	}

	return m
}

// validateConfig validates provider configuration
func validateConfig(config *dnsprovider.Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if config.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	if config.API.BaseURL == "" {
		return fmt.Errorf("API base URL is required")
	}

	if len(config.API.Endpoints) == 0 {
		return fmt.Errorf("at least one API endpoint is required")
	}

	if config.Auth.Method == "" {
		return fmt.Errorf("authentication method is required")
	}

	if len(config.Auth.Credentials) == 0 {
		return fmt.Errorf("authentication credentials are required")
	}

	return nil
}
