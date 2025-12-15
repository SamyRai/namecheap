# GoDaddy OpenAPI Schema

## Source Information

- **Official Documentation**: https://developer.godaddy.com/doc/endpoint/domains
- **OpenAPI Spec URL**: https://developer.godaddy.com/doc/openapi.json
- **Last Updated**: 2024-11-22
- **Status**: Custom spec created from official API documentation

## Schema Details

- **OpenAPI Version**: 3.0.0
- **Base URL**: https://api.godaddy.com/v1
- **Authentication**: Bearer token (API key used as Bearer token)

## DNS Endpoints

- `GET /domains/{domain}/records` - List DNS records
- `PATCH /domains/{domain}/records` - Replace all DNS records
- `POST /domains/{domain}/records` - Add DNS records
- `GET /domains/{domain}/records/{type}/{name}` - Get specific record
- `PUT /domains/{domain}/records/{type}/{name}` - Update record
- `DELETE /domains/{domain}/records/{type}/{name}` - Delete record

## Field Mappings

- `name` → hostname
- `type` → record_type
- `data` → address
- `ttl` → ttl
- `priority` → mx_pref

## Notes

This is a custom OpenAPI spec created from GoDaddy's official API documentation. GoDaddy provides API documentation but the official OpenAPI spec may not be publicly available or may be incomplete.

## Updating

To update this spec:
1. Check GoDaddy's official API documentation
2. Verify endpoint changes
3. Update the OpenAPI spec accordingly
4. Test with the auto-discovery system

