# DNS Provider Infrastructure

This package provides a pluggable architecture for supporting multiple DNS providers with a generic, reusable infrastructure.

## Architecture

### Core Components

1. **Provider Interface** (`provider.go`) - Standard interface all providers implement
2. **ZoneConfigurer** (`zone_configurer.go`) - Optional capability interface for zone-level
   configuration (SSL/TLS settings, bot management, security.txt, redirect rules, DNSSEC),
   discovered via a `ProviderCapabilities` flag plus a type assertion. Not part of `Provider`
   itself, since most providers have no equivalent.
3. **Registry** (`registry.go`) - Thread-safe provider registry
4. **HTTP Client** (`http/client.go`) - Generic HTTP client with retry, timeout, error handling
5. **REST Provider** (`rest/rest.go`) - Generic REST-based provider implementation
6. **Builder** (`builder/builder.go`) - Factory to create providers from config
7. **Authentication** (`auth/auth.go`) - Authentication handlers (API key, Bearer, Basic, OAuth)
8. **Field Mapper** (`mapper/mapper.go`) - Maps between our format and provider formats
9. **Config Loader** (`config/config.go`) - Loads provider configurations from YAML files

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
└── cloudflare/          # Cloudflare provider (typed, hand-rolled net/http client)
    ├── provider.go       # Provider: Name/ListZones/GetZone/ZoneByName/record CRUD/BulkReplaceRecords
    ├── configurer.go      # ZoneConfigurer: zone settings, bot management, security.txt,
    │                       #   redirect rules, DNSSEC
    ├── records.go          # Record <-> wire conversion, TXT canonicalization, diff-based bulk replace
    ├── client.go            # Low-level HTTP client: auth, pagination, error envelope
    ├── config.go             # Config, FromAccount (zonekit account -> Config)
    ├── openapi.yaml            # API-shape reference only - NOT used to build this provider
    └── config.yaml.example      # Example account entry
```

### The Cloudflare Provider (Typed)

Unlike the generic REST/OpenAPI providers above, `cloudflare/` is a typed Go implementation
registered under the name `"cloudflare"` - `autodiscover.DiscoverAndRegister` explicitly skips
the `cloudflare/` directory so it never registers the generic OpenAPI adapter under that name
instead. It uses a hand-rolled `net/http` client (no `cloudflare-go` dependency), Bearer token
auth, and Cloudflare's `result_info` pagination.

```go
p, err := cloudflare.NewFromAccount(accountConfig) // from a zonekit config.AccountConfig
// or: p, err := cloudflare.New(cloudflare.Config{APIToken: "...", AccountID: "..."})
if err != nil { ... }

if err := dnsprovider.Register(p); err != nil { ... } // makes it resolvable via provider.Get("cloudflare")
```

It additionally implements `dnsprovider.ZoneConfigurer` (see `Capabilities().CanConfigure*`)
for zone settings (ssl, min_tls_version, tls_1_3, always_use_https, automatic_https_rewrites),
bot management (read-modify-write, preserving fields it does not model), security.txt, redirect
rules in the `http_request_dynamic_redirect` ruleset (merged by a caller-supplied `Ref` prefix,
never dropping rules owned by someone else), and DNSSEC. Namecheap reports every
`CanConfigure*` capability as `false` and does not implement `ZoneConfigurer`.

`BulkReplaceRecords` diffs the desired record set against what's live and issues only the
create/update/delete calls the diff requires - it never deletes everything and recreates it.

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
