# Namecheap API Integration

## Overview

Namecheap is a special case in our provider architecture because it uses **SOAP/XML** instead of REST, which means it cannot use the OpenAPI-based auto-discovery system.

## API Type

- **Protocol**: SOAP/XML (not REST)
- **Format**: XML requests and responses
- **SDK**: Official Go SDK (`github.com/namecheap/go-namecheap-sdk/v2`)
- **Documentation**: https://www.namecheap.com/get-started/developers.aspx

## Why No OpenAPI?

1. **SOAP vs REST**: Namecheap uses SOAP/XML protocol, while OpenAPI is designed for REST APIs
2. **No Official Spec**: Namecheap does not provide an OpenAPI/Swagger specification
3. **SDK-Based**: Uses official Go SDK rather than direct HTTP calls
4. **Custom Implementation**: Requires custom adapter code, not config-based

## Current Implementation

### Location
- **Adapter**: `pkg/dns/provider/namecheap/adapter.go`
- **Config Example**: `pkg/dns/provider/namecheap/config.yaml.example` (for reference only)

### How It Works

```go
// Namecheap uses custom adapter
type NamecheapProvider struct {
    client *client.Client  // Wraps Namecheap SDK
}

// Uses official SDK methods
nc.DomainsDNS.GetHosts(domainName)
nc.DomainsDNS.SetHosts(args)
```

### Registration

Namecheap is registered separately (not via auto-discovery):

```go
// In cmd/root.go or initialization
namecheap.Register(client)
```

## Authentication

Namecheap requires:
- **Username**: Namecheap account username
- **API User**: API user (may differ from username)
- **API Key**: API key from Namecheap account
- **Client IP**: Whitelisted IP address

These are configured via account config, not provider config.

## API Endpoints

Namecheap DNS operations:
- `domains.dns.getHosts` - Get DNS records
- `domains.dns.setHosts` - Set DNS records (replaces all)

## Differences from REST Providers

| Aspect | REST Providers | Namecheap |
|--------|----------------|-----------|
| Protocol | HTTP/REST | SOAP/XML |
| Config | YAML/OpenAPI | Go code |
| Discovery | Auto-discovered | Manual registration |
| SDK | Generic HTTP client | Namecheap SDK |
| Format | JSON | XML |

## Future Considerations

### Could We Create an OpenAPI Spec?

Technically possible but not practical:
- ❌ OpenAPI doesn't natively support SOAP
- ❌ Would need to manually create spec from docs
- ❌ Wouldn't work with our REST infrastructure
- ✅ Current SDK approach is better for SOAP

### Alternative Approaches

1. **Keep Current Approach** (Recommended)
   - Use official SDK
   - Custom adapter code
   - Works well for SOAP

2. **SOAP-to-REST Wrapper** (Not Recommended)
   - Would add complexity
   - Performance overhead
   - Maintenance burden

## Summary

Namecheap is correctly implemented as a **custom provider** using:
- Official Go SDK
- SOAP/XML protocol
- Custom adapter code
- Manual registration

This is the appropriate approach for SOAP-based APIs and cannot use the OpenAPI auto-discovery system designed for REST APIs.

