# DigitalOcean OpenAPI Schema

## Source Information

- **Official Repository**: https://github.com/digitalocean/openapi
- **Direct Download URL**: https://raw.githubusercontent.com/digitalocean/openapi/main/specification.yaml
- **Documentation**: https://docs.digitalocean.com/reference/api/api-reference/#tag/Domains
- **Last Updated**: 2024-11-22
- **Status**: Custom spec created from official API documentation

## Schema Details

- **OpenAPI Version**: 3.0.0
- **Base URL**: https://api.digitalocean.com/v2
- **Authentication**: Bearer token (API token)

## DNS Endpoints

- `GET /domains/{domain_name}/records` - List all DNS records
- `POST /domains/{domain_name}/records` - Create DNS record
- `GET /domains/{domain_name}/records/{record_id}` - Get specific record
- `PUT /domains/{domain_name}/records/{record_id}` - Update record
- `DELETE /domains/{domain_name}/records/{record_id}` - Delete record

## Field Mappings

- `name` → hostname
- `type` → record_type
- `data` → address
- `ttl` → ttl
- `priority` → mx_pref

## Response Structure

DigitalOcean wraps records in a `domain_records` object:
```json
{
  "domain_records": [
    {
      "id": 123,
      "type": "A",
      "name": "@",
      "data": "192.0.2.1",
      "ttl": 3600
    }
  ]
}
```

## Notes

This is a custom OpenAPI spec created from DigitalOcean's official API documentation. DigitalOcean has an official OpenAPI spec repository, but this minimal spec focuses on DNS operations only.

## Updating

To update this spec:
1. Check DigitalOcean's official OpenAPI repository
2. Extract DNS-related paths if needed
3. Update the spec accordingly
4. Test with the auto-discovery system

