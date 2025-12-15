# DNS Provider Infrastructure

This package provides a pluggable architecture for supporting multiple DNS providers with a generic, reusable infrastructure.

## Architecture

### Core Components

1. **Provider Interface** (`provider.go`) - Standard interface all providers implement
2. **Registry** (`registry.go`) - Thread-safe provider registry
3. **HTTP Client** (`http/client.go`) - Generic HTTP client with retry, timeout, error handling
4. **REST Provider** (`rest/rest.go`) - Generic REST-based provider implementation
5. **Builder** (`builder/builder.go`) - Factory to create providers from config
6. **Authentication** (`auth/auth.go`) - Authentication handlers (API key, Bearer, Basic, OAuth)
7. **Field Mapper** (`mapper/mapper.go`) - Maps between our format and provider formats
8. **Config Loader** (`config/config.go`) - Loads provider configurations from YAML files

### Directory Structure

```
pkg/dns/provider/
├── provider.go          # Provider interface
├── registry.go          # Provider registry
│
├── http/                # Generic HTTP client
│   └── client.go
│
├── rest/                # Generic REST provider
│   └── rest.go
│
├── builder/             # Provider builder/factory
│   └── builder.go
│
├── auth/                # Authentication handlers
│   └── auth.go
│
├── mapper/              # Field mapping utilities
│   └── mapper.go
│
├── config/              # Config loading
│   └── config.go
│
├── namecheap/           # Namecheap provider (SOAP, custom)
│   ├── adapter.go
│   └── config.yaml.example
│
└── cloudflare/          # Cloudflare provider (REST, config-based)
    └── config.yaml.example
```

## Adding a New REST Provider

**OpenAPI-Only Approach** - Just create an OpenAPI spec file!

### Step 1: Create Provider Directory

```bash
mkdir -p pkg/dns/provider/newprovider
```

### Step 2: Create OpenAPI Specification

Create `pkg/dns/provider/newprovider/openapi.yaml`:

```yaml
openapi: 3.0.0
info:
  title: New Provider DNS API
  version: 1.0.0
servers:
  - url: https://api.newprovider.com/v1
paths:
  /domains/{domain}/records:
    get:
      operationId: listDNSRecords
      # ... endpoint definition
    post:
      operationId: createDNSRecord
      # ... endpoint definition
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
  schemas:
    DNSRecord:
      type: object
      properties:
        name: {type: string}      # Maps to hostname
        type: {type: string}      # Maps to record_type
        data: {type: string}       # Maps to address
        ttl: {type: integer}      # Maps to ttl
        priority: {type: integer} # Maps to mx_pref
```

### Step 3: Done!

**That's it!** The provider will be automatically discovered and registered on startup.

No code needed - just the OpenAPI spec file. The auto-discovery system will:
1. Scan `pkg/dns/provider/*/` directories
2. Find `openapi.yaml` files
3. Parse spec and generate provider config automatically
4. Register providers automatically

### Step 4: Use Provider

```go
dnsService, err := dns.NewServiceWithProviderName("newprovider")
if err != nil {
    return err
}

records, err := dnsService.GetRecords("example.com")
```

## Adding a Custom Provider (Non-REST)

For providers that don't fit the REST pattern (like Namecheap with SOAP):

1. Create provider directory: `pkg/dns/provider/customprovider/`
2. Implement the `Provider` interface directly
3. Register in your initialization code

Example:

```go
package customprovider

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
    // Validation logic
}
```

## Authentication Methods

Supported authentication methods:

- **api_key**: API key authentication (with optional email)
- **bearer**: Bearer token authentication
- **basic**: Basic authentication
- **oauth**: OAuth token (treated as Bearer)
- **custom**: Custom headers

## Field Mappings

Field mappings allow you to translate between our standard format and provider-specific formats:

- **Request mappings**: Our format → Provider format
- **Response mappings**: Provider format → Our format
- **List path**: JSON path to records array in response

## Benefits

- **Standardized Interface**: All providers implement the same interface
- **Easy to Add**: REST providers just need a config file
- **Config-based**: Simple REST providers are mostly configuration
- **Well-tested Infrastructure**: Generic components are tested and reliable
- **Sync/Migration Ready**: All providers available for cross-provider operations

## Future Enhancements

1. **Auto-discovery**: Automatically load all provider configs from directory
2. **Provider Testing**: Standardized test suite for providers
3. **OAuth Flow**: Full OAuth implementation for providers requiring it
4. **Rate Limiting**: Built-in rate limiting support
5. **Caching**: Optional response caching
