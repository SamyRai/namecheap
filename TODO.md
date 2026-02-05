# TODO

This file tracks technical debt and upcoming tasks for ZoneKit.

## Priority 1 (Next Release)
- [ ] Implement `dns import` command (Zone file parsing logic in `cmd/dns.go`).
- [ ] Replace hardcoded nameservers (`ns1.namecheap.com`) in `dns export` command.
- [ ] Add integration tests for all providers (requires API credentials).

## Priority 2 (Features)
- [ ] Implement domain registration (`pkg/domain/service.go`).
- [ ] Implement domain renewal (`pkg/domain/service.go`).
- [ ] Add OAuth support for providers that support it (Google, etc.).

## Priority 3 (Refactoring)
- [ ] Refactor `pkg/plugin` to align with the new v2.0.0 architecture if needed.
- [ ] Improve error handling in the `client` package.

## Known Issues
- `dns import` is currently a placeholder and returns an error.
- Domain availability check is not supported by the current Namecheap SDK version.
