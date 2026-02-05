# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - Unreleased

### Added
- **Provider Interface**: A new, explicit `Provider` interface (`pkg/dns/provider`) replacing the old implicit duck-typing.
- **Conformance Harness**: A comprehensive test suite (`pkg/dns/provider/conformance`) to validate provider implementations.
- **OpenAPI Support**: Automatic provider configuration and adapter generation from OpenAPI specifications (`pkg/dns/provider/openapi`).
- **Adapter Pattern**: Specialized adapters for Cloudflare, DigitalOcean, GoDaddy, and Namecheap.
- **Enhanced Record Model**: `dnsrecord.Record` now includes `ID`, `Priority`, `Weight`, `Port`, `Target`, and `Metadata` fields.
- **SRV Record Support**: Full support for SRV records across all providers (where applicable).
- **Zone Discovery**: Automatic zone ID discovery for providers that support it.
- **Capabilities**: Providers now expose their capabilities (e.g., `SupportsRecordID`, `SupportsBulkReplace`) via `Capabilities()`.

### Changed
- **Breaking**: The `Provider` interface has been completely redefined. Custom providers must be updated.
- **Breaking**: `dnsrecord.Record` fields have changed.
- **Refactor**: Generic `RESTProvider` logic has been separated from specific provider implementations.
- **Refactor**: `mapper` package has been rewritten to handle complex field mappings and nested JSON structures.

### Removed
- Legacy Namecheap SOAP-only implementation (replaced by adapter wrapping the SDK).
- Legacy configuration fields and automatic migration support.

### Migration
See `wiki/Migration-Guide.md` for detailed instructions on upgrading from v0.x to v2.0.0.
