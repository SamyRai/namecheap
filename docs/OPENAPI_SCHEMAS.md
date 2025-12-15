# Official OpenAPI Schemas for DNS Providers

This document tracks official OpenAPI/Swagger specifications available from DNS providers.

## Available Official Schemas

### ✅ Cloudflare

**Source**: [cloudflare/api-schemas](https://github.com/cloudflare/api-schemas)
**URL**: https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml
**Size**: ~16MB (full API, includes DNS endpoints)
**Status**: Official, actively maintained
**Last Updated**: Regularly updated by Cloudflare

**DNS Endpoints**:
- `/zones/{zone_id}/dns_records` - DNS record management
- Part of the full Cloudflare API spec

**Usage**:
```bash
# Download full spec
curl -L https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml \
  -o pkg/dns/provider/cloudflare/openapi.yaml

# Or extract just DNS-related paths (recommended for smaller file)
```

### ❌ Namecheap

**Status**: No official OpenAPI spec available
**API Type**: SOAP/XML (not REST)
**Implementation**: Custom Go adapter using official SDK
**SDK**: `github.com/namecheap/go-namecheap-sdk/v2`
**API Docs**: https://www.namecheap.com/get-started/developers.aspx

**Why No OpenAPI?**
- Namecheap uses SOAP/XML protocol, not REST
- OpenAPI is designed for REST APIs
- Uses official Go SDK for integration
- Custom adapter required (not config-based)

**Current Implementation**:
- Uses `pkg/dns/provider/namecheap/adapter.go`
- Direct integration with Namecheap SDK
- Authentication via account config (username, API key, client IP)
- Not part of auto-discovery (registered separately)

**Note**: Namecheap cannot use the OpenAPI approach because it's SOAP-based, not REST. The current custom adapter is the correct approach.

### ❌ GoDaddy

**Status**: No official OpenAPI spec found
**Alternative**: Manual config.yaml (current approach)
**API Docs**: https://developer.godaddy.com/doc/endpoint/domains

**Note**: GoDaddy provides REST API documentation but no OpenAPI spec. Manual configuration required.

### ❌ DigitalOcean

**Status**: No official OpenAPI spec found
**Alternative**: Manual config.yaml (current approach)
**API Docs**: https://docs.digitalocean.com/reference/api/api-reference/

**Note**: DigitalOcean has API documentation but no OpenAPI spec. Manual configuration required.

### ✅ DNSimple

**Source**: DNSimple API v2
**Status**: OpenAPI definition available
**Reference**: https://blog.dnsimple.com/2018/04/openapi-in-depth/
**Note**: Need to verify current availability and download URL

### ✅ Other Providers with OpenAPI

- **Rackspace Cloud DNS**: OpenAPI document via Swagger UI
- **Authava**: OpenAPI specification available
- **PocketDNS**: OpenAPI document for Partner API
- **Openprovider**: OpenAPI/Swagger documentation

## Schema Sources

### Official Repositories

1. **Cloudflare**
   - GitHub: https://github.com/cloudflare/api-schemas
   - Direct Download: https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml
   - Documentation: https://developers.cloudflare.com/api/

2. **DNSimple**
   - Blog Post: https://blog.dnsimple.com/2018/04/openapi-in-depth/
   - API Docs: https://developer.dnsimple.com/

### Community Resources

- **APIDevTools OpenAPI Schemas**: https://github.com/APIDevTools/openapi-schemas
- **OpenAPI Initiative**: https://www.openapis.org/

## Downloading Schemas

### Cloudflare (Full Spec)

```bash
# Download full Cloudflare OpenAPI spec
curl -L https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml \
  -o pkg/dns/provider/cloudflare/openapi.yaml
```

**Note**: The full spec is ~16MB and includes all Cloudflare services. For DNS-only usage, consider extracting just the DNS-related paths.

### Extracting DNS-Only Paths

For providers with large specs (like Cloudflare), you can extract just DNS-related endpoints:

```bash
# Extract DNS-related paths from Cloudflare spec
yq eval '.paths | with_entries(select(.key | contains("dns")))' \
  /tmp/cloudflare-openapi-full.yaml > cloudflare-dns-only.yaml
```

## Schema Maintenance

### Updating Schemas

1. **Check for Updates**: Visit provider's GitHub repo or documentation
2. **Download Latest**: Use curl/wget to fetch updated spec
3. **Test**: Ensure auto-discovery still works with new spec
4. **Document**: Update this file with last update date

### Version Control

- Store schemas in provider directories: `pkg/dns/provider/{provider}/openapi.yaml`
- Commit to repository for version control
- Document source URL and last update date

## Creating Custom Schemas

For providers without official OpenAPI specs:

1. **Reference API Documentation**: Use official API docs as source
2. **Create Minimal Spec**: Focus on DNS endpoints only
3. **Document Source**: Note that it's a custom/derived spec
4. **Example Structure**:

```yaml
openapi: 3.0.0
info:
  title: Provider DNS API
  version: 1.0.0
  description: Custom OpenAPI spec derived from official documentation
servers:
  - url: https://api.provider.com/v1
paths:
  /domains/{domain}/records:
    get:
      summary: List DNS records
      # ... endpoint definition
```

## Status Summary

| Provider | Official Spec | Status | Location | Notes |
|----------|---------------|--------|----------|-------|
| Cloudflare | ✅ Yes | Available | GitHub | REST API |
| Namecheap | ❌ No | Custom adapter | - | SOAP/XML API, uses Go SDK |
| GoDaddy | ❌ No | Manual config | - | REST API, no spec |
| DigitalOcean | ❌ No | Manual config | - | REST API, no spec |
| DNSimple | ✅ Yes | Available | Documentation | REST API |
| AWS Route 53 | ❌ No | Manual config | - | AWS-specific format |
| Google Cloud DNS | ❌ No | Manual config | - | gRPC primary |

## Next Steps

1. ✅ Download Cloudflare spec
2. ⏳ Extract DNS-only paths (optional, for smaller file)
3. ⏳ Test auto-discovery with Cloudflare spec
4. ⏳ Research and download DNSimple spec
5. ⏳ Create custom specs for providers without official ones
6. ⏳ Document update process

