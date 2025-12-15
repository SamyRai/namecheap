# OpenAPI/Swagger Integration for Provider Configuration

## Overview

Instead of manually writing YAML configs, we can use OpenAPI/Swagger specifications to automatically generate provider configurations. This dramatically simplifies adding new providers.

## Benefits

✅ **Zero Configuration** - Just drop an OpenAPI spec file
✅ **Auto-Discovery** - Endpoints, schemas, auth methods extracted automatically
✅ **Standard Format** - OpenAPI is industry standard
✅ **Always Up-to-Date** - Use provider's official spec
✅ **Type Safety** - Schema validation built-in

## How It Works

### Discovery Priority

1. **OpenAPI Spec** (`openapi.yaml` or `openapi.json` or `swagger.yaml`)
   - Parse OpenAPI spec
   - Extract endpoints (GET, POST, PUT, DELETE)
   - Extract authentication methods
   - Extract request/response schemas
   - Auto-generate field mappings

2. **Manual Config** (`config.yaml`) - Fallback
   - Use existing manual config if no OpenAPI spec found
   - Still fully supported for custom providers

### Auto-Discovery Flow

```
pkg/dns/provider/cloudflare/
├── openapi.yaml          ← Preferred: Auto-generates everything
└── config.yaml           ← Fallback: Manual config
```

## Implementation Plan

### Phase 1: OpenAPI Parser

1. Add OpenAPI parser library (e.g., `github.com/getkin/kin-openapi`)
2. Parse OpenAPI 3.0/3.1 specs
3. Extract:
   - Base URL (`servers[0].url`)
   - Endpoints (paths)
   - Authentication schemes (securitySchemes)
   - Request/response schemas

### Phase 2: Endpoint Mapping

Map OpenAPI operations to DNS operations:

```yaml
# OpenAPI paths
/zones/{zone_id}/dns_records:
  get:    → get_records
  post:   → create_record
  put:    → update_record
  delete: → delete_record
```

### Phase 3: Schema Mapping

Extract field mappings from OpenAPI schemas:

```yaml
# From OpenAPI schema
components:
  schemas:
    DNSRecord:
      properties:
        name:      → hostname
        type:      → record_type
        content:   → address
        ttl:       → ttl
        priority:  → mx_pref
```

### Phase 4: Authentication

Extract auth from OpenAPI:

```yaml
# From OpenAPI
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-Auth-Key
    BearerAuth:
      type: http
      scheme: bearer
```

## Example: Cloudflare with OpenAPI

Instead of manual config:

```yaml
# config.yaml (manual)
name: cloudflare
api:
  base_url: "https://api.cloudflare.com/client/v4"
  endpoints:
    get_records: "/zones/{zone_id}/dns_records"
```

Just use:

```yaml
# openapi.yaml (from Cloudflare)
openapi: 3.0.0
info:
  title: Cloudflare API
servers:
  - url: https://api.cloudflare.com/client/v4
paths:
  /zones/{zone_id}/dns_records:
    get:
      operationId: listDNSRecords
      # ... schema definitions
```

## Hybrid Support

The system will:

1. Check for `openapi.yaml` or `openapi.json` first
2. If found, parse and generate config automatically
3. If not found, fall back to `config.yaml`
4. Both can coexist (OpenAPI for structure, config.yaml for overrides)

## Advantages Over Manual Config

| Feature | Manual Config | OpenAPI |
|---------|--------------|---------|
| Endpoints | Manual mapping | Auto-discovered |
| Field mappings | Manual | From schema |
| Auth methods | Manual | From securitySchemes |
| Validation | Basic | Schema-based |
| Updates | Manual | Use provider's spec |
| Documentation | Separate | Built-in |

## Implementation Status

- [ ] OpenAPI parser integration
- [ ] Endpoint auto-discovery
- [ ] Schema-to-field mapping
- [ ] Authentication extraction
- [ ] Fallback to manual config
- [ ] Tests

## Future Enhancements

1. **Auto-download specs** - Fetch OpenAPI spec from provider's URL
2. **Spec validation** - Validate against OpenAPI schema
3. **Code generation** - Generate type-safe clients from spec
4. **Spec caching** - Cache parsed specs for performance

