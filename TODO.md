# ZoneKit TODO List

This list identifies gaps, technical debt, and areas for improvement in the codebase, prioritized by safety and production readiness.

## 🚨 High Priority (Safety & Correctness)

- [x] **Refactor `Service.BulkUpdate` Strategy**
  - **Issue**: Currently, `Service.BulkUpdate` builds a new record list and calls `SetRecords`. For providers like `RESTProvider` (which implements `SetRecords` via "delete all then create all"), this is **non-atomic and unsafe**. If creation fails, data is lost.
  - **Task**: Update `Service.BulkUpdate` to:
    1. Check `Provider.Capabilities()`.
    2. If the provider supports **Atomic Bulk Replace**, use `SetRecords`.
    3. Otherwise, orchestrate the update using granular `CreateRecord`, `UpdateRecord`, and `DeleteRecord` calls to minimize risk.

- [x] **Enhance `ProviderCapabilities`**
  - **Issue**: `CanBulkReplace` is ambiguous. It doesn't distinguish between a safe, atomic API call and a dangerous client-side loop.
  - **Task**: Add `IsBulkReplaceAtomic bool` to `ProviderCapabilities`.

- [x] **Fix `context.Context` Propagation**
  - **Issue**: `Service` methods (e.g., `GetRecords`) create a new `context.Background()` instead of accepting a context from the caller. This prevents cancellation and timeout propagation from the CLI or API layer.
  - **Task**: Update all `Service` methods to accept `ctx context.Context` as the first argument.

- [x] **Harden `RESTProvider` Error Handling**
  - **Issue**: While `BulkReplaceRecords` now checks errors, it's still a "stop the world" failure.
  - **Task**: Implemented granular updates to minimize data loss risk (stop on error instead of delete-all-then-fail). Rollback attempts are future work.

## 🧪 Medium Priority (Testing & QA)

- [x] **Expand Conformance Test Suite**
  - **Issue**: `pkg/dns/provider/conformance` was missing.
  - **Task**: Created `pkg/dns/provider/conformance` with a test suite covering `GetRecords`, `SetRecords`, `AddRecord`, `UpdateRecord`, and `DeleteRecord`. Added `MockProvider` for validating the suite.

- [ ] **Add Integration Tests**
  - **Issue**: Tests primarily rely on mocks.
  - **Task**: Add integration tests that spin up a local HTTP server (mocking Cloudflare/DigitalOcean APIs) to verify the full `RESTProvider` -> `Mapper` -> `HTTP` stack.

## 🧹 Low Priority (Cleanup & Features)

- [ ] **Implement `BatchUpdate` Interface**
  - **Issue**: Some providers support batch operations (e.g., "create 10 records") which is more efficient than 10 separate calls but less drastic than "replace zone".
  - **Task**: Add `BatchUpdate(ctx, operations)` to `Provider` interface.

- [ ] **Structured Logging**
  - **Issue**: Logging is likely ad-hoc (using `fmt` or basic `log`).
  - **Task**: Integrate a structured logger (like `log/slog`) to provide consistent, machine-readable logs for debugging production issues.

- [ ] **Configuration Validation**
  - **Issue**: Configuration loading could be stricter.
  - **Task**: Use a validation library to ensure all required fields (auth, endpoints) are present and well-formed at startup.
