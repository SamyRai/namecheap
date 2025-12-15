# OpenAPI-Only Approach

## Overview

We've transitioned to an **OpenAPI-only approach** for all REST-based DNS providers. This means:

- ✅ **OpenAPI specs required** - No fallback to manual `config.yaml`
- ✅ **Auto-generated configs** - All provider configs generated from OpenAPI specs
- ✅ **Standardized format** - All providers use the same OpenAPI format
- ✅ **Easy updates** - Update OpenAPI spec, config regenerates automatically

## Current Status

### ✅ Providers with OpenAPI Specs

| Provider | Spec Location | Source | Status |
|----------|--------------|--------|--------|
| **Cloudflare** | `pkg/dns/provider/cloudflare/openapi.yaml` | Official (GitHub) | ✅ Downloaded |
| **GoDaddy** | `pkg/dns/provider/godaddy/openapi.yaml` | Custom (from API docs) | ✅ Created |
| **DigitalOcean** | `pkg/dns/provider/digitalocean/openapi.yaml` | Custom (from API docs) | ✅ Created |

### ❌ Providers Not Using OpenAPI

| Provider | Reason | Implementation |
|----------|--------|----------------|
| **Namecheap** | SOAP/XML API | Custom Go adapter (correct approach) |

## How It Works

### Auto-Discovery Flow

1. **Scan Provider Directories**
   - Looks for `openapi.yaml`, `openapi.json`, `swagger.yaml`, etc.

2. **Parse OpenAPI Spec**
   - Extracts base URL from `servers`
   - Extracts endpoints from `paths`
   - Extracts authentication from `securitySchemes`
   - Extracts field mappings from `schemas`

3. **Generate Provider Config**
   - Automatically creates `dnsprovider.Config` from OpenAPI spec
   - Maps OpenAPI operations to DNS operations
   - Maps OpenAPI schemas to field mappings

4. **Register Provider**
   - Builds provider using generated config
   - Registers in provider registry

### No Manual Configs

The system **no longer** falls back to `config.yaml`. If no OpenAPI spec is found, the provider is skipped.

## Creating OpenAPI Specs

### For Providers with Official Specs

1. Download official OpenAPI spec
2. Place in provider directory: `pkg/dns/provider/{provider}/openapi.yaml`
3. Done! Auto-discovery handles the rest

### For Providers Without Official Specs

Create a minimal OpenAPI spec based on their API documentation:

```yaml
openapi: 3.0.0
info:
  title: Provider DNS API
  version: 1.0.0
servers:
  - url: https://api.provider.com/v1
paths:
  /domains/{domain}/records:
    get:
      operationId: listDNSRecords
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
        data: {type: string}      # Maps to address
        ttl: {type: integer}      # Maps to ttl
        priority: {type: integer} # Maps to mx_pref
```

## Benefits

✅ **Consistency** - All providers use same format
✅ **Auto-generation** - Configs generated automatically
✅ **Easy updates** - Update spec, config regenerates
✅ **Standard format** - OpenAPI is industry standard
✅ **Documentation** - Specs serve as documentation

## Migration from Manual Configs

Old manual `config.yaml` files are now:
- **Not used** - System only looks for OpenAPI specs
- **Can be kept** - For reference/documentation
- **Should be removed** - To avoid confusion

## Provider Requirements

For a provider to work with auto-discovery:

1. ✅ Must have `openapi.yaml` in provider directory
2. ✅ Must define DNS endpoints in `paths`
3. ✅ Must define authentication in `securitySchemes`
4. ✅ Must define DNS record schema in `schemas`

## Examples

### Cloudflare (Official Spec)
- Source: GitHub repository
- Size: 16MB (full API)
- Status: Official, maintained by Cloudflare

### GoDaddy (Custom Spec)
- Source: Created from API documentation
- Size: ~5KB (DNS endpoints only)
- Status: Custom, based on official docs

### DigitalOcean (Custom Spec)
- Source: Created from API documentation
- Size: ~5KB (DNS endpoints only)
- Status: Custom, based on official docs

## Next Steps

1. ✅ Create OpenAPI specs for all REST providers - **DONE**
2. ⏳ Test auto-discovery with all providers
3. ⏳ Remove old `config.yaml` files (optional, for cleanup)
4. ⏳ Document OpenAPI spec creation process
5. ⏳ Add validation for OpenAPI specs

