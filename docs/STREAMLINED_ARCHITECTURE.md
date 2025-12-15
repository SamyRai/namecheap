# Streamlined Provider Architecture

## Overview

After aligning with the OpenAPI-only approach, the codebase has been streamlined for simplicity and consistency.

## Key Changes

### ✅ OpenAPI-Only Approach

- **No manual configs** - All REST providers use OpenAPI specs
- **Auto-generated configs** - Provider configs generated from OpenAPI automatically
- **Minimal specs** - DNS-only OpenAPI specs (not full API specs)

### ✅ Code Cleanup

1. **Removed `builder/register.go`** - No longer needed (was for manual config loading)
2. **Simplified `config/config.go`** - Marked as deprecated/unused
3. **Updated auto-discovery** - OpenAPI-only, no fallback logic
4. **Updated comments** - Reflect OpenAPI-only approach

### ✅ Minimal OpenAPI Specs

| Provider | Spec Size | Lines | Status |
|----------|-----------|-------|--------|
| Cloudflare | 5.6KB | 228 | Minimal (DNS only) |
| GoDaddy | ~5KB | ~150 | Minimal (DNS only) |
| DigitalOcean | ~5KB | ~150 | Minimal (DNS only) |

**Before**: Cloudflare spec was 16MB (full API)
**After**: Cloudflare spec is 5.6KB (DNS only)

## Architecture

```
pkg/dns/provider/
├── autodiscover/     # Auto-discovers OpenAPI specs
├── openapi/          # OpenAPI parser & converter
├── builder/           # Builds providers from configs
├── rest/             # Generic REST provider
├── http/             # Generic HTTP client
├── auth/             # Authentication handlers
├── mapper/           # Field mapping utilities
│
├── cloudflare/
│   └── openapi.yaml  # Minimal DNS-only spec (5.6KB)
├── godaddy/
│   └── openapi.yaml  # Minimal DNS-only spec
├── digitalocean/
│   └── openapi.yaml  # Minimal DNS-only spec
└── namecheap/
    └── adapter.go    # Custom adapter (SOAP)
```

## Flow

1. **Startup** → `initProviders()` called
2. **Auto-Discovery** → Scans provider directories
3. **Find OpenAPI** → Looks for `openapi.yaml` files
4. **Parse Spec** → Extracts endpoints, auth, schemas
5. **Generate Config** → Creates `dnsprovider.Config` from spec
6. **Build Provider** → Creates REST provider from config
7. **Register** → Adds to provider registry

## Benefits

✅ **Simplified** - One approach for all REST providers
✅ **Smaller** - Minimal OpenAPI specs (5KB vs 16MB)
✅ **Faster** - Less code to parse and process
✅ **Cleaner** - No manual config files
✅ **Standardized** - All providers use same format

## Removed/Deprecated

- ❌ `builder/register.go` - Removed (no manual config loading)
- ⚠️ `config/config.go` - Deprecated (kept for potential future use)
- ❌ Manual `config.yaml` files - Not used (OpenAPI-only)

## Adding New Providers

Just create `openapi.yaml` in provider directory:

```bash
mkdir -p pkg/dns/provider/newprovider
# Create openapi.yaml with DNS endpoints
# Done! Auto-discovered on next startup
```

No code changes needed!

