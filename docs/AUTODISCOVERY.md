# Provider Auto-Discovery

## Overview

Providers are now **automatically discovered and registered** from subdirectories. No adapter files needed!

## How It Works

1. On startup, `initProviders()` calls `autodiscover.DiscoverAndRegister()`
2. The system scans `pkg/dns/provider/*/` directories
3. For each directory with a `config.yaml` file:
   - Loads the configuration
   - Builds the provider
   - Registers it automatically

## Adding a New Provider

### Before (with adapters):
1. Create directory
2. Create `config.yaml`
3. Create `adapter.go` with Register() function
4. Add import and call in `cmd/root.go`

### Now (auto-discovery):
1. Create directory
2. Create `config.yaml`
3. **Done!** Provider is automatically discovered

## Directory Structure

```
pkg/dns/provider/
├── cloudflare/
│   ├── config.yaml          # Provider auto-discovered from this
│   └── config.yaml.example
├── godaddy/
│   ├── config.yaml          # Provider auto-discovered from this
│   └── config.yaml.example
├── digitalocean/
│   ├── config.yaml          # Provider auto-discovered from this
│   └── config.yaml.example
└── namecheap/
    ├── adapter.go           # Custom provider (SOAP), needs explicit registration
    └── config.yaml.example
```

## Excluded Directories

The auto-discovery skips:
- Hidden directories (starting with `.`)
- Infrastructure directories: `auth`, `builder`, `config`, `http`, `mapper`, `rest`, `autodiscover`
- `namecheap` (registered separately via `namecheap.Register()`)

## Benefits

✅ **Zero boilerplate** - No adapter.go files needed
✅ **Automatic** - Just add config.yaml and it works
✅ **Simple** - Adding a provider = creating a directory + config file
✅ **Clean** - No code duplication

## Custom Providers

For providers that don't fit the REST pattern (like Namecheap with SOAP):
- Keep the `adapter.go` file
- Register manually in `cmd/root.go` if needed
- Or use `namecheap.Register()` pattern

## Example: Adding Vercel Provider

```bash
# 1. Create directory
mkdir -p pkg/dns/provider/vercel

# 2. Create config.yaml
cat > pkg/dns/provider/vercel/config.yaml <<EOF
name: vercel
display_name: Vercel
type: rest

auth:
  method: bearer
  credentials:
    token: "\${VERCEL_API_TOKEN}"

api:
  base_url: "https://api.vercel.com/v2"
  endpoints:
    get_records: "/domains/{domain}/records"
    create_record: "/domains/{domain}/records"
  # ... rest of config
EOF

# 3. Done! Provider is automatically available on next run
```

No code changes needed!

