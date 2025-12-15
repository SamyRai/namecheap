# Implemented DNS Providers

## Overview

This document lists all implemented DNS providers and how to use them.

## Available Providers

### ✅ Namecheap
- **Status**: Fully implemented
- **Type**: Custom (SOAP API via SDK)
- **Config**: Uses account configuration from main config
- **Usage**: Default provider, automatically registered

### ✅ Cloudflare
- **Status**: Implemented via REST infrastructure
- **Type**: REST API
- **Config**: `pkg/dns/provider/cloudflare/config.yaml`
- **Environment Variables**:
  - `CLOUDFLARE_API_KEY` - Your Cloudflare API key
  - `CLOUDFLARE_EMAIL` - Your Cloudflare email
- **Usage**: Automatically registered on startup if config exists

### ✅ GoDaddy
- **Status**: Implemented via REST infrastructure
- **Type**: REST API
- **Config**: `pkg/dns/provider/godaddy/config.yaml`
- **Environment Variables**:
  - `GODADDY_API_KEY` - Your GoDaddy API key
- **Usage**: Automatically registered on startup if config exists

### ✅ DigitalOcean
- **Status**: Implemented via REST infrastructure
- **Type**: REST API
- **Config**: `pkg/dns/provider/digitalocean/config.yaml`
- **Environment Variables**:
  - `DIGITALOCEAN_API_TOKEN` - Your DigitalOcean API token
- **Usage**: Automatically registered on startup if config exists

## Setup Instructions

### Cloudflare

1. Get your API credentials from: https://dash.cloudflare.com/profile/api-tokens
2. Set environment variables:
   ```bash
   export CLOUDFLARE_API_KEY="your-api-key"
   export CLOUDFLARE_EMAIL="your-email@example.com"
   ```
3. Ensure `pkg/dns/provider/cloudflare/config.yaml` exists (it's created automatically)
4. Provider will be available on next CLI run

### GoDaddy

1. Get your API key from: https://developer.godaddy.com/
2. Set environment variable:
   ```bash
   export GODADDY_API_KEY="your-api-key"
   ```
3. Ensure `pkg/dns/provider/godaddy/config.yaml` exists
4. Provider will be available on next CLI run

### DigitalOcean

1. Get your API token from: https://cloud.digitalocean.com/account/api/tokens
2. Set environment variable:
   ```bash
   export DIGITALOCEAN_API_TOKEN="your-token"
   ```
3. Ensure `pkg/dns/provider/digitalocean/config.yaml` exists
4. Provider will be available on next CLI run

## Using Providers

### List Available Providers

```bash
# Providers are automatically registered on startup
# You can check which providers are available programmatically
```

### Use Specific Provider

```bash
# Use provider by name
zonekit dns list example.com --provider cloudflare
```

### Provider Selection

The CLI will:
1. Use Namecheap by default (if account configured)
2. Allow selection of other providers via `--provider` flag
3. Auto-register all providers with valid configs on startup

## Adding More Providers

To add a new REST-based provider:

1. Create provider directory: `pkg/dns/provider/newprovider/`
2. Create `config.yaml` with provider configuration
3. Create `adapter.go` with Register() function (see Cloudflare example)
4. Add registration call in `cmd/root.go` initProviders()
5. Done! No code needed - just configuration.

## Provider Status Summary

| Provider | Status | Type | Config Required |
|----------|--------|------|----------------|
| Namecheap | ✅ Complete | Custom (SOAP) | Account config |
| Cloudflare | ✅ Complete | REST | config.yaml + env vars |
| GoDaddy | ✅ Complete | REST | config.yaml + env vars |
| DigitalOcean | ✅ Complete | REST | config.yaml + env vars |

## Next Providers to Add

Based on priority list:
- AWS Route 53 (may need custom implementation for AWS SDK)
- Google Cloud DNS
- Vercel
- Others as needed

## Notes

- All REST providers use the same generic infrastructure
- Providers are automatically discovered and registered
- Config files support environment variable substitution
- Binary size remains reasonable (~10-15MB with all providers)

