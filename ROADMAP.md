# Roadmap

This document outlines the strategic vision and development milestones for ZoneKit.

## v2.0.0 (Current Stable)

- **Status:** Released
- **Focus:** Provider Contract Refactor, OpenAPI Support, Adapter Pattern.
- **Key Features:**
  - New `Provider` interface.
  - Conformance test harness.
  - OpenAPI-based provider generation.
  - Support for Namecheap, Cloudflare, DigitalOcean, GoDaddy.

## v2.1.0 (Upcoming)

- **Status:** Released
- **Focus:** Import/Export & Usability
- **Features:**
  - Full BIND zone file import support (`dns import`).
  - Improved zone file export (configurable nameservers).
  - Enhanced bulk operation error reporting.

## v2.2.0

- **Status:** Released
- **Focus:** Domain Lifecycle Management
- **Features:**
  - Domain registration support across providers.
  - Domain renewal management.
  - Whois privacy toggling.
  - Nameserver management unification.

## v2.3.0

- **Status:** Released
- **Focus:** Authentication & Security
- **Features:**
  - OAuth flow support for providers.
  - Secure credential storage improvements (system keyring integration).
  - Audit logging for all operations.

## v3.0.0

- **Focus:** Plugin Ecosystem
- **Timeline:** 2026
- **Features:**
  - Dynamic plugin loading (Go plugins or WASM).
  - Community plugin registry.
  - Enhanced plugin hooks (pre/post operation).
