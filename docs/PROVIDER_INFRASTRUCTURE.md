# DNS Provider Infrastructure

## Overview

A complete, production-ready infrastructure for adding DNS providers easily. The system supports both REST-based providers (via configuration) and custom providers (via direct implementation).

## Architecture

### Core Components

```
pkg/dns/provider/
├── http/          # Generic HTTP client (retry, timeout, error handling)
├── rest/          # Generic REST provider implementation
├── builder/       # Provider factory from config
├── auth/          # Authentication handlers
├── mapper/        # Field mapping utilities
└── config/        # Config loading
```

### Key Features

✅ **Generic HTTP Client**
- Automatic retry with exponential backoff
- Configurable timeout
- Error handling and wrapping
- Support for all HTTP methods

✅ **REST Provider**
- Configuration-driven
- Automatic field mapping
- Placeholder replacement (e.g., `{zone_id}`, `{domain}`)
- JSON response parsing

✅ **Authentication System**
- API Key (with optional email)
- Bearer Token
- Basic Auth
- OAuth (Bearer token)
- Custom headers
- Environment variable support (`${VAR_NAME}`)

✅ **Field Mapping**
- Request transformation (our format → provider format)
- Response transformation (provider format → our format)
- JSON path extraction for nested responses

✅ **Provider Builder**
- Validates configuration
- Creates authenticated HTTP client
- Builds REST provider from config
- Error handling

## Quick Start: Adding a REST Provider

### 1. Create Config File

`pkg/dns/provider/cloudflare/config.yaml`:

```yaml
name: cloudflare
display_name: Cloudflare
type: rest

auth:
  method: api_key
  credentials:
    api_key: "${CLOUDFLARE_API_KEY}"
    email: "${CLOUDFLARE_EMAIL}"

api:
  base_url: "https://api.cloudflare.com/client/v4"
  endpoints:
    get_records: "/zones/{zone_id}/dns_records"
    create_record: "/zones/{zone_id}/dns_records"
    delete_record: "/zones/{zone_id}/dns_records/{record_id}"
  headers:
    Content-Type: "application/json"
  timeout: 30
  retries: 3

mappings:
  request:
    hostname: "name"
    record_type: "type"
    address: "content"
    ttl: "ttl"
    mx_pref: "priority"
  response:
    hostname: "name"
    record_type: "type"
    address: "content"
    ttl: "ttl"
    mx_pref: "priority"
  list_path: "result"

settings:
  zone_id_required: true
```

### 2. Load and Register

```go
import (
    "zonekit/pkg/dns/provider/builder"
    "zonekit/pkg/dns/provider/config"
    dnsprovider "zonekit/pkg/dns/provider"
)

// Load config
cfg, err := config.LoadFromFile("pkg/dns/provider/cloudflare/config.yaml")
if err != nil {
    return err
}

// Build provider
provider, err := builder.BuildProvider(cfg)
if err != nil {
    return err
}

// Register
if err := dnsprovider.Register(provider); err != nil {
    return err
}
```

### 3. Use Provider

```go
service, err := dns.NewServiceWithProviderName("cloudflare")
records, err := service.GetRecords("example.com")
```

**That's it!** No code needed for REST providers - just configuration.

## Adding a Custom Provider

For providers that don't fit REST (e.g., SOAP, custom protocols):

```go
package customprovider

import (
    "zonekit/pkg/dnsrecord"
    dnsprovider "zonekit/pkg/dns/provider"
)

type CustomProvider struct {
    // provider-specific fields
}

func (p *CustomProvider) Name() string {
    return "customprovider"
}

func (p *CustomProvider) GetRecords(domainName string) ([]dnsrecord.Record, error) {
    // Custom implementation
}

func (p *CustomProvider) SetRecords(domainName string, records []dnsrecord.Record) error {
    // Custom implementation
}

func (p *CustomProvider) Validate() error {
    // Validation
}

// Register
func Register() error {
    provider := New()
    return dnsprovider.Register(provider)
}
```

## Infrastructure Details

### HTTP Client (`http/client.go`)

Features:
- Retry logic with exponential backoff
- Configurable timeout
- Automatic error wrapping
- JSON request/response handling

Usage:
```go
client := http.NewClient(http.ClientConfig{
    BaseURL: "https://api.example.com",
    Headers: map[string]string{"Authorization": "Bearer token"},
    Timeout: 30 * time.Second,
    Retries: 3,
})

resp, err := client.Get(ctx, "/records", nil)
```

### Authentication (`auth/auth.go`)

Supported methods:
- `api_key`: API key with optional email
- `bearer`: Bearer token
- `basic`: Basic authentication
- `oauth`: OAuth (treated as Bearer)
- `custom`: Custom headers

Environment variables:
```yaml
credentials:
  api_key: "${API_KEY}"  # Reads from environment
```

### Field Mapper (`mapper/mapper.go`)

Transforms between formats:
```go
// Our format → Provider format
providerRecord := mapper.ToProviderFormat(record, requestMapping)

// Provider format → Our format
record, err := mapper.FromProviderFormat(providerData, responseMapping)
```

### REST Provider (`rest/rest.go`)

Handles:
- Endpoint placeholder replacement
- Record creation/deletion
- Bulk operations (get all, replace all)
- Zone ID resolution

### Builder (`builder/builder.go`)

Validates and builds:
- Config validation
- Authenticator creation
- HTTP client setup
- Provider instantiation

## Benefits

1. **Fast Development**: REST providers = config file only
2. **Reliable**: Tested infrastructure components
3. **Consistent**: All providers use same interface
4. **Flexible**: Supports both REST and custom providers
5. **Maintainable**: Clear separation of concerns

## Next Steps

1. ✅ Infrastructure complete
2. ⏳ Add Cloudflare provider using this infrastructure
3. ⏳ Add tests for generic components
4. ⏳ Add more providers (AWS Route 53, GoDaddy, etc.)

## Example: Complete Provider Addition

See `pkg/dns/provider/cloudflare/config.yaml.example` for a complete example configuration.

