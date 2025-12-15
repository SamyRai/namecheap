# Cloudflare OpenAPI Schema

## Source Information

- **Official Repository**: https://github.com/cloudflare/api-schemas
- **Direct Download URL**: https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml
- **Documentation**: https://developers.cloudflare.com/api/
- **Last Downloaded**: $(date +%Y-%m-%d)

## Schema Details

- **Size**: ~16MB (full Cloudflare API, includes all services)
- **OpenAPI Version**: 3.0+
- **Includes**: All Cloudflare API endpoints including DNS, CDN, Security, etc.

## DNS Endpoints

The schema includes DNS-related endpoints under:
- `/zones/{zone_id}/dns_records` - DNS record management operations

## Usage

This schema is automatically discovered by the auto-discovery system. The system will:
1. Parse the OpenAPI spec
2. Extract DNS-related endpoints
3. Extract authentication methods
4. Extract field mappings from schemas
5. Generate provider configuration automatically

## Updating

To update to the latest schema:

```bash
./scripts/download-openapi-schemas.sh
```

Or manually:

```bash
curl -L https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml \
  -o pkg/dns/provider/cloudflare/openapi.yaml
```

## Notes

- The full spec is large because it includes all Cloudflare services
- For DNS-only usage, you could extract just DNS paths (see script comments)
- The auto-discovery system handles the full spec efficiently

