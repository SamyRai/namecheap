# Provider Contract Refactor - Implementation Plan

## Priority Checklist (Top-Down)

### Phase 1: Foundation (P0 - Critical)

- [x] **1.1** Define explicit Provider interface with CRUD methods
  - [x] `Capabilities() ProviderCapabilities`
  - [x] `ListZones(ctx) ([]Zone, error)`
  - [x] `GetZone(ctx, domain) (*Zone, error)`
  - [x] `ListRecords(ctx, zoneID) ([]Record, error)`
  - [x] `CreateRecord(ctx, zoneID, record) (*Record, error)`
  - [x] `UpdateRecord(ctx, zoneID, recordID, record) (*Record, error)`
  - [x] `DeleteRecord(ctx, zoneID, recordID) error`
  - [x] `BulkReplaceRecords(ctx, zoneID, records) error`

- [x] **1.2** Define ProviderCapabilities struct
  - [x] `SupportsRecordID bool`
  - [x] `SupportsBulkReplace bool`
  - [x] `SupportsZoneDiscovery bool`
  - [x] `SupportedRecordTypes []string`
  - [x] `RequiresPlanApproval bool`

- [x] **1.3** Extend dnsrecord.Record model
  - [x] Add `ID string` (provider record ID)
  - [x] Add `Priority int` (MX, SRV)
  - [x] Add `Weight int` (SRV)
  - [x] Add `Port int` (SRV)
  - [x] Add `Target string` (SRV, MX)
  - [x] Add `Metadata map[string]interface{}` (provider-specific fields)
  - [x] Add `Raw interface{}` (original provider response)

- [x] **1.4** Build conformance test harness
  - [x] Create `pkg/dns/provider/conformance/` package
  - [x] Implement `RunConformanceTests(t *testing.T, factory ProviderFactory)`
  - [x] Add sanity checks (capabilities detection, zone listing)
  - [x] Add CRUD operation tests
  - [x] Add zone discovery tests
  - [x] Add record ID handling tests
  - [x] Add SRV record tests (added SRV checks in harness)
  - [x] Add metadata preservation tests (best-effort assertions added)
  - [x] Add mock provider implementation for testing

### Phase 2: Core Infrastructure (P0 - Critical)

- [x] **2.1** Refactor mapper to support extended Record model
  - [x] Update `FromProviderFormat` to handle ID, SRV fields, metadata
  - [x] Update `ToProviderFormat` to preserve all fields
  - [x] Add SRV-specific field mapping logic
  - [x] Add metadata pass-through logic
  - [x] Add unit tests for all mapping scenarios
  - [x] Add tests for nested JSON structures

- [x] **2.2** Enhance OpenAPI parser for advanced features
  - [x] Detect SRV-specific fields in schemas
  - [x] Detect record ID fields (common patterns: `id`, `record_id`, `recordId`)
  - [x] Detect list response wrappers (e.g., Cloudflare `result`)
  - [x] Detect zone endpoint patterns
  - [x] Add OpenAPI fixture tests (Cloudflare, GoDaddy, DigitalOcean)
  - [x] Improve endpoint heuristics for CRUD operations

- [x] **2.3** Refactor REST provider into full adapter
  - [x] Implement all Provider interface methods
  - [x] Add zone discovery logic (`getZoneID` helper)
  - [x] Implement delete-by-ID with placeholder replacement (`{record_id}`, `{id}`)
  - [x] Add capability detection from mappings
  - [x] Split responsibilities: HTTP client, mapper, orchestration
  - [x] Add comprehensive unit tests
  - [x] Add mock HTTP server tests for CRUD operations

### Phase 3: Provider Migrations (P1 - High)

- [x] **3.1** Cloudflare adapter (reference implementation)
  - [x] Implement Provider interface
  - [x] Use record IDs for CRUD operations
  - [x] Implement zone discovery using `/zones` endpoint
  - [x] Set capabilities: `SupportsRecordID=true`, `SupportsZoneDiscovery=true`
  - [x] Add provider-specific tests (added `pkg/dns/provider/cloudflare/adapter_test.go`)
  - [x] Run conformance tests
  - [ ] Document adapter pattern

- [x] **3.2** Namecheap adapter
  - [x] Implement Provider interface
  - [x] Keep SOAP client but wrap with new interface
  - [x] Implement `BulkReplaceRecords` (native Namecheap semantics)
  - [x] Set capabilities: `SupportsBulkReplace=true`
  - [x] Add adapter tests
  - [x] Run conformance tests (stub mode for bulk operations) (added `TestNamecheapConformance`)

- [x] **3.3** GoDaddy adapter
  - [x] Map OpenAPI spec to REST adapter
  - [x] Implement Provider interface via REST adapter
  - [x] Add zone discovery if supported
  - [x] Set appropriate capabilities
  - [x] Add OpenAPI mapping tests
  - [x] Run conformance tests (verified with adapter tests)

- [x] **3.4** DigitalOcean adapter
  - [x] Map OpenAPI spec to REST adapter
  - [x] Implement Provider interface via REST adapter
  - [x] Add zone discovery if supported
  - [x] Set appropriate capabilities
  - [x] Add OpenAPI mapping tests
  - [x] Run conformance tests (verified with adapter tests)

### Phase 4: Integration & Polish (P2 - Medium)

- [ ] **4.1** CI/CD Integration
  - [x] Add `make test-conformance` target
  - [x] Add GitHub Actions job to run conformance tests on PRs
  - [ ] Add optional integration tests (gated by secrets)
  - [ ] Add coverage reporting for provider packages

- [ ] **4.2** Documentation
  - [x] Document Provider interface contract
  - [x] Create provider development guide
  - [ ] Add OpenAPI cookbook (how to add new provider)
  - [x] Document conformance testing
  - [x] Add migration guide from old implementation
  - [x] Update README with multi-provider examples

- [ ] **4.3** Cleanup & Release
  - [x] Remove legacy/duplicate code
  - [x] Run `gofmt`, `go vet`, `golangci-lint` on all changes
  - [x] Fix critical linter warnings
  - [ ] Update CHANGELOG with breaking changes
  - [ ] Bump major version (v2.0.0 or similar)
  - [ ] Tag release

### Phase 5: Future Enhancements (P3 - Low Priority)

- [ ] **5.1** OAuth improvements
  - [ ] Implement refresh token flow
  - [ ] Implement client credentials flow
  - [ ] Document OAuth limitations per provider

- [ ] **5.2** Zone file import
  - [ ] Implement BIND zone file parser
  - [ ] Support A, AAAA, CNAME, MX, TXT, NS, SRV records
  - [ ] Add zone import tests

- [ ] **5.3** Domain management
  - [ ] Implement domain registration (where supported)
  - [ ] Implement domain renewal (where supported)
  - [ ] Add domain transfer support

---

## Context & Background

### Current State

- Basic multi-provider support exists via OpenAPI mapping and REST adapter
- Record ID mapping and delete-by-id implemented but not consistently applied
- Zone discovery implemented but needs per-provider testing
- No explicit Provider contract - implicit duck-typing
- Namecheap SOAP adapter exists but doesn't follow common pattern
- Conformance harness implemented and running (mock provider + provider-specific conformance tests)

### Goals

1. **Single Provider Contract**: Explicit interface all providers must implement
2. **Clean Adapters**: Each provider is a clean, testable adapter
3. **Conformance Testing**: Automated validation of provider behavior
4. **No Legacy Code**: Breaking changes allowed, clean slate
5. **Maintainable**: Clear separation of concerns, well-tested

### Non-Goals

- Backward compatibility (breaking changes allowed)
- Legacy Namecheap-only behavior preservation
- Supporting all possible DNS record types immediately

---

## Implementation Strategy

### Branch & PR Strategy

- **Single branch**: `refactor/provider-contract-v2`
- **Single PR**: All changes in one atomic PR at the end
- **Systematic approach**: Implement phase-by-phase, testing each phase before proceeding

### Testing Strategy

- **Unit tests**: For each component (mapper, parser, adapters)
- **Conformance tests**: Run against each provider implementation
- **Mock integration**: HTTP server mocks for REST providers
- **Real integration**: Optional, gated by secrets in CI

### Phase Dependencies

```
Phase 1 (Contract + Harness)
    ↓
Phase 2 (Infrastructure)
    ↓
Phase 3 (Provider Migrations)
    ↓
Phase 4 (CI + Docs + Cleanup)
    ↓
Phase 5 (Future Enhancements)
```

---

## Acceptance Criteria

### Phase 1 Complete When:

- [ ] Provider interface compiles and exports successfully
- [ ] Record model extended with all required fields
- [ ] Conformance harness runs against mock provider
- [ ] All tests pass

### Phase 2 Complete When:

- [ ] Mapper handles SRV + metadata roundtrips
- [ ] OpenAPI parser detects advanced features
- [ ] REST adapter implements full Provider interface
- [ ] REST adapter passes conformance tests
- [ ] All tests pass

### Phase 3 Complete When:

- [ ] All 4 providers implement Provider interface
- [ ] Each provider passes conformance tests
- [ ] Provider-specific tests added and passing
- [ ] All tests pass

### Phase 4 Complete When:

- [ ] CI runs conformance tests on PRs
- [ ] Documentation complete and reviewed
- [ ] Linter warnings addressed
- [ ] Release tagged and published

---

## Estimated Timeline

- Phase 1: 2-3 days
- Phase 2: 3-4 days
- Phase 3: 6-8 days (2 days per provider)
- Phase 4: 2-3 days
- **Total: 13-18 days**

---

## Breaking Changes (Document All)

### API Changes

- `Provider` interface: New explicit contract vs implicit duck-typing
- `dnsrecord.Record`: New fields (ID, SRV fields, Metadata, Raw)
- Mapper signatures: Updated to handle new fields
- Provider registry: Updated to require explicit capabilities

### Behavioral Changes

- Zone ID discovery: Now automatic for supporting providers
- Delete operations: Now use record ID when available
- SRV records: Now properly structured vs CNAME fallback
- Metadata: Now preserved through CRUD operations

### Migration Path

1. Update to new Provider interface
2. Update Record instantiation to include new fields
3. Update tests to use conformance harness
4. Update provider configs for new mapping fields

---

## Success Metrics

- [ ] All provider conformance tests passing
- [ ] Code coverage > 80% for provider packages
- [ ] Zero critical linter warnings
- [ ] Documentation complete
- [ ] CI green on all commits

---

## Notes

- No users currently exist, so breaking changes are acceptable
- Focus on clean, maintainable code over backward compatibility
- Test coverage is critical - no merging without tests
- Document all decisions and patterns for future providers
